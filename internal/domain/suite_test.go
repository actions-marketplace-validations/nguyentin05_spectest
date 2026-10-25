package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSuiteMutable_OnlyFourBlocks(t *testing.T) {
	suite := &Suite{
		Version:  1,
		Servers:  []string{"auth"},
		APIs:     []string{"POST {auth}/api/login"},
		Analysis: "original analysis",
		Seed: map[string]Seed{
			"admin": {Vars: map[string]any{"email": "admin@example.com"}},
		},
		Setup: map[string]Setup{
			"user": {Steps: []Step{{Call: "POST {auth}/api/register", Expect: 201}}},
		},
		Scenarios: []Scenario{
			{ID: "login", Name: "Login", Steps: []Step{{Call: "POST {auth}/api/login", Expect: 200}}},
		},
	}

	mutable := suite.Mutable()

	suite.Version = 2
	suite.Servers = []string{"order"}
	suite.APIs = []string{"GET {order}/api/orders"}

	if mutable.Analysis != "original analysis" {
		t.Errorf("Mutable().Analysis = %q, want %q", mutable.Analysis, "original analysis")
	}
	if got := mutable.Seed["admin"].Vars["email"]; got != "admin@example.com" {
		t.Errorf("Mutable().Seed[admin].Vars[email] = %v", got)
	}
	if len(mutable.Setup) != 1 || mutable.Setup["user"].Steps[0].Expect != 201 {
		t.Errorf("Mutable().Setup = %+v", mutable.Setup)
	}
	if len(mutable.Scenarios) != 1 || mutable.Scenarios[0].ID != "login" {
		t.Errorf("Mutable().Scenarios = %+v", mutable.Scenarios)
	}
}

func TestSuiteYAML_NewSchemaShape(t *testing.T) {
	suite := &Suite{
		Version:  1,
		Servers:  []string{"auth"},
		APIs:     []string{"POST {auth}/api/login"},
		Analysis: "short analysis",
		Seed: map[string]Seed{
			"admin": {Vars: map[string]any{"email": "admin@example.com"}},
		},
		Setup: map[string]Setup{
			"user": {
				Use: []string{"seed_setup"},
				Vars: map[string]any{
					"email": "user@example.com",
				},
				Steps: []Step{
					{
						Call:    "POST {auth}/api/login",
						Headers: map[string]string{"Authorization": "Bearer {{token}}"},
						Body:    map[string]any{"email": "{{email}}"},
						Expect:  200,
						Save:    map[string]string{"token": "body.access_token"},
					},
				},
			},
		},
		Scenarios: []Scenario{
			{
				ID:     "login-ok",
				Name:   "Login succeeds",
				Setup:  []string{"user"},
				Steps:  []Step{{Call: "POST {auth}/api/login", Expect: 200}},
				Locked: true,
				Skip:   "temporarily disabled",
			},
		},
	}

	raw, err := yaml.Marshal(suite)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	text := string(raw)

	if !strings.HasPrefix(text, "version: 1\n") {
		t.Errorf("suite should start with version: 1, got:\n%s", text)
	}
	for _, want := range []string{"servers:", "apis:", "analysis:", "seed:", "setup:", "scenarios:", "call:", "headers:", "body:", "expect:", "save:"} {
		if !strings.Contains(text, want) {
			t.Errorf("suite YAML missing key %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"metadata:", "base:", "test_cases:", "assert:", "request:", "target_ref:"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("suite YAML must not contain old schema key %q:\n%s", forbidden, text)
		}
	}

	var roundTrip Suite
	if err := yaml.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if roundTrip.Version != 1 {
		t.Errorf("Version = %d, want 1", roundTrip.Version)
	}
	if got := roundTrip.Scenarios[0].Steps[0].Call; got != "POST {auth}/api/login" {
		t.Errorf("step call = %q", got)
	}
}

func TestChange_AfterNullPreserved(t *testing.T) {
	entry := UpdateEntry{
		N:  1,
		At: "2026-10-24T10:00:00+07:00",
		By: ByUser,
		Change: []Change{
			{
				Target: TargetScenario,
				ID:     "login-ok",
				Reason: "manual edit",
				Before: map[string]any{"skip": "old reason"},
				After:  nil,
			},
		},
	}

	raw, err := yaml.Marshal(entry)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	if !strings.Contains(string(raw), "after: null") {
		t.Errorf("explicit null after must be preserved, got:\n%s", raw)
	}
}

func TestConstants(t *testing.T) {
	if TargetAnalysis != "analysis" || TargetSeed != "seed" || TargetSetup != "setup" || TargetScenario != "scenario" {
		t.Error("unexpected change target constants")
	}
	if ByAI != "ai" || ByUser != "user" {
		t.Error("unexpected author constants")
	}
	if StatusActive != "active" || StatusRemoved != "removed" {
		t.Error("unexpected api status constants")
	}
}
