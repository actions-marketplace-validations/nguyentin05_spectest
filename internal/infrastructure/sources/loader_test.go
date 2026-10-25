package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nguyentin05/spectest/internal/domain"
)

// writeFile writes content to a new file in dir and returns its absolute path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// openAPIDoc builds a minimal OpenAPI 3.x document from alternating
// path/method pairs, e.g. openAPIDoc("/users", "get", "/users", "post").
func openAPIDoc(ops ...string) string {
	var b strings.Builder
	b.WriteString("openapi: 3.0.3\ninfo:\n  title: test\n  version: \"1\"\npaths:\n")
	for i := 0; i+1 < len(ops); i += 2 {
		fmt.Fprintf(&b, "  %s:\n    %s:\n      responses:\n        \"200\":\n          description: ok\n", ops[i], ops[i+1])
	}
	return b.String()
}

// cancelAfterContext reports cancellation only after the first Err check, so a
// test can prove the loader stops between files instead of before reading any.
type cancelAfterContext struct {
	context.Context
	checks int
	after  int
}

func (c *cancelAfterContext) Err() error {
	c.checks++
	if c.checks > c.after {
		return context.Canceled
	}
	return nil
}

func TestLoad_TwoServersPreserveServerOrderAndSortAPIs(t *testing.T) {
	dir := t.TempDir()
	betaPath := writeFile(t, dir, "beta.yaml", openAPIDoc("/z", "get", "/a", "post"))
	alphaPath := writeFile(t, dir, "alpha.yaml", openAPIDoc("/m", "put", "/b", "get"))

	set, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "beta", Path: betaPath},
		{Server: "alpha", Path: alphaPath},
	}, "")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	wantServers := []string{"beta", "alpha"}
	if !reflect.DeepEqual(set.Servers, wantServers) {
		t.Errorf("Servers = %v, want %v (input order)", set.Servers, wantServers)
	}

	var got []string
	for _, api := range set.APIs {
		got = append(got, api.Key)
	}
	want := []string{
		"GET {alpha}/b",
		"PUT {alpha}/m",
		"POST {beta}/a",
		"GET {beta}/z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("APIs = %v, want %v (sorted by server, path, method)", got, want)
	}

	if state, ok := set.OpenAPI["beta"]; !ok || state.File != betaPath {
		t.Errorf("OpenAPI[beta] = %+v, want File %q", state, betaPath)
	}
	if state, ok := set.OpenAPI["alpha"]; !ok || state.File != alphaPath {
		t.Errorf("OpenAPI[alpha] = %+v, want File %q", state, alphaPath)
	}
}

func TestLoad_ComputesRawSourceHash(t *testing.T) {
	dir := t.TempDir()
	base := openAPIDoc("/users", "get")
	withComment := "# added comment\n" + base
	withSpaces := strings.Replace(base, "description: ok", "description: ok  ", 1)

	basePath := writeFile(t, dir, "base.yaml", base)
	commentPath := writeFile(t, dir, "comment.yaml", withComment)
	spacePath := writeFile(t, dir, "space.yaml", withSpaces)

	set, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "base", Path: basePath},
		{Server: "comment", Path: commentPath},
		{Server: "space", Path: spacePath},
	}, "")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	sum := sha256.Sum256([]byte(base))
	want := hex.EncodeToString(sum[:])[:12]
	if got := set.OpenAPI["base"].Hash; got != want {
		t.Errorf("OpenAPI[base].Hash = %q, want raw sha256 first 12 hex %q", got, want)
	}
	if set.OpenAPI["base"].Hash == set.OpenAPI["comment"].Hash {
		t.Errorf("hash did not change for an added comment: %q", set.OpenAPI["base"].Hash)
	}
	if set.OpenAPI["base"].Hash == set.OpenAPI["space"].Hash {
		t.Errorf("hash did not change for changed whitespace: %q", set.OpenAPI["base"].Hash)
	}
}

func TestLoad_RequirementsRawExact(t *testing.T) {
	dir := t.TempDir()
	apiPath := writeFile(t, dir, "api.yaml", openAPIDoc("/users", "get"))

	raw := "first line  \r\nsecond\t\r\n\r\nthird line   "
	reqPath := writeFile(t, dir, "requirements.md", raw)

	set, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "api", Path: apiPath},
	}, reqPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if set.Requirements == nil {
		t.Fatal("Requirements = nil, want raw source")
	}
	if set.Requirements.File != reqPath {
		t.Errorf("Requirements.File = %q, want %q", set.Requirements.File, reqPath)
	}
	if set.Requirements.Raw != raw {
		t.Errorf("Requirements.Raw = %q, want verbatim %q", set.Requirements.Raw, raw)
	}
}

func TestLoad_NoRequirements(t *testing.T) {
	dir := t.TempDir()
	apiPath := writeFile(t, dir, "api.yaml", openAPIDoc("/users", "get"))

	set, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "api", Path: apiPath},
	}, "")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if set.Requirements != nil {
		t.Errorf("Requirements = %+v, want nil", set.Requirements)
	}
}

