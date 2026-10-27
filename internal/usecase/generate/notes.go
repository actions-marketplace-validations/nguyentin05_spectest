package generate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nguyentin05/spectest/internal/domain"
)

// ============================================================================
// Chunking
// ============================================================================

const chunkMaxRunes = 12_000

type reqChunk struct {
	Index int
	Text  string
	norm  string // normalizeEvidence(Text), used for evidence checks
}

var blankLineRe = regexp.MustCompile(`\n[ \t]*\n`)

// chunkRequirements splits raw requirements into chunks of at most
// chunkMaxRunes runes. Paragraphs are packed greedily, a Markdown heading
// prefers to start a new chunk once the current one is half full, and adjacent
// chunks overlap by one paragraph so no sentence is lost at a boundary. It is
// deterministic and never calls the LLM.
func chunkRequirements(raw string) []reqChunk {
	text := strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")

	var paras []string
	for _, p := range blankLineRe.Split(text, -1) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if utf8.RuneCountInString(p) > chunkMaxRunes {
			paras = append(paras, splitLong(p, chunkMaxRunes)...)
		} else {
			paras = append(paras, p)
		}
	}

	var chunks []reqChunk
	emit := func(ps []string) {
		t := strings.Join(ps, "\n\n")
		chunks = append(chunks, reqChunk{Index: len(chunks), Text: t, norm: normalizeEvidence(t)})
	}

	var cur []string
	curLen := 0
	for _, p := range paras {
		pl := utf8.RuneCountInString(p)
		overflow := curLen+2+pl > chunkMaxRunes
		headingBreak := strings.HasPrefix(p, "#") && curLen > chunkMaxRunes/2
		if len(cur) > 0 && (overflow || headingBreak) {
			emit(cur)
			last := cur[len(cur)-1]
			cur, curLen = []string{last}, utf8.RuneCountInString(last)
			if curLen+2+pl > chunkMaxRunes {
				cur, curLen = nil, 0
			}
		}
		if len(cur) > 0 {
			curLen += 2
		}
		cur = append(cur, p)
		curLen += pl
	}
	if len(cur) > 0 {
		emit(cur)
	}
	return chunks
}

func isSentenceEnd(r []rune, i int) bool {
	switch r[i] {
	case '\n', '。':
		return true
	case '.', '!', '?':
		return i+1 < len(r) && unicode.IsSpace(r[i+1])
	}
	return false
}

// splitLong cuts an oversized paragraph at the last sentence boundary in the
// second half of the window, or hard-cuts when there is none.
func splitLong(p string, limit int) []string {
	r := []rune(p)
	var out []string
	for len(r) > limit {
		cut := limit
		for i := limit - 1; i > limit/2; i-- {
			if isSentenceEnd(r, i) {
				cut = i + 1
				break
			}
		}
		if s := strings.TrimSpace(string(r[:cut])); s != "" {
			out = append(out, s)
		}
		r = r[cut:]
	}
	if s := strings.TrimSpace(string(r)); s != "" {
		out = append(out, s)
	}
	return out
}

// ============================================================================
// Contract
// ============================================================================

type note struct {
	Kind     string            `json:"kind"`
	Text     string            `json:"text"`
	Evidence string            `json:"evidence"`
	About    []string          `json:"about"`
	Values   map[string]string `json:"values"`
}

type mentionItem struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Service  string `json:"service"`
	Evidence string `json:"evidence"`
}

type notesOut struct {
	Notes       []note        `json:"notes"`
	APIMentions []mentionItem `json:"api_mentions"`
}

var noteKinds = map[string]struct{}{
	"seed": {}, "relation": {}, "rule": {}, "constraint": {}, "info": {},
}

// sourceWarning is a requirements-vs-OpenAPI warning. Line is rendered into
// the code-managed warnings block of analysis; Message goes to the CLI summary.
type sourceWarning struct {
	Code    string
	Key     string
	Line    string
	Message string
}

type chunkNotes struct {
	Notes    []note
	Mentions []domain.APIMention
	Warnings []string
}

