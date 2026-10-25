package run

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nguyentin05/spectest/internal/adapter/cli/cmdutil"
	"github.com/nguyentin05/spectest/internal/domain"
	"github.com/nguyentin05/spectest/internal/infrastructure/suitefile"
)

func TestRunCmd_Help(t *testing.T) {
	cmd := NewCommand(cmdutil.NewFactory())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run --help failed: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"spectest run",
		"-f, --file",
		"-t, --target",
		"-c, --compose",
		"--timeout",
		"--report-dir",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected help output to contain %q; got:\n%s", want, got)
		}
	}

	// Verify --start-cmd was successfully removed
	if strings.Contains(got, "--start-cmd") {
		t.Error("unexpected --start-cmd flag in help output")
	}
}

func TestRunCmd_Execute_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	suitePath := filepath.Join(tmpDir, "spectest.yaml")

	suite := &domain.Suite{
		Version: 1,
		Servers: []string{"default"},
		APIs:    []string{"GET {default}/api/items"},
		Scenarios: []domain.Scenario{
			{
				ID:   "SCN_1",
				Name: "Cart Checkout",
				Steps: []domain.Step{
					{
						Call:   "GET {default}/api/items",
						Expect: 200,
					},
				},
			},
		},
	}
	data, err := suitefile.EncodeSuite(suite)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(suitePath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	f := cmdutil.NewFactory()
	cmd := NewCommand(f)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"-f", suitePath, "-t", srv.URL, "--timeout", "5"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "SpecTest: ready and executed suite") {
		t.Errorf("expected output to mention suite execution, got: %s", got)
	}
}

func TestRunCmd_MissingSuite(t *testing.T) {
	f := cmdutil.NewFactory()
	cmd := NewCommand(f)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"-f", "non-existent-suite.yaml", "-t", "http://localhost:8080"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for non-existent suite file, got nil")
	}
}
