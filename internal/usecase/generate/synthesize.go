package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/nguyentin05/spectest/internal/domain"
)

const (
	scenarioBatchMax   = 4
	scenarioBatchChars = 20_000
)

// synthesizedSetup is the LLM output for one setup call.
type synthesizedSetup struct {
	Vars  map[string]any `json:"vars,omitempty"`
	Steps []domain.Step  `json:"steps"`
}

type synthesizedScenario struct {
	ID    string        `json:"id"`
	Steps []domain.Step `json:"steps"`
}

type synthesizedBatch struct {
	Scenarios []synthesizedScenario `json:"scenarios"`
}

// dependencySetup carries a setup's name and body together for prompt payloads.
type dependencySetup struct {
	Name  string
	Setup domain.Setup
}

// promptInfo is the context shared by every synthesis prompt.
type promptInfo struct {
	Notes       []promptNote
	Assumptions []assumption
	Warnings    []string
}

// synthCtx holds the read-only state of the synthesis stages.
type synthCtx struct {
	g        *Generator
	apis     map[string]domain.API
	servers  map[string]struct{}
	plan     *fullPlan
	groups   []apiGroup
	seedBase map[string]struct{}
	bareIdx  map[string][]string
	info     promptInfo
}

func newSynthCtx(g *Generator, src *domain.SourceSet, plan *fullPlan, groups []apiGroup, info promptInfo) *synthCtx {
	apis := make(map[string]domain.API, len(src.APIs))
	for _, a := range src.APIs {
		apis[a.Key] = a
	}
	servers := make(map[string]struct{}, len(src.Servers))
	for _, s := range src.Servers {
		servers[s] = struct{}{}
	}
	seedBase, bareIdx := seedScope(plan.Seed)
	return &synthCtx{
		g: g, apis: apis, servers: servers, plan: plan, groups: groups,
		seedBase: seedBase, bareIdx: bareIdx, info: info,
	}
}

// parentScope is the seed scope plus the final scope of every used setup.
// Each setup's final scope already contains its own dependencies.
func (sc *synthCtx) parentScope(use []string, scopes map[string]map[string]struct{}) map[string]struct{} {
	scope := cloneScope(sc.seedBase)
	for _, u := range use {
		if s, ok := scopes[u]; ok {
			mergeScope(scope, s)
		}
	}
	return scope
}

// ============================================================================
// Setups — sequential, catalog frozen after each one
// ============================================================================

// synthesizeSetups writes every setup one at a time in dependency order, common
// setups first. After a setup succeeds its vars/steps/saves are frozen; later
// setups and all scenarios only read them.
func (sc *synthCtx) synthesizeSetups(
	ctx context.Context,
) (map[string]domain.Setup, map[string]map[string]struct{}, error) {
	byName := make(map[string]planSetup, len(sc.plan.Setups))
	names := make([]string, 0, len(sc.plan.Setups))
	for _, s := range sc.plan.Setups {
		byName[s.Name] = s
		names = append(names, s.Name)
	}
	order, err := topoOrder(names, func(n string) ([]string, bool) {
		s, ok := byName[n]
		return s.Use, ok
	})
	if err != nil {
		return nil, nil, fmt.Errorf("setup order: %w", err)
	}
	// Common setups only depend on common setups, so moving them first keeps
	// the order valid.
	sort.SliceStable(order, func(i, j int) bool {
		return byName[order[i]].Group == "" && byName[order[j]].Group != ""
	})

	setups := make(map[string]domain.Setup, len(order))
	scopes := make(map[string]map[string]struct{}, len(order))
	for _, name := range order {
		ps := byName[name]
		allowed, err := allowedAPIs(sc.apis, ps.APIs)
		if err != nil {
			return nil, nil, fmt.Errorf("setup %q: %w", name, err)
		}
		parent := sc.parentScope(ps.Use, scopes)
		deps := collectDependencySetups(ps.Use, setups)

		system, user, err := buildSetupPrompt(sc, ps, deps, allowed)
		if err != nil {
			return nil, nil, fmt.Errorf("setup %q: build prompt: %w", name, err)
		}
		out, err := callValidated(ctx, sc.g.llm, "synthesize setup "+name, system, user,
			func(v *synthesizedSetup) error {
				errs, _ := sc.checkSetup(ps, v, allowed, parent)
				return errors.Join(errs...)
			})
		if err != nil {
			return nil, nil, err
		}
		_, final := sc.checkSetup(ps, out, allowed, parent)

		setups[name] = domain.Setup{
			Use:   slices.Clone(ps.Use),
			Vars:  cloneVars(out.Vars),
			Steps: cloneSteps(out.Steps),
		}
		scopes[name] = final
	}
	return setups, scopes, nil
}

