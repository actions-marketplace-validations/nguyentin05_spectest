package openapi

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func decodeResolved(t *testing.T, raw string, out any) {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatalf("yaml.Unmarshal fixture: %v", err)
	}
	if err := resolveLocalRefs(&root); err != nil {
		t.Fatalf("resolveLocalRefs() error = %v, want nil", err)
	}
	if err := root.Decode(out); err != nil {
		t.Fatalf("decode resolved node: %v", err)
	}
}

func TestResolveLocalRefs_UnescapesJSONPointerTokens(t *testing.T) {
	raw := `
components:
  schemas:
    Foo/Bar:
      type: string
    Tilde~Name:
      type: integer
target:
  slash:
    $ref: "#/components/schemas/Foo~1Bar"
  tilde:
    $ref: "#/components/schemas/Tilde~0Name"
`
	var out struct {
		Target map[string]struct {
			Type string `yaml:"type"`
			Ref  string `yaml:"$ref"`
		} `yaml:"target"`
	}
	decodeResolved(t, raw, &out)

	if got := out.Target["slash"].Type; got != "string" {
		t.Errorf("slash ref Type = %q, want string", got)
	}
	if got := out.Target["tilde"].Type; got != "integer" {
		t.Errorf("tilde ref Type = %q, want integer", got)
	}
	for name, value := range out.Target {
		if value.Ref != "" {
			t.Errorf("%s ref was not resolved: %q", name, value.Ref)
		}
	}
}

func TestResolveLocalRefs_RejectsMissingTarget(t *testing.T) {
	raw := `
target:
  $ref: "#/components/schemas/Missing"
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatalf("yaml.Unmarshal fixture: %v", err)
	}
	err := resolveLocalRefs(&root)
	if err == nil {
		t.Fatal("resolveLocalRefs() error = nil, want missing target error")
	}
	if !strings.Contains(err.Error(), "#/components/schemas/Missing") {
		t.Errorf("resolveLocalRefs() error = %v, want it to name the pointer", err)
	}
}

func TestResolveLocalRefs_RejectsExternalDirectly(t *testing.T) {
	for _, ref := range []string{"other.yaml#/components/schemas/User", "https://example.com/spec.yaml"} {
		t.Run(ref, func(t *testing.T) {
			raw := "target:\n  $ref: " + ref + "\n"
			var root yaml.Node
			if err := yaml.Unmarshal([]byte(raw), &root); err != nil {
				t.Fatalf("yaml.Unmarshal fixture: %v", err)
			}
			err := resolveLocalRefs(&root)
			if err == nil {
				t.Fatalf("resolveLocalRefs() error = nil, want external ref rejection")
			}
			if !strings.Contains(err.Error(), ref) {
				t.Errorf("resolveLocalRefs() error = %v, want it to name %q", err, ref)
			}
		})
	}
}

func TestNormalize_RefSiblingsAreIgnored(t *testing.T) {
	raw := `
openapi: 3.0.3
components:
  schemas:
    Base:
      type: string
paths:
  /values:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Base"
              type: object
              description: sibling fields are ignored for $ref
      responses:
        "200":
          description: ok
`
	apis := normalizeOK(t, "api", raw)
	if len(apis) != 1 {
		t.Fatalf("len(apis) = %d, want 1", len(apis))
	}
	if apis[0].Request == nil {
		t.Fatal("Request = nil")
	}
	if got := apis[0].Request.Schema.Type; got != "string" {
		t.Errorf("resolved schema Type = %q, want string from referenced schema", got)
	}
}
