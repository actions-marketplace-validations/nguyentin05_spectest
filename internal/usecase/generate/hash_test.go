package generate

import (
	"encoding/json"
	"reflect"
	"regexp"
	"slices"
	"testing"

	"github.com/nguyentin05/spectest/internal/domain"
)

// hashTestFixture returns a suite whose mutable content touches every field
// that participates in canonicalization.
func hashTestFixture() *domain.Suite {
	return &domain.Suite{
		Version:  1,
		Servers:  []string{"api", "admin"},
		APIs:     []string{"GET {api}/orders", "POST {api}/orders"},
		Analysis: "Orders are created then read.\nSecond line.\n",
		Seed: map[string]domain.Seed{
			"globals": {Vars: map[string]any{"base_url": "https://api.test", "retries": "3"}},
		},
		Setup: map[string]domain.Setup{
			"auth": {
				Use:  []string{"database", "cache"},
				Vars: map[string]any{"token": "{{random}}"},
				Steps: []domain.Step{{
					Call:   "POST {api}/login",
					Expect: 200,
					Save:   map[string]string{"token": "body.token"},
				}},
			},
			"database": {
				Steps: []domain.Step{{Call: "POST {api}/database", Expect: 200}},
			},
			"cache": {
				Steps: []domain.Step{{Call: "POST {api}/cache", Expect: 200}},
			},
		},
		Scenarios: []domain.Scenario{
			{
				ID:    "list-orders",
				Name:  "List orders",
				Setup: []string{"database", "auth"},
				Steps: []domain.Step{
					{Call: "GET {api}/orders", Expect: 200},
					{Call: "GET {api}/orders/{id}", Expect: 404},
				},
			},
			{
				ID:    "create-order",
				Name:  "Create order",
				Setup: []string{"auth"},
				Steps: []domain.Step{{
					Call:    "POST {api}/orders",
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    map[string]any{"items": []any{map[string]any{"sku": "abc"}}},
					Expect:  201,
				}},
			},
		},
	}
}

// hashTestClone deep-copies a suite through JSON so tests do not share any
// nested map or slice with the fixture.
func hashTestClone(t *testing.T, suite *domain.Suite) *domain.Suite {
	t.Helper()
	data, err := json.Marshal(suite)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var clone domain.Suite
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return &clone
}

func TestSuiteHash_FormattingInvariant(t *testing.T) {
	base := hashTestFixture()
	variant := hashTestClone(t, base)

	variant.Analysis = "Orders are created then read.   \r\nSecond line.\t\r\n"

	slices.Reverse(variant.Scenarios)
	for name, setup := range variant.Setup {
		slices.Reverse(setup.Use)
		variant.Setup[name] = setup
	}
	for i := range variant.Scenarios {
		slices.Reverse(variant.Scenarios[i].Setup)
		for j := range variant.Scenarios[i].Steps {
			// Empty optional containers must normalize to absent.
			if len(variant.Scenarios[i].Steps[j].Headers) == 0 {
				variant.Scenarios[i].Steps[j].Headers = map[string]string{}
			}
			if len(variant.Scenarios[i].Steps[j].Save) == 0 {
				variant.Scenarios[i].Steps[j].Save = map[string]string{}
			}
		}
	}
	// An explicit empty use list must normalize to absent.
	database := variant.Setup["database"]
	database.Use = []string{}
	variant.Setup["database"] = database

	if got, want := suiteHash(variant), suiteHash(base); got != want {
		t.Fatalf("suiteHash is not formatting invariant: got %s, want %s", got, want)
	}
}

func TestSuiteHash_ContentSensitive(t *testing.T) {
	base := hashTestFixture()
	baseHash := suiteHash(base)

	mutations := []struct {
		name   string
		mutate func(*domain.Suite)
	}{
		{
			name:   "analysis",
			mutate: func(s *domain.Suite) { s.Analysis = "A different analysis.\n" },
		},
		{
			name: "step order",
			mutate: func(s *domain.Suite) {
				slices.Reverse(s.Scenarios[0].Steps)
			},
		},
		{
			name: "body",
			mutate: func(s *domain.Suite) {
				s.Scenarios[1].Steps[0].Body = map[string]any{
					"items": []any{map[string]any{"sku": "xyz"}},
				}
			},
		},
		{
			name:   "locked",
			mutate: func(s *domain.Suite) { s.Scenarios[0].Locked = true },
		},
		{
			name:   "skip reason",
			mutate: func(s *domain.Suite) { s.Scenarios[0].Skip = "maintenance" },
		},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			variant := hashTestClone(t, base)
			mutation.mutate(variant)
			if got := suiteHash(variant); got == baseHash {
				t.Fatalf("suiteHash ignored %s change: %s", mutation.name, got)
			}
		})
	}
}

