package openapi

import (
	"sort"
	"strconv"

	"github.com/nguyentin05/spectest/internal/domain"
)

// buildSignature builds the compact, human- and LLM-readable summary of an
// operation. The signature is derived from the same normalized data as the
// hash, but it never participates in hashing.
func buildSignature(op normalizedOperation) domain.APISignature {
	sig := domain.APISignature{
		Request:   make(map[string]string),
		Responses: responseCodes(op.Responses),
		Auth:      securitySchemeNames(op.Security),
	}

	// The body is flattened first so a parameter with the same name is
	// disambiguated with an `[in]` suffix.
	if op.Request != nil && op.Request.JSON {
		flattenSchema("", op.Request.Schema, sig.Request)
	}

	parameters := make([]domain.APIParameter, len(op.Parameters))
	copy(parameters, op.Parameters)
	sort.SliceStable(parameters, func(i, j int) bool {
		if parameters[i].In != parameters[j].In {
			return parameters[i].In < parameters[j].In
		}
		return parameters[i].Name < parameters[j].Name
	})
	for _, parameter := range parameters {
		key := parameter.Name
		if _, taken := sig.Request[key]; taken {
			key = parameter.Name + "[" + parameter.In + "]"
		}
		sig.Request[key] = parameterSignature(parameter)
	}

	return sig
}

// flattenSchema writes one entry per body property: `field` or dotted
// `parent.field` for nested objects, with a trailing `!` when the property is
// required by its parent. Arrays stop at "array" because their items have no
// name to extend the dotted path with; objects recurse and also keep their own
// entry so an optional object's required children are interpreted correctly.
func flattenSchema(prefix string, schema domain.Schema, out map[string]string) {
	required := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}

	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		property := schema.Properties[name]
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		value := schemaTypeLabel(property)
		if _, ok := required[name]; ok {
			value += "!"
		}
		out[key] = value
		if isObjectLike(property) {
			flattenSchema(key, property, out)
		}
	}
}

// parameterSignature maps a parameter location to its signature value. Path is
// always `path!`; query, header, and cookie use their `in` name verbatim.
func parameterSignature(parameter domain.APIParameter) string {
	if parameter.In == "path" {
		return "path!"
	}
	return parameter.In
}

// schemaTypeLabel picks the short type shown in the signature. A schema with no
// explicit type falls back to the shape implied by its properties or items.
func schemaTypeLabel(schema domain.Schema) string {
	if schema.Type != "" {
		return schema.Type
	}
	if len(schema.Properties) > 0 {
		return "object"
	}
	if schema.Items != nil {
		return "array"
	}
	if schema.Unsupported {
		return "unsupported"
	}
	return "unknown"
}

func isObjectLike(schema domain.Schema) bool {
	return schema.Type == "object" || (schema.Type == "" && len(schema.Properties) > 0)
}

// responseCodes returns the sorted, deduplicated integer status codes. Range
// codes such as `2XX` and `default` are not integers and are omitted.
func responseCodes(responses map[string]domain.Schema) []int {
	codes := make([]int, 0, len(responses))
	for code := range responses {
		if !allDigits(code) {
			continue
		}
		value, err := strconv.Atoi(code)
		if err != nil {
			continue
		}
		codes = append(codes, value)
	}
	sort.Ints(codes)

	unique := codes[:0]
	for _, value := range codes {
		if len(unique) > 0 && unique[len(unique)-1] == value {
			continue
		}
		unique = append(unique, value)
	}
	return unique
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
