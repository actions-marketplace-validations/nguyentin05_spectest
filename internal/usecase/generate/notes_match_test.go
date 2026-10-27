package generate

import (
	"testing"

	"github.com/nguyentin05/spectest/internal/domain"
)

func TestMatchMentions_SingleServer_IgnoresAliasMismatch(t *testing.T) {
	apis := []domain.API{
		{Server: "server", Method: "POST", Path: "/api/v1/register"},
		{Server: "server", Method: "POST", Path: "/api/v1/login"},
		{Server: "server", Method: "GET", Path: "/api/v1/me"},
		{Server: "server", Method: "GET", Path: "/health"},
	}

	mentions := []domain.APIMention{
		{Method: "POST", Path: "/api/v1/register", Service: "auth"},
		{Method: "POST", Path: "/api/v1/login", Service: "auth"},
		{Method: "GET", Path: "/api/v1/me", Service: "auth"},
		{Method: "GET", Path: "/health", Service: "auth"},
	}

	matched, warns := matchMentions(mentions, apis)

	if len(warns) != 0 {
		t.Fatalf("expected 0 warnings for single-server alias mismatch, got %d: %+v", len(warns), warns)
	}

	for _, m := range matched {
		if !m.Known {
			t.Errorf("expected mention %s %s to be Known, got false", m.Method, m.Path)
		}
	}
}

func TestMatchMentions_MultiServer_MatchesKnownAlias(t *testing.T) {
	apis := []domain.API{
		{Server: "auth", Method: "POST", Path: "/login"},
		{Server: "billing", Method: "POST", Path: "/checkout"},
	}

	mentions := []domain.APIMention{
		{Method: "POST", Path: "/login", Service: "auth"},
		{Method: "POST", Path: "/checkout", Service: "billing"},
		{Method: "GET", Path: "/unknown", Service: "billing"},
	}

	matched, warns := matchMentions(mentions, apis)

	if len(warns) != 1 || warns[0].Code != "unknown_api" {
		t.Fatalf("expected 1 unknown_api warning, got %+v", warns)
	}

	if !matched[0].Known || !matched[1].Known || matched[2].Known {
		t.Errorf("unexpected Known statuses: %+v", matched)
	}
}