// checkSetup runs the full step validation on one synthesized setup and also
// requires that it calls every API it declared.
func (sc *synthCtx) checkSetup(
	ps planSetup, v *synthesizedSetup, allowed map[string]domain.API, parent map[string]struct{},
) ([]error, map[string]struct{}) {
	var errs []error
	for k, val := range v.Vars {
		errs = append(errs, checkTemplates(parent, sc.bareIdx,
			fmt.Sprintf("setup %q var %q", ps.Name, k), val)...)
	}
	stepScope := cloneScope(parent)
	for k := range v.Vars {
		stepScope[k] = struct{}{}
	}
	stepErrs, final := validateStepsDeep("setup "+ps.Name, v.Steps, allowed, sc.servers, stepScope, sc.bareIdx)
	errs = append(errs, stepErrs...)

	called := calledKeys(v.Steps, allowed)
	for _, k := range ps.APIs {
		if _, ok := called[k]; !ok {
			errs = append(errs, fmt.Errorf("setup %q must call api %q", ps.Name, k))
		}
	}
	return errs, final
}

// ============================================================================
// Scenarios — batched, bounded parallel
// ============================================================================

type scenarioJob struct {
	ps       planScenario
	allowed  map[string]domain.API
	closure  []dependencySetup
	parent   map[string]struct{}
	mustCall []string
}

func (sc *synthCtx) newJob(
	ps planScenario, setups map[string]domain.Setup, scopes map[string]map[string]struct{}, mustCall []string,
) (scenarioJob, error) {
	allowed, err := allowedAPIs(sc.apis, ps.APIs)
	if err != nil {
		return scenarioJob{}, fmt.Errorf("scenario %q: %w", ps.ID, err)
	}
	closure, err := setupClosure(ps.Setup, setups)
	if err != nil {
		return scenarioJob{}, fmt.Errorf("scenario %q: %w", ps.ID, err)
	}
	return scenarioJob{
		ps: ps, allowed: allowed, closure: closure,
		parent: sc.parentScope(ps.Setup, scopes), mustCall: mustCall,
	}, nil
}

func jobSize(j scenarioJob) int {
	b, _ := json.Marshal(newPromptScenarioJob(j))
	return len(b)
}

