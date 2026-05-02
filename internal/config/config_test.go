package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}

	provider, ok := cfg.Providers["zai"]
	if !ok {
		t.Fatal("providers['zai'] not found")
	}
	if provider.Name != "Z.AI" {
		t.Errorf("providers['zai'].Name = %q, want %q", provider.Name, "Z.AI")
	}
	if provider.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("providers['zai'].BaseURL = %q, want %q", provider.BaseURL, "https://api.z.ai/api/anthropic")
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) should error when file not found")
	}
	if !errors.Is(err, kairoerrors.ErrConfigNotFound) {
		t.Errorf("LoadConfig(context.Background(), ) error = %v, want %v", err, kairoerrors.ErrConfigNotFound)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `invalid: yaml: content: [`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) should error on invalid YAML")
	}
}

func TestLoadConfigUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test that unknown fields are rejected with clear error message
	// This helps users catch configuration typos and ensures forward compatibility
	// (older binaries will fail gracefully when reading newer config formats)
	configWithUnknownField := `default_provider: zai
unknown_field: some_value
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
	if err := os.WriteFile(configPath, []byte(configWithUnknownField), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) should reject unknown fields")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error message should mention 'unknown' field, got: %v", err)
	}
}

func TestLoadConfigEmptyProviders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: zai
providers: {}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if cfg.Providers == nil {
		t.Error("Providers should not be nil")
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("Providers count = %d, want 0", len(cfg.Providers))
	}
}

func TestLoadConfigNoProviders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: zai
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if cfg.Providers == nil {
		t.Error("Providers should not be nil when omitted")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		DefaultProvider: "anthropic",
		Providers: map[string]Provider{
			"anthropic": {
				Name:    "Native Anthropic",
				BaseURL: "",
				Model:   "",
			},
		},
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("config file is empty")
	}
}

func TestSaveConfigCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		DefaultProvider: "test",
		Providers:       make(map[string]Provider),
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestSaveConfigPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		DefaultProvider: "test",
		Providers:       make(map[string]Provider),
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Skip strict permission check on Windows (doesn't support Unix-style 0600)
	if runtime.GOOS != "windows" {
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("File permissions = %o, want 0600", perm)
		}
	}
}

func TestProviderStruct(t *testing.T) {
	p := Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		Model:   "test-model",
		EnvVars: []string{"VAR1=value1", "VAR2=value2"},
	}

	if p.Name != "Test Provider" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Provider")
	}
	if p.BaseURL != "https://api.test.com" {
		t.Errorf("BaseURL = %q, want %q", p.BaseURL, "https://api.test.com")
	}
	if p.Model != "test-model" {
		t.Errorf("Model = %q, want %q", p.Model, "test-model")
	}
	if len(p.EnvVars) != 2 {
		t.Errorf("EnvVars count = %d, want 2", len(p.EnvVars))
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DefaultProvider: "test-provider",
		Providers: map[string]Provider{
			"test": {
				Name:    "Test",
				BaseURL: "https://test.com",
				Model:   "model",
			},
		},
	}

	if cfg.DefaultProvider != "test-provider" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "test-provider")
	}
	if len(cfg.Providers) != 1 {
		t.Errorf("Providers count = %d, want 1", len(cfg.Providers))
	}
}

func TestMigrateConfigFile(t *testing.T) {
	t.Run("NoMigrationWhenNoOldConfig", func(t *testing.T) {
		tmpDir := t.TempDir()

		migrated, err := migrateConfigFile(tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile() error = %v", err)
		}
		if migrated {
			t.Error("Expected no migration when old config doesn't exist")
		}

		oldPath := filepath.Join(tmpDir, "config")
		newPath := filepath.Join(tmpDir, "config.yaml")
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Error("Old config file should not exist")
		}
		if _, err := os.Stat(newPath); !os.IsNotExist(err) {
			t.Error("New config file should not exist")
		}
	})

	t.Run("SuccessfulMigration", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
		if err := os.WriteFile(oldConfigPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile() error = %v", err)
		}
		if !migrated {
			t.Error("Expected migration to occur")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		data, err := os.ReadFile(newConfigPath)
		if err != nil {
			t.Fatalf("Failed to read new config file: %v", err)
		}

		if string(data) != configContent {
			t.Errorf("Migrated content mismatch.\nGot:\n%s\nWant:\n%s", string(data), configContent)
		}

		backupPath := oldConfigPath + ".backup"
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("Old config file should be backed up, not deleted")
		}

		if _, err := os.Stat(oldConfigPath); !os.IsNotExist(err) {
			t.Error("Old config file should be renamed to backup")
		}
	})

	t.Run("NoMigrationWhenNewConfigExists", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")
		newConfigPath := filepath.Join(tmpDir, "config.yaml")

		oldContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
`
		newContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
