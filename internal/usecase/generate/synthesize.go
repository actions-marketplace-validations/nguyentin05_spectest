package generate

import (
	"context"
	"fmt"
	"sort"

	"github.com/nguyentin05/spectest/internal/domain"
)

// synthesizedSetup is the LLM output for one setup call.
type synthesizedSetup struct {
	Vars  map[string]any `json:"vars,omitempty"`
	Steps []domain.Step  `json:"steps"`
}

// synthesizedScenario is the LLM output for one scenario call.
// Strict decode: a stray "vars" field is an error.
type synthesizedScenario struct {
	Steps []domain.Step `json:"steps"`
}

// synthesizeInitial produces the full suite by calling the LLM once per
// setup (in topo order) and once per scenario (sorted by ID).
func (g *Generator) synthesizeInitial(
	ctx context.Context, src *domain.SourceSet, plan *initialPlan,
) (*domain.Suite, error) {
	ordered, err := planSetupsTopo(plan.Setups)
	if err != nil {
		return nil, fmt.Errorf("topo setup order: %w", err)
	}

	srcByKey := make(map[string]domain.API, len(src.APIs))
	for _, a := range src.APIs {
		srcByKey[a.Key] = a
	}

	setups := make(map[string]domain.Setup, len(ordered))
	for _, ps := range ordered {
		setup, err := g.synthesizeSetup(ctx, srcByKey, plan, ps, setups)
		if err != nil {
			return nil, fmt.Errorf("synthesize setup %q: %w", ps.Name, err)
		}
		setups[ps.Name] = setup
	}

	sorted := append([]planScenario(nil), plan.Scenarios...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	scenarios := make([]domain.Scenario, 0, len(sorted))
	for _, ps := range sorted {
		sc, err := g.synthesizeScenario(ctx, srcByKey, plan, ps, setups)
		if err != nil {
			return nil, fmt.Errorf("synthesize scenario %q: %w", ps.ID, err)
		}
		scenarios = append(scenarios, sc)
	}

	return assembleSuite(src, plan, setups, scenarios)
}

// synthesizeSetup calls the LLM for one setup outline.
func (g *Generator) synthesizeSetup(
	ctx context.Context,
	srcByKey map[string]domain.API,
	plan *initialPlan,
	ps planSetup,
	done map[string]domain.Setup,
) (domain.Setup, error) {
	allowed, err := allowedAPIs(srcByKey, ps.APIs)
	if err != nil {
		return domain.Setup{}, err
	}

	deps := collectDependencySetups(ps.Use, done)

	system, user, err := buildSetupSynthPrompt(plan, ps, deps, allowed)
	if err != nil {
		return domain.Setup{}, fmt.Errorf("build prompt: %w", err)
	}

	out, err := callValidated(ctx, g.llm, "synthesize setup "+ps.Name, system, user,
		func(v *synthesizedSetup) error { return validateEntitySteps(ps.Name, v.Steps, allowed) })
	if err != nil {
		return domain.Setup{}, err
	}

	return domain.Setup{
		Use:   append([]string(nil), ps.Use...),
		Vars:  cloneAnyMap(out.Vars),
		Steps: out.Steps,
	}, nil
}

// synthesizeScenario calls the LLM for one scenario outline.
func (g *Generator) synthesizeScenario(
	ctx context.Context,
	srcByKey map[string]domain.API,
	plan *initialPlan,
	ps planScenario,
	setups map[string]domain.Setup,
) (domain.Scenario, error) {
	allowed, err := allowedAPIs(srcByKey, ps.APIs)
	if err != nil {
		return domain.Scenario{}, err
	}

	closure, err := setupClosure(ps.Setup, setups)
	if err != nil {
		return domain.Scenario{}, err
	}

	system, user, err := buildScenarioSynthPrompt(plan, ps, closure, allowed)
	if err != nil {
		return domain.Scenario{}, fmt.Errorf("build prompt: %w", err)
	}

	out, err := callValidated(ctx, g.llm, "synthesize scenario "+ps.ID, system, user,
		func(v *synthesizedScenario) error { return validateEntitySteps(ps.ID, v.Steps, allowed) })
	if err != nil {
		return domain.Scenario{}, err
	}

	return domain.Scenario{
		ID:     ps.ID,
		Name:   ps.Name,
		Setup:  append([]string(nil), ps.Setup...),
		Steps:  out.Steps,
		Locked: false,
		Skip:   "",
	}, nil
}

// validateEntitySteps performs shallow checks only. Full rule enforcement
// (variable resolution, body schema, status ranges) is deferred to
// validateSuite in the next step.
func validateEntitySteps(owner string, steps []domain.Step, allowed map[string]domain.API) error {
	if len(steps) == 0 {
		return fmt.Errorf("%s: steps must not be empty", owner)
	}
	for i, s := range steps {
		if s.Call == "" {
			return fmt.Errorf("%s: step %d: call is empty", owner, i)
		}
		if s.Expect <= 0 {
			return fmt.Errorf("%s: step %d: expect must be > 0", owner, i)
		}
		if _, found, err := matchAPI(allowed, s.Call); err != nil {
			return fmt.Errorf("%s: step %d: %w", owner, i, err)
		} else if !found {
			return fmt.Errorf("%s: step %d: call %q not in entity apis", owner, i, s.Call)
		}
	}
	return nil
}

// planSetupsTopo returns setups in dependency order, tie-broken by name.
// Cycles are rejected (validateInitialPlan should catch them first).
func planSetupsTopo(setups []planSetup) ([]planSetup, error) {
	byName := make(map[string]planSetup, len(setups))
	for _, s := range setups {
		byName[s.Name] = s
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var order []planSetup
	var visit func(name string) error
	visit = func(name string) error {
		switch color[name] {
		case gray:
			return fmt.Errorf("cycle at %q", name)
		case black:
			return nil
		}
		s, ok := byName[name]
		if !ok {
			return fmt.Errorf("missing setup %q", name)
		}
		color[name] = gray
		deps := append([]string(nil), s.Use...)
		sort.Strings(deps)
		for _, d := range deps {
			if err := visit(d); err != nil {
				return err
			}
		}
		color[name] = black
		order = append(order, s)
		return nil
	}
	names := make([]string, 0, len(setups))
	for _, s := range setups {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// allowedAPIs builds a map from the given key list, erroring if any key is
// missing from the full source map.
func allowedAPIs(srcByKey map[string]domain.API, keys []string) (map[string]domain.API, error) {
	out := make(map[string]domain.API, len(keys))
	for _, k := range keys {
		a, ok := srcByKey[k]
		if !ok {
			return nil, fmt.Errorf("api key %q not in source", k)
		}
		out[k] = a
	}
	return out, nil
}

// collectDependencySetups returns the closure of setups reachable via use,
// preserving topo order.
func collectDependencySetups(names []string, done map[string]domain.Setup) []dependencySetup {
	seen := map[string]struct{}{}
	var out []dependencySetup
	var visit func(n string)
	visit = func(n string) {
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		s, ok := done[n]
		if !ok {
			return
		}
		for _, u := range s.Use {
			visit(u)
		}
		out = append(out, dependencySetup{Name: n, Setup: s})
	}
	for _, n := range names {
		visit(n)
	}
	return out
}

// setupClosure returns the transitive closure of setups used by a scenario.
func setupClosure(names []string, setups map[string]domain.Setup) ([]dependencySetup, error) {
	seen := map[string]struct{}{}
	var out []dependencySetup
	var visit func(n string) error
	visit = func(n string) error {
		if _, ok := seen[n]; ok {
			return nil
		}
		s, ok := setups[n]
		if !ok {
			return fmt.Errorf("setup %q not found", n)
		}
		seen[n] = struct{}{}
		for _, u := range s.Use {
			if err := visit(u); err != nil {
				return err
			}
		}
		out = append(out, dependencySetup{Name: n, Setup: s})
		return nil
	}
	for _, n := range names {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// assembleSuite merges the plan and synthesized content into a domain.Suite.
func assembleSuite(
	src *domain.SourceSet,
	plan *initialPlan,
	setups map[string]domain.Setup,
	scenarios []domain.Scenario,
) (*domain.Suite, error) {
	apiKeys := make([]string, 0, len(src.APIs))
	for _, a := range src.APIs {
		apiKeys = append(apiKeys, a.Key)
	}
	sort.Strings(apiKeys)

	seed := make(map[string]domain.Seed, len(plan.Seed))
	for k, v := range plan.Seed {
		seed[k] = domain.Seed{Vars: cloneAnyMap(v.Vars)}
	}

	return &domain.Suite{
		Version:   1,
		Servers:   append([]string(nil), src.Servers...),
		APIs:      apiKeys,
		Analysis:  plan.Analysis,
		Seed:      seed,
		Setup:     setups,
		Scenarios: scenarios,
	}, nil
}

// cloneAnyMap deep-copies a map[string]any. Nested maps are cloned; slices are
// copied shallowly (sufficient for suite data).
func cloneAnyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneAny(v)
	}
	return out
}

func cloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneAnyMap(t)
	case []any:
		cp := make([]any, len(t))
		for i, e := range t {
			cp[i] = cloneAny(e)
		}
		return cp
	default:
		return v
	}
}

// dependencySetup carries a setup's name and body together for prompt payload.
type dependencySetup struct {
	Name  string
	Setup domain.Setup
}
