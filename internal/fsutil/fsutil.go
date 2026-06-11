// Package fsutil provides atomic file writing utilities for safe config persistence.
package fsutil

import (
	"os"
	"path/filepath"

	"github.com/dkmnx/kairo/internal/errors"
)

// WriteAtomic writes to a file atomically by writing to a temp file in the
// same directory then renaming. The writeFn receives the open temp file and
// should handle all writing. On any error, the temp file is cleaned up
// automatically. The temp file is created with 0600 permissions (os.CreateTemp
// default), so the final file is never world-readable.
func WriteAtomic(path string, writeFn func(f *os.File) error) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp.*")
	if err != nil {
		return errors.FileError("failed to create temp file", path, err)
	}
	tempPath := f.Name()

	if err := writeFn(f); err != nil {
		f.Close()
		os.Remove(tempPath)

		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)

		return errors.FileError("failed to close temp file", tempPath, err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)

		return errors.FileError("failed to rename temp file", path, err)
	}

	return nil
}
