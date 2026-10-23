// Package generate implements the application usecase for converting OpenAPI specifications
// and Markdown PRD documents into executable `spectest.yaml` test suites and hybrid datasets.
package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nguyentin05/spectest/internal/domain"
	"github.com/nguyentin05/spectest/internal/infrastructure/openapi"
	"github.com/nguyentin05/spectest/internal/infrastructure/prd"
	"github.com/nguyentin05/spectest/internal/infrastructure/storage"
	"gopkg.in/yaml.v3"
)

// Request holds the input parameters for executing the `spectest generate` workflow.
type Request struct {
	APIPath   string
	ReqPath   string
	OutputDir string
	ExportFmt string
	Enrich    bool
	// Concurrency caps the number of domain chunks processed in parallel by the
	// LLM scenario synthesis stage. Values <= 0 default to 1, and values above
	// the number of domain chunks are clamped to the chunk count.
	Concurrency int
}

// Result contains the generated TestSuiteFile artifact, output file paths, and
// preprocessing telemetry produced by a completed generation run.
type Result struct {
	Suite            *domain.TestSuiteFile
	SuitePath        string
	MatrixPath       string
	ExportPath       string
	CompressionRatio float64
	ChunksProcessed  int
	AuthProducer     string
}

// Generator coordinates the end-to-end test suite generation pipeline across
// OpenAPI preprocessing, PRD slicing, concurrent LLM scenario synthesis, and hybrid data expansion.
type Generator struct {
	llm  domain.LLMService
	repo *storage.YAMLRepository
}

// NewGenerator constructs a Generator wired with the given LLMService and YAMLRepository.
func NewGenerator(llmSvc domain.LLMService, repo *storage.YAMLRepository) *Generator {
	if repo == nil {
		repo = storage.NewYAMLRepository()
	}
	return &Generator{
		llm:  llmSvc,
		repo: repo,
	}
}

// Execute runs the 6-stage test generation pipeline:
//  1. Prune OpenAPI metadata and resolve `$ref` schemas at zero token cost.
//  2. Detect authentication producer and consumer endpoints via security schemes and regex.
//  3. Partition normalized endpoints into domain chunks by OpenAPI tags.
//  4. Slice the optional Markdown PRD and pair matching sections with each domain chunk.
//  5. Invoke the configured LLM concurrently across chunks to synthesize stateful scenarios.
//  6. Augment mutating steps with deterministic Go fuzzing/security cases and write artifacts.
//
// Execute fails fast and returns an error if the LLM service is unconfigured, unreachable,
// or returns invalid scenario JSON.
func (g *Generator) Execute(ctx context.Context, req Request) (*Result, error) {
	if g.llm == nil {
		return nil, fmt.Errorf("llm service is required to generate test scenarios")
	}

	rawSpec, err := os.ReadFile(req.APIPath)
	if err != nil {
		return nil, fmt.Errorf("read openapi file %s: %w", req.APIPath, err)
	}

	pruned, err := openapi.PruneSpec(rawSpec)
	if err != nil {
		return nil, err
	}

	authProducer := ""
	for _, op := range pruned.Operations {
		if op.IsTokenProducer {
			authProducer = fmt.Sprintf("%s %s", op.Method, op.Endpoint)
			break
		}
	}

	chunks := openapi.ChunkByDomain(pruned.Operations, 5)

	prdSections, err := prd.SliceMarkdownFile(req.ReqPath)
	if err != nil {
		return nil, fmt.Errorf("read prd file %s: %w", req.ReqPath, err)
	}
	for i := range chunks {
		chunks[i].PRDSection = prd.MatchSectionToTag(prdSections, chunks[i].Tag)
	}

	scenarios, err := g.buildScenariosConcurrently(ctx, chunks, req.Concurrency)
	if err != nil {
		return nil, err
	}

	suiteName := pruned.Title
	if suiteName == "" {
		suiteName = "SpecTest Automated Test Suite"
	}

	suite := &domain.TestSuiteFile{
		Version: "1.0",
		Metadata: domain.SuiteMetadata{
			Name:          suiteName,
			TargetBaseURL: pruned.BaseURL,
		},
		Base: domain.BaseSuite{
			Scenarios: scenarios,
		},
	}

	outDir := req.OutputDir
	if strings.TrimSpace(outDir) == "" {
		outDir = "./tests"
	}

	suitePath := filepath.Join(outDir, "spectest.yaml")
	if err := g.repo.Save(suitePath, suite); err != nil {
		return nil, err
	}

	matrixPath := filepath.Join(outDir, "test_matrix.md")
	if err := g.repo.ExportTestMatrix(matrixPath, suite); err != nil {
		return nil, err
	}

	exportPath := ""
	if strings.EqualFold(req.ExportFmt, "postman") {
		exportPath = filepath.Join(outDir, "postman_collection.json")
		if err := g.repo.ExportPostmanCollection(exportPath, suite); err != nil {
			return nil, err
		}
	}

	return &Result{
		Suite:            suite,
		SuitePath:        suitePath,
		MatrixPath:       matrixPath,
		ExportPath:       exportPath,
		CompressionRatio: pruned.CompressionRatio,
		ChunksProcessed:  len(chunks),
		AuthProducer:     authProducer,
	}, nil
}

