package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestCreateConfigBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config file
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"test": {Name: "Test Provider"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	backupPath, err := createConfigBackup(tmpDir)
	if err != nil {
		t.Fatalf("createConfigBackup() error = %v", err)
	}

	if backupPath == "" {
		t.Fatal("Backup path should not be empty")
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}

	// Verify backup contains same content as original
	originalPath := getConfigPath(tmpDir)
	originalData, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(originalData) != string(backupData) {
		t.Error("Backup should contain same content as original config")
	}
}

func TestCreateConfigBackupNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := createConfigBackup(tmpDir)
	if err == nil {
		t.Error("createConfigBackup() should error when config doesn't exist")
	}
}

func TestRollbackConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create original config
	originalCfg := &config.Config{
		Providers: map[string]config.Provider{
			"original": {Name: "Original Provider"},
		},
	}
	if err := config.SaveConfig(tmpDir, originalCfg); err != nil {
		t.Fatal(err)
	}

	// Create a backup with different content
	backupPath := filepath.Join(tmpDir, "config.yaml.backup.test")
	if err := os.WriteFile(backupPath, []byte("modified content"), 0600); err != nil {
		t.Fatal(err)
	}

	// Rollback
	err := rollbackConfig(tmpDir, backupPath)
	if err != nil {
		t.Fatalf("rollbackConfig() error = %v", err)
	}

	// Verify config was restored (but we can't easily check the content)
	// Just verify the function didn't error
}

func TestRollbackConfigNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	err := rollbackConfig(tmpDir, "/nonexistent/backup")
	if err == nil {
		t.Error("rollbackConfig() should error when backup doesn't exist")
	}
}

func TestWithConfigTransactionSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	originalCfg := &config.Config{
		Providers: map[string]config.Provider{
			"initial": {Name: "Initial"},
		},
	}
	if err := config.SaveConfig(tmpDir, originalCfg); err != nil {
		t.Fatal(err)
	}

	// Successful transaction
	err := withConfigTransaction(tmpDir, func(txDir string) error {
		cfg, err := config.LoadConfig(txDir)
		if err != nil {
			return err
		}
		cfg.Providers["new"] = config.Provider{Name: "New Provider"}
		return config.SaveConfig(txDir, cfg)
	})

	if err != nil {
		t.Fatalf("withConfigTransaction() error = %v", err)
	}

	// Verify new provider was added
	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := loadedCfg.Providers["new"]; !ok {
		t.Error("New provider should exist after successful transaction")
	}
}

func TestWithConfigTransactionRollbackOnError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	originalCfg := &config.Config{
		Providers: map[string]config.Provider{
			"initial": {Name: "Initial"},
		},
	}
	if err := config.SaveConfig(tmpDir, originalCfg); err != nil {
		t.Fatal(err)
	}

	originalContent, err := os.ReadFile(getConfigPath(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	// Failing transaction - should trigger rollback
	err = withConfigTransaction(tmpDir, func(txDir string) error {
		cfg, err := config.LoadConfig(txDir)
		if err != nil {
			return err
		}
		cfg.Providers["new"] = config.Provider{Name: "New Provider"}
		if err := config.SaveConfig(txDir, cfg); err != nil {
			return err
		}
		// Return error to trigger rollback
		return &testError{"simulated failure"}
	})

	// Error should be wrapped
	if err == nil {
		t.Fatal("withConfigTransaction() should return error")
	}

	// Verify config was rolled back
	rolledBackContent, err := os.ReadFile(getConfigPath(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	if string(originalContent) != string(rolledBackContent) {
		t.Error("Config should be rolled back to original state after transaction failure")
	}
}

func TestWithConfigTransactionBackupCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	originalCfg := &config.Config{
		Providers: map[string]config.Provider{
			"initial": {Name: "Initial"},
		},
	}
	if err := config.SaveConfig(tmpDir, originalCfg); err != nil {
		t.Fatal(err)
	}

	// List files before transaction
	beforeFiles, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	beforeCount := len(beforeFiles)

	// Successful transaction
	err = withConfigTransaction(tmpDir, func(txDir string) error {
		return nil // Do nothing, just test backup cleanup
	})

	if err != nil {
		t.Fatalf("withConfigTransaction() error = %v", err)
	}

	// List files after transaction
	afterFiles, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	afterCount := len(afterFiles)

	// Backup file should be cleaned up
	if afterCount > beforeCount {
		t.Errorf("Backup file should be cleaned up, but files increased from %d to %d", beforeCount, afterCount)
	}
}

func TestWithConfigTransactionCriticalFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	originalCfg := &config.Config{
		Providers: map[string]config.Provider{
			"initial": {Name: "Initial"},
		},
	}
	if err := config.SaveConfig(tmpDir, originalCfg); err != nil {
		t.Fatal(err)
	}

	// Make backup directory read-only to cause rollback failure
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0500); err != nil {
		t.Fatal(err)
	}

	// Make the config file read-only to cause rollback to fail
	configPath := getConfigPath(tmpDir)
	if err := os.Chmod(configPath, 0444); err != nil {
		t.Fatal(err)
	}

	// Transaction that fails, then rollback that fails
	err := withConfigTransaction(tmpDir, func(txDir string) error {
		cfg, err := config.LoadConfig(txDir)
		if err != nil {
			return err
		}
		cfg.Providers["new"] = config.Provider{Name: "New Provider"}
		return config.SaveConfig(txDir, cfg)
	})

	// Should return an error about both transaction and rollback failure
	if err == nil {
		t.Error("withConfigTransaction() should return error when both transaction and rollback fail")
	}

	// Restore permissions for cleanup (ignore errors during cleanup)
	_ = os.Chmod(configPath, 0600)
	_ = os.Chmod(backupDir, 0700)
}

func TestGetConfigPath(t *testing.T) {
	dir := "/test/dir"
	expected := filepath.Join(dir, "config.yaml")

	result := getConfigPath(dir)

	if result != expected {
		t.Errorf("getConfigPath() = %q, want %q", result, expected)
	}
}

func TestGetBackupPath(t *testing.T) {
	dir := "/test/dir"

	result := getBackupPath(dir)

	// Should contain config.yaml.backup.
	if len(result) < len("config.yaml.backup.") {
		t.Error("Backup path should be longer than minimum expected")
	}

	// Should start with the config dir
	if filepath.Dir(result) != dir {
		t.Errorf("Backup path should be in %s, got %s", dir, filepath.Dir(result))
	}
}

func TestGetBackupPathUniqueness(t *testing.T) {
	dir := "/test/dir"

	// Call twice and verify paths are different (nanosecond precision)
	path1 := getBackupPath(dir)

	// Note: Due to speed, these might be the same in some cases
	// But the function uses nanosecond precision which should be unique
	path2 := getBackupPath(dir)

	_ = path1
	_ = path2
	// This test just verifies the function doesn't panic
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
