// Package generate implements the `spectest generate` CLI subcommand, which
// creates the spectest suite, history log, and track files from one or more
// OpenAPI documents and optional raw requirements.
package generate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nguyentin05/spectest/internal/adapter/cli/cmdutil"
	"github.com/nguyentin05/spectest/internal/infrastructure/sources"
	usecasegenerate "github.com/nguyentin05/spectest/internal/usecase/generate"
	"github.com/spf13/cobra"
)

// options holds the parsed flag values for a single `spectest generate` invocation.
type options struct {
	apiValues []string
	reqPath   string
	outputDir string
	mode      string
	dryRun    bool
}

// NewCommand creates the `spectest generate` subcommand. Only init mode is
// implemented: the command validates flags, confirms before overwriting
// existing suite files, resolves an LLM provider, and writes a new suite.
// Update mode is recognized but reports that it is not implemented.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	opts := &options{outputDir: "./tests", mode: "init"}

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate a new SpecTest suite from OpenAPI and requirements",
		GroupID: "core",
		Args:    cobra.NoArgs,
		Long: `Generate a new spectest.yaml suite from one or more OpenAPI documents and optional
raw requirements. The command manages spectest.yaml, spectest.log.yaml, and
spectest.track.yaml together. If any of these files already exist, you must confirm
before they are overwritten.

Only init mode is implemented. --mode update is reserved and currently reports that
it is not implemented yet, without reading or writing any files.`,
		Example: `  spectest generate --mode init --api auth=openapi-auth.yaml --api order=openapi-order.yaml
  spectest generate --mode init -a auth=openapi.yaml -r requirements.md -o ./tests --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd, opts, f)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.apiValues, "api", "a", nil,
		"OpenAPI document as <server>=<path> (repeatable; order defines servers)")
	cmd.Flags().StringVarP(&opts.reqPath, "req", "r", "",
		"Path to raw business requirements (markdown)")
	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", opts.outputDir,
		"Directory containing spectest.yaml, spectest.log.yaml, and spectest.track.yaml")
	cmd.Flags().StringVar(&opts.mode, "mode", opts.mode,
		"Generation mode: init or update")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false,
		"Run the full pipeline but write no files")

	return cmd
}

func validateMode(mode string) error {
	switch mode {
	case "init":
		return nil
	case "update":
		return errors.New("update mode is not implemented yet; use --mode init")
	default:
		return fmt.Errorf("invalid --mode %q: must be init or update", mode)
	}
}

func runGenerate(cmd *cobra.Command, opts *options, f *cmdutil.Factory) error {
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	if err := validateMode(opts.mode); err != nil {
		// The update stub is an expected runtime outcome, not a usage error.
		if opts.mode == "update" {
			cmd.SilenceUsage = true
		}
		return err
	}
	if len(opts.apiValues) == 0 {
		return fmt.Errorf("required flag --api (-a) is missing")
	}
	inputs, err := parseAPIFlags(opts.apiValues)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.outputDir) == "" {
		return fmt.Errorf("--output must not be empty")
	}

	// Declining the prompt and aborting on a pending marker are expected
	// outcomes, so suppress Cobra's usage block from here on.
	cmd.SilenceUsage = true
	if err := guardOverwrite(
		cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), opts.outputDir, opts.dryRun,
	); err != nil {
		return err
	}

	llmSvc, err := f.LLMService()
	if err != nil {
		return err
	}

	loader := sources.NewLoader()
	gen := usecasegenerate.NewGenerator(llmSvc, loader, f.SuiteStore(), realClock{})

	res, err := gen.Execute(cmd.Context(), usecasegenerate.Request{
		APIs:             inputs,
		RequirementsPath: opts.reqPath,
		OutputDir:        opts.outputDir,
		DryRun:           opts.dryRun,
		Concurrency:      1,
	})
	if err != nil {
		return err
	}
	printSummary(cmd.OutOrStdout(), res)
	return nil
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
