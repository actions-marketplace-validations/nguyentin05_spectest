package generate

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/nguyentin05/spectest/internal/domain"
)

var (
	idRegex   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	nameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// planSetup is the outline of a setup before steps are synthesized. Group is
// the owning group ID for local setups and empty for common ones.
type planSetup struct {
	Name    string   `json:"name"`
	Use     []string `json:"use,omitempty"`
	Purpose string   `json:"purpose"`
	APIs    []string `json:"apis"`
	Group   string   `json:"-"`
}

// planScenario is the outline of a scenario before steps are synthesized.
type planScenario struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Setup   []string `json:"setup,omitempty"`
	Purpose string   `json:"purpose"`
	APIs    []string `json:"apis"`
	Group   string   `json:"-"`
}

// globalPlan is the output of the global stage: assumptions, seed and the
// setups shared by several groups. It never contains steps or analysis.
type globalPlan struct {
	Assumptions  []assumption           `json:"assumptions"`
	Seed         map[string]domain.Seed `json:"seed"`
	CommonSetups []planSetup            `json:"common_setups"`
}

// groupOutline is the output of the per-group outline stage.
type groupOutline struct {
	Setups    []planSetup    `json:"setups"`
	Scenarios []planScenario `json:"scenarios"`
}

// fullPlan is the merged structure handed to synthesis.
type fullPlan struct {
	Seed        map[string]domain.Seed
	Setups      []planSetup    // common + local, pruned, sorted by name
	Scenarios   []planScenario // sorted by ID
	Assumptions []assumption
	Warnings    []string
}

// planInput is everything the planning stages read.
type planInput struct {
	Src      *domain.SourceSet
	Groups   []apiGroup
	Notes    []note
	Warnings []sourceWarning
	reqNorm  string
	hasReq   bool
}

func newPlanInput(src *domain.SourceSet, groups []apiGroup, nr *notesResult) *planInput {
	in := &planInput{Src: src, Groups: groups}
	if nr != nil {
		in.Notes = nr.Notes
		in.Warnings = nr.SourceWarnings
	}
	if src.Requirements != nil {
		in.hasReq = true
		in.reqNorm = normalizeEvidence(src.Requirements.Raw)
	}
	return in
}

func (in *planInput) apiSet() map[string]struct{} {
	s := make(map[string]struct{}, len(in.Src.APIs))
	for _, a := range in.Src.APIs {
		s[a.Key] = struct{}{}
	}
	return s
}

func (in *planInput) warningMessages() []string {
	out := make([]string, 0, len(in.Warnings))
	for _, w := range in.Warnings {
		out = append(out, w.Message)
	}
	return out
}

// ============================================================================
// Global plan ("chỗ mơ hồ"): assumptions, seed, common setups
// ============================================================================

func (g *Generator) planGlobal(ctx context.Context, in *planInput) (*globalPlan, []string, error) {
	system, user, err := buildGlobalPrompt(in)
	if err != nil {
		return nil, nil, fmt.Errorf("build prompt: %w", err)
	}
	plan, err := callValidatedLast(ctx, g.llm, "plan global", system, user, defaultRepairs,
		func(p *globalPlan, last bool) error { return validateGlobalPlan(p, in, last) })
	if err != nil {
		return nil, nil, err
	}

	plan.Assumptions = cleanAssumptions(plan.Assumptions, in.hasReq)
	sort.SliceStable(plan.CommonSetups, func(i, j int) bool {
		return plan.CommonSetups[i].Name < plan.CommonSetups[j].Name
	})

	var warns []string
	warns = append(warns, seedNoteProblems(in.Notes, plan.Seed)...)
	return plan, warns, nil
}

