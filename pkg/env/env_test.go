package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("returns default config dir from home", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("cannot find home directory")
		}

		expected := filepath.Join(home, ".config", "kairo")
		dir := GetConfigDir()
		if dir != expected {
			t.Errorf("GetConfigDir() = %q, want %q", dir, expected)
		}
	})
}
