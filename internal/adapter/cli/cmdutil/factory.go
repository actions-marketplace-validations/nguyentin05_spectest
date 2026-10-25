// Package cmdutil provides shared dependency wiring for SpecTest CLI subcommands,
// following the `cmdutil.Factory` pattern used by kubectl and the GitHub CLI.
package cmdutil

import (
	"github.com/nguyentin05/spectest/internal/domain"
	"github.com/nguyentin05/spectest/internal/infrastructure/llm"
	"github.com/nguyentin05/spectest/internal/infrastructure/storage"
	"github.com/nguyentin05/spectest/internal/infrastructure/suitefile"
	"github.com/spf13/pflag"
)

// Factory holds global CLI runtime state (such as the `--verbose` flag) and lazily
// constructs infrastructure services shared across subcommands, so each command package
// receives its dependencies through a single injected value instead of global variables.
type Factory struct {
	Verbose bool

	credStore *storage.FileCredentialStore
	suiteStore domain.SuiteStore
}

// NewFactory returns an empty Factory. Services are created on first access.
func NewFactory() *Factory {
	return &Factory{}
}

// NewFactoryWith constructs a Factory with a pre-configured credential store, useful in tests.
func NewFactoryWith(credStore *storage.FileCredentialStore) *Factory {
	return &Factory{credStore: credStore}
}

// BindFlags registers global persistent flags (currently `--verbose`) on the given flag set,
// typically the root command's PersistentFlags.
func (f *Factory) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&f.Verbose, "verbose", false, "Enable verbose debug logging")
}

// CredentialStore returns the shared FileCredentialStore, creating it on first call.
func (f *Factory) CredentialStore() *storage.FileCredentialStore {
	if f.credStore == nil {
		f.credStore = storage.NewFileCredentialStore()
	}
	return f.credStore
}

func (f *Factory) SuiteStore() domain.SuiteStore {
	if f.suiteStore == nil {
		f.suiteStore = suitefile.NewStore()
	}
	return f.suiteStore
}

// LLMService resolves the active provider, model, endpoint, and API key from the credential
// store and environment variables, returning an error if any required setting is missing.
func (f *Factory) LLMService() (domain.LLMService, error) {
	return llm.NewProvider(f.CredentialStore(), "", "", "")
}