// notesResult is the merged output of the notes stage.
type notesResult struct {
	Notes          []note
	Mentions       []domain.APIMention // with Known set
	SourceWarnings []sourceWarning
	Warnings       []string
}

// ============================================================================
// Validation
// ============================================================================

// noteProblems lists why a note is not trustworthy. Empty means valid.
func noteProblems(n note, chunkNorm string) []string {
	var p []string
	if !containsEvidence(chunkNorm, n.Text) {
		p = append(p, "text is not verbatim from the chunk")
	}
	if !containsEvidence(chunkNorm, n.Evidence) {
		p = append(p, "evidence is not verbatim from the chunk")
	}
	if n.Kind == "seed" {
		if len(n.Values) == 0 {
			p = append(p, "seed note has no values")
		}
		tn, en := normalizeEvidence(n.Text), normalizeEvidence(n.Evidence)
		for k, v := range n.Values {
			vn := normalizeEvidence(v)
			if strings.TrimSpace(k) == "" || vn == "" {
				p = append(p, "seed value has an empty name or value")
				continue
			}
			if !strings.Contains(tn, vn) && !strings.Contains(en, vn) {
				p = append(p, fmt.Sprintf("seed value %q does not appear in text or evidence", v))
			}
		}
	}
	return p
}

// validateNotesOut is strict while repairs remain. On the last attempt only
// problems that cannot be tolerated (bad kind, invalid seed note) still fail;
// everything else is dropped later by sanitizeChunkNotes.
func validateNotesOut(out *notesOut, chunkNorm string, last bool) error {
	var errs []error
	for i, n := range out.Notes {
		if _, ok := noteKinds[n.Kind]; !ok {
			errs = append(errs, fmt.Errorf("notes[%d]: invalid kind %q", i, n.Kind))
			continue
		}
		if n.Kind != "seed" && last {
			continue
		}
		for _, p := range noteProblems(n, chunkNorm) {
			errs = append(errs, fmt.Errorf("notes[%d] (%s): %s", i, n.Kind, p))
		}
	}
	if !last {
		for i, m := range out.APIMentions {
			if !containsEvidence(chunkNorm, m.Evidence) {
				errs = append(errs, fmt.Errorf("api_mentions[%d]: evidence is not verbatim from the chunk", i))
			}
		}
	}
	return errors.Join(errs...)
}

// sanitizeChunkNotes drops what failed validation on the last attempt and
// cleans what is kept: unknown "about" names are removed, values on non-seed
// notes are ignored.
func sanitizeChunkNotes(out *notesOut, chunkNorm string, resources map[string]struct{}) chunkNotes {
	var res chunkNotes
	for _, n := range out.Notes {
		if probs := noteProblems(n, chunkNorm); len(probs) > 0 {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("dropped %s note %q: %s", n.Kind, truncate(n.Text, 60), probs[0]))
			continue
		}
		clean := note{
			Kind:     n.Kind,
			Text:     normalizeEvidence(n.Text),
			Evidence: normalizeEvidence(n.Evidence),
		}
		for _, a := range n.About {
			a = strings.ToLower(strings.TrimSpace(a))
			if _, ok := resources[a]; ok {
				clean.About = append(clean.About, a)
			} else if a != "" {
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("note %q: unknown resource %q ignored", truncate(n.Text, 40), a))
			}
		}
		clean.About = unionSorted(clean.About, nil)
		if n.Kind == "seed" {
			clean.Values = n.Values
		} else if len(n.Values) > 0 {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s note %q: values ignored (only seed notes carry values)", n.Kind, truncate(n.Text, 40)))
		}
		res.Notes = append(res.Notes, clean)
	}
	for _, m := range out.APIMentions {
		if !containsEvidence(chunkNorm, m.Evidence) {
			res.Warnings = append(res.Warnings, "dropped api mention: evidence is not verbatim")
			continue
		}
		res.Mentions = append(res.Mentions, domain.APIMention{
			Method:   m.Method,
			Path:     m.Path,
			Service:  m.Service,
			Evidence: normalizeEvidence(m.Evidence),
		})
	}
	return res
}

