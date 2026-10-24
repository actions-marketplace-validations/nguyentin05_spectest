package domain

import (
	"context"
	"time"
)

// LLMService defines the outbound port for interacting with Large Language Model providers.
// Implementations communicate over the standard Chat Completions protocol to synthesize
// structured test scenarios from normalized APIs and raw requirements.
type LLMService interface {
	// Name returns a human-readable identifier for the active provider, model, and endpoint.
	Name() string

	// GenerateStructured sends a system instruction and user prompt to the configured model
	// and returns the raw JSON payload extracted from the completion response.
	GenerateStructured(ctx context.Context, systemPrompt string, userPrompt string, responseSchema interface{}) (string, error)
}

// CredentialStore defines the configuration port for resolving and persisting LLM provider
// credentials, base endpoints, and model selections across environment variables and config files.
type CredentialStore interface {
	// GetAPIKey resolves the API key for the specified provider.
	GetAPIKey(provider string) (string, error)

	// SetAPIKey persists the API key for the specified provider with restricted (0600) permissions.
	SetAPIKey(provider, key string) error

	// GetEndpoint resolves the base API endpoint URL for the specified provider.
	GetEndpoint(provider string) (string, error)

	// SetEndpoint persists the base API endpoint URL for the specified provider.
	SetEndpoint(provider, endpoint string) error
}

// FileSet groups the three state files managed by the generate pipeline.
// A nil member means that file does not exist.
type FileSet struct {
	Suite *Suite
	Log   *LogFile
	Track *TrackFile
}

// WriteOptions selects which files to persist and carries the run mode for
// diagnostics or the pending marker.
type WriteOptions struct {
	Suite bool
	Log   bool
	Track bool
	Mode  string
}

// SourceLoader loads, normalizes, and fingerprints OpenAPI and requirements inputs.
type SourceLoader interface {
	Load(ctx context.Context, inputs []APIInput, requirementsPath string) (*SourceSet, error)
}

// SuiteStore loads and saves the three state files in an output directory.
type SuiteStore interface {
	Load(ctx context.Context, dir string) (*FileSet, error)
	Save(ctx context.Context, dir string, files *FileSet, opts WriteOptions) error
}

// Clock supplies the current time so history entries are deterministic in tests.
type Clock interface {
	Now() time.Time
}
