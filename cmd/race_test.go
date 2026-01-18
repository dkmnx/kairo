package cmd

import (
	"testing"
)

func TestGlobalVariableAccess(t *testing.T) {
	t.Run("configDir can be set and retrieved", func(t *testing.T) {
		originalConfigDir := getConfigDir()
		defer func() { setConfigDir(originalConfigDir) }()

		testDir := "/tmp/test-config-dir"
		setConfigDir(testDir)
		if got := getConfigDir(); got != testDir {
			t.Errorf("getConfigDir() = %q, want %q", got, testDir)
		}
	})

	t.Run("configDir falls back to default when empty", func(t *testing.T) {
		originalConfigDir := getConfigDir()
		defer func() { setConfigDir(originalConfigDir) }()

		setConfigDir("")
		got := getConfigDir()
		if got == "" {
			t.Errorf("getConfigDir() should return default directory when empty, got empty string")
		}
	})

	t.Run("verbose can be set and retrieved", func(t *testing.T) {
		originalVerbose := getVerbose()
		defer func() { setVerbose(originalVerbose) }()

		setVerbose(true)
		if !getVerbose() {
			t.Error("getVerbose() should return true after setVerbose(true)")
		}

		setVerbose(false)
		if getVerbose() {
			t.Error("getVerbose() should return false after setVerbose(false)")
		}
	})
}
