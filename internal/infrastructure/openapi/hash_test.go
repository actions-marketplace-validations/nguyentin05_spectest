package openapi

import (
	"strings"
	"testing"
)

// hashOf normalizes a single-operation document and returns the API hash.
func hashOf(t *testing.T, raw string) string {
	t.Helper()
	apis := normalizeOK(t, "api", raw)
	if len(apis) != 1 {
		t.Fatalf("len(apis) = %d, want 1", len(apis))
	}
	if apis[0].Hash == "" {
		t.Fatalf("Hash is empty, want a canonical hash")
	}
	return apis[0].Hash
}

// TestAPIHash_IgnoresDescription proves that only semantic content participates
// in the hash: summary, description, tags, externalDocs, operationId, and
// examples are all ignored.
func TestAPIHash_IgnoresDescription(t *testing.T) {
	first := `
openapi: 3.0.3
paths:
  /users:
    post:
      operationId: createUser
      summary: Create a user
      description: A long description that must be ignored.
      tags: [users, docs]
      externalDocs:
        url: https://example.com/users
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  description: the user name
                  example: Ada
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: string
`
	second := `
openapi: 3.0.3
paths:
  /users:
    post:
      operationId: createUserV2
      summary: Register a user
      description: |
        A completely different description
        spanning several lines.
      tags: [accounts]
      externalDocs:
        url: https://example.org/register
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  description: displayed name
                  examples: [Bob]
                  example: Bob
      responses:
        "200":
          description: everything is fine
          content:
            application/json:
              schema:
                type: string
                example: done
`
	got, want := hashOf(t, first), hashOf(t, second)
	if got != want {
		t.Errorf("hash changed with only non-semantic edits: got %q, want %q", got, want)
	}
	if len(got) != 12 {
		t.Errorf("len(Hash) = %d, want 12", len(got))
	}
}

// TestAPIHash_CanonicalOrder proves the hash is independent of input ordering
// for parameters, response codes, schema properties, required lists, and
// security requirement alternatives.
func TestAPIHash_CanonicalOrder(t *testing.T) {
	first := `
openapi: 3.0.3
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    bearerAuth:
      type: http
      scheme: bearer
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
      - name: verbose
        in: query
        schema:
          type: boolean
    post:
      security:
        - apiKey: []
        - bearerAuth: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name, age]
              properties:
                name:
                  type: string
                age:
                  type: integer
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: string
        "404":
          description: missing
          content:
            application/json:
              schema:
                type: integer
`
	second := `
openapi: 3.0.3
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    bearerAuth:
      type: http
      scheme: bearer
paths:
  /users/{id}:
    parameters:
      - name: verbose
        in: query
        schema:
          type: boolean
      - name: id
        in: path
        required: true
        schema:
          type: string
    post:
      security:
        - bearerAuth: []
        - apiKey: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [age, name]
              properties:
                age:
                  type: integer
                name:
                  type: string
      responses:
        "404":
          description: missing
          content:
            application/json:
              schema:
                type: integer
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: string
`
	got, want := hashOf(t, first), hashOf(t, second)
	if got != want {
		t.Errorf("hash changed with reordered input: got %q, want %q", got, want)
	}
}

// semanticBase is a fully featured operation used to prove that each semantic
// field participates in the hash.
const semanticBase = `
openapi: 3.0.3
security:
  - apiKey: []
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    bearerAuth:
      type: http
      scheme: bearer
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: string
`

// TestAPIHash_SemanticChanges proves that changing any semantic part of the
// operation changes the hash.
func TestAPIHash_SemanticChanges(t *testing.T) {
	base := hashOf(t, semanticBase)

	cases := []struct {
		name   string
		mutate func(string) string
	}{
		{
			name: "schema property type",
			mutate: func(spec string) string {
				return strings.Replace(spec,
					"                name:\n                  type: string",
					"                name:\n                  type: integer", 1)
			},
		},
		{
			name: "schema property required",
			mutate: func(spec string) string {
				return strings.Replace(spec, "              required: [name]", "              required: []", 1)
			},
		},
		{
			name: "request media type",
			mutate: func(spec string) string {
				return strings.Replace(spec,
					"          application/json:\n            schema:\n              type: object",
					"          application/xml:\n            schema:\n              type: object", 1)
			},
		},
		{
			name: "response status",
			mutate: func(spec string) string {
				return strings.Replace(spec, `        "200":`, `        "201":`, 1)
			},
		},
		{
			name: "security scheme",
			mutate: func(spec string) string {
				return strings.Replace(spec, "  - apiKey: []", "  - bearerAuth: []", 1)
			},
		},
		{
			name: "path parameter required",
			mutate: func(spec string) string {
				return strings.Replace(spec, "        required: true", "        required: false", 1)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := tc.mutate(semanticBase)
			if mutated == semanticBase {
				t.Fatalf("mutation %q did not change the spec text", tc.name)
			}
			if got := hashOf(t, mutated); got == base {
				t.Errorf("hash unchanged after %s: %q", tc.name, got)
			}
		})
	}
}

// TestAPIHash_CircularRefStable proves a recursive schema hashes deterministically
// and without hanging.
func TestAPIHash_CircularRefStable(t *testing.T) {
	raw := `
openapi: 3.0.3
components:
  schemas:
    Node:
      type: object
      properties:
        value:
          type: string
        next:
          $ref: "#/components/schemas/Node"
paths:
  /nodes:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Node"
`
	first := hashOf(t, raw)
	second := hashOf(t, raw)
	if first != second {
		t.Errorf("hash is not stable across normalizations: %q != %q", first, second)
	}
	if len(first) != 12 || strings.ToLower(first) != first {
		t.Errorf("Hash = %q, want 12 lowercase hex characters", first)
	}
}
