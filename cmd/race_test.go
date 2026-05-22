package cmd

import (
	"testing"
)

func TestGlobalVariableAccess(t *testing.T) {
	t.Run("configDir can be set and retrieved", func(t *testing.T) {
		originalConfigDir := configDir()
		defer func() { setConfigDir(originalConfigDir) }()

		testDir := "/tmp/test-config-dir"
		setConfigDir(testDir)
		if got := configDir(); got != testDir {
			t.Errorf("configDir() = %q, want %q", got, testDir)
		}
	})

	t.Run("configDir falls back to default when empty", func(t *testing.T) {
		originalConfigDir := configDir()
		defer func() { setConfigDir(originalConfigDir) }()

		setConfigDir("")
		got := configDir()
		if got == "" {
			t.Errorf("configDir() should return default directory when empty, got empty string")
		}
	})

	t.Run("verbose can be set and retrieved", func(t *testing.T) {
		originalVerbose := verbose()
		defer func() { setVerbose(originalVerbose) }()

		setVerbose(true)
		if !verbose() {
			t.Error("verbose() should return true after setVerbose(true)")
		}

		setVerbose(false)
		if verbose() {
			t.Error("verbose() should return false after setVerbose(false)")
		}
	})
}