func validateGlobalPlan(p *globalPlan, in *planInput, last bool) error {
	var errs []error
	errs = append(errs, validateAssumptions(p.Assumptions, in.reqNorm, in.hasReq)...)

	// seed: names, literal values, no variable name shared between seeds
	varOwner := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(p.Seed)) {
		if !nameRegex.MatchString(name) {
			errs = append(errs, fmt.Errorf("seed name %q does not match %s", name, nameRegex))
		}
		for _, k := range slices.Sorted(maps.Keys(p.Seed[name].Vars)) {
			if other, dup := varOwner[k]; dup {
				errs = append(errs, fmt.Errorf("seed var %q is defined in both %q and %q", k, other, name))
			}
			varOwner[k] = name
			walkStrings(p.Seed[name].Vars[k], func(s string) {
				if strings.Contains(s, "{{") {
					errs = append(errs, fmt.Errorf("seed %q var %q must be literal, got %q", name, k, s))
				}
			})
		}
	}

	// common setups
	apiSet := in.apiSet()
	commons := make(map[string]planSetup, len(p.CommonSetups))
	for _, s := range p.CommonSetups {
		if !nameRegex.MatchString(s.Name) {
			errs = append(errs, fmt.Errorf("common setup name %q does not match %s", s.Name, nameRegex))
		}
		if _, dup := commons[s.Name]; dup {
			errs = append(errs, fmt.Errorf("duplicate common setup %q", s.Name))
		}
		commons[s.Name] = s
		for _, g := range in.Groups {
			if strings.HasPrefix(s.Name, g.ID+"_") {
				errs = append(errs, fmt.Errorf(
					"common setup %q uses prefix %q reserved for local setups of that group", s.Name, g.ID+"_"))
			}
		}
		if len(s.APIs) == 0 {
			errs = append(errs, fmt.Errorf("common setup %q has no apis", s.Name))
		}
		for _, k := range s.APIs {
			if _, ok := apiSet[k]; !ok {
				errs = append(errs, fmt.Errorf("common setup %q refers to unknown api %q", s.Name, k))
			}
		}
	}
	for _, s := range p.CommonSetups {
		for _, u := range s.Use {
			if _, ok := commons[u]; !ok {
				errs = append(errs, fmt.Errorf("common setup %q uses %q, which is not a common setup", s.Name, u))
			}
		}
	}
	if len(errs) == 0 {
		names := slices.Collect(maps.Keys(commons))
		if _, err := topoOrder(names, func(n string) ([]string, bool) {
			s, ok := commons[n]
			return s.Use, ok
		}); err != nil {
			errs = append(errs, err)
		}
	}

	// Seed notes should be used. Strict while a repair is left, a warning after.
	if !last {
		for _, msg := range seedNoteProblems(in.Notes, p.Seed) {
			errs = append(errs, errors.New(msg))
		}
	}
	return errors.Join(errs...)
}

// seedNoteProblems reports seed notes none of whose values reached the seed.
func seedNoteProblems(notes []note, seed map[string]domain.Seed) []string {
	have := map[string]struct{}{}
	for _, s := range seed {
		for _, v := range s.Vars {
			have[normalizeEvidence(fmt.Sprint(v))] = struct{}{}
		}
	}
	var out []string
	for _, n := range notes {
		if n.Kind != "seed" {
			continue
		}
		used := false
		for _, v := range n.Values {
			if _, ok := have[normalizeEvidence(v)]; ok {
				used = true
				break
			}
		}
		if !used {
			out = append(out, fmt.Sprintf("seed note %q is not reflected in seed", truncate(n.Text, 60)))
		}
	}
	sort.Strings(out)
	return out
}

// ============================================================================
// Outline per group
// ============================================================================

type outlineCtx struct {
	in      *planInput
	global  *globalPlan
	commons map[string]planSetup
}

func newOutlineCtx(in *planInput, gp *globalPlan) *outlineCtx {
	c := &outlineCtx{in: in, global: gp, commons: map[string]planSetup{}}
	for _, s := range gp.CommonSetups {
		c.commons[s.Name] = s
	}
	return c
}

func (g *Generator) outlineGroup(
	ctx context.Context, c *outlineCtx, gr apiGroup, hints []string,
) (*groupOutline, error) {
	system, user, err := buildOutlinePrompt(c, gr, hints)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}
	out, err := callValidated(ctx, g.llm, "outline "+gr.ID, system, user,
		func(o *groupOutline) error { return c.validateGroup(o, gr) })
	if err != nil {
		return nil, err
	}
	for i := range out.Setups {
		out.Setups[i].Group = gr.ID
	}
	for i := range out.Scenarios {
		out.Scenarios[i].Group = gr.ID
	}
	return out, nil
}

