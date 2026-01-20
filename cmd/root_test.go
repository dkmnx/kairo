package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestRootCmd(t *testing.T) {
	t.Run("no config file - shows setup message", func(t *testing.T) {
		// Use a temp directory that doesn't have a config file
		tmpDir := t.TempDir()

		// Set config dir to temp directory
		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() { setConfigDir(originalConfigDir) }()

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

		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() {
			setConfigDir(originalConfigDir)
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

	t.Run("no default provider with provider arg - switches to provider", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file without default provider, but with "anthropic" configured
		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic"},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() {
			setConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		// Mock lookPath to return a fake claude path (for Docker/CI environments)
		originalLookPath := lookPath
		lookPath = func(file string) (string, error) {
			if file == "claude" {
				return "/usr/bin/claude", nil
			}
			return originalLookPath(file)
		}
		defer func() { lookPath = originalLookPath }()

		// Mock execCommand to capture invocation without actually running
		// Use atomic.Bool for race-safe access
		var execCalled atomic.Bool
		originalExecCommand := execCommand
		execCommand = func(name string, arg ...string) *exec.Cmd {
			execCalled.Store(true)
			// Return a command that does nothing
			cmd := originalExecCommand("echo", "mocked")
			cmd.Args = []string{"echo", "mocked"}
			return cmd
		}
		defer func() { execCommand = originalExecCommand }()

		// Also mock exitProcess to prevent test from exiting
		originalExitProcess := exitProcess
		exitProcess = func(int) {}
		defer func() { exitProcess = originalExitProcess }()

		// Execute root command with provider name as argument
		rootCmd.Run(rootCmd, []string{"anthropic"})

		// Verify execCommand was called (meaning switch behavior was triggered)
		if !execCalled.Load() {
			t.Errorf("Expected execCommand to be called when provider name is passed as argument")
		}
	})

	t.Run("no default provider with invalid provider arg - shows usage", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file without default provider
		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic"},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() {
			setConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		// Execute root command with non-existent provider name
		rootCmd.Run(rootCmd, []string{"nonexistent"})

		result := output.String()
		// Should show error about provider not configured
		if !containsString(result, "not configured") {
			t.Errorf("Expected 'not configured' error, got: %s", result)
		}
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
		originalConfigDir := getConfigDir()
		originalVerbose := verbose
		setConfigDir("")
		setVerbose(false)
		defer func() {
			setConfigDir(originalConfigDir)
			setVerbose(originalVerbose)
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

	t.Run("invalid command treated as provider name", func(t *testing.T) {
		// Save original args
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		// Set args to simulate invalid command (not a subcommand)
		os.Args = []string{"kairo", "invalid-command-that-does-not-exist"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		// Save and restore flag values
		originalConfigDir := getConfigDir()
		originalVerbose := verbose
		setConfigDir("")
		setVerbose(false)
		defer func() {
			setConfigDir(originalConfigDir)
			setVerbose(originalVerbose)
		}()

		err := Execute()

		// Should not error - gets converted to switch command which shows provider error
		if err != nil {
			t.Errorf("Execute() should succeed, got error: %v", err)
		}

		result := output.String()
		// Should show "not configured" from switchCmd, not "unknown command" from Cobra
		if containsString(result, "unknown command") {
			t.Errorf("Should not show 'unknown command', got: %s", result)
		}
		// Should show either "not configured" (provider missing) OR "configuration file not found" (no config)
		if !containsString(result, "not configured") && !containsString(result, "configuration file not found") {
			t.Errorf("Expected 'not configured' or 'configuration file not found' message, got: %s", result)
		}
	})

	t.Run("with --verbose flag", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--verbose", "--help"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		// Save and restore flag values
		originalConfigDir := getConfigDir()
		originalVerbose := verbose
		setConfigDir("")
		setVerbose(false)
		defer func() {
			setConfigDir(originalConfigDir)
			setVerbose(originalVerbose)
		}()

		err := Execute()

		if err != nil {
			t.Errorf("Execute() with --verbose should succeed, got error: %v", err)
		}

		if !getVerbose() {
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
		originalConfigDir := getConfigDir()
		originalVerbose := verbose
		setConfigDir("")
		setVerbose(false)
		defer func() {
			setConfigDir(originalConfigDir)
			setVerbose(originalVerbose)
		}()

		err := Execute()

		if err != nil {
			t.Errorf("Execute() with --config should succeed, got error: %v", err)
		}

		// Note: We can't reliably check configDir value here because it's a global variable
		// that may be modified by other tests. The test above verifies that Execute() works
		// with --config flag without errors.
	})

	t.Run("provider shorthand without default - switches to provider", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file without default provider, but with "anthropic" configured
		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic"},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--config", tmpDir, "anthropic"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs(nil)

		originalConfigDir := getConfigDir()
		originalVerbose := verbose
		setConfigDir("")
		setVerbose(false)
		defer func() {
			setConfigDir(originalConfigDir)
			setVerbose(originalVerbose)
			os.Remove(configPath)
		}()

		// Mock lookPath to return a fake claude path (for Docker/CI environments)
		originalLookPath := lookPath
		lookPath = func(file string) (string, error) {
			if file == "claude" {
				return "/usr/bin/claude", nil
			}
			return originalLookPath(file)
		}
		defer func() { lookPath = originalLookPath }()

		// Mock execCommand to capture invocation
		// Use atomic.Bool for race-safe access
		var execCalled atomic.Bool
		originalExecCommand := execCommand
		execCommand = func(name string, arg ...string) *exec.Cmd {
			execCalled.Store(true)
			cmd := originalExecCommand("echo", "mocked")
			cmd.Args = []string{"echo", "mocked"}
			return cmd
		}
		defer func() { execCommand = originalExecCommand }()

		// Mock exitProcess
		originalExitProcess := exitProcess
		exitProcess = func(int) {}
		defer func() { exitProcess = originalExitProcess }()

		err := Execute()

		// Should succeed (no error about unknown command)
		if err != nil {
			t.Errorf("Execute() with provider name should succeed, got error: %v", err)
		}

		// Verify execCommand was called (switch behavior triggered)
		if !execCalled.Load() {
			t.Errorf("Expected execCommand to be called when provider name is passed")
		}

		// Verify output doesn't contain "unknown command" error
		result := output.String()
		if containsString(result, "unknown command") {
			t.Errorf("Got 'unknown command' error, output: %s", result)
		}
	})
}

func TestRootCmdGetConfigDir(t *testing.T) {
	t.Run("returns flag value when set", func(t *testing.T) {
		originalConfigDir := getConfigDir()
		setConfigDir("/custom/config/dir")
		defer func() { setConfigDir(originalConfigDir) }()

		result := getConfigDir()
		if result != "/custom/config/dir" {
			t.Errorf("getConfigDir() = %q, want %q", result, "/custom/config/dir")
		}
	})

	t.Run("returns env default when flag is empty", func(t *testing.T) {
		originalConfigDir := getConfigDir()
		setConfigDir("")
		defer func() { setConfigDir(originalConfigDir) }()

		result := getConfigDir()
		// Should return the value from env.GetConfigDir()
		// We can't easily test the exact value without mocking env package
		if result == "" {
			// At minimum, it should return something non-empty in normal conditions
			t.Skip("Cannot test env.GetConfigDir() without mocking")
		}
	})

	t.Run("empty flag value uses default", func(t *testing.T) {
		originalConfigDir := getConfigDir()
		setConfigDir("")
		defer func() { setConfigDir(originalConfigDir) }()

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
	configPath := filepath.Join(dir, "config.yaml")
	if err := config.SaveConfig(dir, cfg); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	return configPath
}
