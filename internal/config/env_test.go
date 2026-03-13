package config

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
		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("GetConfigDir() returned error: %v", err)
		}
		if dir != expected {
			t.Errorf("GetConfigDir() = %q, want %q", dir, expected)
		}
	})
}

func TestGetConfigDirWithOverride(t *testing.T) {
	original, _ := GetConfigDir()
	defer SetConfigDir(original)

	tmpDir := t.TempDir()
	SetConfigDir(tmpDir)

	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() returned error: %v", err)
	}
	if dir != tmpDir {
		t.Errorf("GetConfigDir() = %q, want %q", dir, tmpDir)
	}
}

func TestGetConfigDirEmptyOverride(t *testing.T) {
	original, _ := GetConfigDir()
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
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() returned error: %v", err)
	}
	if dir != expected {
		t.Errorf("GetConfigDir() = %q, want %q", dir, expected)
	}
}

func TestEnv_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent GetConfigDir calls are safe", func(t *testing.T) {
		original, _ := GetConfigDir()
		defer SetConfigDir(original)

		tmpDir := t.TempDir()
		SetConfigDir(tmpDir)

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				_, _ = GetConfigDir()
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent SetConfigDir and GetConfigDir calls are safe", func(t *testing.T) {
		original, _ := GetConfigDir()
		defer SetConfigDir(original)

		SetConfigDir(t.TempDir())

		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func() {
				_, _ = GetConfigDir()
				done <- true
			}()
		}

		for i := 0; i < 5; i++ {
			go func(n int) {
				SetConfigDir(t.TempDir())
				_, _ = GetConfigDir()
				done <- true
			}(i)
		}

		for i := 0; i < 15; i++ {
			<-done
		}
	})
}
