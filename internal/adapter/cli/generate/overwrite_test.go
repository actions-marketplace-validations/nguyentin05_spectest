package generate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nguyentin05/spectest/internal/infrastructure/suitefile"
)

func TestConfirmOverwrite(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "yes", input: "y\n", want: true},
		{name: "YES", input: "YES\n", want: true},
		{name: "no", input: "n\n", want: false},
		{name: "empty", input: "\n", want: false},
		{name: "eof", input: "", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := confirmOverwrite(strings.NewReader(tc.input), &out, []string{"a", "b"})
			if err != nil {
				t.Fatalf("confirmOverwrite() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("confirmOverwrite() = %v, want %v", got, tc.want)
			}
			if !strings.Contains(out.String(), "Overwrite") {
				t.Fatalf("prompt output = %q, want Overwrite", out.String())
			}
		})
	}
}

func TestGuardOverwrite(t *testing.T) {
	dir := t.TempDir()
	suitePath := filepath.Join(dir, suitefile.SuiteFileName)
	if err := os.WriteFile(suitePath, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := guardOverwrite(strings.NewReader("n\n"), &out, dir, false)
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Fatalf("error = %v, want abort", err)
	}
	got, _ := os.ReadFile(suitePath)
	if string(got) != "keep" {
		t.Fatalf("suite changed to %q, want keep", got)
	}

	out.Reset()
	if err := guardOverwrite(strings.NewReader(""), &out, dir, true); err != nil {
		t.Fatalf("dry-run guard error = %v", err)
	}
	if strings.Contains(out.String(), "Overwrite") {
		t.Fatal("dry-run must not prompt")
	}
}

func TestGuardOverwrite_PendingMarker(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, suitefile.PendingMarkerName)
	if err := os.WriteFile(marker, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := guardOverwrite(strings.NewReader("y\n"), &out, dir, false)
	if err == nil || !strings.Contains(err.Error(), "pending write marker") {
		t.Fatalf("error = %v, want pending marker", err)
	}
	if strings.Contains(out.String(), "Overwrite") {
		t.Fatal("pending marker must abort before prompting")
	}

	out.Reset()
	err = guardOverwrite(strings.NewReader("y\n"), &out, dir, true)
	if err == nil || !strings.Contains(err.Error(), "pending write marker") {
		t.Fatalf("dry-run error = %v, want pending marker", err)
	}
}