// ============================================================================
// Stage
// ============================================================================

// extractNotes runs the notes stage over every chunk (bounded parallel) and
// merges the results deterministically. Only used when requirements exist.
func (g *Generator) extractNotes(
	ctx context.Context, src *domain.SourceSet, groups []apiGroup, concurrency int,
) (*notesResult, error) {
	chunks := chunkRequirements(src.Requirements.Raw)
	resources := resourceList(src, groups)
	resSet := make(map[string]struct{}, len(resources))
	for _, r := range resources {
		resSet[r] = struct{}{}
	}
	apis := promptAPIsLite(src.APIs)

	parts, err := runBounded(ctx, chunks, concurrency,
		func(ctx context.Context, _ int, ch reqChunk) (chunkNotes, error) {
			system, user, err := buildNotesPrompt(src.Servers, resources, apis, ch)
			if err != nil {
				return chunkNotes{}, fmt.Errorf("build prompt: %w", err)
			}
			stage := fmt.Sprintf("notes chunk %d", ch.Index)
			out, err := callValidatedLast(ctx, g.llm, stage, system, user, 2,
				func(v *notesOut, last bool) error { return validateNotesOut(v, ch.norm, last) })
			if err != nil {
				return chunkNotes{}, err
			}
			return sanitizeChunkNotes(out, ch.norm, resSet), nil
		})
	if err != nil {
		return nil, fmt.Errorf("notes: %w", err)
	}

	notes, mentions, warns := mergeChunkNotes(parts)
	matched, sw := matchMentions(mentions, src.APIs)
	return &notesResult{Notes: notes, Mentions: matched, SourceWarnings: sw, Warnings: warns}, nil
}

// mergeChunkNotes dedupes by (kind, normalized text), merging about/values.
// Equal text with a different kind is kept as two notes.
func mergeChunkNotes(parts []chunkNotes) ([]note, []domain.APIMention, []string) {
	var (
		notes    []note
		mentions []domain.APIMention
		warns    []string
	)
	idx := map[string]int{}
	seenMention := map[string]struct{}{}

	for _, p := range parts {
		warns = append(warns, p.Warnings...)
		for _, n := range p.Notes {
			key := n.Kind + "\x00" + normalizeEvidence(n.Text)
			i, ok := idx[key]
			if !ok {
				idx[key] = len(notes)
				notes = append(notes, n)
				continue
			}
			cur := &notes[i]
			cur.About = unionSorted(cur.About, n.About)
			for k, v := range n.Values {
				if old, has := cur.Values[k]; has {
					if old != v {
						warns = append(warns, fmt.Sprintf(
							"seed value %q has conflicting values (%q, %q); kept the first", k, old, v))
					}
					continue
				}
				if cur.Values == nil {
					cur.Values = map[string]string{}
				}
				cur.Values[k] = v
			}
		}
		for _, m := range p.Mentions {
			key := strings.Join([]string{m.Method, m.Path, m.Service, normalizeEvidence(m.Evidence)}, "\x00")
			if _, dup := seenMention[key]; dup {
				continue
			}
			seenMention[key] = struct{}{}
			mentions = append(mentions, m)
		}
	}

	sort.SliceStable(notes, func(i, j int) bool {
		if notes[i].Kind != notes[j].Kind {
			return notes[i].Kind < notes[j].Kind
		}
		return notes[i].Text < notes[j].Text
	})
	sort.Strings(warns)
	return notes, mentions, warns
}

