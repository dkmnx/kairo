package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigErrorMessages(t *testing.T) {
	t.Run("nonexistent directory returns descriptive error", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/directory")
		if err != ErrConfigNotFound {
			t.Errorf("LoadConfig() error = %v, want ErrConfigNotFound", err)
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "config") {
			t.Errorf("Error message should mention config, got: %s", errMsg)
		}
	})

	t.Run("invalid YAML returns parsing error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config")

		invalidYAML := `
default_provider: "zai"
providers:
  - invalid: [yaml, structure
`
		if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := LoadConfig(tmpDir)
		if err == nil {
			t.Fatal("LoadConfig() should error for invalid YAML")
		}

		errMsg := err.Error()
		// Error should mention YAML parsing or provide context about the format issue
		if !strings.Contains(errMsg, "yaml") && !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "unmarshal") {
			t.Errorf("Error message should mention YAML parsing, got: %s", errMsg)
		}
	})
}

func TestSaveConfigErrorMessages(t *testing.T) {
	t.Run("readonly directory returns permission error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping readonly test when running as root")
		}

		tmpDir := t.TempDir()
		readonlyDir := filepath.Join(tmpDir, "readonly")
		if err := os.MkdirAll(readonlyDir, 0555); err != nil {
			t.Fatal(err)
		}

		cfg := &Config{
			DefaultProvider: "test",
			Providers:       make(map[string]Provider),
		}

		err := SaveConfig(readonlyDir, cfg)
		if err == nil {
			t.Fatal("SaveConfig() should error for readonly directory")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "config") && !strings.Contains(errMsg, "write") && !strings.Contains(errMsg, "permission") {
			t.Errorf("Error message should mention config, write, or permission, got: %s", errMsg)
		}
	})

	t.Run("invalid directory path returns descriptive error", func(t *testing.T) {
		cfg := &Config{
			DefaultProvider: "test",
			Providers:       make(map[string]Provider),
		}

		err := SaveConfig("/invalid/path/that/does/not/exist", cfg)
		if err == nil {
			t.Fatal("SaveConfig() should error for invalid path")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "no such file") && !strings.Contains(errMsg, "directory") && !strings.Contains(errMsg, "config") {
			t.Errorf("Error message should mention path or directory issue, got: %s", errMsg)
		}
	})
}

func TestConfigNotFoundError(t *testing.T) {
	t.Run("ErrConfigNotFound is a known error type", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path")
		if err != ErrConfigNotFound {
			t.Errorf("Expected ErrConfigNotFound, got: %T", err)
		}
	})

	t.Run("ErrConfigNotFound has descriptive message", func(t *testing.T) {
		errMsg := ErrConfigNotFound.Error()
		if errMsg == "" {
			t.Error("ErrConfigNotFound should have a descriptive error message")
		}
		if !strings.Contains(errMsg, "config") {
			t.Errorf("ErrConfigNotFound message should mention 'config', got: %s", errMsg)
		}
	})
}

func TestNilProvidersMapHandling(t *testing.T) {
	t.Run("handles nil providers map in loaded config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config")

		yamlContent := `default_provider: "zai"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if cfg.Providers == nil {
			t.Error("Providers map should be initialized even if empty in YAML")
		}
	})
}

func TestErrorWrapping(t *testing.T) {
	t.Run("SaveConfig wraps permission errors", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping readonly test when running as root")
		}

		tmpDir := t.TempDir()
		readonlyDir := filepath.Join(tmpDir, "readonly")
		if err := os.MkdirAll(readonlyDir, 0555); err != nil {
			t.Fatal(err)
		}

		cfg := &Config{
			DefaultProvider: "test",
			Providers:       make(map[string]Provider),
		}

		err := SaveConfig(readonlyDir, cfg)
		if err == nil {
			t.Fatal("Expected error")
		}

		// Error should contain context about what operation failed
		errMsg := err.Error()
		if strings.Contains(errMsg, "permission denied") && !strings.Contains(errMsg, "config") {
			t.Errorf("Error should be wrapped with context, got raw error: %s", errMsg)
		}
	})
}