// buildScenariosConcurrently synthesizes scenarios for domain chunks using a bounded worker
// pool that runs at most `concurrency` chunks in parallel. Results keep deterministic order by
// chunk index regardless of completion order. The first chunk error cancels all remaining work
// (peers and queued jobs) and all concurrent errors are joined into a single error;
// cancellation noise from peer workers is not surfaced.
func (g *Generator) buildScenariosConcurrently(ctx context.Context, chunks []openapi.DomainChunk, concurrency int) ([]domain.Scenario, error) {
	if len(chunks) == 0 {
		return nil, nil
	}
	concurrency = effectiveConcurrency(concurrency, len(chunks))

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan int)
	results := make([]domain.Scenario, len(chunks))
	errs := make([]error, len(chunks))

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if runCtx.Err() != nil {
					return
				}
				ch := chunks[idx]
				ops := ch.Operations
				if ch.TokenProducer != nil {
					ops = append([]openapi.NormalizedOperation{*ch.TokenProducer}, ops...)
				}
				sc, err := g.generateChunkScenario(runCtx, ch.Tag, ops, ch.PRDSection)
				if err != nil {
					if runCtx.Err() != nil && errors.Is(err, context.Canceled) {
						return
					}
					errs[idx] = fmt.Errorf("domain %q: %w", ch.Tag, err)
					cancel()
					return
				}
				results[idx] = sc
				slog.Debug("chunk generated", "tag", ch.Tag, "index", idx+1, "total", len(chunks))
			}
		}()
	}

feed:
	for i := range chunks {
		select {
		case jobs <- i:
		case <-runCtx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// effectiveConcurrency clamps the requested worker count between 1 and the number of
// chunks: values <= 0 default to 1, and values above the chunk count are clamped to
// the chunk count. When there are no chunks, it returns 0.
func effectiveConcurrency(requested, chunks int) int {
	if chunks <= 0 {
		return 0
	}
	if requested < 1 {
		requested = 1
	}
	if requested > chunks {
		requested = chunks
	}
	return requested
}

// generateChunkScenario prompts the configured LLM with a single domain chunk's pruned
// operations and matched PRD requirements, unmarshals the returned JSON into a domain.Scenario,
// and enriches mutating steps with deterministic Go fuzzing and security test cases.
func (g *Generator) generateChunkScenario(ctx context.Context, tag string, ops []openapi.NormalizedOperation, prdText string) (domain.Scenario, error) {
	opsYAML, err := yaml.Marshal(ops)
	if err != nil {
		return domain.Scenario{}, fmt.Errorf("marshal operations for tag %q: %w", tag, err)
	}

	sysPrompt := scenarioSystemPrompt
	userPrompt := buildScenarioUserPrompt(tag, string(opsYAML), prdText)

	rawJSON, err := g.llm.GenerateStructured(ctx, sysPrompt, userPrompt, nil)
	if err != nil {
		return domain.Scenario{}, fmt.Errorf("llm generation failed for domain %q: %w", tag, err)
	}
	if strings.TrimSpace(rawJSON) == "" {
		return domain.Scenario{}, fmt.Errorf("llm returned empty response for domain %q", tag)
	}

	var sc domain.Scenario
	if err := json.Unmarshal([]byte(rawJSON), &sc); err != nil {
		return domain.Scenario{}, fmt.Errorf("parse llm scenario json for domain %q: %w", tag, err)
	}
	if len(sc.Steps) == 0 {
		return domain.Scenario{}, fmt.Errorf("llm scenario for domain %q contains no steps", tag)
	}

	for i := range sc.Steps {
		if matchedOp, found := findMatchingOp(sc.Steps[i], ops); found {
			EnrichStepWithHybridData(&sc.Steps[i], matchedOp)
		}
	}
	return sc, nil
}

// findMatchingOp locates the NormalizedOperation corresponding to a generated Step
// by comparing TargetRef or Request method and endpoint paths.
func findMatchingOp(step domain.Step, ops []openapi.NormalizedOperation) (openapi.NormalizedOperation, bool) {
	for _, op := range ops {
		if step.TargetRef != nil && strings.EqualFold(step.TargetRef.Endpoint, op.Endpoint) && strings.EqualFold(step.TargetRef.Method, op.Method) {
			return op, true
		}
		if strings.EqualFold(step.Request.Endpoint, op.Endpoint) && strings.EqualFold(step.Request.Method, op.Method) {
			return op, true
		}
	}
	return openapi.NormalizedOperation{}, false
}