// resourceList is the closed vocabulary the notes stage may use for "about":
// services, group IDs and resources, static path segments and request fields.
func resourceList(src *domain.SourceSet, groups []apiGroup) []string {
	set := map[string]struct{}{}
	add := func(s string) {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			set[s] = struct{}{}
		}
	}
	for _, s := range src.Servers {
		add(s)
	}
	for _, g := range groups {
		add(g.ID)
		add(g.Resource)
	}
	for _, a := range src.APIs {
		for _, seg := range splitCallPath(a.Path) {
			if seg != "" && !isTemplateSegment(seg) {
				add(seg)
			}
		}
		for f := range a.Sig.Request {
			add(f)
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// ============================================================================
// API mentions vs OpenAPI
// ============================================================================

func normalizeMentionPath(p string) string {
	p = strings.TrimSpace(p)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") && len(s) > 1 {
			segs[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(segs, "/")
}

// pathsMatch compares two path templates; a {param} segment on either side
// matches any segment.
func pathsMatch(a, b string) bool {
	as, bs := splitCallPath(a), splitCallPath(b)
	if len(as) != len(bs) {
		return false
	}
	for i := range as {
		if as[i] == bs[i] || isTemplateSegment(as[i]) || isTemplateSegment(bs[i]) {
			continue
		}
		return false
	}
	return true
}

// matchMentions marks each mention Known when OpenAPI has the API and returns
// the warnings for the rest. It never adds APIs: OpenAPI is authoritative.
func matchMentions(ms []domain.APIMention, apis []domain.API) ([]domain.APIMention, []sourceWarning) {
	var warns []sourceWarning
	seenWarn := map[string]struct{}{}
	warn := func(w sourceWarning) {
		k := w.Code + "\x00" + w.Key
		if _, dup := seenWarn[k]; dup {
			return
		}
		seenWarn[k] = struct{}{}
		warns = append(warns, w)
	}

	knownServers := make(map[string]struct{})
	for _, a := range apis {
		knownServers[a.Server] = struct{}{}
	}

	out := make([]domain.APIMention, 0, len(ms))
	for _, m := range ms {
		m.Method = strings.ToUpper(strings.TrimSpace(m.Method))
		m.Path = normalizeMentionPath(m.Path)
		m.Service = strings.TrimSpace(m.Service)
		m.Known = false

		if m.Method == "" || m.Path == "" {
			warn(sourceWarning{
				Code: "insufficient_info", Key: "(unknown)",
				Line:    "`(unknown)` — mention lacks method or path; cannot be matched",
				Message: "Requirements mention an API without method or path; cannot match it against OpenAPI",
			})
			out = append(out, m)
			continue
		}

		key := m.Method + " " + m.Path
		servers := map[string]struct{}{}
		_, serviceIsKnown := knownServers[m.Service]
		for _, a := range apis {
			if strings.EqualFold(a.Method, m.Method) && pathsMatch(a.Path, m.Path) {
				// Match if mention service is empty, there is only one server in spec,
				// the service name is not a recognized server alias (fallback), or it directly matches.
				if m.Service == "" || len(knownServers) == 1 || !serviceIsKnown || a.Server == m.Service {
					servers[a.Server] = struct{}{}
				}
			}
		}
		switch {
		case len(servers) == 0:
			warn(sourceWarning{
				Code: "unknown_api", Key: key,
				Line: fmt.Sprintf("`%s` — requirements mention but not in OpenAPI (ignored)", key),
				Message: fmt.Sprintf(
					"Requirements mention API %q but OpenAPI does not have it; ignored (OpenAPI is the only API source)", key),
			})
		default:
			m.Known = true
			if m.Service == "" && len(servers) > 1 {
				warn(sourceWarning{
					Code: "ambiguous_service", Key: key,
					Line:    fmt.Sprintf("`%s` — mentioned without a service; matched multiple servers", key),
					Message: fmt.Sprintf("Requirements mention %q without a service; it matches several servers", key),
				})
			}
		}
		out = append(out, m)
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		return a.Service < b.Service
	})
	sort.SliceStable(warns, func(i, j int) bool {
		if warns[i].Code != warns[j].Code {
			return warns[i].Code < warns[j].Code
		}
		return warns[i].Key < warns[j].Key
	})
	return out, warns
}

// ============================================================================
// Small helpers
// ============================================================================

func unionSorted(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := slices.Clone(a)
	out = append(out, b...)
	sort.Strings(out)
	return slices.Compact(out)
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
