package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateConfigOnUpdate(t *testing.T) {
	t.Run("UpdatesModelWhenBuiltinDefaultChanges", func(t *testing.T) {
		// Simulate config with old model that needs migration
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Old config with MiniMax using old default model
		configContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2
default_models:
  minimax: MiniMax-M2
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		changes, err := MigrateConfigOnUpdate(tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate() error = %v", err)
		}

		// Verify changes were recorded
		if len(changes) == 0 {
			t.Error("Expected migration changes, got none")
		}

		// Verify config was updated
		cfg, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		provider, ok := cfg.Providers["minimax"]
		if !ok {
			t.Fatal("minimax provider not found")
		}

		// The builtin default is MiniMax-M2.5, so model should be updated
		if provider.Model != "MiniMax-M2.5" {
			t.Errorf("Provider model = %q, want %q", provider.Model, "MiniMax-M2.5")
		}

		// DefaultModels should also be updated
		if cfg.DefaultModels["minimax"] != "MiniMax-M2.5" {
			t.Errorf("DefaultModels[minimax] = %q, want %q", cfg.DefaultModels["minimax"], "MiniMax-M2.5")
		}
	})

	t.Run("UpdatesEmptyModelWithBuiltinDefault", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Config with empty model - should get builtin default
		configContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: ""
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		changes, err := MigrateConfigOnUpdate(tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate() error = %v", err)
		}

		if len(changes) == 0 {
			t.Error("Expected migration changes for empty model")
		}

		cfg, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if cfg.Providers["minimax"].Model != "MiniMax-M2.5" {
			t.Errorf("Provider model = %q, want %q", cfg.Providers["minimax"].Model, "MiniMax-M2.5")
		}
	})

	t.Run("NoChangesWhenModelAlreadyMatchesBuiltin", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Config already using current builtin default
		configContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2.5
default_models:
  minimax: MiniMax-M2.5
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		changes, err := MigrateConfigOnUpdate(tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate() error = %v", err)
		}

		// No changes expected since already matches builtin
		if len(changes) != 0 {
			t.Errorf("Expected no changes, got %d", len(changes))
		}

		cfg, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Should remain unchanged
		if cfg.Providers["minimax"].Model != "MiniMax-M2.5" {
			t.Errorf("Provider model = %q, want %q", cfg.Providers["minimax"].Model, "MiniMax-M2.5")
		}
	})

	t.Run("UpdatesUserSetModelToNewBuiltin", func(t *testing.T) {
		// This is the actual bug: user set MiniMax-M2.5 but builtin changed
		// to MiniMax-M2.5 (or similar), and the model wasn't being updated
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// User explicitly set a model that differs from current builtin
		// After migration, it should be updated to the new builtin default
		configContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2
default_models:
  minimax: MiniMax-M2
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		changes, err := MigrateConfigOnUpdate(tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate() error = %v", err)
		}

		// Should have changes since model differs from new builtin
		if len(changes) == 0 {
			t.Error("Expected migration changes when user model differs from builtin")
		}

		cfg, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Should be updated to new builtin
		if cfg.Providers["minimax"].Model != "MiniMax-M2.5" {
			t.Errorf("Provider model = %q, want %q", cfg.Providers["minimax"].Model, "MiniMax-M2.5")
		}
	})

	t.Run("NoMigrationForNonBuiltinProvider", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Custom provider should not be migrated
		configContent := `default_provider: custom
providers:
  custom:
    name: My Custom Provider
    base_url: https://api.custom.com
    model: my-model
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		changes, err := MigrateConfigOnUpdate(tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate() error = %v", err)
		}

		if len(changes) != 0 {
			t.Errorf("Expected no changes for custom provider, got %d", len(changes))
		}
	})

	t.Run("NoMigrationWhenNoConfig", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := MigrateConfigOnUpdate(tmpDir)
		// When config doesn't exist, LoadConfig returns ErrConfigNotFound
		// which is not os.IsNotExist, so it returns an error
		if err == nil {
			t.Error("Expected error for missing config")
		}
	})

	t.Run("NoMigrationWhenNoProviders", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: ""
providers: {}
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		changes, err := MigrateConfigOnUpdate(tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate() error = %v", err)
		}

		if len(changes) != 0 {
			t.Errorf("Expected no changes for empty providers, got %d", len(changes))
		}
	})

	t.Run("ProviderWithNoBuiltinModelNotMigrated", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Anthropic has no builtin default model, should not be migrated
		configContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
    base_url: ""
    model: ""
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		changes, err := MigrateConfigOnUpdate(tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate() error = %v", err)
		}

		if len(changes) != 0 {
			t.Errorf("Expected no changes for provider without builtin model, got %d", len(changes))
		}
	})
}

func TestFormatMigrationChanges(t *testing.T) {
	t.Run("FormatsChangesCorrectly", func(t *testing.T) {
		changes := []MigrationChange{
			{Provider: "minimax", Field: "model", Old: "MiniMax-M2", New: "MiniMax-M2.5"},
			{Provider: "zai", Field: "model", Old: "glm-4.5", New: "glm-4.7"},
		}

		result := FormatMigrationChanges(changes)
		if result == "" {
			t.Error("Expected non-empty formatted output")
		}

		// Check that it contains expected content
		if !contains(result, "minimax") {
			t.Error("Formatted output should contain 'minimax'")
		}
		if !contains(result, "MiniMax-M2") || !contains(result, "MiniMax-M2.5") {
			t.Error("Formatted output should show old and new values")
		}
	})

	t.Run("ReturnsEmptyForNoChanges", func(t *testing.T) {
		result := FormatMigrationChanges([]MigrationChange{})
		if result != "" {
			t.Errorf("Expected empty string for no changes, got %q", result)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
