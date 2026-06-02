package cmd

import (
	"testing"
)

func TestGlobalVariableAccess(t *testing.T) {
	t.Run("configDir can be set and retrieved", func(t *testing.T) {
		originalConfigDir := defaultCLIContext.ConfigDir()
		defer func() { defaultCLIContext.SetConfigDir(originalConfigDir) }()

		testDir := "/tmp/test-config-dir"
		defaultCLIContext.SetConfigDir(testDir)
		if got := defaultCLIContext.ConfigDir(); got != testDir {
			t.Errorf("defaultCLIContext.ConfigDir() = %q, want %q", got, testDir)
		}
	})

	t.Run("configDir falls back to default when empty", func(t *testing.T) {
		originalConfigDir := defaultCLIContext.ConfigDir()
		defer func() { defaultCLIContext.SetConfigDir(originalConfigDir) }()

		defaultCLIContext.SetConfigDir("")
		got := defaultCLIContext.ConfigDir()
		if got == "" {
			t.Errorf("defaultCLIContext.ConfigDir() should return default directory when empty, got empty string")
		}
	})

	t.Run("verbose can be set and retrieved", func(t *testing.T) {
		originalVerbose := verbose()
		defer func() { defaultCLIContext.SetVerbose(originalVerbose) }()

		defaultCLIContext.SetVerbose(true)
		if !verbose() {
			t.Error("verbose() should return true after defaultCLIContext.SetVerbose(true)")
		}

		defaultCLIContext.SetVerbose(false)
		if verbose() {
			t.Error("verbose() should return false after defaultCLIContext.SetVerbose(false)")
		}
	})
}
