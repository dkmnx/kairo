package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func CreateBackup(configDir string) (string, error) {
	backupDir := filepath.Join(configDir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"create backup dir", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("kairo_backup_%s.zip", timestamp))

	zipFile, err := os.Create(backupPath)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"create zip", err)
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
			return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
				fmt.Sprintf("open %s", f), err)
		}

		w, err := zipWriter.Create(f)
		if err != nil {
			src.Close()
			return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
				fmt.Sprintf("create zip entry %s", f), err)
		}

		if _, err := io.Copy(w, src); err != nil {
			src.Close()
			return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
				fmt.Sprintf("write %s", f), err)
		}
		src.Close()
	}

	// Explicitly close and check for flush errors
	if err := zipWriter.Close(); err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"close zip writer", err)
	}
	return backupPath, nil
}

func RestoreBackup(configDir, backupPath string) error {
	r, err := zip.OpenReader(backupPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"open backup", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Skip directory entries
		if f.Mode().IsDir() {
			continue
		}

		destPath := filepath.Join(configDir, f.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				fmt.Sprintf("create dir for %s", f.Name), err)
		}

		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				fmt.Sprintf("create %s", f.Name), err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				fmt.Sprintf("open %s in zip", f.Name), err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				fmt.Sprintf("extract %s", f.Name), err)
		}

		outFile.Close()
		rc.Close()
	}

	return nil
}
