package env

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("returns platform-specific config dir from home", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("cannot find home directory")
		}

		var expected string
		if runtime.GOOS == "windows" {
			expected = filepath.Join(home, "AppData", "Roaming", "kairo")
		} else {
			expected = filepath.Join(home, ".config", "kairo")
		}
		dir := GetConfigDir()
		if dir != expected {
			t.Errorf("GetConfigDir() = %q, want %q", dir, expected)
		}
	})
}

func TestGetConfigDirWithOverride(t *testing.T) {
	original := GetConfigDir()
	defer SetConfigDir(original)

	tmpDir := t.TempDir()
	SetConfigDir(tmpDir)

	dir := GetConfigDir()
	if dir != tmpDir {
		t.Errorf("GetConfigDir() = %q, want %q", dir, tmpDir)
	}
}

func TestGetConfigDirEmptyOverride(t *testing.T) {
	original := GetConfigDir()
	defer SetConfigDir(original)

	SetConfigDir("")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot find home directory")
	}

	var expected string
	if runtime.GOOS == "windows" {
		expected = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		expected = filepath.Join(home, ".config", "kairo")
	}
	dir := GetConfigDir()
	if dir != expected {
		t.Errorf("GetConfigDir() = %q, want %q", dir, expected)
	}
}

func TestEnv_ConcurrentAccess(t *testing.T) {
	// This test verifies that concurrent access to configDir is safe.
	// Without proper synchronization (sync.RWMutex), -race would detect a data race.

	t.Run("concurrent GetConfigDir calls are safe", func(t *testing.T) {
		original := GetConfigDir()
		defer SetConfigDir(original)

		tmpDir := t.TempDir()
		SetConfigDir(tmpDir)

		// Simulate concurrent reads
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				_ = GetConfigDir()
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent SetConfigDir and GetConfigDir calls are safe", func(t *testing.T) {
		original := GetConfigDir()
		defer SetConfigDir(original)

		SetConfigDir(t.TempDir())

		done := make(chan bool)

		// Concurrent reads
		for i := 0; i < 10; i++ {
			go func() {
				_ = GetConfigDir()
				done <- true
			}()
		}

		// Concurrent writes
		for i := 0; i < 5; i++ {
			go func(n int) {
				SetConfigDir(t.TempDir())
				_ = GetConfigDir()
				done <- true
			}(i)
		}

		for i := 0; i < 15; i++ {
			<-done
		}
	})
}
