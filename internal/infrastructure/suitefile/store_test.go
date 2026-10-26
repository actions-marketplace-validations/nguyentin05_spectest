package suitefile

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nguyentin05/spectest/internal/domain"
)

func sampleFileSet() *domain.FileSet {
	suite := &domain.Suite{
		Version:  1,
		Input:    domain.InputBoth,
		Servers:  []string{"auth", "order"},
		APIs:     []string{"POST {auth}/api/login", "POST {order}/api/orders"},
		Analysis: "Test the authentication and ordering flows.",
		Seed: map[string]domain.Seed{
			"global": {Vars: map[string]any{"base_url": "https://api.test"}},
		},
		Setup: map[string]domain.Setup{
			"logged_in": {
				Vars: map[string]any{"email": "user@example.com"},
				Steps: []domain.Step{
					{
						Call:    "POST {auth}/api/login",
						Headers: map[string]string{"Content-Type": "application/json"},
						Body:    map[string]any{"email": "user@example.com", "password": "P@ssw0rd123"},
						Expect:  200,
						Save:    map[string]string{"token": "body.token"},
					},
				},
			},
		},
		Scenarios: []domain.Scenario{
			{
				ID:    "order-create",
				Name:  "Create an order",
				Setup: []string{"logged_in"},
				Steps: []domain.Step{
					{
						Call:    "POST {order}/api/orders",
						Headers: map[string]string{"Authorization": "Bearer {{token}}"},
						Body:    map[string]any{"items": []any{map[string]any{"product_id": "p-1", "quantity": 2}}},
						Expect:  201,
						Save:    map[string]string{"order_id": "body.id"},
					},
				},
			},
			{
				ID:     "order-cancel",
				Name:   "Cancel a pending order",
				Setup:  []string{"logged_in"},
				Steps:  []domain.Step{},
				Locked: true,
			},
		},
	}

	log := &domain.LogFile{
		Base: domain.BaseEntry{
			At:        "2026-10-24T10:00:00+07:00",
			By:        domain.ByAI,
			Spectest:  suite,
			AfterHash: "5c1d02a9b7e4",
		},
		Update: []domain.UpdateEntry{
			{
				N:  1,
				At: "2026-10-24T10:15:00+07:00",
				By: domain.ByUser,
				Change: []domain.Change{
					{
						Target: domain.TargetScenario,
						ID:     "order-cancel",
						Reason: "user deleted scenario",
						After:  nil,
					},
				},
				AfterHash: "8e41ab77c9d0",
			},
		},
	}

	track := &domain.TrackFile{
		LastUpdate: 1,
		Input:      domain.InputBoth,
		Sources: domain.Sources{
			OpenAPI: map[string]domain.FileState{
				"auth":  {File: "openapi/auth.yaml", Hash: "111111111111"},
				"order": {File: "openapi/order.yaml", Hash: "222222222222"},
			},
			Requirements: &domain.RequirementState{
				File: "requirements.md",
				Hash: "333333333333",
				Raw:  "line one\nline two\n",
			},
		},
		APIs: map[string]domain.APIState{
			"POST {auth}/api/login": {
				Hash:   "444444444444",
				Status: domain.StatusActive,
				Sig: domain.APISignature{
					Request:   map[string]string{"body": "LoginRequest"},
					Responses: []int{200, 401},
					Auth:      []string{"none"},
				},
			},
			"POST {order}/api/orders": {
				Hash:   "555555555555",
				Status: domain.StatusRemoved,
				Sig: domain.APISignature{
					Request:   map[string]string{},
					Responses: []int{201},
					Auth:      []string{"bearer"},
				},
			},
		},
	}

	return &domain.FileSet{Suite: suite, Log: log, Track: track}
}

func TestStore_LoadMissingFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")

	files, err := NewStore().Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if files == nil {
		t.Fatal("Load() files = nil, want non-nil FileSet")
	}
	if files.Suite != nil || files.Log != nil || files.Track != nil {
		t.Errorf("Load() = (%v, %v, %v), want all nil", files.Suite, files.Log, files.Track)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Load() created %q; Stat error = %v, want not-exist", dir, err)
	}
}

