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
	original := configDir
	defer func() { configDir = original }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	dir := GetConfigDir()
	if dir != tmpDir {
		t.Errorf("GetConfigDir() = %q, want %q", dir, tmpDir)
	}
}

func TestGetConfigDirEmptyOverride(t *testing.T) {
	original := configDir
	defer func() { configDir = original }()

	configDir = ""
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