func TestSuiteHash_ExcludesDerivedFields(t *testing.T) {
	base := hashTestFixture()
	variant := hashTestClone(t, base)
	variant.Version = 2
	variant.Servers = []string{"other"}
	variant.APIs = []string{"DELETE {api}/gone"}

	if got, want := suiteHash(variant), suiteHash(base); got != want {
		t.Fatalf("suiteHash included derived fields: got %s, want %s", got, want)
	}
}

func TestSuiteHash_DropsEmptyOptional(t *testing.T) {
	withEmpty := &domain.Suite{
		Seed: map[string]domain.Seed{},
		Setup: map[string]domain.Setup{
			"session": {
				Use: []string{},
				Steps: []domain.Step{{
					Call:    "POST {api}/session",
					Headers: map[string]string{},
					Save:    map[string]string{},
					Expect:  200,
				}},
			},
		},
		Scenarios: []domain.Scenario{{
			ID:     "ping",
			Name:   "Ping",
			Setup:  []string{},
			Locked: false,
			Skip:   "",
			Steps:  []domain.Step{{Call: "GET {api}/ping", Expect: 200}},
		}},
	}
	withoutEmpty := &domain.Suite{
		Setup: map[string]domain.Setup{
			"session": {
				Steps: []domain.Step{{Call: "POST {api}/session", Expect: 200}},
			},
		},
		Scenarios: []domain.Scenario{{
			ID:    "ping",
			Name:  "Ping",
			Steps: []domain.Step{{Call: "GET {api}/ping", Expect: 200}},
		}},
	}

	if got, want := suiteHash(withEmpty), suiteHash(withoutEmpty); got != want {
		t.Fatalf("empty optional fields changed the hash: got %s, want %s", got, want)
	}

	// An empty body is meaningful and must not be dropped.
	withBody := &domain.Suite{
		Scenarios: []domain.Scenario{{
			ID:    "ping",
			Steps: []domain.Step{{Call: "GET {api}/ping", Body: map[string]any{}, Expect: 200}},
		}},
	}
	withoutBody := &domain.Suite{
		Scenarios: []domain.Scenario{{
			ID:    "ping",
			Steps: []domain.Step{{Call: "GET {api}/ping", Expect: 200}},
		}},
	}
	if suiteHash(withBody) == suiteHash(withoutBody) {
		t.Fatal("suiteHash dropped an empty body")
	}
}

func TestNormalizeSuite_SortsForWrite(t *testing.T) {
	input := hashTestFixture()
	input.APIs = []string{"POST {api}/orders", "GET {admin}/health", "GET {api}/orders"}
	input.Servers = []string{"admin", "api"}
	input.Setup["auth"] = domain.Setup{
		Use:   []string{"cache", "database"},
		Vars:  input.Setup["auth"].Vars,
		Steps: input.Setup["auth"].Steps,
	}
	input.Scenarios[0].Setup = []string{"database", "auth"}

	snapshot := hashTestClone(t, input)
	normalized := normalizeSuite(input)

	ids := make([]string, len(normalized.Scenarios))
	for i, scenario := range normalized.Scenarios {
		ids[i] = scenario.ID
	}
	if want := []string{"create-order", "list-orders"}; !slices.Equal(ids, want) {
		t.Fatalf("scenarios not sorted by id: got %v, want %v", ids, want)
	}

	if got, want := normalized.Setup["auth"].Use, []string{"cache", "database"}; !slices.Equal(got, want) {
		t.Fatalf("setup use not sorted: got %v, want %v", got, want)
	}
	var listOrders *domain.Scenario
	for i := range normalized.Scenarios {
		if normalized.Scenarios[i].ID == "list-orders" {
			listOrders = &normalized.Scenarios[i]
		}
	}
	if listOrders == nil {
		t.Fatal("list-orders scenario missing after normalization")
	}
	if got, want := listOrders.Setup, []string{"auth", "database"}; !slices.Equal(got, want) {
		t.Fatalf("scenario setup not sorted: got %v, want %v", got, want)
	}
	if got, want := listOrders.Steps[0].Call+","+listOrders.Steps[1].Call, "GET {api}/orders,GET {api}/orders/{id}"; got != want {
		t.Fatalf("step order changed: got %s, want %s", got, want)
	}

	if want := []string{"GET {admin}/health", "GET {api}/orders", "POST {api}/orders"}; !slices.Equal(normalized.APIs, want) {
		t.Fatalf("apis not sorted: got %v, want %v", normalized.APIs, want)
	}
	if want := []string{"admin", "api"}; !slices.Equal(normalized.Servers, want) {
		t.Fatalf("servers order changed: got %v, want %v", normalized.Servers, want)
	}

	if !reflect.DeepEqual(input, snapshot) {
		t.Fatal("normalizeSuite mutated its input")
	}

	// The result must own every nested map and slice.
	normalized.Seed["globals"].Vars["base_url"] = "mutated"
	if input.Seed["globals"].Vars["base_url"] == "mutated" {
		t.Fatal("normalizeSuite shared the seed vars map with its input")
	}
	headers := normalized.Scenarios[0].Steps[0].Headers
	headers["X-Test"] = "mutated"
	if _, ok := input.Scenarios[1].Steps[0].Headers["X-Test"]; ok {
		t.Fatal("normalizeSuite shared step headers with its input")
	}
	body := normalized.Scenarios[0].Steps[0].Body.(map[string]any)
	body["items"].([]any)[0].(map[string]any)["sku"] = "mutated"
	originalBody := input.Scenarios[1].Steps[0].Body.(map[string]any)
	if originalBody["items"].([]any)[0].(map[string]any)["sku"] == "mutated" {
		t.Fatal("normalizeSuite shared a nested body value with its input")
	}
}

