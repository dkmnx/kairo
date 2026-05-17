package fsutil

import (
	"os"
)

const atomicFilePerms = 0o600

// WriteAtomic writes to a file atomically by writing to a temp file then renaming.
// The writeFn receives the open temp file and should handle all writing.
// On any error, the temp file is cleaned up automatically.
func WriteAtomic(path string, writeFn func(f *os.File) error) error {
	tempPath := path + ".tmp"

	f, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, atomicFilePerms)
	if err != nil {
		return err
	}

	if err := writeFn(f); err != nil {
		f.Close()
		os.Remove(tempPath)

		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)

		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)

		return err
	}

	return nil
}
