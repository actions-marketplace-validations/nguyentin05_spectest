package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nguyentin05/spectest/internal/domain"
)

// RunEvent represents a real-time event dispatched during a test run execution.
type RunEvent struct {
	Type      string         `json:"type"` // "start", "step_start", "case_result", "complete", "cancelled", "error"
	Timestamp time.Time      `json:"timestamp"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// CaseResult holds the execution outcome of an individual test step or case.
type CaseResult struct {
	ScenarioID string         `json:"scenario_id"`
	StepID     string         `json:"step_id,omitempty"`
	CaseID     string         `json:"case_id,omitempty"`
	Call       string         `json:"call,omitempty"`
	Endpoint   string         `json:"endpoint,omitempty"`
	URL        string         `json:"url,omitempty"`
	Method     string         `json:"method"`
	Status     string         `json:"status"` // "PASS", "FAIL", "SKIPPED", "CANCELLED"
	HTTPStatus int            `json:"http_status"`
	Expected   int            `json:"expected_http_status"`
	LatencyMs  int64          `json:"latency_ms"`
	Error      string         `json:"error,omitempty"`
	SavedVars  map[string]any `json:"saved_vars,omitempty"`
}

// RunReport stores the full outcome of a test suite execution.
type RunReport struct {
	ID               string            `json:"id"`
	TargetURL        string            `json:"target_url"`
	Targets          map[string]string `json:"targets,omitempty"`
	SuitePath        string            `json:"suite_path"`
	Status           string            `json:"status"` // "passed", "failed", "cancelled", "running"
	TotalScenarios   int               `json:"total_scenarios"`
	PassedScenarios  int               `json:"passed_scenarios"`
	FailedScenarios  int               `json:"failed_scenarios"`
	SkippedScenarios int               `json:"skipped_scenarios"`
	TotalCases       int               `json:"total_cases"` // total checks
	Passed           int               `json:"passed"`
	Failed           int               `json:"failed"`
	Skipped          int               `json:"skipped"`
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	DurationMs       int64             `json:"duration_ms"`
	Results          []CaseResult      `json:"results"`
}

// RunSession encapsulates a live test execution session.
type RunSession struct {
	ID        string
	TargetURL string
	SuitePath string
	Cancel    context.CancelFunc
	Status    string
	Report    *RunReport
	mu        sync.RWMutex
	clients   map[chan RunEvent]struct{}
}

// NewRunSession creates an active session.
func NewRunSession(id, targetURL, suitePath string, cancel context.CancelFunc) *RunSession {
	return &RunSession{
		ID:        id,
		TargetURL: targetURL,
		SuitePath: suitePath,
		Cancel:    cancel,
		Status:    "running",
		clients:   make(map[chan RunEvent]struct{}),
		Report: &RunReport{
			ID:        id,
			TargetURL: targetURL,
			SuitePath: suitePath,
			Status:    "running",
			StartTime: time.Now(),
			Results:   make([]CaseResult, 0),
		},
	}
}

// Broadcast sends an event to all connected SSE clients.
func (s *RunSession) Broadcast(evt RunEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- evt:
		default:
			// client slow or blocked, drop event to prevent hanging
		}
	}
}

// Subscribe registers a new SSE listener channel.
func (s *RunSession) Subscribe() chan RunEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan RunEvent, 50)
	s.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe removes an SSE listener channel.
func (s *RunSession) Unsubscribe(ch chan RunEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, ch)
	close(ch)
}

// RecordResult appends a test case result and updates metrics.
func (s *RunSession) RecordResult(res CaseResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Report.Results = append(s.Report.Results, res)
	switch res.Status {
	case "PASS":
		s.Report.Passed++
	case "FAIL":
		s.Report.Failed++
	default:
		s.Report.Skipped++
	}
}

// Finish marks the session as finished and calculates duration.
func (s *RunSession) Finish(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	s.Report.Status = status
	s.Report.EndTime = time.Now()
	s.Report.DurationMs = s.Report.EndTime.Sub(s.Report.StartTime).Milliseconds()
}

// SessionManager manages active and past test runs.
type SessionManager struct {
	mu            sync.RWMutex
	activeSession *RunSession
	lastReport    *RunReport
	recentLogs    []string
}

// NewSessionManager creates a SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		recentLogs: make([]string, 0),
	}
}

// ActiveSession returns the currently running test session, if any.
func (m *SessionManager) ActiveSession() *RunSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeSession
}

// StartSession sets up a new active session.
func (m *SessionManager) StartSession(id, targetURL, suitePath string, cancel context.CancelFunc) (*RunSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeSession != nil && m.activeSession.Status == "running" {
		return nil, fmt.Errorf("a test run is already in progress (ID: %s)", m.activeSession.ID)
	}
	sess := NewRunSession(id, targetURL, suitePath, cancel)
	m.activeSession = sess
	return sess, nil
}

// CancelActive aborts the active session if running.
func (m *SessionManager) CancelActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeSession != nil && m.activeSession.Status == "running" {
		m.activeSession.Cancel()
		m.activeSession.Finish("cancelled")
		m.lastReport = m.activeSession.Report
		return true
	}
	return false
}

// SetLastReport stores completed report.
func (m *SessionManager) SetLastReport(rep *RunReport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastReport = rep
	if m.activeSession != nil && m.activeSession.ID == rep.ID {
		m.activeSession.Finish(rep.Status)
	}
}

// LastReport retrieves the latest completed run report.
func (m *SessionManager) LastReport() *RunReport {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.activeSession != nil {
		return m.activeSession.Report
	}
	return m.lastReport
}

// AppendLog adds a message to the recent logs buffer.
func (m *SessionManager) AppendLog(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	m.recentLogs = append(m.recentLogs, entry)
	if len(m.recentLogs) > 200 {
		m.recentLogs = m.recentLogs[len(m.recentLogs)-200:]
	}
}

// Logs returns a copy of recent logs.
func (m *SessionManager) Logs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copied := make([]string, len(m.recentLogs))
	copy(copied, m.recentLogs)
	return copied
}

// countSuiteSteps calculates the total number of steps in setups and scenarios.
func countSuiteSteps(suite *domain.Suite) int {
	if suite == nil {
		return 0
	}
	total := 0
	for _, sc := range suite.Scenarios {
		total += len(sc.Steps)
	}
	for _, st := range suite.Setup {
		total += len(st.Steps)
	}
	return total
}
