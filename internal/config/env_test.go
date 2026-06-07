package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigDir(t *testing.T) {
	t.Run("returns platform-specific config dir from home", func(t *testing.T) {
		// Ensure KAIRO_CONFIG_DIR is not set for this test
		origEnv := os.Getenv("KAIRO_CONFIG_DIR")
		defer os.Setenv("KAIRO_CONFIG_DIR", origEnv)
		os.Unsetenv("KAIRO_CONFIG_DIR")

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

	t.Run("uses KAIRO_CONFIG_DIR env var when set", func(t *testing.T) {
		origEnv := os.Getenv("KAIRO_CONFIG_DIR")
		defer os.Setenv("KAIRO_CONFIG_DIR", origEnv)
		os.Setenv("KAIRO_CONFIG_DIR", "/custom/kairo/path")

		dir, err := ConfigDir()
		if err != nil {
			t.Fatalf("ConfigDir() returned error: %v", err)
		}
		if dir != "/custom/kairo/path" {
			t.Errorf("ConfigDir() = %q, want %q", dir, "/custom/kairo/path")
		}
	})
}

func TestConfigDirConcurrentAccess(t *testing.T) {
	t.Run("concurrent ConfigDir calls are safe", func(t *testing.T) {
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
}
