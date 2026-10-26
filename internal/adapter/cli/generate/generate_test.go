package generate

import (
	"bytes"
	"context"
	"errors"
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
	for _, flag := range []string{"--api", "--req", "--output", "--dry-run", "--mode"} {
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

func TestGenerateCmd_UpdateModeNotImplemented(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())
	cmd.SetArgs([]string{"--mode", "update"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("error = %v, want update not implemented", err)
	}
}

func TestGenerateCmd_InvalidMode(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())
	cmd.SetArgs([]string{"--mode", "migrate"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "must be init or update") {
		t.Fatalf("error = %v, want invalid mode", err)
	}
}

func TestGenerateCmd_UsesCommandContext(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())
	cmd.SetArgs([]string{"--api", "auth=whatever.yaml", "--output", t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cmd.ExecuteContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
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