func (c *outlineCtx) validateGroup(o *groupOutline, gr apiGroup) error {
	var errs []error
	groupAPIs := make(map[string]struct{}, len(gr.APIs))
	for _, a := range gr.APIs {
		groupAPIs[a.Key] = struct{}{}
	}

	local := make(map[string]planSetup, len(o.Setups))
	for _, s := range o.Setups {
		switch {
		case !nameRegex.MatchString(s.Name):
			errs = append(errs, fmt.Errorf("setup name %q does not match %s", s.Name, nameRegex))
		case setupOwner(s.Name, c.in.Groups) != gr.ID:
			errs = append(errs, fmt.Errorf("local setup %q must start with %q", s.Name, gr.ID+"_"))
		}
		if _, dup := local[s.Name]; dup {
			errs = append(errs, fmt.Errorf("duplicate setup %q", s.Name))
		}
		local[s.Name] = s
		if len(s.APIs) == 0 {
			errs = append(errs, fmt.Errorf("setup %q has no apis", s.Name))
		}
		for _, k := range s.APIs {
			if _, ok := groupAPIs[k]; !ok {
				errs = append(errs, fmt.Errorf(
					"setup %q refers to api %q outside group %q; cross-group flows must use a common setup", s.Name, k, gr.ID))
			}
		}
	}
	known := func(name string) bool {
		_, isLocal := local[name]
		_, isCommon := c.commons[name]
		return isLocal || isCommon
	}
	for _, s := range o.Setups {
		for _, u := range s.Use {
			if u == s.Name {
				errs = append(errs, fmt.Errorf("setup %q uses itself", s.Name))
			} else if !known(u) {
				errs = append(errs, fmt.Errorf("setup %q uses unknown setup %q", s.Name, u))
			}
		}
	}
	if len(errs) == 0 {
		if _, err := topoOrder(slices.Collect(maps.Keys(local)), func(n string) ([]string, bool) {
			if s, ok := local[n]; ok {
				return s.Use, true
			}
			_, isCommon := c.commons[n]
			return nil, isCommon
		}); err != nil {
			errs = append(errs, err)
		}
	}

	ids := map[string]struct{}{}
	covered := map[string]struct{}{}
	for _, s := range o.Setups {
		for _, k := range s.APIs {
			covered[k] = struct{}{}
		}
	}
	for _, s := range c.commons {
		for _, k := range s.APIs {
			covered[k] = struct{}{}
		}
	}
	for _, sc := range o.Scenarios {
		if !idRegex.MatchString(sc.ID) {
			errs = append(errs, fmt.Errorf("scenario id %q does not match %s", sc.ID, idRegex))
		}
		if _, dup := ids[sc.ID]; dup {
			errs = append(errs, fmt.Errorf("duplicate scenario id %q", sc.ID))
		}
		ids[sc.ID] = struct{}{}
		if strings.TrimSpace(sc.Name) == "" {
			errs = append(errs, fmt.Errorf("scenario %q has empty name", sc.ID))
		}
		if len(sc.APIs) == 0 {
			errs = append(errs, fmt.Errorf("scenario %q has no apis", sc.ID))
		}
		for _, k := range sc.APIs {
			if _, ok := groupAPIs[k]; !ok {
				errs = append(errs, fmt.Errorf("scenario %q refers to api %q outside group %q", sc.ID, k, gr.ID))
			}
			covered[k] = struct{}{}
		}
		for _, u := range sc.Setup {
			if !known(u) {
				errs = append(errs, fmt.Errorf("scenario %q uses unknown setup %q", sc.ID, u))
			}
		}
	}

	var missing []string
	for k := range groupAPIs {
		if _, ok := covered[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		errs = append(errs, fmt.Errorf("group %q does not cover: %s", gr.ID, strings.Join(missing, ", ")))
	}
	return errors.Join(errs...)
}

// merge combines per-group outlines. It returns either the merged plan or
// per-group issues (keyed by group index) to repair:
//
//  1. scenario IDs/names must be unique across groups;
//  2. setups no scenario can reach are pruned (and reported as warnings);
//  3. every API must be covered by a scenario or a reachable setup;
//  4. the suite must have at least one scenario.
func (c *outlineCtx) merge(outlines []*groupOutline) (*fullPlan, map[int][]string) {
	issues := map[int][]string{}
	var scenarios []planScenario
	var locals []planSetup
	seenID, seenName := map[string]struct{}{}, map[string]struct{}{}

	for gi, o := range outlines {
		scs := slices.Clone(o.Scenarios)
		sort.SliceStable(scs, func(i, j int) bool { return scs[i].ID < scs[j].ID })
		for _, s := range scs {
			if _, dup := seenID[s.ID]; dup {
				issues[gi] = append(issues[gi], fmt.Sprintf(
					"scenario id %q is already used by another group; choose a different id (prefix it with %q)", s.ID, c.in.Groups[gi].ID))
				continue
			}
			if _, dup := seenName[s.Name]; dup {
				issues[gi] = append(issues[gi], fmt.Sprintf(
					"scenario name %q is already used by another group; choose a different name", s.Name))
				continue
			}
			seenID[s.ID], seenName[s.Name] = struct{}{}, struct{}{}
			scenarios = append(scenarios, s)
		}
		locals = append(locals, o.Setups...)
	}
	if len(issues) > 0 {
		return nil, issues
	}

	if len(scenarios) == 0 {
		biggest := 0
		for i, g := range c.in.Groups {
			if len(g.APIs) > len(c.in.Groups[biggest].APIs) {
				biggest = i
			}
		}
		issues[biggest] = []string{"the suite has no scenario; add at least one smoke scenario to this group"}
		return nil, issues
	}

	byName := map[string]planSetup{}
	for _, s := range c.commons {
		byName[s.Name] = s
	}
	for _, s := range locals {
		byName[s.Name] = s
	}
	reach := map[string]struct{}{}
	var visit func(string)
	visit = func(n string) {
		if _, ok := reach[n]; ok {
			return
		}
		reach[n] = struct{}{}
		for _, u := range byName[n].Use {
			visit(u)
		}
	}
	for _, sc := range scenarios {
		for _, n := range sc.Setup {
			visit(n)
		}
	}

	var warns []string
	var kept []planSetup
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		if _, ok := reach[name]; ok {
			kept = append(kept, byName[name])
		} else {
			warns = append(warns, fmt.Sprintf("setup %q is not reachable from any scenario; pruned", name))
		}
	}

	covered := map[string]struct{}{}
	for _, sc := range scenarios {
		for _, k := range sc.APIs {
			covered[k] = struct{}{}
		}
	}
	for _, s := range kept {
		for _, k := range s.APIs {
			covered[k] = struct{}{}
		}
	}
	groupIdx := map[string]int{}
	for i, g := range c.in.Groups {
		groupIdx[g.ID] = i
	}
	owner := groupOfAPIs(c.in.Groups)
	for _, a := range c.in.Src.APIs {
		if _, ok := covered[a.Key]; !ok {
			gi := groupIdx[owner[a.Key]]
			issues[gi] = append(issues[gi], fmt.Sprintf(
				"api %q is not covered; a scenario of this group must exercise it", a.Key))
		}
	}
	if len(issues) > 0 {
		return nil, issues
	}

	return &fullPlan{
		Seed:        c.global.Seed,
		Setups:      kept,
		Scenarios:   scenarios, // already ordered by group, then ID
		Assumptions: c.global.Assumptions,
		Warnings:    warns,
	}, nil
}