func TestStore_RoundTripAllFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewStore()
	want := sampleFileSet()

	if err := store.Save(context.Background(), dir, want, domain.WriteOptions{Suite: true, Log: true, Track: true, Mode: "initial"}); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got, err := store.Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(got.Suite, want.Suite) {
		t.Errorf("Suite round trip mismatch:\n got: %#v\nwant: %#v", got.Suite, want.Suite)
	}
	if !reflect.DeepEqual(got.Log, want.Log) {
		t.Errorf("Log round trip mismatch:\n got: %#v\nwant: %#v", got.Log, want.Log)
	}
	if !reflect.DeepEqual(got.Track, want.Track) {
		t.Errorf("Track round trip mismatch:\n got: %#v\nwant: %#v", got.Track, want.Track)
	}

	suiteRaw := readFile(t, filepath.Join(dir, SuiteFileName))
	logRaw := readFile(t, filepath.Join(dir, LogFileName))
	trackRaw := readFile(t, filepath.Join(dir, TrackFileName))

	if !strings.HasPrefix(suiteRaw, "version: 1") {
		t.Errorf("%s does not begin with %q:\n%s", SuiteFileName, "version: 1", suiteRaw)
	}
	if strings.HasPrefix(logRaw, "version:") {
		t.Errorf("%s must not have a top-level version field:\n%s", LogFileName, logRaw)
	}
	if strings.HasPrefix(trackRaw, "version:") {
		t.Errorf("%s must not have a top-level version field:\n%s", TrackFileName, trackRaw)
	}

	for _, name := range []string{SuiteFileName, LogFileName, TrackFileName} {
		fi, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("Stat(%s) error = %v", name, err)
		}
		if fi.Mode().Perm() != 0o644 {
			t.Errorf("%s mode = %04o, want 0644", name, fi.Mode().Perm())
		}
	}
	assertNoMarker(t, dir)
}

func TestStore_LoadRejectsOldSuiteSchema(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "different version",
			yaml: "version: 2\nservers: []\napis: []\nscenarios: []\n",
		},
		{
			name: "missing version",
			yaml: "servers: []\napis: []\nscenarios: []\n",
		},
		{
			name: "legacy metadata and base",
			yaml: "metadata:\n  generated_at: 2026-10-20\nbase:\n  at: 2026-10-20\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, SuiteFileName)
			writeFile(t, path, tc.yaml)

			_, err := NewStore().Load(context.Background(), dir)
			if err == nil {
				t.Fatal("Load() error = nil, want schema error")
			}
			if !strings.Contains(err.Error(), "expected version: 1") {
				t.Errorf("Load() error = %q, want it to contain %q", err, "expected version: 1")
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("Load() error = %q, want it to name %q", err, path)
			}
		})
	}
}

func TestStore_LoadPendingMarker(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, PendingMarkerName)
	writeFile(t, markerPath, `{"mode":"update","at":"2026-10-24T10:00:00Z","suite_hash":"aaaaaaaaaaaa","log_hash":"bbbbbbbbbbbb","track_hash":"cccccccccccc"}`+"\n")
	writeFile(t, filepath.Join(dir, SuiteFileName), "metadata: legacy\n")

	files, err := NewStore().Load(context.Background(), dir)
	if err == nil {
		t.Fatal("Load() error = nil, want pending-marker error")
	}
	if files != nil {
		t.Errorf("Load() files = %#v, want nil when a pending marker exists", files)
	}
	if !strings.Contains(err.Error(), markerPath) {
		t.Errorf("Load() error = %q, want it to name marker %q", err, markerPath)
	}
	for _, name := range []string{SuiteFileName, LogFileName, TrackFileName} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("Load() error = %q, want it to mention %q", err, name)
		}
	}
	if !strings.Contains(err.Error(), "remove") {
		t.Errorf("Load() error = %q, want guidance to remove the marker", err)
	}
}

func TestStore_SaveAbortsOnPendingMarker(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, PendingMarkerName)
	if err := os.WriteFile(marker, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := NewStore().Save(context.Background(), dir, sampleFileSet(), domain.WriteOptions{
		Suite: true, Log: true, Track: true, Mode: "initial",
	})
	if err == nil || !strings.Contains(err.Error(), PendingMarkerName) {
		t.Fatalf("Save() error = %v, want pending marker", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, SuiteFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("Save() wrote suite despite marker: %v", statErr)
	}
}

func TestStore_SaveRespectsWriteFlags(t *testing.T) {
	dir := t.TempDir()
	want := sampleFileSet()

	if err := NewStore().Save(context.Background(), dir, want, domain.WriteOptions{Track: true, Mode: "sync"}); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	if _, err := os.Stat(filepath.Join(dir, TrackFileName)); err != nil {
		t.Errorf("Stat(%s) error = %v, want file", TrackFileName, err)
	}
	for _, name := range []string{SuiteFileName, LogFileName} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s was written despite its flag being false; Stat error = %v", name, err)
		}
	}
	assertNoMarker(t, dir)

	got, err := NewStore().Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.Suite != nil || got.Log != nil {
		t.Errorf("Load() Suite/Log = %v/%v, want nil/nil", got.Suite, got.Log)
	}
	if !reflect.DeepEqual(got.Track, want.Track) {
		t.Errorf("round-tripped Track mismatch:\n got: %#v\nwant: %#v", got.Track, want.Track)
	}
}

