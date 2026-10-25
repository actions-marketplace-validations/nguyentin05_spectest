package generate

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/nguyentin05/spectest/internal/domain"
)

// planSetup is the outline of a setup before steps are synthesized.
type planSetup struct {
	Name    string   `json:"name"`
	Use     []string `json:"use,omitempty"`
	Purpose string   `json:"purpose"`
	APIs    []string `json:"apis"`
}

// planScenario is the outline of a scenario before steps are synthesized.
type planScenario struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Setup   []string `json:"setup,omitempty"`
	Purpose string   `json:"purpose"`
	APIs    []string `json:"apis"`
}

// initialPlan is the LLM output for the initial planning stage.
type initialPlan struct {
	Analysis  string                 `json:"analysis"`
	Seed      map[string]domain.Seed `json:"seed"`
	Setups    []planSetup            `json:"setups"`
	Scenarios []planScenario         `json:"scenarios"`
}

var (
	idRegex   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	nameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// planInitial calls the LLM to produce the suite structure.
func (g *Generator) planInitial(ctx context.Context, src *domain.SourceSet) (*initialPlan, error) {
	if g.llm == nil {
		return nil, fmt.Errorf("llm service is not configured")
	}

	system, user, err := buildInitialPlanPrompt(src)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	plan, err := callValidated(ctx, g.llm, "plan initial", system, user,
		func(p *initialPlan) error { return validateInitialPlan(p, src) })
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// validateInitialPlan enforces the planning contract before we spend more
// LLM calls on synthesizing steps.
func validateInitialPlan(plan *initialPlan, src *domain.SourceSet) error {
	if strings.TrimSpace(plan.Analysis) == "" {
		return fmt.Errorf("analysis is empty")
	}
	if len(plan.Setups) == 0 {
		return fmt.Errorf("setups list is empty")
	}
	if len(plan.Scenarios) == 0 {
		return fmt.Errorf("scenarios list is empty")
	}

	// --- seed literal check ---
	for name, seed := range plan.Seed {
		if !nameRegex.MatchString(name) {
			return fmt.Errorf("seed name %q does not match %s", name, nameRegex)
		}
		for k, v := range seed.Vars {
			if s, ok := v.(string); ok && strings.Contains(s, "{{") {
				return fmt.Errorf("seed %q var %q must be literal, got template %q", name, k, s)
			}
		}
	}

	// --- setup checks ---
	setupNames := make(map[string]struct{}, len(plan.Setups))
	for _, s := range plan.Setups {
		if !nameRegex.MatchString(s.Name) {
			return fmt.Errorf("setup name %q does not match %s", s.Name, nameRegex)
		}
		if _, dup := setupNames[s.Name]; dup {
			return fmt.Errorf("duplicate setup name %q", s.Name)
		}
		setupNames[s.Name] = struct{}{}
		if len(s.APIs) == 0 {
			return fmt.Errorf("setup %q has no apis", s.Name)
		}
	}
	for _, s := range plan.Setups {
		for _, u := range s.Use {
			if _, ok := setupNames[u]; !ok {
				return fmt.Errorf("setup %q uses unknown setup %q", s.Name, u)
			}
		}
	}
	if err := checkSetupGraphAcyclic(plan.Setups); err != nil {
		return err
	}

	// --- scenario checks ---
	scenarioIDs := make(map[string]struct{}, len(plan.Scenarios))
	scenarioNames := make(map[string]struct{}, len(plan.Scenarios))
	for _, sc := range plan.Scenarios {
		if !idRegex.MatchString(sc.ID) {
			return fmt.Errorf("scenario id %q does not match %s", sc.ID, idRegex)
		}
		if _, dup := scenarioIDs[sc.ID]; dup {
			return fmt.Errorf("duplicate scenario id %q", sc.ID)
		}
		scenarioIDs[sc.ID] = struct{}{}
		if strings.TrimSpace(sc.Name) == "" {
			return fmt.Errorf("scenario %q has empty name", sc.ID)
		}
		if _, dup := scenarioNames[sc.Name]; dup {
			return fmt.Errorf("duplicate scenario name %q", sc.Name)
		}
		scenarioNames[sc.Name] = struct{}{}
		if len(sc.APIs) == 0 {
			return fmt.Errorf("scenario %q has no apis", sc.ID)
		}
		for _, u := range sc.Setup {
			if _, ok := setupNames[u]; !ok {
				return fmt.Errorf("scenario %q uses unknown setup %q", sc.ID, u)
			}
		}
	}

	// --- api key membership ---
	validAPIs := make(map[string]struct{}, len(src.APIs))
	for _, a := range src.APIs {
		validAPIs[a.Key] = struct{}{}
	}
	checkAPIs := func(owner string, keys []string) error {
		for _, k := range keys {
			if _, ok := validAPIs[k]; !ok {
				return fmt.Errorf("%s refers to unknown api key %q", owner, k)
			}
		}
		return nil
	}
	for _, s := range plan.Setups {
		if err := checkAPIs("setup "+s.Name, s.APIs); err != nil {
			return err
		}
	}
	for _, sc := range plan.Scenarios {
		if err := checkAPIs("scenario "+sc.ID, sc.APIs); err != nil {
			return err
		}
	}

	// --- coverage: every API must be covered by a scenario or a setup
	// that is reachable from at least one scenario. ---
	covered := make(map[string]struct{})
	for _, sc := range plan.Scenarios {
		for _, k := range sc.APIs {
			covered[k] = struct{}{}
		}
	}
	// collect setups used directly or transitively by any scenario
	usedSetups := make(map[string]struct{})
	var markUsed func(name string)
	markUsed = func(name string) {
		if _, done := usedSetups[name]; done {
			return
		}
		usedSetups[name] = struct{}{}
		for _, s := range plan.Setups {
			if s.Name == name {
				for _, u := range s.Use {
					markUsed(u)
				}
				for _, k := range s.APIs {
					covered[k] = struct{}{}
				}
			}
		}
	}
	for _, sc := range plan.Scenarios {
		for _, u := range sc.Setup {
			markUsed(u)
		}
	}

	var missing []string
	for k := range validAPIs {
		if _, ok := covered[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("plan does not cover these api keys: %s", strings.Join(missing, ", "))
	}
	return nil
}

func checkSetupGraphAcyclic(setups []planSetup) error {
	edges := make(map[string][]string, len(setups))
	for _, s := range setups {
		edges[s.Name] = s.Use
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var visit func(name string) error
	visit = func(name string) error {
		switch color[name] {
		case gray:
			return fmt.Errorf("setup graph has a cycle at %q", name)
		case black:
			return nil
		}
		color[name] = gray
		for _, u := range edges[name] {
			if err := visit(u); err != nil {
				return err
			}
		}
		color[name] = black
		return nil
	}
	for _, s := range setups {
		if err := visit(s.Name); err != nil {
			return err
		}
	}
	return nil
}
