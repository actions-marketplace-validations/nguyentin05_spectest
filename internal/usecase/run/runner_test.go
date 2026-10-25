package run

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/nguyentin05/spectest/internal/domain"
)

type mockLifecycleManager struct {
	upErr       error
	teardownFn  func()
	teardownRan *atomic.Bool
}

func (m *mockLifecycleManager) Up(ctx context.Context) (func(), error) {
	if m.upErr != nil {
		return nil, m.upErr
	}
	return func() {
		if m.teardownRan != nil {
			m.teardownRan.Store(true)
		}
		if m.teardownFn != nil {
			m.teardownFn()
		}
	}, nil
}

func sampleSuite() *domain.Suite {
	return &domain.Suite{
		Version: 1,
		Servers: []string{"order"},
		APIs:    []string{"POST {order}/api/orders"},
		Scenarios: []domain.Scenario{
			{
				ID:   "SCN_1",
				Name: "Checkout flow",
				Steps: []domain.Step{
					{
						Call:   "POST {order}/api/orders",
						Expect: 201,
					},
					{
						Call:   "GET {order}/api/orders/1",
						Expect: 200,
					},
				},
			},
		},
	}
}

func TestRunner_Execute_Success(t *testing.T) {
	mockLoader := func(path string) (*domain.Suite, error) {
		return sampleSuite(), nil
	}
	runner := NewRunner(mockLoader, nil)

	var teardownCalled atomic.Bool
	runner.SetLifecycleFactory(func(composeFile, targetURL string, timeoutSec int) domain.ServerLifecycleManager {
		return &mockLifecycleManager{
			teardownRan: &teardownCalled,
		}
	})

	summary, err := runner.Execute(context.Background(), Options{
		SuitePath: "spectest.yaml",
		TargetURL: "http://localhost:9090",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TargetURL != "http://localhost:9090" {
		t.Errorf("TargetURL = %q, want 'http://localhost:9090'", summary.TargetURL)
	}
	if summary.TotalChecks != 2 {
		t.Errorf("TotalChecks = %d, want 2", summary.TotalChecks)
	}
	if !teardownCalled.Load() {
		t.Error("expected server lifecycle teardown to be executed")
	}
}

func TestRunner_Execute_LoadError(t *testing.T) {
	mockLoader := func(path string) (*domain.Suite, error) {
		return nil, errors.New("file not found")
	}
	runner := NewRunner(mockLoader, nil)

	_, err := runner.Execute(context.Background(), Options{
		SuitePath: "invalid.yaml",
		TargetURL: "http://localhost:8080",
	})
	if err == nil {
		t.Fatal("expected error loading invalid suite path, got nil")
	}
	if !strings.Contains(err.Error(), "load test suite") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunner_Execute_TargetRequired(t *testing.T) {
	mockLoader := func(path string) (*domain.Suite, error) {
		return sampleSuite(), nil
	}
	runner := NewRunner(mockLoader, nil)

	_, err := runner.Execute(context.Background(), Options{
		SuitePath: "spectest.yaml",
		TargetURL: "",
	})
	if err == nil {
		t.Fatal("expected error when target URL is missing, got nil")
	}
}

func TestRunner_Execute_LifecycleError(t *testing.T) {
	mockLoader := func(path string) (*domain.Suite, error) {
		return sampleSuite(), nil
	}
	runner := NewRunner(mockLoader, nil)
	runner.SetLifecycleFactory(func(composeFile, targetURL string, timeoutSec int) domain.ServerLifecycleManager {
		return &mockLifecycleManager{
			upErr: errors.New("compose up failed"),
		}
	})

	_, err := runner.Execute(context.Background(), Options{
		SuitePath: "spectest.yaml",
		TargetURL: "http://localhost:8080",
	})
	if err == nil {
		t.Fatal("expected error when lifecycle Up() fails, got nil")
	}
	if !strings.Contains(err.Error(), "setup server environment") {
		t.Errorf("unexpected error message: %v", err)
	}
}
