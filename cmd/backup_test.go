package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal config structure
	_ = os.MkdirAll(filepath.Join(tmpDir, "backups"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "age.key"), []byte("test"), 0600)
	_ = os.WriteFile(filepath.Join(tmpDir, "secrets.age"), []byte("test"), 0600)

	rootCmd.SetArgs([]string{"--config", tmpDir, "backup"})
	err := rootCmd.Execute()

	if err != nil {
		t.Errorf("backup command failed: %v", err)
	}
}

func TestRestoreCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal config structure
	_ = os.MkdirAll(filepath.Join(tmpDir, "backups"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "age.key"), []byte("test-key"), 0600)
	_ = os.WriteFile(filepath.Join(tmpDir, "secrets.age"), []byte("test-secrets"), 0600)

	// Create a backup first
	rootCmd.SetArgs([]string{"--config", tmpDir, "backup"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("backup command failed: %v", err)
	}

	// Find the backup file
	backups, err := os.ReadDir(filepath.Join(tmpDir, "backups"))
	if err != nil {
		t.Fatalf("failed to read backups dir: %v", err)
	}
	if len(backups) == 0 {
		t.Fatal("no backup file created")
	}

	backupPath := filepath.Join(tmpDir, "backups", backups[0].Name())

	// Remove original files
	os.Remove(filepath.Join(tmpDir, "age.key"))
	os.Remove(filepath.Join(tmpDir, "secrets.age"))

	// Restore
	rootCmd.SetArgs([]string{"--config", tmpDir, "restore", backupPath})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("restore command failed: %v", err)
	}

	// Verify files restored
	if _, err := os.Stat(filepath.Join(tmpDir, "age.key")); os.IsNotExist(err) {
		t.Error("age.key not restored after restore command")
	}
}
