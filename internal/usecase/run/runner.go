package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nguyentin05/spectest/internal/domain"
	"github.com/nguyentin05/spectest/internal/infrastructure/server"
	"github.com/nguyentin05/spectest/internal/infrastructure/suitefile"
)

// Options configures a `spectest run` execution.
type Options struct {
	SuitePath   string
	TargetURL   string
	ComposeFile string
	TimeoutSec  int
	ReportDir   string
}

// ExecutionRecord represents the detailed outcome of a single step execution.
type ExecutionRecord struct {
	ScenarioID     string         `json:"scenario_id"`
	ScenarioName   string         `json:"scenario_name"`
	Call           string         `json:"call"`
	Method         string         `json:"method"`
	Endpoint       string         `json:"endpoint"`
	Passed         bool           `json:"passed"`
	StatusCode     int            `json:"status_code"`
	ExpectedCode   int            `json:"expected_code"`
	LatencyMs      int64          `json:"latency_ms"`
	FailReason     string         `json:"fail_reason,omitempty"`
	AIDiagnosis    string         `json:"ai_diagnosis,omitempty"`
	RequestPayload string         `json:"request_payload,omitempty"`
	ResponseBody   string         `json:"response_body,omitempty"`
	Extracted      map[string]any `json:"extracted,omitempty"`
}

// Summary holds the complete test run telemetry and results for reporting.
type Summary struct {
	SuiteName     string            `json:"suite_name"`
	TargetURL     string            `json:"target_url"`
	TotalChecks   int               `json:"total_checks"`
	PassedChecks  int               `json:"passed_checks"`
	FailedChecks  int               `json:"failed_checks"`
	TotalDuration time.Duration     `json:"total_duration"`
	Records       []ExecutionRecord `json:"records"`
}

// LifecycleFactory creates a domain.ServerLifecycleManager for a given compose file, target URL, and timeout.
type LifecycleFactory func(composeFile, targetURL string, timeoutSec int) domain.ServerLifecycleManager

// SuiteLoader loads a test suite from disk.
type SuiteLoader func(path string) (*domain.Suite, error)

// Runner orchestrates the execution lifecycle for API test suites.
type Runner struct {
	loadFn      SuiteLoader
	lifecycleFn LifecycleFactory
	llm         domain.LLMService
}

// NewRunner creates a new Runner with dependencies.
func NewRunner(loadFn SuiteLoader, llmSvc domain.LLMService) *Runner {
	if loadFn == nil {
		loadFn = suitefile.LoadSuite
	}
	return &Runner{
		loadFn: loadFn,
		lifecycleFn: func(composeFile, targetURL string, timeoutSec int) domain.ServerLifecycleManager {
			return server.NewLifecycleManager(composeFile, targetURL, timeoutSec)
		},
		llm: llmSvc,
	}
}

// SetLifecycleFactory allows overriding the lifecycle manager constructor for testing.
func (r *Runner) SetLifecycleFactory(fn LifecycleFactory) {
	r.lifecycleFn = fn
}

// SetSuiteLoader allows overriding the suite loading function for testing.
func (r *Runner) SetSuiteLoader(fn SuiteLoader) {
	r.loadFn = fn
}

// Execute orchestrates the test suite run.
func (r *Runner) Execute(ctx context.Context, opts Options) (*Summary, error) {
	suitePath := strings.TrimSpace(opts.SuitePath)
	if suitePath == "" {
		suitePath = "./tests/spectest.yaml"
	}

	suite, err := r.loadFn(suitePath)
	if err != nil {
		return nil, fmt.Errorf("load test suite %q: %w", suitePath, err)
	}

	targetURL := strings.TrimSpace(opts.TargetURL)
	if targetURL == "" {
		return nil, fmt.Errorf("target URL is required: specify via -t/--target flag")
	}

	normalizedURL, err := server.NormalizeTargetURL(targetURL)
	if err != nil {
		return nil, fmt.Errorf("target URL %q: %w", targetURL, err)
	}
	targetURL = normalizedURL

	timeoutSec := opts.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	// 1. Setup Server Lifecycle (Existing server healthcheck or Docker Compose up)
	lifecycle := r.lifecycleFn(opts.ComposeFile, targetURL, timeoutSec)
	teardown, err := lifecycle.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("setup server environment: %w", err)
	}
	defer teardown()

	// 2. Execute Suite & Evaluate
	summary, err := r.executeSuite(ctx, suite, targetURL)
	if err != nil {
		return nil, fmt.Errorf("execute suite: %w", err)
	}

	return summary, nil
}

func (r *Runner) executeSuite(_ context.Context, suite *domain.Suite, targetURL string) (*Summary, error) {
	startTime := time.Now()
	summary := &Summary{
		SuiteName:     fmt.Sprintf("SpecTest Suite (v%d)", suite.Version),
		TargetURL:     targetURL,
		TotalDuration: time.Since(startTime),
		Records:       make([]ExecutionRecord, 0),
	}

	for _, sc := range suite.Scenarios {
		summary.TotalChecks += len(sc.Steps)
	}

	return summary, nil
}
