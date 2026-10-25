package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// clearLLMEnv unsets every environment variable the credential store reads so tests
// only observe values written to their temporary config files.
func clearLLMEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"SPECTEST_PROVIDER", "SPECTEST_MODEL", "SPECTEST_ENDPOINT", "SPECTEST_API_KEY",
		"OPENAI_API_KEY", "OPENAI_ENDPOINT", "OPENAI_MODEL",
	} {
		t.Setenv(k, "")
	}
}

func newTestStore(t *testing.T) (*FileCredentialStore, string) {
	t.Helper()
	clearLLMEnv(t)
	dir := t.TempDir()
	globalPath := filepath.Join(dir, ".spectest", "credentials.yaml")
	return NewFileCredentialStoreWithPath(globalPath), globalPath
}

func TestFileCredentialStore_SaveAndResolve(t *testing.T) {
	store, globalPath := newTestStore(t)

	if err := store.SetDefaultProvider("OpenAI"); err != nil {
		t.Fatalf("SetDefaultProvider: %v", err)
	}
	if err := store.SetEndpoint("openai", "https://api.openai.com/v1"); err != nil {
		t.Fatalf("SetEndpoint: %v", err)
	}
	if err := store.SetDefaultModel("openai", "gpt-4o-mini"); err != nil {
		t.Fatalf("SetDefaultModel: %v", err)
	}
	if err := store.SetAPIKey("openai", "sk-test-secret-123456"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	info, err := os.Stat(globalPath)
	if err != nil {
		t.Fatalf("stat credentials file: %v", err)
	}
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm != 0o600 {
		t.Errorf("credentials file permissions = %04o, want 0600", perm)
	}

	if got := store.GetDefaultProvider(); got != "openai" {
		t.Errorf("GetDefaultProvider() = %q, want openai (lowercased)", got)
	}
	if got := store.GetDefaultModel("openai"); got != "gpt-4o-mini" {
		t.Errorf("GetDefaultModel() = %q, want gpt-4o-mini", got)
	}
	if ep, _ := store.GetEndpoint("openai"); ep != "https://api.openai.com/v1" {
		t.Errorf("GetEndpoint() = %q", ep)
	}
	if key, _ := store.GetAPIKey("openai"); key != "sk-test-secret-123456" {
		t.Errorf("GetAPIKey() = %q", key)
	}
	if store.GlobalPath() != globalPath {
		t.Errorf("GlobalPath() = %q, want %q", store.GlobalPath(), globalPath)
	}
}

func TestFileCredentialStore_EmptyWhenUnconfigured(t *testing.T) {
	store, _ := newTestStore(t)

	if got := store.GetDefaultProvider(); got != "" {
		t.Errorf("GetDefaultProvider() = %q, want empty", got)
	}
	if got := store.GetDefaultModel("openai"); got != "" {
		t.Errorf("GetDefaultModel() = %q, want empty", got)
	}
	if ep, _ := store.GetEndpoint("openai"); ep != "" {
		t.Errorf("GetEndpoint() = %q, want empty", ep)
	}
	if key, _ := store.GetAPIKey("openai"); key != "" {
		t.Errorf("GetAPIKey() = %q, want empty", key)
	}
}

func TestFileCredentialStore_EnvironmentOverrides(t *testing.T) {
	store, _ := newTestStore(t)
	if err := store.SetDefaultProvider("openai"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAPIKey("openai", "file-key"); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENAI_API_KEY", "provider-env-key")
	if key, _ := store.GetAPIKey("openai"); key != "provider-env-key" {
		t.Errorf("<PROVIDER>_API_KEY should override file, got %q", key)
	}

	t.Setenv("SPECTEST_API_KEY", "global-env-key")
	if key, _ := store.GetAPIKey("openai"); key != "global-env-key" {
		t.Errorf("SPECTEST_API_KEY should take highest priority, got %q", key)
	}

	t.Setenv("OPENAI_ENDPOINT", "http://env-endpoint")
	if ep, _ := store.GetEndpoint("openai"); ep != "http://env-endpoint" {
		t.Errorf("<PROVIDER>_ENDPOINT override failed, got %q", ep)
	}

	t.Setenv("OPENAI_MODEL", "env-model")
	if m := store.GetDefaultModel("openai"); m != "env-model" {
		t.Errorf("<PROVIDER>_MODEL override failed, got %q", m)
	}

	t.Setenv("SPECTEST_PROVIDER", "Custom")
	if p := store.GetDefaultProvider(); p != "custom" {
		t.Errorf("SPECTEST_PROVIDER override failed, got %q", p)
	}
}

func TestFileCredentialStore_ValidationErrors(t *testing.T) {
	store, _ := newTestStore(t)

	if err := store.SetAPIKey("", "key"); err == nil {
		t.Error("SetAPIKey without provider should fail")
	}
	if err := store.SetEndpoint("", "http://x"); err == nil {
		t.Error("SetEndpoint without provider should fail")
	}
	if err := store.SetDefaultProvider("   "); err == nil {
		t.Error("SetDefaultProvider with blank value should fail")
	}
	if err := store.SetDefaultModel("openai", ""); err == nil {
		t.Error("SetDefaultModel with blank value should fail")
	}
}

func TestMaskKey(t *testing.T) {
	tests := map[string]string{
		"":                  "[not set]",
		"short":             "****",
		"12345678":          "****",
		"sk-abcdefghijklmn": "sk-a...klmn",
	}
	for in, want := range tests {
		if got := MaskKey(in); got != want {
			t.Errorf("MaskKey(%q) = %q, want %q", in, got, want)
		}
	}
}
