package generate

import (
	"testing"

	"github.com/nguyentin05/spectest/internal/domain"
)

func TestValidateSemantics_SaveNameToPath(t *testing.T) {
	api := domain.API{
		Key: "GET {default}/users", Server: "default", Method: "GET", Path: "/users",
		Responses: map[string]domain.Schema{"200": {}},
	}
	steps := []domain.Step{
		{Call: api.Key, Expect: 200, Save: map[string]string{"token": "body.token"}},
		{Call: api.Key, Expect: 200, Headers: map[string]string{"Authorization": "Bearer {{token}}"}},
	}
	suite := &domain.Suite{
		Servers: []string{"default"},
		Scenarios: []domain.Scenario{{
			ID: "test", Name: "Test", Steps: steps,
		}},
	}

	errs := validateSemantics(suite, map[string]domain.API{api.Key: api})
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateSemantics_RejectsReversedSave(t *testing.T) {
	api := domain.API{
		Key: "GET {default}/users", Server: "default", Method: "GET", Path: "/users",
		Responses: map[string]domain.Schema{"200": {}},
	}
	steps := []domain.Step{
		{Call: api.Key, Expect: 200, Save: map[string]string{"body.token": "token"}},
	}
	suite := &domain.Suite{
		Servers: []string{"default"},
		Scenarios: []domain.Scenario{{
			ID: "test", Name: "Test", Steps: steps,
		}},
	}

	errs := validateSemantics(suite, map[string]domain.API{api.Key: api})
	if len(errs) == 0 {
		t.Fatal("expected reversed save to be rejected")
	}
}