// planOutline runs the outline stage for every group in parallel, merges the
// results and repairs each group at most once if the merge finds problems.
func (g *Generator) planOutline(ctx context.Context, c *outlineCtx, concurrency int) (*fullPlan, error) {
	outlines, err := runBounded(ctx, c.in.Groups, concurrency,
		func(ctx context.Context, _ int, gr apiGroup) (*groupOutline, error) {
			return g.outlineGroup(ctx, c, gr, nil)
		})
	if err != nil {
		return nil, fmt.Errorf("outline: %w", err)
	}

	for round := 0; ; round++ {
		plan, issues := c.merge(outlines)
		if len(issues) == 0 {
			sort.SliceStable(plan.Scenarios, func(i, j int) bool { return plan.Scenarios[i].ID < plan.Scenarios[j].ID })
			return plan, nil
		}
		if round >= 1 {
			return nil, fmt.Errorf("outline: unresolved after repair: %s", formatIssues(c.in.Groups, issues))
		}

		idxs := slices.Sorted(maps.Keys(issues))
		redone, err := runBounded(ctx, idxs, concurrency,
			func(ctx context.Context, _ int, gi int) (*groupOutline, error) {
				return g.outlineGroup(ctx, c, c.in.Groups[gi], issues[gi])
			})
		if err != nil {
			return nil, fmt.Errorf("outline repair: %w", err)
		}
		for k, gi := range idxs {
			outlines[gi] = redone[k]
		}
	}
}

func formatIssues(groups []apiGroup, issues map[int][]string) string {
	var parts []string
	for _, gi := range slices.Sorted(maps.Keys(issues)) {
		parts = append(parts, fmt.Sprintf("%s: %s", groups[gi].ID, strings.Join(issues[gi], "; ")))
	}
	return strings.Join(parts, " | ")
}