`

		if err := os.WriteFile(oldConfigPath, []byte(oldContent), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newConfigPath, []byte(newContent), 0600); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile() error = %v", err)
		}
		if migrated {
			t.Error("Should not migrate when new config already exists")
		}

		data, err := os.ReadFile(newConfigPath)
		if err != nil {
			t.Fatalf("Failed to read new config: %v", err)
		}
		if string(data) != newContent {
			t.Error("New config file should not be overwritten")
		}

		if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
			t.Error("Old config file should still exist")
		}
	})

	t.Run("MigrationFailsWithInvalidYAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		invalidYAML := `invalid: yaml: content: [`
		if err := os.WriteFile(oldConfigPath, []byte(invalidYAML), 0600); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(tmpDir)
		if err == nil {
			t.Error("Expected error when migrating invalid YAML")
		}
		if migrated {
			t.Error("Should not report migration on error")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		if _, err := os.Stat(newConfigPath); !os.IsNotExist(err) {
			t.Error("New config file should not be created when old has invalid YAML")
		}

		if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
			t.Error("Old config file should still exist after failed migration")
		}
	})

	t.Run("MigrationPreservesPermissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
`
		if err := os.WriteFile(oldConfigPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile() error = %v", err)
		}
		if !migrated {
			t.Error("Expected migration to occur")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		info, err := os.Stat(newConfigPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0644 {
			t.Errorf("Permissions not preserved: got %o, want %o", info.Mode().Perm(), 0644)
		}
	})
}

func TestLoadConfigWithMigration(t *testing.T) {
	t.Run("LoadConfigMigratesOldFormat", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
		if err := os.WriteFile(oldConfigPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		if cfg.DefaultProvider != "zai" {
			t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
		}

		provider, ok := cfg.Providers["zai"]
		if !ok {
			t.Fatal("zai provider not found")
		}
		if provider.Name != "Z.AI" {
			t.Errorf("Provider name = %q, want %q", provider.Name, "Z.AI")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		if _, err := os.Stat(newConfigPath); os.IsNotExist(err) {
			t.Error("New config.yaml should exist after LoadConfig with migration")
		}

		backupPath := oldConfigPath + ".backup"
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("Old config should be backed up")
		}
	})

	t.Run("LoadConfigWorksWhenAlreadyMigrated", func(t *testing.T) {
		tmpDir := t.TempDir()
		newConfigPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
`
		if err := os.WriteFile(newConfigPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		if cfg.DefaultProvider != "anthropic" {
			t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "anthropic")
		}
	})
}

func TestLoadConfigEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Empty file should return error (not valid YAML)
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) on empty file should error")
	}
}

func TestLoadConfigWhitespaceOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte("   \n\n   \n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Whitespace-only file should return error (not valid YAML)
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) on whitespace-only file should error")
	}
}

func TestLoadConfigCommentOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	commentContent := `# This is a comment
# Another comment
`
	if err := os.WriteFile(configPath, []byte(commentContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Comment-only file returns error (YAML parser requires content)
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) on comment-only file should error")
	}
}
