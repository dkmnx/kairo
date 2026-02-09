package backup

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateBackup(t *testing.T) {
	// Setup: create temp dir with mock key files
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	// Create mock files
	os.WriteFile(keyPath, []byte("test-key"), 0600)
	os.WriteFile(secretsPath, []byte("test-secrets"), 0600)

	// Test backup creation
	backupPath, err := CreateBackup(tmpDir)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if backupPath == "" {
		t.Fatal("backupPath is empty")
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file does not exist: %s", backupPath)
	}

	// Verify zip contents
	r, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer r.Close()

	if len(r.File) != 2 {
		t.Errorf("Expected 2 files, got %d", len(r.File))
	}

	// Verify each file is readable
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Errorf("Failed to open %s: %v", f.Name, err)
		}
		io.ReadAll(rc)
		rc.Close()
	}
}
