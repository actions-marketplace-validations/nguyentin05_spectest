package openapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/nguyentin05/spectest/internal/domain"
)

// hash12 returns the first 12 lowercase hex characters of the SHA-256 digest.
// Twelve characters is part of the track format version 1 contract, so changing
// it requires a migration.
func hash12(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// apiHash hashes the canonical JSON encoding of a normalized operation and
// returns the first 12 lowercase hex characters of the SHA-256 digest.
//
// Only semantic content participates: method, path, parameters, request body,
// responses, and security. Description, summary, tags, examples, operationId,
// and externalDocs are deliberately excluded, as is the server label (the same
// operation is the same API regardless of which server hosts it).
func apiHash(op normalizedOperation) string {
	data, err := json.Marshal(canonicalOperation(op))
	if err != nil {
		// canonicalOperation only produces structs, slices, maps, and scalars,
		// so a marshal failure means an impossible value reached the hasher.
		// Fail loudly rather than silently hash a partial operation.
		panic("openapi: marshal canonical operation: " + err.Error())
	}
	return hash12(data)
}

// canonicalOperation projects a normalized operation onto an order-independent
// JSON shape. Every map is materialized as a key-sorted slice (or relies on
// encoding/json's deterministic map key ordering), so two documents that differ
// only in YAML ordering hash identically.
func canonicalOperation(op normalizedOperation) any {
	return canonicalAPI{
		Method:     op.Method,
		Path:       op.Path,
		Parameters: canonicalParameters(op.Parameters),
		Request:    canonicalRequestOf(op.Request),
		Responses:  canonicalResponses(op.Responses),
		Security:   canonicalSecurity(op.Security),
	}
}

// canonicalAPI is the hash input for one operation. A missing request body is
// the only nullable block; empty parameters/responses/security encode as empty
// arrays so "absent" and "empty" never collide with distinct hashes.
type canonicalAPI struct {
	Method     string                         `json:"method"`
	Path       string                         `json:"path"`
	Parameters []canonicalParameter           `json:"parameters"`
	Request    *canonicalRequest              `json:"request,omitempty"`
	Responses  []canonicalResponse            `json:"responses"`
	Security   []canonicalSecurityRequirement `json:"security"`
}

type canonicalParameter struct {
	Name     string          `json:"name"`
	In       string          `json:"in"`
	Required bool            `json:"required"`
	Schema   canonicalSchema `json:"schema"`
}

type canonicalRequest struct {
	Required  bool            `json:"required"`
	MediaType string          `json:"media_type"`
	Schema    canonicalSchema `json:"schema"`
}

type canonicalResponse struct {
	Code   string          `json:"code"`
	Schema canonicalSchema `json:"schema"`
}

type canonicalSecurityRequirement struct {
	Schemes []canonicalSecurityScheme `json:"schemes"`
}

type canonicalSecurityScheme struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// canonicalSchema mirrors domain.Schema with a marker for constructs the domain
// model cannot express. The domain schema keeps only an Unsupported boolean, so
// a circular `$ref` cut point is represented by that stable marker rather than
// the literal pointer text.
type canonicalSchema struct {
	Type                 string                     `json:"type,omitempty"`
	Required             []string                   `json:"required,omitempty"`
	Properties           map[string]canonicalSchema `json:"properties,omitempty"`
	Items                *canonicalSchema           `json:"items,omitempty"`
	AdditionalProperties *bool                      `json:"additional_properties,omitempty"`
	Unsupported          bool                       `json:"unsupported,omitempty"`
}

// canonicalParameters sorts a copy by (in, name), matching the normalized
// parameter order but independent of the caller's slice order.
func canonicalParameters(parameters []domain.APIParameter) []canonicalParameter {
	sorted := make([]domain.APIParameter, len(parameters))
	copy(sorted, parameters)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].In != sorted[j].In {
			return sorted[i].In < sorted[j].In
		}
		return sorted[i].Name < sorted[j].Name
	})

	canonical := make([]canonicalParameter, 0, len(sorted))
	for _, parameter := range sorted {
		canonical = append(canonical, canonicalParameter{
			Name:     parameter.Name,
			In:       parameter.In,
			Required: parameter.Required,
			Schema:   canonicalSchemaOf(parameter.Schema),
		})
	}
	return canonical
}

func canonicalRequestOf(request *domain.RequestSpec) *canonicalRequest {
	if request == nil {
		return nil
	}
	return &canonicalRequest{
		Required:  request.Required,
		MediaType: request.MediaType,
		Schema:    canonicalSchemaOf(request.Schema),
	}
}

// canonicalResponses flattens the response map into a code-sorted slice. `2XX`
// and `default` are hashed like any other code; only the signature drops them.
func canonicalResponses(responses map[string]domain.Schema) []canonicalResponse {
	codes := make([]string, 0, len(responses))
	for code := range responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	canonical := make([]canonicalResponse, 0, len(codes))
	for _, code := range codes {
		canonical = append(canonical, canonicalResponse{
			Code:   code,
			Schema: canonicalSchemaOf(responses[code]),
		})
	}
	return canonical
}

// canonicalSecurity sorts the alternatives (OR semantics) and each
// requirement's schemes (AND semantics) so security ordering cannot affect the
// hash.
func canonicalSecurity(requirements []securityRequirement) []canonicalSecurityRequirement {
	canonical := make([]canonicalSecurityRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		schemes := make([]canonicalSecurityScheme, 0, len(requirement.Schemes))
		for _, scheme := range requirement.Schemes {
			scopes := make([]string, len(scheme.Scopes))
			copy(scopes, scheme.Scopes)
			sort.Strings(scopes)
			schemes = append(schemes, canonicalSecurityScheme{Name: scheme.Name, Scopes: scopes})
		}
		sort.SliceStable(schemes, func(i, j int) bool {
			return schemes[i].Name < schemes[j].Name
		})
		canonical = append(canonical, canonicalSecurityRequirement{Schemes: schemes})
	}
	sort.SliceStable(canonical, func(i, j int) bool {
		return securityKey(canonical[i]) < securityKey(canonical[j])
	})
	return canonical
}

// securityKey renders one requirement as a stable sort key.
func securityKey(requirement canonicalSecurityRequirement) string {
	key := ""
	for _, scheme := range requirement.Schemes {
		key += scheme.Name + "("
		for _, scope := range scheme.Scopes {
			key += scope + ","
		}
		key += ");"
	}
	return key
}

// canonicalSchemaOf returns a deep canonical copy with sorted required names.
// encoding/json sorts the property map keys while marshaling.
func canonicalSchemaOf(schema domain.Schema) canonicalSchema {
	canonical := canonicalSchema{
		Type:        schema.Type,
		Unsupported: schema.Unsupported,
	}
	if len(schema.Required) > 0 {
		canonical.Required = make([]string, len(schema.Required))
		copy(canonical.Required, schema.Required)
		sort.Strings(canonical.Required)
	}
	if len(schema.Properties) > 0 {
		canonical.Properties = make(map[string]canonicalSchema, len(schema.Properties))
		for name, property := range schema.Properties {
			canonical.Properties[name] = canonicalSchemaOf(property)
		}
	}
	if schema.Items != nil {
		items := canonicalSchemaOf(*schema.Items)
		canonical.Items = &items
	}
	if schema.AdditionalProperties != nil {
		allowed := *schema.AdditionalProperties
		canonical.AdditionalProperties = &allowed
	}
	return canonical
}
