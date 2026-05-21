package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteAtomic(t *testing.T) {
	t.Run("writes data to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "test.txt")

		err := WriteAtomic(path, func(f *os.File) error {
			_, err := f.WriteString("hello world")
			return err
		})
		if err != nil {
			t.Fatalf("WriteAtomic() error = %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		if string(data) != "hello world" {
			t.Errorf("content = %q, want %q", string(data), "hello world")
		}
	})

	t.Run("creates file with 0600 permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix permissions not enforced on Windows")
		}

		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "secure.txt")

		err := WriteAtomic(path, func(f *os.File) error {
			_, err := f.WriteString("secret")
			return err
		})
		if err != nil {
			t.Fatalf("WriteAtomic() error = %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}

		if info.Mode().Perm() != 0o600 {
			t.Errorf("permissions = %o, want %o", info.Mode().Perm(), 0o600)
		}
	})

	t.Run("cleans up temp file on write error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "fail.txt")

		writeErr := errors.New("write failed")
		err := WriteAtomic(path, func(f *os.File) error {
			return writeErr
		})
		if !errors.Is(err, writeErr) {
			t.Errorf("error = %v, want %v", err, writeErr)
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("target file should not exist after write error")
		}

		matches, _ := filepath.Glob(filepath.Join(tmpDir, "fail.txt.tmp.*"))
		if len(matches) > 0 {
			t.Errorf("temp file should be cleaned up after write error, found: %v", matches)
		}
	})

	t.Run("cleans up temp file on close error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "close-fail.txt")

		err := WriteAtomic(path, func(f *os.File) error {
			_, err := f.WriteString("data")
			if err != nil {
				return err
			}
			return f.Close()
		})
		if err == nil {
			t.Error("expected error from double close")
		}

		matches, _ := filepath.Glob(filepath.Join(tmpDir, "close-fail.txt.tmp.*"))
		if len(matches) > 0 {
			t.Errorf("temp file should be cleaned up after close error, found: %v", matches)
		}
	})

	t.Run("overwrites existing file atomically", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "existing.txt")

		if err := os.WriteFile(path, []byte("old content"), 0o600); err != nil {
			t.Fatal(err)
		}

		err := WriteAtomic(path, func(f *os.File) error {
			_, err := f.WriteString("new content")
			return err
		})
		if err != nil {
			t.Fatalf("WriteAtomic() error = %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		if string(data) != "new content" {
			t.Errorf("content = %q, want %q", string(data), "new content")
		}
	})
}
