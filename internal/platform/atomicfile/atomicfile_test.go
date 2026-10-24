package atomicfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// poisonTMPDIR points TMPDIR at a nonexistent directory inside dir. A correct
// implementation creates its temp file next to the destination (never through
// os.TempDir()), so it is unaffected; an implementation that fell back to
// os.TempDir() would fail to create the temp file at all.
func poisonTMPDIR(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("TMPDIR", filepath.Join(dir, "does-not-exist"))
}

func TestWrite_CreatesFileAndMode(t *testing.T) {
	dir := t.TempDir()
	poisonTMPDIR(t, dir)

	path := filepath.Join(dir, "out.txt")
	data := []byte("hello atomic")

	if err := Write(path, data, 0o644); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", got, data)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %04o, want 0644", fi.Mode().Perm())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "out.txt" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("destination dir contains %v, want only [out.txt]", names)
	}
}

func TestWrite_ReplacesExistingWithoutTempLeftover(t *testing.T) {
	dir := t.TempDir()
	poisonTMPDIR(t, dir)

	path := filepath.Join(dir, "config.yaml")
	if err := Write(path, []byte("old: true"), 0o644); err != nil {
		t.Fatalf("Write() initial error = %v, want nil", err)
	}

	if err := Write(path, []byte("new: content"), 0o600); err != nil {
		t.Fatalf("Write() replace error = %v, want nil", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "new: content" {
		t.Errorf("content = %q, want %q", got, "new: content")
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %04o, want 0600", fi.Mode().Perm())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %q in destination dir", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("destination dir contains %d entries, want 1 (no .tmp leftovers)", len(entries))
	}
}

func TestWrite_ErrorDoesNotLeaveTemp(t *testing.T) {
	dir := t.TempDir()
	poisonTMPDIR(t, dir)

	path := filepath.Join(dir, "missing-parent", "out.txt")
	if err := Write(path, []byte("data"), 0o644); err == nil {
		t.Fatal("Write() error = nil, want error for missing parent dir")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Stat(%q) error = %v, want IsNotExist", path, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("destination dir contains %v after failed write, want empty", names)
	}
}
