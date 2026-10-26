package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nguyentin05/spectest/internal/adapter/cli/cmdutil"
)

func TestUICmd_Help(t *testing.T) {
	f := cmdutil.NewFactory()
	cmd := NewCommand(f)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	got := out.String()
	for _, flag := range []string{"--port", "--host", "--no-open"} {
		if !strings.Contains(got, flag) {
			t.Errorf("expected ui help to contain flag %q, got:\n%s", flag, got)
		}
	}
}

func TestUICmd_Flags(t *testing.T) {
	f := cmdutil.NewFactory()
	cmd := NewCommand(f)

	cmd.SetArgs([]string{"--port", "9099", "--host", "0.0.0.0", "--no-open"})
	if err := cmd.ParseFlags([]string{"--port", "9099", "--host", "0.0.0.0", "--no-open"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}

	p, err := cmd.Flags().GetInt("port")
	if err != nil || p != 9099 {
		t.Errorf("port = %d, want 9099", p)
	}

	h, err := cmd.Flags().GetString("host")
	if err != nil || h != "0.0.0.0" {
		t.Errorf("host = %q, want 0.0.0.0", h)
	}

	noOpen, err := cmd.Flags().GetBool("no-open")
	if err != nil || !noOpen {
		t.Errorf("no-open = %v, want true", noOpen)
	}
}
