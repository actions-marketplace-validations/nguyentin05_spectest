package generate

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nguyentin05/spectest/internal/adapter/cli/cmdutil"
)

func TestGenerateCmd_HelpFlags(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	got := out.String()
	for _, flag := range []string{"--api", "--req", "--output", "--dry-run"} {
		if !strings.Contains(got, flag) {
			t.Errorf("expected help output to document %q, got:\n%s", flag, got)
		}
	}
	for _, removed := range []string{"--enrich", "--export", "--concurrency"} {
		if strings.Contains(got, removed) {
			t.Errorf("help output must not document removed flag %q, got:\n%s", removed, got)
		}
	}
}

func TestGenerateCmd_ValidFlagsNotImplemented(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--api", "auth=openapi-auth.yaml",
		"--api", "order=openapi-order.yaml",
		"--req", "requirements.md",
		"--output", "./tests",
		"--dry-run",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != "generate: not implemented" {
		t.Fatalf("error = %v, want generate: not implemented", err)
	}
}

func TestGenerateCmd_RejectsPositionalArgs(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"auth=openapi.yaml"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected positional argument to be rejected")
	}
}

func TestGenerateCmd_MissingAPI(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--api (-a) is missing") {
		t.Fatalf("error = %v, want missing --api validation", err)
	}
}
