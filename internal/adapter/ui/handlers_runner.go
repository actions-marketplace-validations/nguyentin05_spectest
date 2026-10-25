package ui

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nguyentin05/spectest/internal/domain"
	"github.com/nguyentin05/spectest/internal/infrastructure/server"
	"github.com/nguyentin05/spectest/internal/infrastructure/suitefile"
)

type runStartRequest struct {
	TargetURL string            `json:"target_url"`
	Targets   map[string]string `json:"targets"`
	SuitePath string            `json:"suite_path"`
	Timeout   int               `json:"timeout_seconds"`
}

type runStartResponse struct {
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	TargetURL string `json:"target_url"`
	SuitePath string `json:"suite_path"`
}

// handleRunStart triggers a test execution session in background.
func (s *Server) handleRunStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req runStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	suitePath := strings.TrimSpace(req.SuitePath)
	if suitePath == "" {
		candidates := []string{
			"spectest.yaml",
			"spectest.yml",
			"tests/spectest.yaml",
		}
		for _, c := range candidates {
			if _, err := suitefile.LoadSuite(c); err == nil {
				suitePath = c
				break
			}
		}
		if suitePath == "" {
			suitePath = "spectest.yaml"
		}
	}

	suite, err := suitefile.LoadSuite(suitePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("cannot load suite at %q: %v", suitePath, err))
		return
	}

	targets := make(map[string]string)
	for k, v := range req.Targets {
		normURL := normalizeURL(v)
		if normURL != "" {
			targets[k] = normURL
		}
	}

	defaultTarget := normalizeURL(req.TargetURL)
	if defaultTarget == "" && len(targets) == 0 {
		writeError(w, http.StatusBadRequest, "target URL is required: specify 'target_url' or 'targets' map")
		return
	}

	runID := fmt.Sprintf("run-%x", time.Now().UnixNano()%0x100000000)
	ctx, cancel := context.WithCancel(context.Background())

	sess, err := s.sessionMgr.StartSession(runID, defaultTarget, suitePath, cancel)
	if err != nil {
		cancel()
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	sess.Report.Targets = targets
	s.sessionMgr.AppendLog(fmt.Sprintf("[%s] Test run scheduled against %s", runID, defaultTarget))

	// Launch async execution
	go s.executeTestSuite(ctx, sess, suite, targets, defaultTarget)

	writeJSON(w, http.StatusAccepted, runStartResponse{
		RunID:     runID,
		Status:    "running",
		Message:   "Test execution started",
		TargetURL: defaultTarget,
		SuitePath: suitePath,
	})
}

// handleRunStream provides Server-Sent Events (SSE) for real-time progress.
func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sess := s.sessionMgr.ActiveSession()
	if sess == nil {
		raw, _ := json.Marshal(RunEvent{
			Type:      "idle",
			Timestamp: time.Now(),
			Message:   "No active test run is in progress",
		})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
		flusher.Flush()
		return
	}

	events := sess.Subscribe()
	defer sess.Unsubscribe(events)

	// Send initial status event
	initEvt, _ := json.Marshal(RunEvent{
		Type:      "connected",
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Connected to test run %s (status: %s)", sess.ID, sess.Status),
		Data: map[string]any{
			"run_id":     sess.ID,
			"target_url": sess.TargetURL,
			"suite_path": sess.SuitePath,
		},
	})
	_, _ = fmt.Fprintf(w, "data: %s\n\n", initEvt)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			raw, err := json.Marshal(evt)
			if err == nil {
				_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
				flusher.Flush()
			}
			if evt.Type == "complete" || evt.Type == "cancelled" || evt.Type == "error" {
				return
			}
		}
	}
}

// handleRunCancel aborts the active run session.
func (s *Server) handleRunCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cancelled := s.sessionMgr.CancelActive()
	if !cancelled {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "no active test run is in progress",
		})
		return
	}

	s.sessionMgr.AppendLog("Active test run was cancelled by user")
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "test run cancelled successfully",
	})
}

// handleRunReport returns the report of the latest test run.
func (s *Server) handleRunReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rep := s.sessionMgr.LastReport()
	if rep == nil {
		writeError(w, http.StatusNotFound, "no test runs recorded yet")
		return
	}

	writeJSON(w, http.StatusOK, rep)
}

