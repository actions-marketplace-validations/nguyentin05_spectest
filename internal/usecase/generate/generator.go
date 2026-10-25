package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nguyentin05/spectest/internal/domain"
)

// Generator is the facade for the generate pipeline.
type Generator struct {
	llm     domain.LLMService
	sources domain.SourceLoader
	store   domain.SuiteStore
	clock   domain.Clock
}

func NewGenerator(
	llm domain.LLMService,
	sources domain.SourceLoader,
	store domain.SuiteStore,
	clock domain.Clock,
) *Generator {
	return &Generator{llm: llm, sources: sources, store: store, clock: clock}
}

// Request is the input to Execute.
type Request struct {
	APIs             []domain.APIInput
	RequirementsPath string
	OutputDir        string
	DryRun           bool
}

// Result is what the CLI prints.
type Result struct {
	Mode           Mode
	Servers        []string
	APITotal       int
	APIAdded       int
	APIChanged     int
	APIRemoved     int
	ManualEdits    int
	Warnings       []string
	SuitePath      string
	LogPath        string
	TrackPath      string
	NoFilesWritten bool
}

// Execute runs the pipeline for a first run (initial).
// Flow: load → plan → synthesize → validate → persist.
func (g *Generator) Execute(ctx context.Context, req Request) (*Result, error) {
	// 1. Load sources.
	src, err := g.sources.Load(ctx, req.APIs, req.RequirementsPath)
	if err != nil {
		return nil, fmt.Errorf("load sources: %w", err)
	}

	fmt.Println("=== Normalized APIs ===")
	for i, api := range src.APIs {
		fmt.Printf("[%d] %s\n", i+1, api.Key)
	}
	fmt.Printf("Total: %d APIs across %d servers (%v)\n\n",
		len(src.APIs), len(src.Servers), src.Servers)

	if g.llm == nil {
		return nil, fmt.Errorf("llm service is not configured")
	}

	// 2. Plan initial.
	fmt.Printf("=== Plan Initial (llm=%s) ===\n", g.llm.Name())
	plan, err := g.planInitial(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("plan initial: %w", err)
	}
	pretty, _ := json.MarshalIndent(plan, "", "  ")
	fmt.Println(string(pretty))
	fmt.Printf("\nPlan summary: %d setups, %d scenarios | validation PASSED\n\n",
		len(plan.Setups), len(plan.Scenarios))

	// 3. Synthesize.
	fmt.Println("=== Synthesize (setup → scenario) ===")
	suite, err := g.synthesizeInitial(ctx, src, plan)
	if err != nil {
		return nil, fmt.Errorf("synthesize: %w", err)
	}

	// 4. Validate.
	fmt.Println("=== Validate Suite ===")
	vr := validateSuite(suite, src, validationOptions{
		RequireAnalysis:  true,
		RequiredCoverage: allAPIKeys(src),
		AllowStaleAPIs:   false,
		WarnUncovered:    true,
	})
	if err := vr.Err(); err != nil {
		for _, e := range vr.Errors {
			fmt.Printf("  ✗ %v\n", e)
		}
		return nil, fmt.Errorf("validate: %w", err)
	}
	fmt.Println("Validation PASSED")
	for _, w := range vr.Warnings {
		fmt.Printf("  ⚠ %s\n", w)
	}
	fmt.Printf("\nSynthesis summary: %d setups, %d scenarios\n",
		len(suite.Setup), len(suite.Scenarios))
	fmt.Printf("Suite hash: %s\n\n", suiteHash(suite))

	fmt.Println("=== Assembled Suite (JSON) ===")
	suiteJSON, _ := json.MarshalIndent(suite, "", "  ")
	fmt.Println(string(suiteJSON))
	fmt.Println()

	// 5. Build file set (normalize + base entry + track).
	files, err := g.buildInitialFileSet(src, suite)
	if err != nil {
		return nil, fmt.Errorf("build files: %w", err)
	}

	// 6. Persist.
	fmt.Println("=== Persist ===")
	writeOpts := domain.WriteOptions{Suite: true, Log: true, Track: true}
	if err := g.persist(ctx, req, files, writeOpts); err != nil {
		return nil, fmt.Errorf("persist: %w", err)
	}

	suitePath := joinDisplayPath(req.OutputDir, "spectest.yaml")
	logPath := joinDisplayPath(req.OutputDir, "spectest.log.yaml")
	trackPath := joinDisplayPath(req.OutputDir, "spectest.track.yaml")

	if req.DryRun {
		fmt.Println("  [dry-run] no files written")
	} else {
		fmt.Printf("  ✓ %s\n", suitePath)
		fmt.Printf("  ✓ %s\n", logPath)
		fmt.Printf("  ✓ %s\n", trackPath)
	}

	mode := ModeInitial
	if req.DryRun {
		mode = ModeDryRun
	}

	return &Result{
		Mode:           mode,
		Servers:        src.Servers,
		APITotal:       len(src.APIs),
		Warnings:       vr.Warnings,
		SuitePath:      suitePath,
		LogPath:        logPath,
		TrackPath:      trackPath,
		NoFilesWritten: req.DryRun,
	}, nil
}

// realClock is a temporary domain.Clock implementation.
// Sẽ được thay bằng internal/platform/clock khi cần.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
