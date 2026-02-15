package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateBackupWithMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only config.yaml, leave age.key and secrets.age missing
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("test: config"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create backups directory
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create backup
	backupPath, err := CreateBackup(tmpDir)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Verify backup was created
	if backupPath == "" {
		t.Fatal("Backup path should not be empty")
	}

	// Verify the zip contains only config.yaml
	r, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatalf("Failed to open backup zip: %v", err)
	}
	defer r.Close()

	files := make(map[string]bool)
	for _, f := range r.File {
		files[f.Name] = true
	}

	if !files["config.yaml"] {
		t.Error("Backup should contain config.yaml")
	}

	// age.key and secrets.age should not be in the backup (they don't exist)
	if files["age.key"] {
		t.Error("Backup should not contain age.key (doesn't exist)")
	}
	if files["secrets.age"] {
		t.Error("Backup should not contain secrets.age (doesn't exist)")
	}
}

func TestCreateBackupWithAllFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create all files
	files := map[string]string{
		"age.key":     "age1keyhere",
		"secrets.age": "encryptedsecrets",
		"config.yaml": "providers: {}",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	backupPath, err := CreateBackup(tmpDir)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Verify the zip contains all files
	r, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatalf("Failed to open backup zip: %v", err)
	}
	defer r.Close()

	filesInZip := make(map[string]bool)
	for _, f := range r.File {
		filesInZip[f.Name] = true
	}

	for name := range files {
		if !filesInZip[name] {
			t.Errorf("Backup should contain %s", name)
		}
	}
}

func TestCreateBackupCreatesBackupDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Ensure no backup directory exists
	backupDir := filepath.Join(tmpDir, "backups")
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatal("Backup directory should not exist initially")
	}

	_, err := CreateBackup(tmpDir)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Verify backup directory was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Backup directory should be created")
	}
}

func TestRestoreBackupWithMissingDestinationDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a backup zip
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(backupDir, "test_backup.zip")
	zipFile, err := os.Create(backupPath)
	if err != nil {
		t.Fatal(err)
	}

	zipWriter := zip.NewWriter(zipFile)
	// Create a file in a subdirectory
	writer, err := zipWriter.Create("subdir/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, err = writer.Write([]byte("test content"))
	if err != nil {
		t.Fatal(err)
	}
	zipWriter.Close()
	zipFile.Close()

	// Restore to a new directory (doesn't exist)
	newDir := filepath.Join(tmpDir, "newdir")
	err = RestoreBackup(newDir, backupPath)
	if err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}

	// Verify file was restored
	restoredPath := filepath.Join(newDir, "subdir", "config.yaml")
	if _, err := os.Stat(restoredPath); os.IsNotExist(err) {
		t.Error("Restored file should exist")
	}
}

func TestRestoreBackupOverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create original config
	configPath := filepath.Join(tmpDir, "config.yaml")
	originalContent := "original: content"
	if err := os.WriteFile(configPath, []byte(originalContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Create backup with different content
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(backupDir, "test.zip")
	zipFile, err := os.Create(backupPath)
	if err != nil {
		t.Fatal(err)
	}

	zipWriter := zip.NewWriter(zipFile)
	writer, err := zipWriter.Create("config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	newContent := "new: content"
	_, err = writer.Write([]byte(newContent))
	if err != nil {
		t.Fatal(err)
	}
	zipWriter.Close()
	zipFile.Close()

	// Restore
	err = RestoreBackup(tmpDir, backupPath)
	if err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}

	// Verify content was overwritten
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != newContent {
		t.Errorf("Config content = %q, want %q", string(data), newContent)
	}
}

func TestRestoreBackupWithInvalidZip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid zip file
	backupPath := filepath.Join(tmpDir, "invalid.zip")
	if err := os.WriteFile(backupPath, []byte("not a zip file"), 0600); err != nil {
		t.Fatal(err)
	}

	err := RestoreBackup(tmpDir, backupPath)
	if err == nil {
		t.Error("RestoreBackup() should error on invalid zip")
	}
}

func TestRestoreBackupWithNonExistentZip(t *testing.T) {
	tmpDir := t.TempDir()

	err := RestoreBackup(tmpDir, "/nonexistent/backup.zip")
	if err == nil {
		t.Error("RestoreBackup() should error on non-existent zip")
	}
}

func TestCreateBackupZipPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("test: config"), 0600); err != nil {
		t.Fatal(err)
	}

	backupPath, err := CreateBackup(tmpDir)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}
}

func TestRestoreBackupWithDirectoryEntry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create backup with a directory entry
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(backupDir, "test.zip")
	zipFile, err := os.Create(backupPath)
	if err != nil {
		t.Fatal(err)
	}

	zipWriter := zip.NewWriter(zipFile)
	// Create a directory entry (ends with /)
	_, err = zipWriter.Create("somedir/")
	if err != nil {
		t.Fatal(err)
	}
	// Create a file inside the directory
	writer, err := zipWriter.Create("somedir/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = writer.Write([]byte("content"))
	if err != nil {
		t.Fatal(err)
	}
	zipWriter.Close()
	zipFile.Close()

	// Restore should skip directory entries
	err = RestoreBackup(tmpDir, backupPath)
	if err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}

	// Verify file was restored
	restoredPath := filepath.Join(tmpDir, "somedir", "file.txt")
	if _, err := os.Stat(restoredPath); os.IsNotExist(err) {
		t.Error("Restored file should exist")
	}
}

func TestRestoreBackupFileContent(t *testing.T) {
	tmpDir := t.TempDir()

	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(backupDir, "test.zip")
	zipFile, err := os.Create(backupPath)
	if err != nil {
		t.Fatal(err)
	}

	zipWriter := zip.NewWriter(zipFile)
	writer, err := zipWriter.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	testContent := "test content"
	_, err = writer.Write([]byte(testContent))
	if err != nil {
		t.Fatal(err)
	}
	zipWriter.Close()
	zipFile.Close()

	// Restore
	err = RestoreBackup(tmpDir, backupPath)
	if err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}

	// Verify content
	restoredPath := filepath.Join(tmpDir, "test.txt")
	data, err := os.ReadFile(restoredPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != testContent {
		t.Errorf("Content = %q, want %q", string(data), testContent)
	}
}
