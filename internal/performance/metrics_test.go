package performance

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if registry.Metrics == nil {
		t.Error("Metrics map is not initialized")
	}

	if registry.Enabled {
		t.Error("Metrics should be disabled by default (privacy-aware)")
	}
}

func TestRegistry_Enable(t *testing.T) {
	registry := NewRegistry()

	registry.Enable()

	if !registry.Enabled {
		t.Error("Enable() should set Enabled to true")
	}
}

func TestRegistry_Disable(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()
	registry.Disable()

	if registry.Enabled {
		t.Error("Disable() should set Enabled to false")
	}
}

func TestRegistry_RecordOperation(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		operationName string
		provider      string
		duration      time.Duration
		success       bool
	}{
		{
			name:          "record when enabled",
			enabled:       true,
			operationName: "api_call",
			provider:      "anthropic",
			duration:      100 * time.Millisecond,
			success:       true,
		},
		{
			name:          "record when disabled",
			enabled:       false,
			operationName: "api_call",
			provider:      "anthropic",
			duration:      100 * time.Millisecond,
			success:       true,
		},
		{
			name:          "record failed operation",
			enabled:       true,
			operationName: "api_call",
			provider:      "zai",
			duration:      50 * time.Millisecond,
			success:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			registry.Enabled = tt.enabled

			registry.RecordOperation(tt.operationName, tt.provider, tt.duration, tt.success)

			if tt.enabled {
				metrics, exists := registry.Metrics[tt.operationName]
				if !exists {
					t.Fatalf("Operation '%s' not found in metrics", tt.operationName)
				}

				providerMetrics, exists := metrics[tt.provider]
				if !exists {
					t.Fatalf("Provider '%s' not found in metrics for '%s'", tt.provider, tt.operationName)
				}

				if providerMetrics.Count != 1 {
					t.Errorf("Expected count 1, got %d", providerMetrics.Count)
				}

				if providerMetrics.TotalDuration != tt.duration {
					t.Errorf("Expected duration %v, got %v", tt.duration, providerMetrics.TotalDuration)
				}

				if tt.success && providerMetrics.FailureCount != 0 {
					t.Errorf("Expected failure count 0, got %d", providerMetrics.FailureCount)
				}

				if !tt.success && providerMetrics.FailureCount != 1 {
					t.Errorf("Expected failure count 1, got %d", providerMetrics.FailureCount)
				}
			} else {
				if len(registry.Metrics) > 0 {
					t.Error("Metrics should be empty when disabled")
				}
			}
		})
	}
}

func TestRegistry_RecordOperation_Aggregates(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()

	// Record multiple operations for the same provider
	registry.RecordOperation("api_call", "anthropic", 100*time.Millisecond, true)
	registry.RecordOperation("api_call", "anthropic", 200*time.Millisecond, true)
	registry.RecordOperation("api_call", "anthropic", 50*time.Millisecond, false)

	metrics := registry.Metrics["api_call"]["anthropic"]

	if metrics.Count != 3 {
		t.Errorf("Expected count 3, got %d", metrics.Count)
	}

	expectedDuration := 350 * time.Millisecond
	if metrics.TotalDuration != expectedDuration {
		t.Errorf("Expected total duration %v, got %v", expectedDuration, metrics.TotalDuration)
	}

	if metrics.MinDuration != 50*time.Millisecond {
		t.Errorf("Expected min duration %v, got %v", 50*time.Millisecond, metrics.MinDuration)
	}

	if metrics.MaxDuration != 200*time.Millisecond {
		t.Errorf("Expected max duration %v, got %v", 200*time.Millisecond, metrics.MaxDuration)
	}

	if metrics.FailureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", metrics.FailureCount)
	}
}

func TestRegistry_GetStats(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()

	registry.RecordOperation("api_call", "anthropic", 100*time.Millisecond, true)
	registry.RecordOperation("api_call", "anthropic", 200*time.Millisecond, false)

	stats := registry.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if len(stats) == 0 {
		t.Error("Stats should not be empty")
	}

	opStats, exists := stats["api_call"]
	if !exists {
		t.Fatal("Operation 'api_call' not found in stats")
	}

	providerStats, exists := opStats["anthropic"]
	if !exists {
		t.Fatal("Provider 'anthropic' not found in stats")
	}

	if providerStats.Count != 2 {
		t.Errorf("Expected count 2, got %d", providerStats.Count)
	}

	if providerStats.FailureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", providerStats.FailureCount)
	}

	avgDuration := providerStats.TotalDuration / time.Duration(providerStats.Count)
	if avgDuration != 150*time.Millisecond {
		t.Errorf("Expected avg duration 150ms, got %v", avgDuration)
	}
}

func TestRegistry_GetStats_Empty(t *testing.T) {
	registry := NewRegistry()

	stats := registry.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if len(stats) != 0 {
		t.Errorf("Expected empty stats, got %d entries", len(stats))
	}
}

func TestRegistry_Reset(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()

	registry.RecordOperation("api_call", "anthropic", 100*time.Millisecond, true)

	if len(registry.Metrics) == 0 {
		t.Error("Metrics should not be empty before reset")
	}

	registry.Reset()

	if len(registry.Metrics) != 0 {
		t.Errorf("Metrics should be empty after reset, got %d entries", len(registry.Metrics))
	}
}

