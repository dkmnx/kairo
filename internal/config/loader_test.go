package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	_, err := LoadConfig(ctx, tmpDir)
	if err == nil {
		t.Error("LoadConfig with cancelled context should return error")
	}
}

func TestSaveConfig_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	cfg := &Config{Providers: map[string]Provider{}}
	err := SaveConfig(ctx, tmpDir, cfg)
	if err == nil {
		t.Error("SaveConfig with cancelled context should return error")
	}
}

func TestMigrateConfigFile_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	migrated, err := migrateConfigFile(ctx, tmpDir)
	if err == nil {
		t.Error("migrateConfigFile with cancelled context should return error")
	}
	if migrated {
		t.Error("migrateConfigFile should not report migration on cancelled context")
	}
}

func TestValidate_ClearsInvalidDefaultProvider(t *testing.T) {
	cfg := &Config{
		DefaultProvider: "nonexistent",
		Providers: map[string]Provider{
			"actual": {Name: "Actual"},
		},
	}
	cfg.validate()

	if cfg.DefaultProvider != "" {
		t.Errorf("validate() should clear DefaultProvider when not in Providers, got %q", cfg.DefaultProvider)
	}
}

func TestValidate_KeepsValidDefaultProvider(t *testing.T) {
	cfg := &Config{
		DefaultProvider: "actual",
		Providers: map[string]Provider{
			"actual": {Name: "Actual"},
		},
	}
	cfg.validate()

	if cfg.DefaultProvider != "actual" {
		t.Errorf("validate() should keep valid DefaultProvider, got %q", cfg.DefaultProvider)
	}
}

func TestValidate_EmptyDefaultProvider_NoOp(t *testing.T) {
	cfg := &Config{
		DefaultProvider: "",
		Providers:       map[string]Provider{},
	}
	cfg.validate()

	if cfg.DefaultProvider != "" {
		t.Errorf("validate() should not change empty DefaultProvider, got %q", cfg.DefaultProvider)
	}
}

func TestLoadConfig_MigrationOldFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "config")
	newPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `providers: [invalid
`
	if err := os.WriteFile(oldPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig should error when old config has invalid YAML")
	}

	if _, statErr := os.Stat(newPath); statErr == nil {
		t.Error("new config file should not be created when old config is invalid YAML")
	}
}

func TestMigrateConfigFile_NoOldFile(t *testing.T) {
	tmpDir := t.TempDir()

	migrated, err := migrateConfigFile(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("migrateConfigFile with no old file should not error, got: %v", err)
	}
	if migrated {
		t.Error("migrateConfigFile should report no migration when old file doesn't exist")
	}
}

func TestMigrateConfigFile_OldAndNewBothExist(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "config")
	newPath := filepath.Join(tmpDir, "config.yaml")

	content := `default_provider: test
providers:
  test:
    name: Test
    base_url: https://test.com
    model: test-model
`
	if err := os.WriteFile(oldPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	migrated, err := migrateConfigFile(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("migrateConfigFile should not error, got: %v", err)
	}
	if migrated {
		t.Error("migrateConfigFile should not migrate when new file already exists")
	}
}

func TestMigrateConfigFile_SuccessCreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "config")
	newPath := filepath.Join(tmpDir, "config.yaml")
	backupPath := oldPath + ".backup"

	content := `default_provider: test
providers:
  test:
    name: Test
    base_url: https://test.com
    model: test-model
`
	if err := os.WriteFile(oldPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	migrated, err := migrateConfigFile(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("migrateConfigFile should succeed, got: %v", err)
	}
	if !migrated {
		t.Error("migrateConfigFile should report migration")
	}

	if _, statErr := os.Stat(newPath); statErr != nil {
		t.Errorf("new config file should exist: %v", statErr)
	}
	if _, statErr := os.Stat(backupPath); statErr != nil {
		t.Errorf("backup file should exist: %v", statErr)
	}
	if _, statErr := os.Stat(oldPath); statErr == nil {
		t.Error("old config file should have been renamed")
	}

	newData, readErr := os.ReadFile(newPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(newData) != content {
		t.Errorf("new config content mismatch: got %q", string(newData))
	}
}

func TestLoadConfig_NonExistentDir(t *testing.T) {
	_, err := LoadConfig(context.Background(), "/nonexistent/dir/that/does/not/exist")
	if err == nil {
		t.Error("LoadConfig on nonexistent directory should return error")
	}
}
