package cmd

import (
	"testing"
)

func TestCLIContextAccessors(t *testing.T) {
	t.Run("configDir can be set and retrieved", func(t *testing.T) {
		originalConfigDir := testCLI.ConfigDir()
		defer func() { testCLI.SetConfigDir(originalConfigDir) }()

		testDir := "/tmp/test-config-dir"
		testCLI.SetConfigDir(testDir)
		if got := testCLI.ConfigDir(); got != testDir {
			t.Errorf("testCLI.ConfigDir() = %q, want %q", got, testDir)
		}
	})

	t.Run("configDir falls back to default when empty", func(t *testing.T) {
		originalConfigDir := testCLI.ConfigDir()
		defer func() { testCLI.SetConfigDir(originalConfigDir) }()

		testCLI.SetConfigDir("")
		got := testCLI.ConfigDir()
		if got == "" {
			t.Errorf("testCLI.ConfigDir() should return default directory when empty, got empty string")
		}
	})

	t.Run("verbose can be set and retrieved", func(t *testing.T) {
		originalVerbose := testCLI.Verbose()
		defer func() { testCLI.SetVerbose(originalVerbose) }()

		testCLI.SetVerbose(true)
		if !testCLI.Verbose() {
			t.Error("verbose() should return true after testCLI.SetVerbose(true)")
		}

		testCLI.SetVerbose(false)
		if testCLI.Verbose() {
			t.Error("verbose() should return false after testCLI.SetVerbose(false)")
		}
	})
}
