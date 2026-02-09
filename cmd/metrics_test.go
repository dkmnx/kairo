package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMetricsCommand_Disabled(t *testing.T) {
	// Ensure metrics are disabled
	registry := getMetricsRegistry()
	registry.Disable()

	// Run metrics command - should not panic
	metricsCmd.Run(metricsCmd, []string{})

	// Verify metrics are still disabled
	if registry.Enabled {
		t.Error("Metrics should remain disabled")
	}
}

func TestMetricsCommand_NoData(t *testing.T) {
	// Enable metrics but don't record anything
	registry := getMetricsRegistry()
	registry.Reset()
	registry.Enable()

	// Run metrics command - should not panic
	metricsCmd.Run(metricsCmd, []string{})

	// Verify still enabled
	if !registry.Enabled {
		t.Error("Metrics should remain enabled")
	}
}

func TestMetricsCommand_WithData(t *testing.T) {
	// Enable and record some metrics
	registry := getMetricsRegistry()
	registry.Reset()
	registry.Enable()

	registry.RecordOperation("api_call", "anthropic", 100*time.Millisecond, true)
	registry.RecordOperation("api_call", "zai", 200*time.Millisecond, true)
	registry.RecordOperation("config_save", "anthropic", 50*time.Millisecond, true)

	// Run metrics command - should not panic
	metricsCmd.Run(metricsCmd, []string{})

	// Verify metrics are still recorded
	stats := registry.GetStats()
	if len(stats) == 0 {
		t.Error("Metrics should still be recorded after display")
	}
}

func TestMetricsEnableCommand(t *testing.T) {
	// Disable first
	registry := getMetricsRegistry()
	registry.Disable()

	// Run enable command
	metricsEnableCmd.Run(metricsEnableCmd, []string{})

	// Verify actually enabled
	if !registry.Enabled {
		t.Error("Registry should be enabled after enable command")
	}
}

func TestMetricsDisableCommand(t *testing.T) {
	// Enable first
	registry := getMetricsRegistry()
	registry.Enable()

	// Run disable command
	metricsDisableCmd.Run(metricsDisableCmd, []string{})

	// Verify actually disabled
	if registry.Enabled {
		t.Error("Registry should be disabled after disable command")
	}
}

func TestMetricsResetCommand(t *testing.T) {
	// Enable and record some metrics
	registry := getMetricsRegistry()
	registry.Enable()
	registry.RecordOperation("test", "provider", 100*time.Millisecond, true)

	// Verify metrics exist
	stats := registry.GetStats()
	if len(stats) == 0 {
		t.Fatal("Expected metrics to be recorded")
	}

	// Run reset command
	metricsResetCmd.Run(metricsResetCmd, []string{})

	// Verify metrics cleared
	stats = registry.GetStats()
	if len(stats) != 0 {
		t.Errorf("Expected empty metrics after reset, got %d entries", len(stats))
	}
}

func TestMetricsResetCommand_Disabled(t *testing.T) {
	// Disable metrics
	registry := getMetricsRegistry()
	registry.Disable()

	// Record some data before disabling
	registry.RecordOperation("test", "provider", 100*time.Millisecond, true)
	registry.Disable()

	// Run reset command - should warn but not panic
	metricsResetCmd.Run(metricsResetCmd, []string{})

	// Metrics should still be disabled
	if registry.Enabled {
		t.Error("Metrics should remain disabled")
	}
}

func TestGetMetricsRegistry_EnvVar(t *testing.T) {
	// Save original value
	original := os.Getenv("KAIRO_METRICS_ENABLED")
	defer func() {
		if original == "" {
			os.Unsetenv("KAIRO_METRICS_ENABLED")
		} else {
			os.Setenv("KAIRO_METRICS_ENABLED", original)
		}
	}()

	// Reset global registry
	globalMetricsRegistry = nil

	// Test with env var set
	os.Setenv("KAIRO_METRICS_ENABLED", "true")
	registry := getMetricsRegistry()

	if !registry.Enabled {
		t.Error("Registry should be enabled when KAIRO_METRICS_ENABLED=true")
	}
}

func TestGetMetricsRegistry_Singleton(t *testing.T) {
	// Reset global registry
	globalMetricsRegistry = nil

	r1 := getMetricsRegistry()
	r2 := getMetricsRegistry()

	if r1 != r2 {
		t.Error("getMetricsRegistry should return the same instance (singleton)")
	}
}

func TestExportMetrics_InvalidFormat(t *testing.T) {
	// Enable metrics
	registry := getMetricsRegistry()
	registry.Enable()
	registry.RecordOperation("test", "provider", 100*time.Millisecond, true)

	// Set invalid format
	metricsFormat = "invalid"
	metricsOutputPath = filepath.Join(os.TempDir(), "test_metrics_invalid.json")
	defer os.Remove(metricsOutputPath)

	err := exportMetrics(registry)

	if err == nil {
		t.Error("Expected error for invalid format")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}
}

func TestExportMetrics_JSON(t *testing.T) {
	// Enable metrics
	registry := getMetricsRegistry()
	registry.Reset()
	registry.Enable()
	registry.RecordOperation("test", "provider", 100*time.Millisecond, true)

	// Set JSON format
	metricsFormat = "json"
	metricsOutputPath = filepath.Join(os.TempDir(), "test_metrics_export.json")
	defer os.Remove(metricsOutputPath)

	err := exportMetrics(registry)

	if err != nil {
		t.Fatalf("Failed to export metrics: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(metricsOutputPath); os.IsNotExist(err) {
		t.Error("Export file was not created")
	}

	// Verify file contains valid JSON
	content, err := os.ReadFile(metricsOutputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if !strings.Contains(string(content), "test") {
		t.Errorf("Export file should contain operation data, got: %s", string(content))
	}
}