func TestCloneSuite_DeepCopy(t *testing.T) {
	original := hashTestFixture()
	snapshot := hashTestClone(t, original)

	clone := cloneSuite(original)
	if clone == original {
		t.Fatal("cloneSuite returned the same pointer")
	}

	clone.Servers[0] = "mutated"
	clone.APIs[0] = "mutated"
	clone.Seed["globals"].Vars["base_url"] = "mutated"
	auth := clone.Setup["auth"]
	auth.Use[0] = "mutated"
	auth.Steps[0].Save["token"] = "mutated"
	clone.Setup["auth"] = auth
	clone.Scenarios[0].Setup[0] = "mutated"
	clone.Scenarios[1].Steps[0].Headers["Content-Type"] = "mutated"
	body := clone.Scenarios[1].Steps[0].Body.(map[string]any)
	body["items"].([]any)[0].(map[string]any)["sku"] = "mutated"

	if !reflect.DeepEqual(original, snapshot) {
		t.Fatal("cloneSuite shares nested data with its input")
	}
}

func TestRequirementsHash_NormalizesLineEndingsAndTrailingWhitespace(t *testing.T) {
	equivalent := []string{
		"line one\nline two\n",
		"line one  \r\nline two\t\r\n",
		"line one \nline two   \n\n\n",
	}
	want := requirementsHash(equivalent[0])
	for _, raw := range equivalent[1:] {
		if got := requirementsHash(raw); got != want {
			t.Fatalf("requirementsHash(%q) = %s, want %s", raw, got, want)
		}
	}

	empty := []string{"", "\n", "\r\n", "   \n\t\n \r\n"}
	for _, raw := range empty {
		if got := requirementsHash(raw); got != hash12([]byte("")) {
			t.Fatalf("requirementsHash(%q) = %s, want empty-text hash %s", raw, got, hash12([]byte("")))
		}
	}
	if want := "e3b0c44298fc"; requirementsHash("") != want {
		t.Fatalf("requirementsHash(\"\") = %s, want %s", requirementsHash(""), want)
	}

	if requirementsHash("line one\nline two\n") == requirementsHash("line one\nline 2\n") {
		t.Fatal("requirementsHash ignored a content change")
	}
}

func TestNormalizeRequirements(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "newlines only", raw: "\n\n", want: ""},
		{name: "whitespace only", raw: " \t\r\n  \r\n", want: ""},
		{name: "crlf", raw: "a\r\nb\r\n", want: "a\nb\n"},
		{name: "trailing spaces", raw: "a  \nb\t \n\n", want: "a\nb\n"},
		{name: "no final newline", raw: "a\nb", want: "a\nb\n"},
		{name: "keeps interior blank lines", raw: "a\n\nb\n", want: "a\n\nb\n"},
		{name: "leading spaces preserved", raw: "  a  \n b\n", want: "  a\n b\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeRequirements(tc.raw); got != tc.want {
				t.Fatalf("normalizeRequirements(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestHash12_IsLowercaseTwelveHex(t *testing.T) {
	got := hash12([]byte("hello world"))
	if len(got) != 12 {
		t.Fatalf("hash12 length = %d, want 12", len(got))
	}
	if !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(got) {
		t.Fatalf("hash12(%q) = %q, want 12 lowercase hex characters", "hello world", got)
	}
	if got != hash12([]byte("hello world")) {
		t.Fatal("hash12 is not deterministic")
	}
	if got == hash12([]byte("hello worle")) {
		t.Fatal("hash12 ignored a content change")
	}
	if want := "e3b0c44298fc"; hash12(nil) != want || hash12([]byte{}) != want {
		t.Fatalf("hash12 of empty data = %s, want %s", hash12(nil), want)
	}
}

func TestSuiteHash_FormattingInvariant_NormalizesAnalysisLineEndings(t *testing.T) {
	base := hashTestFixture()
	base.Analysis = "  indented  \nlast"
	variant := hashTestClone(t, base)
	variant.Analysis = "  indented\t\r\nlast\r\n\r\n"

	if got, want := suiteHash(variant), suiteHash(base); got != want {
		t.Fatalf("analysis line endings changed the hash: got %s, want %s", got, want)
	}
}