type stepExecResult struct {
	Passed     bool
	StatusCode int
	LatencyMs  int64
	Method     string
	URL        string
	Error      string
	RawBody    []byte
	Headers    http.Header
	SavedVars  map[string]any
}

// executeTestSuite executes the v1 Suite step by step with SSE dispatches.
func (s *Server) executeTestSuite(ctx context.Context, sess *RunSession, suite *domain.Suite, targets map[string]string, defaultBaseURL string) {
	totalChecks := countSuiteSteps(suite)
	sess.Report.TotalCases = totalChecks
	sess.Report.TotalScenarios = len(suite.Scenarios)

	sess.Broadcast(RunEvent{
		Type:      "start",
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Beginning execution of suite (version %d) with %d scenarios", suite.Version, len(suite.Scenarios)),
		Data: map[string]any{
			"total_scenarios": len(suite.Scenarios),
			"servers":         suite.Servers,
			"targets":         targets,
		},
	})

	// 1. Pre-flight healthcheck of target base URLs
	probeURLs := make(map[string]struct{})
	if defaultBaseURL != "" {
		probeURLs[defaultBaseURL] = struct{}{}
	}
	for _, u := range targets {
		if u != "" {
			probeURLs[u] = struct{}{}
		}
	}

	for u := range probeURLs {
		checkCtx, checkCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := server.WaitForReady(checkCtx, u, 10*time.Second); err != nil {
			checkCancel()
			sess.Broadcast(RunEvent{
				Type:      "error",
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("Pre-flight healthcheck failed for %s: %s", u, err.Error()),
			})
			sess.Finish("failed")
			s.sessionMgr.SetLastReport(sess.Report)
			return
		}
		checkCancel()
	}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 2. Initialize Seed Variables (global scope)
	globalSeedVars := make(map[string]any)
	for _, seedDef := range suite.Seed {
		for k, v := range seedDef.Vars {
			globalSeedVars[k] = v
		}
	}

	suiteStartTime := time.Now()

	// 3. Iterate Scenarios
	for _, sc := range suite.Scenarios {
		if ctx.Err() != nil {
			break
		}

		// Handle Skip
		if sc.Skip != "" {
			sess.Report.SkippedScenarios++
			sess.RecordResult(CaseResult{
				ScenarioID: sc.ID,
				Status:     "SKIPPED",
				Error:      sc.Skip,
			})
			sess.Broadcast(RunEvent{
				Type:      "scenario_complete",
				Timestamp: time.Now(),
				Data: map[string]any{
					"scenario_id": sc.ID,
					"passed":      true,
					"skipped":     true,
					"reason":      sc.Skip,
				},
			})
			continue
		}

		sess.Broadcast(RunEvent{
			Type:      "scenario_start",
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Executing scenario: %s (%s)", sc.Name, sc.ID),
			Data: map[string]any{
				"scenario_id":   sc.ID,
				"scenario_name": sc.Name,
				"setups_to_run": sc.Setup,
			},
		})

		scenarioScope := make(map[string]any)
		for k, v := range globalSeedVars {
			scenarioScope[k] = v
		}

		scenarioPassed := true
		scenarioStart := time.Now()

		// A. Execute Setups for this scenario
		for _, setupName := range sc.Setup {
			setupDef, exists := suite.Setup[setupName]
			if !exists {
				continue
			}

			// Local setup vars
			for k, v := range setupDef.Vars {
				scenarioScope[k] = interpolateValue(v, scenarioScope)
			}

			for _, step := range setupDef.Steps {
				if ctx.Err() != nil {
					break
				}
				stepRes := executeSingleStep(ctx, httpClient, step, targets, defaultBaseURL, scenarioScope)
				if !stepRes.Passed {
					scenarioPassed = false
				}
				// Save variables into scenario scope
				for varName, expr := range step.Save {
					val := extractSavedVar(stepRes.RawBody, stepRes.Headers, expr)
					if val != nil {
						scenarioScope[varName] = val
						stepRes.SavedVars[varName] = val
					}
				}

				sess.Broadcast(RunEvent{
					Type:      "step_result",
					Timestamp: time.Now(),
					Data: map[string]any{
						"scenario_id": sc.ID,
						"setup":       setupName,
						"call":        step.Call,
						"method":      stepRes.Method,
						"url":         stepRes.URL,
						"status_code": stepRes.StatusCode,
						"expect":      step.Expect,
						"latency_ms":  stepRes.LatencyMs,
						"passed":      stepRes.Passed,
						"error":       stepRes.Error,
					},
				})
			}
		}

		// B. Execute Scenario Steps
		for _, step := range sc.Steps {
			if ctx.Err() != nil {
				break
			}

			stepRes := executeSingleStep(ctx, httpClient, step, targets, defaultBaseURL, scenarioScope)
			if !stepRes.Passed {
				scenarioPassed = false
			}

			// Extract saved variables
			for varName, expr := range step.Save {
				val := extractSavedVar(stepRes.RawBody, stepRes.Headers, expr)
				if val != nil {
					scenarioScope[varName] = val
					stepRes.SavedVars[varName] = val
				}
			}

			statusStr := "PASS"
			if !stepRes.Passed {
				statusStr = "FAIL"
			}

			caseRes := CaseResult{
				ScenarioID: sc.ID,
				Call:       step.Call,
				Method:     stepRes.Method,
				URL:        stepRes.URL,
				Status:     statusStr,
				HTTPStatus: stepRes.StatusCode,
				Expected:   step.Expect,
				LatencyMs:  stepRes.LatencyMs,
				Error:      stepRes.Error,
				SavedVars:  stepRes.SavedVars,
			}

			sess.RecordResult(caseRes)
			sess.Broadcast(RunEvent{
				Type:      "step_result",
				Timestamp: time.Now(),
				Data: map[string]any{
					"scenario_id": sc.ID,
					"call":        step.Call,
					"method":      stepRes.Method,
					"url":         stepRes.URL,
					"status_code": stepRes.StatusCode,
					"expect":      step.Expect,
					"latency_ms":  stepRes.LatencyMs,
					"passed":      stepRes.Passed,
					"error":       stepRes.Error,
					"saved_vars":  stepRes.SavedVars,
				},
			})
		}

		if scenarioPassed {
			sess.Report.PassedScenarios++
		} else {
			sess.Report.FailedScenarios++
		}

		sess.Broadcast(RunEvent{
			Type:      "scenario_complete",
			Timestamp: time.Now(),
			Data: map[string]any{
				"scenario_id": sc.ID,
				"passed":      scenarioPassed,
				"duration_ms": time.Since(scenarioStart).Milliseconds(),
			},
		})
	}

	finalStatus := "passed"
	if ctx.Err() != nil {
		finalStatus = "cancelled"
		sess.Broadcast(RunEvent{
			Type:      "cancelled",
			Timestamp: time.Now(),
			Message:   "Test run was cancelled by user",
		})
	} else if sess.Report.FailedScenarios > 0 || sess.Report.Failed > 0 {
		finalStatus = "failed"
		sess.Broadcast(RunEvent{
			Type:      "complete",
			Timestamp: time.Now(),
			Message: fmt.Sprintf("Run finished: %d passed, %d failed",
				sess.Report.PassedScenarios, sess.Report.FailedScenarios),
			Data: map[string]any{"report": sess.Report},
		})
	} else {
		sess.Broadcast(RunEvent{
			Type:      "complete",
			Timestamp: time.Now(),
			Message: fmt.Sprintf("Run finished: all %d scenarios passed",
				sess.Report.PassedScenarios),
			Data: map[string]any{"report": sess.Report},
		})
	}

	sess.Report.DurationMs = time.Since(suiteStartTime).Milliseconds()
	sess.Finish(finalStatus)
	s.sessionMgr.SetLastReport(sess.Report)
	s.sessionMgr.AppendLog(fmt.Sprintf("[%s] Suite run finished: %s (%d/%d scenarios passed)",
		sess.ID, finalStatus, sess.Report.PassedScenarios, sess.Report.TotalScenarios))
}

