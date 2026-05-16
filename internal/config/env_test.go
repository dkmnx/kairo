package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigDir(t *testing.T) {
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
		dir, err := ConfigDir()
		if err != nil {
			t.Fatalf("ConfigDir() returned error: %v", err)
		}
		if dir != expected {
			t.Errorf("ConfigDir() = %q, want %q", dir, expected)
		}
	})
}

func TestConfigDirWithOverride(t *testing.T) {
	original, _ := ConfigDir()
	defer SetConfigDir(original)

	tmpDir := t.TempDir()
	SetConfigDir(tmpDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() returned error: %v", err)
	}
	if dir != tmpDir {
		t.Errorf("ConfigDir() = %q, want %q", dir, tmpDir)
	}
}

func TestConfigDirEmptyOverride(t *testing.T) {
	original, _ := ConfigDir()
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
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() returned error: %v", err)
	}
	if dir != expected {
		t.Errorf("ConfigDir() = %q, want %q", dir, expected)
	}
}

func TestEnv_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent ConfigDir calls are safe", func(t *testing.T) {
		original, _ := ConfigDir()
		defer SetConfigDir(original)

		tmpDir := t.TempDir()
		SetConfigDir(tmpDir)

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				_, _ = ConfigDir()
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent SetConfigDir and ConfigDir calls are safe", func(t *testing.T) {
		original, _ := ConfigDir()
		defer SetConfigDir(original)

		SetConfigDir(t.TempDir())

		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func() {
				_, _ = ConfigDir()
				done <- true
			}()
		}

		for i := 0; i < 5; i++ {
			go func(n int) {
				SetConfigDir(t.TempDir())
				_, _ = ConfigDir()
				done <- true
			}(i)
		}

		for i := 0; i < 15; i++ {
			<-done
		}
	})
}