func TestLoad_MissingFileErrorNamesPath(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.yaml")

	t.Run("api", func(t *testing.T) {
		_, err := NewLoader().Load(context.Background(), []domain.APIInput{
			{Server: "api", Path: missing},
		}, "")
		if err == nil {
			t.Fatal("Load() error = nil, want missing-file error")
		}
		if !strings.Contains(err.Error(), missing) {
			t.Errorf("Load() error = %v, want it to name %q", err, missing)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Load() error = %v, want it to wrap fs.ErrNotExist", err)
		}
	})

	t.Run("requirements", func(t *testing.T) {
		apiPath := writeFile(t, dir, "api.yaml", openAPIDoc("/users", "get"))
		_, err := NewLoader().Load(context.Background(), []domain.APIInput{
			{Server: "api", Path: apiPath},
		}, missing)
		if err == nil {
			t.Fatal("Load() error = nil, want missing-file error")
		}
		if !strings.Contains(err.Error(), missing) {
			t.Errorf("Load() error = %v, want it to name %q", err, missing)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Load() error = %v, want it to wrap fs.ErrNotExist", err)
		}
	})
}

func TestLoad_InvalidServerName(t *testing.T) {
	dir := t.TempDir()
	apiPath := writeFile(t, dir, "api.yaml", openAPIDoc("/users", "get"))

	for _, server := range []string{"", "Bad", "1bad", "has space", "with.dot"} {
		t.Run(server, func(t *testing.T) {
			_, err := NewLoader().Load(context.Background(), []domain.APIInput{
				{Server: server, Path: apiPath},
			}, "")
			if err == nil {
				t.Fatalf("Load() error = nil for server %q, want rejection", server)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%q", server)) {
				t.Errorf("Load() error = %v, want it to name server %q", err, server)
			}
		})
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	_, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "api", Path: ""},
	}, "")
	if err == nil {
		t.Fatal("Load() error = nil, want empty-path rejection")
	}
	if !strings.Contains(err.Error(), "api") {
		t.Errorf("Load() error = %v, want it to name server %q", err, "api")
	}
}

func TestLoad_DuplicateServer(t *testing.T) {
	dir := t.TempDir()
	first := writeFile(t, dir, "first.yaml", openAPIDoc("/a", "get"))
	second := writeFile(t, dir, "second.yaml", openAPIDoc("/b", "get"))

	_, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "api", Path: first},
		{Server: "api", Path: second},
	}, "")
	if err == nil {
		t.Fatal("Load() error = nil, want duplicate-server rejection")
	}
	if !strings.Contains(err.Error(), `"api"`) {
		t.Errorf("Load() error = %v, want it to name server %q", err, "api")
	}
}

func TestLoad_WrapsOpenAPIErrorWithServer(t *testing.T) {
	dir := t.TempDir()
	swagger := "swagger: \"2.0\"\ninfo:\n  title: legacy\n  version: \"1\"\npaths:\n  /users:\n    get:\n      responses:\n        \"200\":\n          description: ok\n"
	apiPath := writeFile(t, dir, "swagger.yaml", swagger)

	_, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "legacy", Path: apiPath},
	}, "")
	if err == nil {
		t.Fatal("Load() error = nil, want OpenAPI error")
	}
	if !strings.Contains(err.Error(), "legacy") {
		t.Errorf("Load() error = %v, want it to name server %q", err, "legacy")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "swagger") {
		t.Errorf("Load() error = %v, want it to retain the underlying OpenAPI error", err)
	}
}

func TestLoad_EmptyOperationsError(t *testing.T) {
	dir := t.TempDir()
	empty := "openapi: 3.0.3\ninfo:\n  title: test\n  version: \"1\"\npaths:\n  /users: {}\n"
	apiPath := writeFile(t, dir, "empty.yaml", empty)

	_, err := NewLoader().Load(context.Background(), []domain.APIInput{
		{Server: "api", Path: apiPath},
	}, "")
	if err == nil {
		t.Fatal("Load() error = nil, want empty-operations rejection")
	}
	if !strings.Contains(err.Error(), "api") {
		t.Errorf("Load() error = %v, want it to name server %q", err, "api")
	}
}

func TestLoad_ContextCanceled(t *testing.T) {
	dir := t.TempDir()
	first := writeFile(t, dir, "first.yaml", openAPIDoc("/a", "get"))
	missing := filepath.Join(dir, "second.yaml")

	ctx := &cancelAfterContext{Context: context.Background(), after: 1}
	_, err := NewLoader().Load(ctx, []domain.APIInput{
		{Server: "first", Path: first},
		{Server: "second", Path: missing},
	}, "")
	if err == nil {
		t.Fatal("Load() error = nil, want context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Load() error = %v, want it to wrap context.Canceled", err)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load() error = %v, want the next file left unread", err)
	}
	if strings.Contains(err.Error(), missing) {
		t.Errorf("Load() error = %v, want it not to name the unread file %q", err, missing)
	}
}