var serverPlaceholderPattern = regexp.MustCompile(`\{([a-zA-Z0-9_-]+)\}`)

// executeSingleStep runs a single HTTP step, checks expectation, and extracts telemetry.
func executeSingleStep(ctx context.Context, client *http.Client, step domain.Step, targets map[string]string, defaultBaseURL string, scope map[string]any) stepExecResult {
	result := stepExecResult{
		SavedVars: make(map[string]any),
	}

	callStr := strings.TrimSpace(step.Call)
	parts := strings.SplitN(callStr, " ", 2)
	method := "GET"
	urlTmpl := ""
	if len(parts) == 1 {
		urlTmpl = parts[0]
	} else {
		method = strings.ToUpper(parts[0])
		urlTmpl = parts[1]
	}
	result.Method = method

	// Replace {server} placeholder
	fullURL := serverPlaceholderPattern.ReplaceAllStringFunc(urlTmpl, func(placeholder string) string {
		serverName := strings.Trim(placeholder, "{}")
		if base, ok := targets[serverName]; ok && base != "" {
			return strings.TrimRight(base, "/")
		}
		if defaultBaseURL != "" {
			return strings.TrimRight(defaultBaseURL, "/")
		}
		return placeholder
	})

	// Interpolate {{var}} in URL
	fullURL = interpolateString(fullURL, scope)
	result.URL = fullURL

	// Prepare request body
	var bodyReader io.Reader
	if step.Body != nil {
		interpolatedBody := interpolateValue(step.Body, scope)
		bodyBytes, err := json.Marshal(interpolatedBody)
		if err == nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		result.Error = fmt.Sprintf("invalid request: %v", err)
		return result
	}

	if step.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range step.Headers {
		req.Header.Set(k, interpolateString(v, scope))
	}

	t0 := time.Now()
	resp, reqErr := client.Do(req)
	result.LatencyMs = time.Since(t0).Milliseconds()

	if reqErr != nil {
		result.Error = reqErr.Error()
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode
	result.Headers = resp.Header

	bodyBytes, _ := io.ReadAll(resp.Body)
	result.RawBody = bodyBytes

	expected := step.Expect
	if expected <= 0 {
		expected = 200
	}

	if resp.StatusCode == expected {
		result.Passed = true
	} else {
		result.Passed = false
		result.Error = fmt.Sprintf("expected HTTP %d, got %d", expected, resp.StatusCode)
	}

	return result
}

var varTokenPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_-]+)\}\}`)

// interpolateString replaces {{var}} and {{random}} placeholders with runtime values.
func interpolateString(input string, scope map[string]any) string {
	return varTokenPattern.ReplaceAllStringFunc(input, func(token string) string {
		key := strings.Trim(token, "{}")
		if key == "random" {
			b := make([]byte, 4)
			_, _ = rand.Read(b)
			return hex.EncodeToString(b)
		}
		if val, ok := scope[key]; ok {
			return fmt.Sprintf("%v", val)
		}
		return token
	})
}

// interpolateValue replaces {{var}} in strings or deeply inside maps/slices.
func interpolateValue(val any, scope map[string]any) any {
	switch v := val.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
			key := strings.Trim(trimmed, "{}")
			if key == "random" {
				b := make([]byte, 4)
				_, _ = rand.Read(b)
				return hex.EncodeToString(b)
			}
			if realVal, ok := scope[key]; ok {
				return realVal
			}
		}
		return interpolateString(v, scope)
	case map[string]any:
		res := make(map[string]any, len(v))
		for k, item := range v {
			res[k] = interpolateValue(item, scope)
		}
		return res
	case []any:
		res := make([]any, len(v))
		for i, item := range v {
			res[i] = interpolateValue(item, scope)
		}
		return res
	default:
		return val
	}
}

// extractSavedVar extracts a value from HTTP response body or headers.
func extractSavedVar(bodyBytes []byte, headers http.Header, expr string) any {
	trimmed := strings.TrimSpace(expr)
	if strings.HasPrefix(trimmed, "header.") {
		headerName := strings.TrimPrefix(trimmed, "header.")
		return headers.Get(headerName)
	}

	if strings.HasPrefix(trimmed, "body.") {
		dotPath := strings.TrimPrefix(trimmed, "body.")
		var parsed any
		if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
			return nil
		}
		return extractJSONPath(parsed, dotPath)
	}

	return nil
}

// extractJSONPath traverses a nested JSON structure using a dot-delimited path.
func extractJSONPath(root any, path string) any {
	parts := strings.Split(path, ".")
	current := root

	for _, part := range parts {
		if current == nil {
			return nil
		}
		switch cur := current.(type) {
		case map[string]any:
			val, ok := cur[part]
			if !ok {
				return nil
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(cur) {
				return nil
			}
			current = cur[idx]
		default:
			return nil
		}
	}
	return current
}

func normalizeURL(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "http://" + target
	}
	return strings.TrimRight(target, "/")
}
