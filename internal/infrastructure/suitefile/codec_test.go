package suitefile

import (
	"strings"
	"testing"

	"github.com/nguyentin05/spectest/internal/domain"
)

// TestCodec_PreservesExplicitNulls guards the two places where an explicit YAML
// null carries meaning: deleting an entity (change.after: null) and removing a
// field from an entity (a nil field inside an after patch, e.g. scenario.skip).
func TestCodec_PreservesExplicitNulls(t *testing.T) {
	log := &domain.LogFile{
		Base: domain.BaseEntry{
			At: "2026-10-24T10:00:00+07:00",
			By: domain.ByAI,
			Spectest: &domain.Suite{
				Version:   1,
				Servers:   []string{"auth"},
				APIs:      []string{"POST {auth}/api/login"},
				Scenarios: []domain.Scenario{},
			},
			AfterHash: "aaaaaaaaaaaa",
		},
		Update: []domain.UpdateEntry{
			{
				N:  1,
				At: "2026-10-24T10:15:00+07:00",
				By: domain.ByUser,
				Change: []domain.Change{
					{
						Target: domain.TargetScenario,
						ID:     "order-cancel",
						Reason: "user deleted scenario",
						After:  nil,
					},
					{
						Target: domain.TargetScenario,
						ID:     "order-cancel",
						Reason: "API restored",
						Before: map[string]any{"skip": "endpoint removed"},
						After:  map[string]any{"skip": nil},
					},
				},
				AfterHash: "bbbbbbbbbbbb",
			},
		},
	}

	data, err := encodeLog(log)
	if err != nil {
		t.Fatalf("encodeLog() error = %v, want nil", err)
	}
	text := string(data)
	if !strings.Contains(text, "after: null") {
		t.Errorf("encoded log does not contain %q:\n%s", "after: null", text)
	}
	if !strings.Contains(text, "skip: null") {
		t.Errorf("encoded log does not contain %q:\n%s", "skip: null", text)
	}

	decoded, err := decodeLog(data)
	if err != nil {
		t.Fatalf("decodeLog() error = %v, want nil", err)
	}
	if len(decoded.Update) != 1 || len(decoded.Update[0].Change) != 2 {
		t.Fatalf("decoded update/change = %d/%d, want 1/2", len(decoded.Update), len(decoded.Update[0].Change))
	}
	if decoded.Update[0].Change[0].After != nil {
		t.Errorf("entity deletion After = %#v, want nil", decoded.Update[0].Change[0].After)
	}
	patch, ok := decoded.Update[0].Change[1].After.(map[string]any)
	if !ok {
		t.Fatalf("field patch After type = %T, want map[string]any", decoded.Update[0].Change[1].After)
	}
	value, present := patch["skip"]
	if !present {
		t.Fatal("field patch After lost the explicit skip key")
	}
	if value != nil {
		t.Errorf("field patch skip = %#v, want nil", value)
	}

	again, err := encodeLog(decoded)
	if err != nil {
		t.Fatalf("re-encodeLog() error = %v, want nil", err)
	}
	if !strings.Contains(string(again), "after: null") || !strings.Contains(string(again), "skip: null") {
		t.Errorf("re-encoded log lost explicit nulls:\n%s", again)
	}
}

// TestCodec_EncodeIndentTwoAndTrailingNewline pins the on-disk formatting that
// the three-file artifacts are expected to follow.
func TestCodec_EncodeIndentTwoAndTrailingNewline(t *testing.T) {
	suite := &domain.Suite{
		Version:   1,
		Servers:   []string{"auth"},
		APIs:      []string{"POST {auth}/api/login"},
		Scenarios: []domain.Scenario{},
	}

	data, err := encodeSuite(suite)
	if err != nil {
		t.Fatalf("encodeSuite() error = %v, want nil", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("encoded suite does not end with a newline")
	}
	if !strings.Contains(string(data), "\n  - auth") {
		t.Errorf("encoded suite does not use two-space indentation:\n%s", data)
	}
}
