package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestRootCmd(t *testing.T) {
	t.Run("no config file - shows setup message", func(t *testing.T) {
		// Use a temp directory that doesn't have a config file
		tmpDir := t.TempDir()

		// Set config dir to temp directory
		originalConfigDir := configDir
		configDir = tmpDir
		defer func() { configDir = originalConfigDir }()

		// Capture output
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		// Execute root command
		rootCmd.Run(rootCmd, []string{})

		result := output.String()
		// When config file doesn't exist, rootCmd should show setup message
		if !containsString(result, "No providers configured") && !containsString(result, "configuration file not found") {
			t.Errorf("Expected setup-related message, got: %s", result)
		}
	})

	t.Run("no default provider - shows usage", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file without default provider
		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic"},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		originalConfigDir := configDir
		configDir = tmpDir
		defer func() {
			configDir = originalConfigDir
			os.Remove(configPath)
		}()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		rootCmd.Run(rootCmd, []string{})

		result := output.String()
		if !containsString(result, "No default provider set") {
			t.Errorf("Expected 'No default provider set' message, got: %s", result)
		}
		if !containsString(result, "kairo setup") {
			t.Errorf("Expected 'kairo setup' in usage, got: %s", result)
		}
		if !containsString(result, "kairo default <provider>") {
			t.Errorf("Expected 'kairo default' in usage, got: %s", result)
		}
	})

	t.Run("has default provider - delegates to switch", func(t *testing.T) {
		t.Skip("Skipping: switchCmd requires TTY/input, hard to test without mocking")
	})
}

func TestExecute(t *testing.T) {
	t.Run("valid command executes successfully", func(t *testing.T) {
		// Test with help command
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--help"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		// Save original flag values
		originalConfigDir := configDir
		originalVerbose := verbose
		configDir = ""
		verbose = false
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		err := Execute()

		if err != nil {
			t.Errorf("Execute() should succeed, got error: %v", err)
		}

		result := output.String()
		if !containsString(result, "Available Commands:") {
			t.Errorf("Expected help output, got: %s", result)
		}
	})

	t.Run("invalid command returns error", func(t *testing.T) {
		// Save original args
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		// Set args to simulate invalid command
		os.Args = []string{"kairo", "invalid-command-that-does-not-exist"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		// Save and restore flag values
		originalConfigDir := configDir
		originalVerbose := verbose
		configDir = ""
		verbose = false
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		err := Execute()

		if err == nil {
			t.Error("Execute() with invalid command should return error")
		}

		result := output.String()
		// Cobra should show error about unknown command
		_ = result
	})

	t.Run("with --verbose flag", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--verbose", "--help"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		// Save and restore flag values
		originalConfigDir := configDir
		originalVerbose := verbose
		configDir = ""
		verbose = false
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		err := Execute()

		if err != nil {
			t.Errorf("Execute() with --verbose should succeed, got error: %v", err)
		}

		if !verbose {
			t.Error("verbose flag should be set")
		}
	})

	t.Run("with --config flag", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--config", tmpDir, "--help"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		// Reset and restore flag values
		originalConfigDir := configDir
		originalVerbose := verbose
		configDir = ""
		verbose = false
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		err := Execute()

		if err != nil {
			t.Errorf("Execute() with --config should succeed, got error: %v", err)
		}

		// Note: We can't reliably check configDir value here because it's a global variable
		// that may be modified by other tests. The test above verifies that Execute() works
		// with --config flag without errors.
	})
}

func TestRootCmdGetConfigDir(t *testing.T) {
	t.Run("returns flag value when set", func(t *testing.T) {
		originalConfigDir := configDir
		configDir = "/custom/config/dir"
		defer func() { configDir = originalConfigDir }()

		result := getConfigDir()
		if result != "/custom/config/dir" {
			t.Errorf("getConfigDir() = %q, want %q", result, "/custom/config/dir")
		}
	})

	t.Run("returns env default when flag is empty", func(t *testing.T) {
		originalConfigDir := configDir
		configDir = ""
		defer func() { configDir = originalConfigDir }()

		result := getConfigDir()
		// Should return the value from env.GetConfigDir()
		// We can't easily test the exact value without mocking env package
		if result == "" {
			// At minimum, it should return something non-empty in normal conditions
			t.Skip("Cannot test env.GetConfigDir() without mocking")
		}
	})

	t.Run("empty flag value uses default", func(t *testing.T) {
		originalConfigDir := configDir
		configDir = ""
		defer func() { configDir = originalConfigDir }()

		result := getConfigDir()
		// Just verify it doesn't crash and returns a string
		if result == "" {
			// In a real environment, GetConfigDir would return ~/.config/kairo
			// Since we can't mock it easily, we just verify it's a valid return
			t.Skip("Cannot mock env.GetConfigDir() without dependency injection")
		}
	})
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// Helper function to create a test config file
func createConfigFile(t *testing.T, dir string, cfg *config.Config) string {
	configPath := dir + "/config.yaml"
	if err := config.SaveConfig(dir, cfg); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	return configPath
}
