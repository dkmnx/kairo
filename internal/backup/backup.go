package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func CreateBackup(configDir string) (string, error) {
	backupDir := filepath.Join(configDir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("kairo_backup_%s.zip", timestamp))

	zipFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	files := []string{"age.key", "secrets.age", "config.yaml"}
	for _, f := range files {
		srcPath := filepath.Join(configDir, f)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		src, err := os.Open(srcPath)
		if err != nil {
			return "", fmt.Errorf("open %s: %w", f, err)
		}

		w, err := zipWriter.Create(f)
		if err != nil {
			src.Close()
			return "", fmt.Errorf("create zip entry %s: %w", f, err)
		}

		if _, err := io.Copy(w, src); err != nil {
			src.Close()
			return "", fmt.Errorf("write %s: %w", f, err)
		}
		src.Close()
	}

	// Explicitly close and check for flush errors
	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("close zip writer: %w", err)
	}
	return backupPath, nil
}
