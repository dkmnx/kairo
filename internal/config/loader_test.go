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
		t.Error("LoadConfig with canceled context should return error")
	}
}

func TestSaveConfig_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	cfg := &Config{Providers: map[string]Provider{}}
	err := SaveConfig(ctx, tmpDir, cfg)
	if err == nil {
		t.Error("SaveConfig with canceled context should return error")
	}
}

func TestMigrateConfigFile_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	migrated, err := migrateConfigFile(ctx, tmpDir)
	if err == nil {
		t.Error("migrateConfigFile with canceled context should return error")
	}
	if migrated {
		t.Error("migrateConfigFile should not report migration on canceled context")
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

func TestReconcileDefaultModels_PrunesStaleEntries(t *testing.T) {
	cfg := &Config{
		Providers: map[string]Provider{
			"active": {Name: "Active", Model: "model-v2"},
		},
		DefaultModels: map[string]string{
			"active":  "model-v1",
			"removed": "model-old",
			"deleted": "model-old",
		},
	}
	cfg.reconcileDefaultModels()

	if _, ok := cfg.DefaultModels["removed"]; ok {
		t.Error("reconcileDefaultModels should prune DefaultModels entry for removed provider 'removed'")
	}
	if _, ok := cfg.DefaultModels["deleted"]; ok {
		t.Error("reconcileDefaultModels should prune DefaultModels entry for removed provider 'deleted'")
	}
	if v := cfg.DefaultModels["active"]; v != "model-v1" {
		t.Errorf("reconcileDefaultModels should preserve existing DefaultModels value, got %q", v)
	}
}

func TestReconcileDefaultModels_PopulatesMissingEntries(t *testing.T) {
	cfg := &Config{
		Providers: map[string]Provider{
			"new-one": {Name: "New One", Model: "model-v2"},
			"new-two": {Name: "New Two", Model: "model-v1"},
		},
		DefaultModels: map[string]string{
			"existing": "model-old",
		},
	}
	cfg.reconcileDefaultModels()

	if v := cfg.DefaultModels["new-one"]; v != "model-v2" {
		t.Errorf("reconcileDefaultModels should populate missing entry for 'new-one', got %q", v)
	}
	if v := cfg.DefaultModels["new-two"]; v != "model-v1" {
		t.Errorf("reconcileDefaultModels should populate missing entry for 'new-two', got %q", v)
	}
}

func TestReconcileDefaultModels_PreservesExistingEntries(t *testing.T) {
	cfg := &Config{
		Providers: map[string]Provider{
			"keep": {Name: "Keep", Model: "model-v2"},
		},
		DefaultModels: map[string]string{
			"keep": "model-v1",
		},
	}
	cfg.reconcileDefaultModels()

	if v := cfg.DefaultModels["keep"]; v != "model-v1" {
		t.Errorf("reconcileDefaultModels should preserve explicit DefaultModels override, got %q", v)
	}
}

func TestReconcileDefaultModels_MixedAddAndPrune(t *testing.T) {
	cfg := &Config{
		Providers: map[string]Provider{
			"stays": {Name: "Stays", Model: "model-v2"},
			"new-p": {Name: "New P", Model: "model-v3"},
		},
		DefaultModels: map[string]string{
			"stays": "model-v1",
			"old-p": "model-old",
		},
	}
	cfg.reconcileDefaultModels()

	// stale removed
	if _, ok := cfg.DefaultModels["old-p"]; ok {
		t.Error("reconcileDefaultModels should prune removed provider 'old-p'")
	}
	// existing preserved
	if v := cfg.DefaultModels["stays"]; v != "model-v1" {
		t.Errorf("reconcileDefaultModels should preserve existing entry for 'stays', got %q", v)
	}
	// new populated
	if v := cfg.DefaultModels["new-p"]; v != "model-v3" {
		t.Errorf("reconcileDefaultModels should populate new entry for 'new-p', got %q", v)
	}
	if len(cfg.DefaultModels) != 2 {
		t.Errorf("reconcileDefaultModels should result in exactly 2 entries, got %d", len(cfg.DefaultModels))
	}
}

func TestReconcileDefaultModels_EmptyProviders(t *testing.T) {
	cfg := &Config{
		Providers:     map[string]Provider{},
		DefaultModels: map[string]string{"old": "model-old"},
	}
	cfg.reconcileDefaultModels()

	if len(cfg.DefaultModels) != 0 {
		t.Errorf("reconcileDefaultModels should prune all entries when no providers remain, got %d", len(cfg.DefaultModels))
	}
}

func TestReconcileDefaultModels_NilDefaultModels(t *testing.T) {
	cfg := &Config{
		Providers: map[string]Provider{
			"a": {Name: "A", Model: "model-a"},
		},
		DefaultModels: nil,
	}
	cfg.reconcileDefaultModels()

	if len(cfg.DefaultModels) != 1 {
		t.Errorf("reconcileDefaultModels should populate from nil, got %d entries", len(cfg.DefaultModels))
	}
	if v := cfg.DefaultModels["a"]; v != "model-a" {
		t.Errorf("reconcileDefaultModels should set model for 'a', got %q", v)
	}
}

func TestReconcileDefaultModels_NilProvidersNoPanic(t *testing.T) {
	cfg := &Config{
		Providers:     nil,
		DefaultModels: map[string]string{"old": "model-old"},
	}
	cfg.reconcileDefaultModels()

	// Should initialize DefaultModels to a non-nil empty map when nil, then
	// prune all entries since Providers is nil.
	if cfg.DefaultModels == nil {
		t.Error("reconcileDefaultModels should initialize DefaultModels map even when Providers is nil")
	}
	if len(cfg.DefaultModels) != 0 {
		t.Errorf("reconcileDefaultModels should prune all entries when Providers is nil, got %d", len(cfg.DefaultModels))
	}
}
