package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateConfigOnUpdate(t *testing.T) {
	t.Run("PreservesUserModelWhenDifferentFromBuiltin", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

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

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if len(result.Changes) != 0 {
			t.Errorf("Expected no migration changes, got %d", len(result.Changes))
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		provider, ok := cfg.Providers["minimax"]
		if !ok {
			t.Fatal("minimax provider not found")
		}

		if provider.Model != "MiniMax-M2" {
			t.Errorf("Provider model = %q, want %q (user model should be preserved)", provider.Model, "MiniMax-M2")
		}

		if cfg.DefaultModels["minimax"] != "MiniMax-M2.7" {
			t.Errorf("DefaultModels[minimax] = %q, want %q", cfg.DefaultModels["minimax"], "MiniMax-M2.7")
		}
	})

	t.Run("UpdatesEmptyModelWithBuiltinDefault", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

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

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if len(result.Changes) == 0 {
			t.Error("Expected migration changes for empty model, got none")
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		provider, ok := cfg.Providers["minimax"]
		if !ok {
			t.Fatal("minimax provider not found")
		}

		if provider.Model != "MiniMax-M2.7" {
			t.Errorf("Provider model = %q, want %q", provider.Model, "MiniMax-M2.7")
		}
	})

	t.Run("NoChangeWhenModelAlreadyMatches", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2.7
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if len(result.Changes) != 0 {
			t.Errorf("Expected no changes when model matches, got %d", len(result.Changes))
		}
	})

	t.Run("NoChangeForNonBuiltinProvider", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: myprovider
providers:
  myprovider:
    name: My Provider
    base_url: https://myprovider.api.com
    model: myprovider-v1
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if len(result.Changes) != 0 {
			t.Errorf("Expected no changes for non-builtin provider, got %d", len(result.Changes))
		}

		if len(result.SkippedProviders) != 1 || result.SkippedProviders[0] != "myprovider" {
			t.Errorf("Expected 'myprovider' in skipped providers, got %v", result.SkippedProviders)
		}
	})

	t.Run("NoChangeForBuiltinWithoutModel", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: anthropic
providers:
  anthropic:
    name: Anthropic
    base_url: https://api.anthropic.com
    model: claude-3-opus
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if len(result.Changes) != 0 {
			t.Errorf("Expected no changes for provider without builtin model, got %d", len(result.Changes))
		}
	})

	t.Run("NoSaveWhenNoChanges", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		originalContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2.7
default_models:
  minimax: MiniMax-M2.7
`
		if err := os.WriteFile(configPath, []byte(originalContent), 0600); err != nil {
			t.Fatal(err)
		}

		originalStat, err := os.Stat(configPath)
		if err != nil {
			t.Fatal(err)
		}

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if len(result.Changes) != 0 {
			t.Errorf("Expected no changes, got %d", len(result.Changes))
		}

		newStat, err := os.Stat(configPath)
		if err != nil {
			t.Fatal(err)
		}

		if originalStat.ModTime() != newStat.ModTime() {
			t.Error("Config file was modified despite no changes being needed")
		}
	})

	t.Run("ReturnsNilForMissingConfig", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if result != nil {
			t.Errorf("Expected nil result for missing config, got %v", result)
		}
	})

	t.Run("ReturnsNilForEmptyProviders", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: ""
providers: {}
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if result != nil {
			t.Errorf("Expected nil result for empty providers, got %v", result)
		}
	})

	t.Run("InitializesDefaultModelsIfNil", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2.7
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		if cfg.DefaultModels == nil {
			t.Error("DefaultModels should be initialized, got nil")
		}

		if cfg.DefaultModels["minimax"] != "MiniMax-M2.7" {
			t.Errorf("DefaultModels[minimax] = %q, want %q", cfg.DefaultModels["minimax"], "MiniMax-M2.7")
		}
	})

	t.Run("PopulatesDefaultModelsForMatchingModels", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: minimax
providers:
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2.7
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		if cfg.DefaultModels["minimax"] != "MiniMax-M2.7" {
			t.Errorf("DefaultModels[minimax] = %q, want %q", cfg.DefaultModels["minimax"], "MiniMax-M2.7")
		}
	})

	t.Run("HandlesContextCancellation", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

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

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := MigrateConfigOnUpdate(ctx, tmpDir)
		if err == nil {
			t.Error("Expected error for canceled context")
		}
	})

	t.Run("PreservesCustomModelAndUpdatesDefaultModels", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: kimi
providers:
  kimi:
    name: Moonshot AI
    base_url: https://api.kimi.com/coding/
    model: my-custom-kimi-model
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		result, err := MigrateConfigOnUpdate(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("MigrateConfigOnUpdate(context.Background(), ) error = %v", err)
		}

		if len(result.Changes) != 0 {
			t.Errorf("Expected no changes for custom model, got %d", len(result.Changes))
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		provider, ok := cfg.Providers["kimi"]
		if !ok {
			t.Fatal("kimi provider not found")
		}

		if provider.Model != "my-custom-kimi-model" {
			t.Errorf("Provider model = %q, want %q (custom model should be preserved)", provider.Model, "my-custom-kimi-model")
		}

		if cfg.DefaultModels["kimi"] != "kimi-for-coding" {
			t.Errorf("DefaultModels[kimi] = %q, want %q", cfg.DefaultModels["kimi"], "kimi-for-coding")
		}
	})
}

func TestFormatMigrationChanges(t *testing.T) {
	t.Run("EmptyChanges", func(t *testing.T) {
		result := FormatMigrationChanges(nil)
		if result != "" {
			t.Errorf("FormatMigrationChanges(nil) = %q, want empty", result)
		}
	})

	t.Run("SingleChange", func(t *testing.T) {
		changes := []MigrationChange{
			{Provider: "minimax", Field: "model", Old: "MiniMax-M2", New: "MiniMax-M2.7"},
		}
		result := FormatMigrationChanges(changes)
		if !strings.Contains(result, "minimax") {
			t.Error("Expected result to contain provider name")
		}
		if !strings.Contains(result, "MiniMax-M2") {
			t.Error("Expected result to contain old model")
		}
		if !strings.Contains(result, "MiniMax-M2.7") {
			t.Error("Expected result to contain new model")
		}
	})

	t.Run("MultipleChanges", func(t *testing.T) {
		changes := []MigrationChange{
			{Provider: "minimax", Field: "model", Old: "MiniMax-M2", New: "MiniMax-M2.7"},
			{Provider: "kimi", Field: "model", Old: "kimi-old", New: "kimi-for-coding"},
		}
		result := FormatMigrationChanges(changes)
		if !strings.Contains(result, "minimax") || !strings.Contains(result, "kimi") {
			t.Error("Expected result to contain both provider names")
		}
	})
}