func TestRegistry_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "with data",
			enabled: true,
		},
		{
			name:    "empty when disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			registry.Enabled = tt.enabled

			if tt.enabled {
				registry.RecordOperation("api_call", "anthropic", 100*time.Millisecond, true)
			}

			jsonData, err := registry.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON returned error: %v", err)
			}

			if jsonData == nil {
				t.Fatal("ToJSON returned nil")
			}

			if tt.enabled {
				var result map[string]interface{}
				if err := json.Unmarshal(jsonData, &result); err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				if _, exists := result["api_call"]; !exists {
					t.Error("JSON should contain 'api_call' operation")
				}
			} else {
				var result map[string]interface{}
				if err := json.Unmarshal(jsonData, &result); err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				if len(result) != 0 {
					t.Error("JSON should be empty when metrics are disabled")
				}
			}
		})
	}
}

func TestRecordDuration(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()

	operationCalled := false
	providerCalled := false
	successCalled := false

	// Mock operation that takes some time
	mockOperation := func() error {
		operationCalled = true
		time.Sleep(10 * time.Millisecond)
		providerCalled = true
		successCalled = true
		return nil
	}

	err := RecordDuration(registry, "test_operation", "test_provider", mockOperation)
	if err != nil {
		t.Fatalf("RecordDuration returned error: %v", err)
	}

	if !operationCalled {
		t.Error("Operation was not called")
	}

	if !providerCalled {
		t.Error("Provider was not recorded")
	}

	if !successCalled {
		t.Error("Success status was not recorded")
	}

	// Verify metrics were recorded
	stats := registry.GetStats()
	if len(stats) == 0 {
		t.Fatal("No metrics recorded")
	}

	opStats, exists := stats["test_operation"]
	if !exists {
		t.Fatal("test_operation not found in metrics")
	}

	providerStats, exists := opStats["test_provider"]
	if !exists {
		t.Fatal("test_provider not found in metrics")
	}

	if providerStats.Count != 1 {
		t.Errorf("Expected count 1, got %d", providerStats.Count)
	}

	if providerStats.TotalDuration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", providerStats.TotalDuration)
	}
}

func TestRecordDuration_Error(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()

	expectedErr := fmt.Errorf("test error")
	mockOperation := func() error {
		return expectedErr
	}

	err := RecordDuration(registry, "test_operation", "test_provider", mockOperation)
	if err != expectedErr {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}

	// Verify failure was recorded
	stats := registry.GetStats()
	providerStats := stats["test_operation"]["test_provider"]

	if providerStats.FailureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", providerStats.FailureCount)
	}
}

func TestRecordDuration_Disabled(t *testing.T) {
	registry := NewRegistry()
	// Keep disabled (default)

	called := false
	mockOperation := func() error {
		called = true
		return nil
	}

	err := RecordDuration(registry, "test_operation", "test_provider", mockOperation)
	if err != nil {
		t.Fatalf("RecordDuration returned error: %v", err)
	}

	if !called {
		t.Error("Operation should still be called when metrics disabled")
	}

	// Verify no metrics were recorded
	stats := registry.GetStats()
	if len(stats) != 0 {
		t.Errorf("Metrics should be empty when disabled, got %d entries", len(stats))
	}
}

func TestStatsSummary(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()

	registry.RecordOperation("api_call", "anthropic", 100*time.Millisecond, true)
	registry.RecordOperation("api_call", "zai", 200*time.Millisecond, true)
	registry.RecordOperation("config_save", "anthropic", 50*time.Millisecond, true)

	stats := registry.GetStats()
	summary := stats.Summary()

	if summary == "" {
		t.Fatal("Summary returned empty string")
	}

	// Check that summary contains expected information
	// Summary format: "operation: X ops, avg Yms, Z failures"
	if !contains(summary, "api_call") {
		t.Error("Summary should mention 'api_call'")
	}

	if !contains(summary, "config_save") {
		t.Error("Summary should mention 'config_save'")
	}

	// api_call has 2 ops, config_save has 1
	if !contains(summary, "2 ops") {
		t.Error("Summary should show '2 ops' for api_call")
	}
}

func TestStatsSummary_Empty(t *testing.T) {
	stats := make(Stats)
	summary := stats.Summary()

	if summary != "" {
		t.Errorf("Expected empty summary, got: %s", summary)
	}
}

func TestStatsSummary_MultipleProviders(t *testing.T) {
	registry := NewRegistry()
	registry.Enable()

	registry.RecordOperation("api_call", "anthropic", 100*time.Millisecond, true)
	registry.RecordOperation("api_call", "anthropic", 150*time.Millisecond, true)
	registry.RecordOperation("api_call", "zai", 200*time.Millisecond, false)

	stats := registry.GetStats()
	summary := stats.Summary()

	// Summary aggregates across providers, so we check aggregated values
	if !contains(summary, "api_call") {
		t.Error("Summary should mention 'api_call'")
	}

	// Total: 3 ops, avg (100+150+200)/3 = 150ms, 1 failure
	if !contains(summary, "3 ops") {
		t.Error("Summary should show '3 ops'")
	}

	if !contains(summary, "1 failures") {
		t.Error("Summary should show '1 failures'")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
