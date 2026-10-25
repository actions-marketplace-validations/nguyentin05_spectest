// Package generate implements the `spectest generate` CLI subcommand, which
// creates or incrementally updates the spectest suite, history log, and track
// files from one or more OpenAPI documents and optional raw requirements.
package generate

import (
	"context"
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
	dryRun    bool
}

// NewCommand creates the `spectest generate` subcommand. The pipeline is not
// wired yet: this skeleton validates flags and reports that generation is not
// implemented, without resolving an LLM provider or touching the filesystem.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	opts := &options{outputDir: "./tests"}

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate or update the SpecTest suite from OpenAPI and requirements",
		GroupID: "core",
		Args:    cobra.NoArgs,
		Long: `Generate the spectest.yaml suite from one or more OpenAPI documents and optional
raw requirements, or update it incrementally by comparing against spectest.track.yaml.
The command manages spectest.yaml, spectest.log.yaml, and spectest.track.yaml together.`,
		Example: `  spectest generate --api auth=openapi-auth.yaml --api order=openapi-order.yaml
  spectest generate -a auth=openapi.yaml -r requirements.md -o ./tests --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(opts, f)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.apiValues, "api", "a", nil,
		"OpenAPI document as <server>=<path> (repeatable; order defines servers)")
	cmd.Flags().StringVarP(&opts.reqPath, "req", "r", "",
		"Path to raw business requirements (markdown)")
	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", opts.outputDir,
		"Directory containing spectest.yaml, spectest.log.yaml, and spectest.track.yaml")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false,
		"Run the full pipeline but write no files")

	return cmd
}

func runGenerate(opts *options, f *cmdutil.Factory) error {
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

	llmSvc, err := f.LLMService()
	if err != nil {
		return err
	}

	loader := sources.NewLoader()
	gen := usecasegenerate.NewGenerator(llmSvc, loader, f.SuiteStore(), realClock{})

	_, err = gen.Execute(context.Background(), usecasegenerate.Request{
		APIs:             inputs,
		RequirementsPath: opts.reqPath,
		OutputDir:        opts.outputDir,
		DryRun:           opts.dryRun,
	})
	return err
}


type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
