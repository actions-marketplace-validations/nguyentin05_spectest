package openapi

import (
	"reflect"
	"testing"
)

// TestSig_FlattensBodyAndParameters covers the flattened request signature:
// dotted body fields, required markers, arrays stopping at "array", and one
// entry per parameter location.
func TestSig_FlattensBodyAndParameters(t *testing.T) {
	raw := `
openapi: 3.0.3
paths:
  /customers/{customerId}:
    parameters:
      - name: customerId
        in: path
        required: true
        schema:
          type: string
    post:
      parameters:
        - name: limit
          in: query
          required: true
          schema:
            type: integer
        - name: X-Trace
          in: header
          schema:
            type: string
        - name: session
          in: cookie
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name, customer]
              properties:
                name:
                  type: string
                age:
                  type: integer
                customer:
                  type: object
                  required: [name]
                  properties:
                    name:
                      type: string
                    tags:
                      type: array
                      items:
                        type: string
      responses:
        "200":
          description: ok
`
	api := findAPI(t, normalizeOK(t, "api", raw), "POST {api}/customers/{customerId}")

	want := map[string]string{
		"name":          "string!",
		"age":           "integer",
		"customer":      "object!",
		"customer.name": "string!",
		"customer.tags": "array",
		"customerId":    "path!",
		"limit":         "query",
		"X-Trace":       "header",
		"session":       "cookie",
	}
	if !reflect.DeepEqual(api.Sig.Request, want) {
		t.Errorf("Sig.Request = %#v, want %#v", api.Sig.Request, want)
	}
}

// TestSig_ResponsesOnlyIntegers proves that range codes and `default` are kept
// out of the integer response list.
func TestSig_ResponsesOnlyIntegers(t *testing.T) {
	raw := `
openapi: 3.0.3
paths:
  /users:
    get:
      responses:
        "404":
          description: missing
        "2XX":
          description: broad
        default:
          description: fallback
        "201":
          description: created
        "200":
          description: ok
`
	api := findAPI(t, normalizeOK(t, "api", raw), "GET {api}/users")

	want := []int{200, 201, 404}
	if !reflect.DeepEqual(api.Sig.Responses, want) {
		t.Errorf("Sig.Responses = %v, want %v", api.Sig.Responses, want)
	}
}

// TestSig_AuthUnionSorted proves Sig.Auth is the sorted union of scheme names
// across the operation's effective security requirements.
func TestSig_AuthUnionSorted(t *testing.T) {
	raw := `
openapi: 3.0.3
security:
  - apiKey: []
paths:
  /global:
    get:
      responses:
        "200":
          description: ok
  /public:
    get:
      security: []
      responses:
        "200":
          description: ok
  /override:
    get:
      security:
        - bearerAuth: []
      responses:
        "200":
          description: ok
  /alternatives:
    get:
      security:
        - oauth: [read]
        - apiKey: []
          bearerAuth: []
      responses:
        "200":
          description: ok
`
	apis := normalizeOK(t, "api", raw)

	cases := map[string][]string{
		"GET {api}/global":       {"apiKey"},
		"GET {api}/override":     {"bearerAuth"},
		"GET {api}/alternatives": {"apiKey", "bearerAuth", "oauth"},
	}
	for key, want := range cases {
		api := findAPI(t, apis, key)
		if !reflect.DeepEqual(api.Sig.Auth, want) {
			t.Errorf("%s Sig.Auth = %v, want %v", key, api.Sig.Auth, want)
		}
	}

	if public := findAPI(t, apis, "GET {api}/public"); len(public.Sig.Auth) != 0 {
		t.Errorf("GET {api}/public Sig.Auth = %v, want empty", public.Sig.Auth)
	}
}

// TestSig_NonJSONBodyHasNoBodyFields proves a non-JSON body contributes no
// request fields while parameters, responses, and auth are still summarized.
func TestSig_NonJSONBodyHasNoBodyFields(t *testing.T) {
	raw := `
openapi: 3.0.3
security:
  - apiKey: []
paths:
  /upload/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    post:
      parameters:
        - name: verbose
          in: query
          schema:
            type: boolean
      requestBody:
        content:
          text/plain:
            schema:
              type: object
              required: [payload]
              properties:
                payload:
                  type: string
      responses:
        "201":
          description: created
`
	api := findAPI(t, normalizeOK(t, "api", raw), "POST {api}/upload/{id}")

	if want := map[string]string{"id": "path!", "verbose": "query"}; !reflect.DeepEqual(api.Sig.Request, want) {
		t.Errorf("Sig.Request = %#v, want %#v", api.Sig.Request, want)
	}
	if want := []int{201}; !reflect.DeepEqual(api.Sig.Responses, want) {
		t.Errorf("Sig.Responses = %v, want %v", api.Sig.Responses, want)
	}
	if want := []string{"apiKey"}; !reflect.DeepEqual(api.Sig.Auth, want) {
		t.Errorf("Sig.Auth = %v, want %v", api.Sig.Auth, want)
	}
}

// TestSig_CollisionSuffix proves a parameter that shares its name with a body
// field is disambiguated with the `[in]` suffix and stays deterministic.
func TestSig_CollisionSuffix(t *testing.T) {
	raw := `
openapi: 3.0.3
paths:
  /things:
    post:
      parameters:
        - name: status
          in: query
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [status]
              properties:
                status:
                  type: string
      responses:
        "200":
          description: ok
`
	want := map[string]string{
		"status":        "string!",
		"status[query]": "query",
	}

	first := findAPI(t, normalizeOK(t, "api", raw), "POST {api}/things")
	if !reflect.DeepEqual(first.Sig.Request, want) {
		t.Fatalf("Sig.Request = %#v, want %#v", first.Sig.Request, want)
	}

	second := findAPI(t, normalizeOK(t, "api", raw), "POST {api}/things")
	if !reflect.DeepEqual(first.Sig.Request, second.Sig.Request) {
		t.Errorf("collision signature is not deterministic: %v != %v", first.Sig.Request, second.Sig.Request)
	}
}
