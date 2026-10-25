package openapi

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// refResolver inlines local JSON Pointer references in a YAML node tree.
type refResolver struct {
	root *yaml.Node
}

// resolveLocalRefs replaces local `$ref` objects reachable from the document
// content with a deep copy of their target. External references are rejected
// everywhere, including unused components. Components themselves are inlined on
// demand from paths, which keeps a recursive component at its first cycle level.
//
// OpenAPI 3.0 ignores any sibling fields next to `$ref`; this implementation
// replaces the whole node, so siblings (description, nullable, ...) are dropped
// rather than merged.
//
// A reference cycle is not inlined forever. When a pointer is already on the
// active resolution stack, the `$ref` node is left in place; schemaToDomain later
// marks that schema Unsupported.
func resolveLocalRefs(root *yaml.Node) error {
	if root == nil {
		return nil
	}
	resolver := &refResolver{root: documentRoot(root)}
	if err := resolver.validate(root, make(map[*yaml.Node]bool)); err != nil {
		return err
	}
	return resolver.resolve(root, nil, make(map[*yaml.Node]bool))
}

// validate walks the whole document (including unused components) and rejects
// non-local or dangling references. It does not inline anything.
func (r *refResolver) validate(node *yaml.Node, seen map[*yaml.Node]bool) error {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		if node.Alias == nil {
			return nil
		}
		return r.validate(node.Alias, seen)
	}
	if seen[node] {
		return nil
	}
	seen[node] = true

	if node.Kind == yaml.MappingNode {
		if refNode := mapValue(node, "$ref"); refNode != nil {
			ref, err := localRef(refNode)
			if err != nil {
				return err
			}
			if _, err := r.lookup(ref); err != nil {
				return fmt.Errorf("resolve $ref %q: %w", ref, err)
			}
			return nil
		}
		for i := 1; i < len(node.Content); i += 2 {
			if err := r.validate(node.Content[i], seen); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range node.Content {
		if err := r.validate(child, seen); err != nil {
			return err
		}
	}
	return nil
}

// localRef validates a $ref scalar and returns its local pointer.
func localRef(refNode *yaml.Node) (string, error) {
	if refNode.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("$ref must be a string")
	}
	ref := strings.TrimSpace(refNode.Value)
	if !strings.HasPrefix(ref, "#/") {
		return "", fmt.Errorf(`unsupported $ref %q: only local references starting with "#/" are allowed`, ref)
	}
	if ref == "#/" {
		return "", fmt.Errorf("$ref %q has an empty JSON Pointer", ref)
	}
	return ref, nil
}

func (r *refResolver) resolve(node *yaml.Node, stack []string, seen map[*yaml.Node]bool) error {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		if node.Alias == nil {
			return nil
		}
		return r.resolve(node.Alias, stack, seen)
	}
	if seen[node] {
		return nil
	}

	if node.Kind == yaml.MappingNode {
		if refNode := mapValue(node, "$ref"); refNode != nil {
			ref, err := localRef(refNode)
			if err != nil {
				return err
			}
			for _, active := range stack {
				if active == ref {
					// Leave the node intact; this is the cycle marker consumed by
					// schemaToDomain.
					return nil
				}
			}
			target, err := r.lookup(ref)
			if err != nil {
				return fmt.Errorf("resolve $ref %q: %w", ref, err)
			}
			replacement := cloneNode(target)
			next := make([]string, 0, len(stack)+1)
			next = append(next, stack...)
			next = append(next, ref)
			if err := r.resolve(replacement, next, seen); err != nil {
				return err
			}
			*node = *replacement
			return nil
		}
	}

	seen[node] = true
	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range node.Content {
			if err := r.resolve(child, stack, seen); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		for i := 1; i < len(node.Content); i += 2 {
			// Components are only inlined on demand from paths. Pre-expanding
			// them would add an unnecessary level before a recursive schema is
			// marked as a cycle.
			if node == r.root && node.Content[i-1].Value == "components" {
				continue
			}
			if err := r.resolve(node.Content[i], stack, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// lookup resolves a local JSON Pointer of the form `#/a/b` against the document
// root, decoding `~1` to `/` and `~0` to `~`.
func (r *refResolver) lookup(ref string) (*yaml.Node, error) {
	if r.root == nil {
		return nil, fmt.Errorf("document has no root")
	}
	current := derefAlias(r.root)
	for _, rawToken := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		token := unescapePointerToken(rawToken)
		current = derefAlias(current)
		if current == nil {
			return nil, fmt.Errorf("pointer segment %q hits an empty alias", token)
		}
		switch current.Kind {
		case yaml.MappingNode:
			var next *yaml.Node
			for i := 0; i+1 < len(current.Content); i += 2 {
				if current.Content[i].Value == token {
					next = current.Content[i+1]
					break
				}
			}
			if next == nil {
				return nil, fmt.Errorf("pointer segment %q not found", token)
			}
			current = next
		case yaml.SequenceNode:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(current.Content) {
				return nil, fmt.Errorf("invalid sequence index %q", token)
			}
			current = current.Content[index]
		default:
			return nil, fmt.Errorf("pointer segment %q hits a non-container node", token)
		}
	}
	return current, nil
}

func derefAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

// unescapePointerToken applies RFC 6901 escaping in the required order.
func unescapePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~1", "/")
	return strings.ReplaceAll(token, "~0", "~")
}

// cloneNode deep-copies a YAML node so inlined references do not share mutable
// state with their component definition.
func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	clone := *node
	clone.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		clone.Content[i] = cloneNode(child)
	}
	return &clone
}
