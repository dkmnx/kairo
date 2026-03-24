package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

func TestRootCmd(t *testing.T) {
	t.Run("no config file - shows setup message", func(t *testing.T) {
		tmpDir := t.TempDir()

		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() { setConfigDir(originalConfigDir) }()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		rootCmd.Run(rootCmd, []string{})

		result := output.String()
		if !containsString(result, "No providers configured") && !containsString(result, "configuration file not found") {
			t.Errorf("Expected setup-related message, got: %s", result)
		}
	})

	t.Run("no default provider - shows usage", func(t *testing.T) {
		tmpDir := t.TempDir()

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
		if !containsString(result, "kairo list") && !containsString(result, "kairo <provider>") {
			t.Errorf("Expected kairo commands in usage, got: %s", result)
		}
	})

	t.Run("has default provider - delegates to switch", func(t *testing.T) {
		t.Skip("Skipping: switchCmd requires TTY/input, hard to test without mocking")
	})

	t.Run("no default provider with provider arg - switches to provider", func(t *testing.T) {
		tmpDir := t.TempDir()

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

		originalLookPath := lookPath
		lookPath = func(file string) (string, error) {
			if file == "claude" {
				return "/usr/bin/claude", nil
			}
			return originalLookPath(file)
		}
		defer func() { lookPath = originalLookPath }()

		var execCalled atomic.Bool
		originalExecCommand := execCommand
		execCommand = func(name string, arg ...string) *exec.Cmd {
			execCalled.Store(true)
			cmd := originalExecCommand("echo", "mocked")
			cmd.Args = []string{"echo", "mocked"}
			return cmd
		}
		defer func() { execCommand = originalExecCommand }()

		originalExecCommandContext := execCommandContext
		execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			execCalled.Store(true)
			cmd := originalExecCommandContext(ctx, "echo", "mocked")
			cmd.Args = []string{"echo", "mocked"}
			return cmd
		}
		defer func() { execCommandContext = originalExecCommandContext }()

		originalExitProcess := exitProcess
		exitProcess = func(int) {}
		defer func() { exitProcess = originalExitProcess }()

		rootCmd.Run(rootCmd, []string{"anthropic"})

		if !execCalled.Load() {
			t.Errorf("Expected execCommand to be called when provider name is passed as argument")
		}
	})

	t.Run("no default provider with invalid provider arg - shows usage", func(t *testing.T) {
		tmpDir := t.TempDir()

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

		rootCmd.Run(rootCmd, []string{"nonexistent"})

		result := output.String()
		if !containsString(result, "not configured") {
			t.Errorf("Expected 'not configured' error, got: %s", result)
		}
	})
}

func TestExecute(t *testing.T) {
	t.Run("valid command executes successfully", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--help"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		originalConfigDir := getConfigDir()
		originalVerbose := getVerbose()
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
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "invalid-command-that-does-not-exist"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		originalConfigDir := getConfigDir()
		originalVerbose := getVerbose()
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
		if containsString(result, "unknown command") {
			t.Errorf("Should not show 'unknown command' from Cobra parser, got: %s", result)
		}
	})

	t.Run("with --verbose flag", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--verbose", "--help"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		originalConfigDir := getConfigDir()
		originalVerbose := getVerbose()
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

		originalConfigDir := getConfigDir()
		originalVerbose := getVerbose()
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
	})

	t.Run("provider shorthand without default - switches to provider", func(t *testing.T) {
		tmpDir := t.TempDir()

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
		originalVerbose := getVerbose()
		setConfigDir(tmpDir) // Use tmpDir, not empty string
		setVerbose(false)
		defer func() {
			setConfigDir(originalConfigDir)
			setVerbose(originalVerbose)
			os.Remove(configPath)
		}()

		rootCmd.SetArgs([]string{"--config", tmpDir, "anthropic"})

		// Note: This test verifies the provider shorthand behavior with rootCmd.Run
		rootCmd.Run(rootCmd, []string{"--config", tmpDir, "anthropic"})

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
		// We can't easily test the exact value without mocking env package
		if result == "" {
			t.Skip("Cannot test env.GetConfigDir() without mocking")
		}
	})

	t.Run("empty flag value uses default", func(t *testing.T) {
		originalConfigDir := getConfigDir()
		setConfigDir("")
		defer func() { setConfigDir(originalConfigDir) }()

		result := getConfigDir()
		if result == "" {
			t.Skip("Cannot mock env.GetConfigDir() without dependency injection")
		}
	})
}

