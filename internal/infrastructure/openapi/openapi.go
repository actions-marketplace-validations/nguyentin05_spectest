// Package openapi parses OpenAPI 3.x documents, resolves local JSON Pointer
// references, and normalizes operations into the domain.API model.
//
// Swagger 2.0 and external references are rejected: the generator only supports
// self-contained OpenAPI 3.x inputs.
package openapi

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// spec is the typed subset of an OpenAPI document needed for normalization.
// Unused sections (info, servers, tags, ...) are intentionally ignored.
type spec struct {
	Paths    map[string]*pathItem   `yaml:"paths"`
	Security *[]securityRequirement `yaml:"security"`
}

// pathItem holds the path-level parameters and the supported HTTP operations.
type pathItem struct {
	Get        *operation   `yaml:"get"`
	Put        *operation   `yaml:"put"`
	Post       *operation   `yaml:"post"`
	Delete     *operation   `yaml:"delete"`
	Options    *operation   `yaml:"options"`
	Head       *operation   `yaml:"head"`
	Patch      *operation   `yaml:"patch"`
	Trace      *operation   `yaml:"trace"`
	Parameters []*parameter `yaml:"parameters"`
}

// operation is one HTTP operation in a path item.
type operation struct {
	Summary     string                 `yaml:"summary"`
	Description string                 `yaml:"description"`
	Parameters  []*parameter           `yaml:"parameters"`
	RequestBody *requestBody           `yaml:"requestBody"`
	Responses   map[string]*response   `yaml:"responses"`
	Security    *[]securityRequirement `yaml:"security"`
}

// parameter is a path, query, header, or cookie parameter. Schema is kept as a
// yaml.Node because the domain schema is a deliberately shallow subset.
type parameter struct {
	Name     string    `yaml:"name"`
	In       string    `yaml:"in"`
	Required bool      `yaml:"required"`
	Schema   yaml.Node `yaml:"schema"`
}

// requestBody is the request payload definition of an operation.
type requestBody struct {
	Required bool                  `yaml:"required"`
	Content  map[string]*mediaType `yaml:"content"`
}

// response is one status-code response. Only its media types matter here.
type response struct {
	Content map[string]*mediaType `yaml:"content"`
}

// mediaType holds the schema associated with one request or response media type.
type mediaType struct {
	Schema yaml.Node `yaml:"schema"`
}

// securityRequirement is one alternative in an OpenAPI security requirement
// list. Schemes are sorted by name and scopes by value so the representation is
// deterministic before Task 6 hashes it.
type securityRequirement struct {
	Schemes []securityScheme
}

// securityScheme is one named scheme inside a security requirement.
type securityScheme struct {
	Name   string
	Scopes []string
}

// UnmarshalYAML decodes an OpenAPI security requirement object, which is a
// mapping of scheme name to a list of scopes.
func (r *securityRequirement) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("security requirement must be a mapping")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name := strings.TrimSpace(node.Content[i].Value)
		var scopes []string
		if err := node.Content[i+1].Decode(&scopes); err != nil {
			return fmt.Errorf("security scheme %q scopes: %w", name, err)
		}
		sort.Strings(scopes)
		r.Schemes = append(r.Schemes, securityScheme{Name: name, Scopes: scopes})
	}
	sort.Slice(r.Schemes, func(i, j int) bool {
		return r.Schemes[i].Name < r.Schemes[j].Name
	})
	return nil
}

// securitySchemeNames returns the union of scheme names across all security
// alternatives, sorted. An empty result means the operation needs no auth.
func securitySchemeNames(requirements []securityRequirement) []string {
	seen := make(map[string]struct{})
	for _, requirement := range requirements {
		for _, scheme := range requirement.Schemes {
			if scheme.Name != "" {
				seen[scheme.Name] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// parseSpec validates the OpenAPI version, resolves local references, and
// decodes the paths consumed by normalization.
func parseSpec(raw []byte) (*spec, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	document := documentRoot(&root)
	if document == nil || document.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected an OpenAPI document mapping")
	}
	if err := validateVersion(document); err != nil {
		return nil, err
	}
	if err := resolveLocalRefs(&root); err != nil {
		return nil, err
	}

	var decoded spec
	if err := root.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode specification: %w", err)
	}
	if len(decoded.Paths) == 0 {
		return nil, fmt.Errorf("no paths defined")
	}
	return &decoded, nil
}

// validateVersion accepts only OpenAPI 3.x. Swagger 2.0 is rejected explicitly
// so a legacy document does not fall through to a confusing decode error.
func validateVersion(document *yaml.Node) error {
	if swagger := mapValue(document, "swagger"); swagger != nil {
		return fmt.Errorf("swagger %q is not supported; only OpenAPI 3.x is accepted", swagger.Value)
	}
	version := mapValue(document, "openapi")
	if version == nil || version.Kind != yaml.ScalarNode {
		return fmt.Errorf(`missing required "openapi: 3.x" version field`)
	}
	value := strings.TrimSpace(version.Value)
	if !isOpenAPI3(value) {
		return fmt.Errorf("unsupported openapi version %q; only 3.x is accepted", value)
	}
	return nil
}

// isOpenAPI3 reports whether value is a 3.x version such as "3.0.3" or "3.1.0".
func isOpenAPI3(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 2 || parts[0] != "3" {
		return false
	}
	for _, part := range parts[1:] {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// documentRoot unwraps a YAML document node, or returns nil for an empty input.
func documentRoot(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			return nil
		}
		return root.Content[0]
	}
	if root.Kind == 0 {
		return nil
	}
	return root
}

// mapValue returns the value node for key in a mapping node, or nil.
func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var found *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			found = node.Content[i+1]
		}
	}
	return found
}
