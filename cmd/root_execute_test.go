package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"gopkg.in/yaml.v3"
)

func TestExecute(t *testing.T) {
	t.Run("valid command executes successfully", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--help"}
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetArgs(nil) // Reset any args from previous tests

		originalConfigDir := testCLI.ConfigDir()
		originalVerbose := testCLI.Verbose()
		testCLI.SetConfigDir("")
		testCLI.SetVerbose(false)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			testCLI.SetVerbose(originalVerbose)
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

		originalConfigDir := testCLI.ConfigDir()
		originalVerbose := testCLI.Verbose()
		testCLI.SetConfigDir("")
		testCLI.SetVerbose(false)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			testCLI.SetVerbose(originalVerbose)
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

		originalConfigDir := testCLI.ConfigDir()
		originalVerbose := testCLI.Verbose()
		testCLI.SetConfigDir("")
		testCLI.SetVerbose(false)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			testCLI.SetVerbose(originalVerbose)
		}()

		err := Execute()

		if err != nil {
			t.Errorf("Execute() with --verbose should succeed, got error: %v", err)
		}

		if !verboseFlag {
			t.Error("verbose flag should be set after Execute() with --verbose")
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

		originalConfigDir := testCLI.ConfigDir()
		originalVerbose := testCLI.Verbose()
		testCLI.SetConfigDir("")
		testCLI.SetVerbose(false)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			testCLI.SetVerbose(originalVerbose)
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

		originalConfigDir := testCLI.ConfigDir()
		originalVerbose := testCLI.Verbose()
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			testCLI.SetVerbose(originalVerbose)
			os.Remove(configPath)
		}()

		var execCalled bool
		d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
			mp.LookPathFn = func(file string) (string, error) {
				if file == "claude" {
					return "/usr/bin/claude", nil
				}
				return "", fmt.Errorf("not found: %s", file)
			}
			mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
				execCalled = true
				cmd := exec.CommandContext(ctx, "echo", "mocked")
				cmd.Args = []string{"echo", "mocked"}
				return cmd
			}
			mp.ExitProcessFn = func(int) {}
		})

		cliCtx := NewCLIContext()
		cliCtx.SetConfigDir(tmpDir)
		cliCtx.SetVerbose(false)
		cliCtx.SetDeps(d)
		rootCmd.SetContext(WithCLIContext(context.Background(), cliCtx))

		rootCmd.Run(rootCmd, []string{"anthropic"})

		result := output.String()
		if containsString(result, "unknown command") {
			t.Errorf("Got 'unknown command' error, output: %s", result)
		}
		if !execCalled {
			t.Errorf("Expected execCommand to be called when provider name is passed as argument")
		}
	})
}

func TestHandleConfigError(t *testing.T) {
	t.Run("unknown field error shows helpful guide", func(t *testing.T) {
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		err := kairoerrors.WrapError(kairoerrors.ConfigError,
			"configuration file contains field(s) not recognized by this version of kairo",
			&yaml.TypeError{Errors: []string{"field default_harness not found in type config.Config"}})

		originalVerbose := testCLI.Verbose()
		testCLI.SetVerbose(false)
		defer func() { testCLI.SetVerbose(originalVerbose) }()

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

		err := kairoerrors.WrapError(kairoerrors.ConfigError,
			"configuration file contains field(s) not recognized by this version of kairo",
			&yaml.TypeError{Errors: []string{"field default_harness not found in type config.Config"}})

		verboseFlag = true
		defer func() { verboseFlag = false }()

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

		originalVerbose := testCLI.Verbose()
		testCLI.SetVerbose(false)
		defer func() { testCLI.SetVerbose(originalVerbose) }()

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