func containsString(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

func createConfigFile(t *testing.T, dir string, cfg *config.Config) string {
	configPath := filepath.Join(dir, "config.yaml")
	if err := config.SaveConfig(context.Background(), dir, cfg); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	return configPath
}

func TestHandleConfigError(t *testing.T) {
	t.Run("unknown field error shows helpful guide", func(t *testing.T) {
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		err := fmt.Errorf("field default_harness not found in type config.Config (path=/home/user/.config/kairo/config.yaml)")

		originalVerbose := getVerbose()
		setVerbose(false)
		defer func() { setVerbose(originalVerbose) }()

		handleConfigError(rootCmd, err)

		result := output.String()

		installScript := "install.ps1"
		if runtime.GOOS != "windows" {
			installScript = "install.sh"
		}
		expectedMessages := []string{
			"Your kairo binary is outdated",
			"configuration file contains newer fields",
			"installation script",
			"github.com/dkmnx/kairo",
			installScript,
		}

		for _, msg := range expectedMessages {
			if !containsString(result, msg) {
				t.Errorf("Expected message %q not found in output:\n%s", msg, result)
			}
		}
	})

	t.Run("unknown field error with verbose shows technical details", func(t *testing.T) {
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		err := fmt.Errorf("field default_harness not found in type config.Config")

		originalVerbose := getVerbose()
		setVerbose(true)
		defer func() { setVerbose(originalVerbose) }()

		handleConfigError(rootCmd, err)

		result := output.String()

		if !containsString(result, "Technical details:") {
			t.Errorf("Expected 'Technical details:' in verbose output:\n%s", result)
		}
		if !containsString(result, "field default_harness") {
			t.Errorf("Expected error details in verbose output:\n%s", result)
		}
	})

	t.Run("other errors show default message", func(t *testing.T) {
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		err := fmt.Errorf("some other config error")

		originalVerbose := getVerbose()
		setVerbose(false)
		defer func() { setVerbose(originalVerbose) }()

		handleConfigError(rootCmd, err)

		result := output.String()

		if !containsString(result, "Error loading config:") {
			t.Errorf("Expected default error message, got:\n%s", result)
		}
		if !containsString(result, "some other config error") {
			t.Errorf("Expected error text in output:\n%s", result)
		}
	})
}

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"substring exists", "hello world", "world", true},
		{"substring at start", "hello world", "hello", true},
		{"substring at end", "hello world", "world", true},
		{"substring in middle", "hello world test", "world", true},
		{"exact match", "hello", "hello", true},
		{"empty substring", "hello", "", true},
		{"substring not found", "hello world", "goodbye", false},
		{"case sensitive", "Hello World", "hello", false},
		{"longer substring than string", "hi", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.Contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("strings.Contains(%q, %q) = %v, want %v",
					tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestGetProviderFromArgs(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		defaultProvider string
		wantProvider    string
		wantArgs        []string
	}{
		{
			name:            "single provider arg",
			args:            []string{"anthropic"},
			defaultProvider: "",
			wantProvider:    "anthropic",
			wantArgs:        []string{},
		},
		{
			name:            "provider with harness args",
			args:            []string{"anthropic", "--model", "claude-sonnet"},
			defaultProvider: "",
			wantProvider:    "anthropic",
			wantArgs:        []string{"--model", "claude-sonnet"},
		},
		{
			name:            "flag-like first arg with default set",
			args:            []string{"--model", "claude-sonnet"},
			defaultProvider: "anthropic",
			wantProvider:    "anthropic",
			wantArgs:        []string{"--model", "claude-sonnet"},
		},
		{
			name:            "flag-like first arg without default",
			args:            []string{"--model", "claude-sonnet"},
			defaultProvider: "",
			wantProvider:    "",
			wantArgs:        nil,
		},
		{
			name:            "multiple args uses first as provider",
			args:            []string{"anthropic", "arg2", "arg3"},
			defaultProvider: "",
			wantProvider:    "anthropic",
			wantArgs:        []string{"arg2", "arg3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var output bytes.Buffer
			cmd.SetOut(&output)

			cfg := &config.Config{
				DefaultProvider: tt.defaultProvider,
			}

			provider, args := getProviderFromArgs(cmd, cfg, tt.args)

			if provider != tt.wantProvider {
				t.Errorf("getProviderFromArgs() provider = %q, want %q", provider, tt.wantProvider)
			}

			if len(args) != len(tt.wantArgs) {
				t.Errorf("getProviderFromArgs() args length = %d, want %d", len(args), len(tt.wantArgs))
				return
			}

			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("getProviderFromArgs() args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}
