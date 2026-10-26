package generate

import (
	"fmt"
	"slices"
	"sort"
)

// topoOrder returns names with dependencies first. Ties are broken by name so
// the result is deterministic. deps returns a node's dependencies and whether
// the node exists. A cycle or a missing node is an error.
func topoOrder(names []string, deps func(name string) ([]string, bool)) ([]string, error) {
	const (
		unvisited = iota
		visiting
		done
	)
	state := make(map[string]int, len(names))
	order := make([]string, 0, len(names))

	var visit func(string) error
	visit = func(n string) error {
		switch state[n] {
		case visiting:
			return fmt.Errorf("setup graph has a cycle at %q", n)
		case done:
			return nil
		}
		ds, ok := deps(n)
		if !ok {
			return fmt.Errorf("setup %q not found", n)
		}
		state[n] = visiting
		ds = slices.Clone(ds)
		sort.Strings(ds)
		for _, d := range ds {
			if err := visit(d); err != nil {
				return err
			}
		}
		state[n] = done
		order = append(order, n)
		return nil
	}

	sorted := slices.Clone(names)
	sort.Strings(sorted)
	for _, n := range sorted {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return order, nil
}