func TestStore_SaveNoWritesDoesNothing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")

	if err := NewStore().Save(context.Background(), dir, sampleFileSet(), domain.WriteOptions{}); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Save() with no flags touched %q; Stat error = %v, want not-exist", dir, err)
	}
}

func TestStore_SaveFailureLeavesMarker(t *testing.T) {
	dir := t.TempDir()
	// A directory at the log destination makes the temp-file rename fail after
	// the suite has already been written.
	if err := os.Mkdir(filepath.Join(dir, LogFileName), 0o755); err != nil {
		t.Fatalf("Mkdir(%s) error = %v", LogFileName, err)
	}

	err := NewStore().Save(context.Background(), dir, sampleFileSet(), domain.WriteOptions{Suite: true, Log: true, Track: true, Mode: "initial"})
	if err == nil {
		t.Fatal("Save() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), LogFileName) {
		t.Errorf("Save() error = %q, want it to name %q", err, LogFileName)
	}
	if _, statErr := os.Stat(filepath.Join(dir, PendingMarkerName)); statErr != nil {
		t.Errorf("pending marker missing after failed Save: %v", statErr)
	}

	files, loadErr := NewStore().Load(context.Background(), dir)
	if loadErr == nil {
		t.Fatal("Load() after failed Save = nil error, want abort on pending marker")
	}
	if files != nil {
		t.Errorf("Load() after failed Save = %#v, want nil", files)
	}
	if !strings.Contains(loadErr.Error(), PendingMarkerName) {
		t.Errorf("Load() error = %q, want it to name the pending marker", loadErr)
	}
}

func TestStore_PendingMarkerContents(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, LogFileName), 0o755); err != nil {
		t.Fatalf("Mkdir(%s) error = %v", LogFileName, err)
	}
	want := sampleFileSet()

	if err := NewStore().Save(context.Background(), dir, want, domain.WriteOptions{Suite: true, Log: true, Track: true, Mode: "initial"}); err == nil {
		t.Fatal("Save() error = nil, want failure so the marker survives")
	}

	raw, err := os.ReadFile(filepath.Join(dir, PendingMarkerName))
	if err != nil {
		t.Fatalf("ReadFile(marker) error = %v", err)
	}
	var marker struct {
		Mode      string `json:"mode"`
		At        string `json:"at"`
		SuiteHash string `json:"suite_hash"`
		LogHash   string `json:"log_hash"`
		TrackHash string `json:"track_hash"`
	}
	if err := json.Unmarshal(raw, &marker); err != nil {
		t.Fatalf("marker is not valid JSON: %v\n%s", err, raw)
	}
	if marker.Mode != "initial" {
		t.Errorf("marker mode = %q, want %q", marker.Mode, "initial")
	}
	if _, err := time.Parse(time.RFC3339, marker.At); err != nil {
		t.Errorf("marker at = %q is not RFC3339: %v", marker.At, err)
	}

	suiteData, err := encodeSuite(want.Suite)
	if err != nil {
		t.Fatalf("encodeSuite() error = %v", err)
	}
	logData, err := encodeLog(want.Log)
	if err != nil {
		t.Fatalf("encodeLog() error = %v", err)
	}
	trackData, err := encodeTrack(want.Track)
	if err != nil {
		t.Fatalf("encodeTrack() error = %v", err)
	}
	checks := []struct {
		name string
		got  string
		want string
	}{
		{"suite_hash", marker.SuiteHash, hash12(suiteData)},
		{"log_hash", marker.LogHash, hash12(logData)},
		{"track_hash", marker.TrackHash, hash12(trackData)},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("marker %s = %q, want %q", check.name, check.got, check.want)
		}
		if len(check.got) != 12 {
			t.Errorf("marker %s length = %d, want 12", check.name, len(check.got))
		}
	}
}

func TestStore_SaveRemovesMarkerOnSuccess(t *testing.T) {
	dir := t.TempDir()

	if err := NewStore().Save(context.Background(), dir, sampleFileSet(), domain.WriteOptions{Suite: true, Log: true, Track: true, Mode: "initial"}); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	assertNoMarker(t, dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Errorf("leftover temp file %q after Save()", entry.Name())
		}
	}
	if len(entries) != 3 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("output dir contains %v, want exactly the three artifacts", names)
	}
}

func TestStore_LoadInvalidLogIncludesPath(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, LogFileName)
	writeFile(t, logPath, "base:\n  at: now\n  by: ai\n  unexpected: true\n")

	_, err := NewStore().Load(context.Background(), dir)
	if err == nil {
		t.Fatal("Load() error = nil, want strict-decode error")
	}
	if !strings.Contains(err.Error(), logPath) {
		t.Errorf("Load() error = %q, want it to name %q", err, logPath)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func assertNoMarker(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, PendingMarkerName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s still exists; Stat error = %v, want not-exist", PendingMarkerName, err)
	}
}