// makeBatches groups jobs group by group (jobs arrive already ordered), at most
// scenarioBatchMax per batch or scenarioBatchChars of estimated input.
func makeBatches(jobs []scenarioJob) [][]scenarioJob {
	var batches [][]scenarioJob
	var cur []scenarioJob
	size := 0
	for _, j := range jobs {
		js := jobSize(j)
		newGroup := len(cur) > 0 && cur[0].ps.Group != j.ps.Group
		if len(cur) > 0 && (newGroup || len(cur) >= scenarioBatchMax || size+js > scenarioBatchChars) {
			batches = append(batches, cur)
			cur, size = nil, 0
		}
		cur = append(cur, j)
		size += js
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

// orderedScenarios returns the plan's scenarios group by group, by ID inside a
// group, so batches are stable.
func (sc *synthCtx) orderedScenarios() []planScenario {
	rank := make(map[string]int, len(sc.groups))
	for i, g := range sc.groups {
		rank[g.ID] = i
	}
	out := slices.Clone(sc.plan.Scenarios)
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := rank[out[i].Group], rank[out[j].Group]; ri != rj {
			return ri < rj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// synthesizeScenarios writes the given scenarios in batches. Batches run in
// parallel, bounded by concurrency; the setup catalog is read-only here.
func (sc *synthCtx) synthesizeScenarios(
	ctx context.Context,
	concurrency int,
	list []planScenario,
	setups map[string]domain.Setup,
	scopes map[string]map[string]struct{},
	mustCall map[string][]string,
) ([]domain.Scenario, error) {
	jobs := make([]scenarioJob, 0, len(list))
	for _, ps := range list {
		j, err := sc.newJob(ps, setups, scopes, mustCall[ps.ID])
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	parts, err := runBounded(ctx, makeBatches(jobs), concurrency,
		func(ctx context.Context, _ int, batch []scenarioJob) ([]domain.Scenario, error) {
			res, err := sc.synthBatch(ctx, batch)
			if err == nil || len(batch) == 1 || ctx.Err() != nil {
				return res, err
			}
			// The batch failed even after its repair: retry each scenario on
			// its own so one bad scenario does not sink its neighbours.
			res = res[:0]
			for _, j := range batch {
				one, e := sc.synthBatch(ctx, []scenarioJob{j})
				if e != nil {
					return nil, errors.Join(err, e)
				}
				res = append(res, one...)
			}
			return res, nil
		})
	if err != nil {
		return nil, fmt.Errorf("synthesize scenarios: %w", err)
	}

	var out []domain.Scenario
	for _, p := range parts {
		out = append(out, p...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (sc *synthCtx) synthBatch(ctx context.Context, jobs []scenarioJob) ([]domain.Scenario, error) {
	system, user, err := buildScenarioBatchPrompt(sc, jobs)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}
	ids := make([]string, len(jobs))
	for i, j := range jobs {
		ids[i] = j.ps.ID
	}
	stage := "synthesize scenarios " + strings.Join(ids, ",")

	out, err := callValidated(ctx, sc.g.llm, stage, system, user,
		func(v *synthesizedBatch) error { return sc.checkBatch(v, jobs) })
	if err != nil {
		return nil, err
	}

	byID := make(map[string]synthesizedScenario, len(out.Scenarios))
	for _, s := range out.Scenarios {
		byID[s.ID] = s
	}
	res := make([]domain.Scenario, 0, len(jobs))
	for _, j := range jobs {
		res = append(res, domain.Scenario{
			ID:    j.ps.ID,
			Name:  j.ps.Name,
			Setup: slices.Clone(j.ps.Setup),
			Steps: cloneSteps(byID[j.ps.ID].Steps),
		})
	}
	return res, nil
}

func (sc *synthCtx) checkBatch(v *synthesizedBatch, jobs []scenarioJob) error {
	var errs []error
	want := make(map[string]struct{}, len(jobs))
	for _, j := range jobs {
		want[j.ps.ID] = struct{}{}
	}
	got := make(map[string]synthesizedScenario, len(v.Scenarios))
	for _, s := range v.Scenarios {
		if _, ok := want[s.ID]; !ok {
			errs = append(errs, fmt.Errorf("unexpected scenario id %q", s.ID))
			continue
		}
		if _, dup := got[s.ID]; dup {
			errs = append(errs, fmt.Errorf("duplicate scenario id %q", s.ID))
			continue
		}
		got[s.ID] = s
	}
	for _, j := range jobs {
		s, ok := got[j.ps.ID]
		if !ok {
			errs = append(errs, fmt.Errorf("missing scenario %q", j.ps.ID))
			continue
		}
		stepErrs, _ := validateStepsDeep("scenario "+j.ps.ID, s.Steps, j.allowed, sc.servers, j.parent, sc.bareIdx)
		errs = append(errs, stepErrs...)
		called := calledKeys(s.Steps, j.allowed)
		for _, k := range j.mustCall {
			if _, ok := called[k]; !ok {
				errs = append(errs, fmt.Errorf("scenario %q must call api %q", j.ps.ID, k))
			}
		}
	}
	return errors.Join(errs...)
}

// ============================================================================
// Coverage — repair the scenario that declared a missing API, once
// ============================================================================

// repairCoverage makes sure every API is called by some scenario or setup.
// Setups already call everything they declare (checkSetup), so a gap can only
// come from a scenario. Each such scenario is rewritten once with a must_call
// list. Nothing new is invented: if no scenario declared the missing API, or it
// is still missing afterwards, the run aborts.
func (sc *synthCtx) repairCoverage(
	ctx context.Context,
	concurrency int,
	scenarios []domain.Scenario,
	setups map[string]domain.Setup,
	scopes map[string]map[string]struct{},
) ([]domain.Scenario, error) {
	missing := sc.missingAPIs(scenarios, setups)
	if len(missing) == 0 {
		return scenarios, nil
	}

	mustCall := map[string][]string{}
	for _, key := range missing {
		owner := ""
		for _, ps := range sc.orderedScenarios() {
			if slices.Contains(ps.APIs, key) {
				owner = ps.ID
				break
			}
		}
		if owner == "" {
			return nil, fmt.Errorf("coverage: api %q is not called and no scenario declares it", key)
		}
		mustCall[owner] = append(mustCall[owner], key)
	}

	var redo []planScenario
	for _, ps := range sc.orderedScenarios() {
		if _, ok := mustCall[ps.ID]; ok {
			redo = append(redo, ps)
		}
	}
	repaired, err := sc.synthesizeScenarios(ctx, concurrency, redo, setups, scopes, mustCall)
	if err != nil {
		return nil, fmt.Errorf("coverage repair: %w", err)
	}

	byID := make(map[string]domain.Scenario, len(repaired))
	for _, s := range repaired {
		byID[s.ID] = s
	}
	out := slices.Clone(scenarios)
	for i, s := range out {
		if r, ok := byID[s.ID]; ok {
			out[i] = r
		}
	}

	if still := sc.missingAPIs(out, setups); len(still) > 0 {
		return nil, fmt.Errorf("coverage: apis still not called after repair: %s", strings.Join(still, ", "))
	}
	return out, nil
}

func (sc *synthCtx) missingAPIs(scenarios []domain.Scenario, setups map[string]domain.Setup) []string {
	covered := map[string]struct{}{}
	for _, s := range scenarios {
		for k := range calledKeys(s.Steps, sc.apis) {
			covered[k] = struct{}{}
		}
	}
	for _, s := range setups {
		for k := range calledKeys(s.Steps, sc.apis) {
			covered[k] = struct{}{}
		}
	}
	var missing []string
	for k := range sc.apis {
		if _, ok := covered[k]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

// ============================================================================
// Helpers
// ============================================================================

// calledKeys returns the API keys that steps call, resolved against lookup.
func calledKeys(steps []domain.Step, lookup map[string]domain.API) map[string]struct{} {
	out := map[string]struct{}{}
	for _, st := range steps {
		if api, found, err := matchAPI(lookup, st.Call); err == nil && found {
			out[api.Key] = struct{}{}
		}
	}
	return out
}

// allowedAPIs builds a map from keys, erroring when a key is not in the source.
func allowedAPIs(all map[string]domain.API, keys []string) (map[string]domain.API, error) {
	out := make(map[string]domain.API, len(keys))
	for _, k := range keys {
		a, ok := all[k]
		if !ok {
			return nil, fmt.Errorf("api key %q not in source", k)
		}
		out[k] = a
	}
	return out, nil
}

// collectDependencySetups returns the transitive closure of setups reachable
// through use, dependencies first.
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

// setupClosure is collectDependencySetups that fails on an unknown setup.
func setupClosure(names []string, setups map[string]domain.Setup) ([]dependencySetup, error) {
	for _, n := range names {
		if _, ok := setups[n]; !ok {
			return nil, fmt.Errorf("setup %q not found", n)
		}
	}
	return collectDependencySetups(names, setups), nil
}

// assembleSuite merges the plan and synthesized content into a domain.Suite.
func assembleSuite(
	src *domain.SourceSet,
	plan *fullPlan,
	setups map[string]domain.Setup,
	scenarios []domain.Scenario,
	analysis string,
) *domain.Suite {
	apiKeys := make([]string, 0, len(src.APIs))
	for _, a := range src.APIs {
		apiKeys = append(apiKeys, a.Key)
	}
	sort.Strings(apiKeys)

	seed := make(map[string]domain.Seed, len(plan.Seed))
	for k, v := range plan.Seed {
		seed[k] = domain.Seed{Vars: cloneVars(v.Vars)}
	}

	return &domain.Suite{
		Version:   1,
		Input:     src.Input,
		Servers:   slices.Clone(src.Servers),
		APIs:      apiKeys,
		Analysis:  analysis,
		Seed:      seed,
		Setup:     setups,
		Scenarios: scenarios,
	}
}
