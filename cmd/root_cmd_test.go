package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

func TestRootCmd(t *testing.T) {
	originalOut := rootCmd.OutOrStdout()
	originalErr := rootCmd.ErrOrStderr()
	t.Cleanup(func() {
		rootCmd.SetOut(originalOut)
		rootCmd.SetErr(originalErr)
	})

	t.Run("no config file - shows setup message", func(t *testing.T) {
		tmpDir := t.TempDir()

		originalConfigDir := testCLI.ConfigDir()
		testCLI.SetConfigDir(tmpDir)
		defer func() { testCLI.SetConfigDir(originalConfigDir) }()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		rootCmd.SetContext(WithCLIContext(context.Background(), testCLI))
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

		originalConfigDir := testCLI.ConfigDir()
		testCLI.SetConfigDir(tmpDir)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)

		rootCmd.SetContext(WithCLIContext(context.Background(), testCLI))
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

		originalConfigDir := testCLI.ConfigDir()
		testCLI.SetConfigDir(tmpDir)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		var execCalled atomic.Bool
		d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
			mp.LookPathFn = func(file string) (string, error) {
				if file == "claude" {
					return "/usr/bin/claude", nil
				}
				return "", fmt.Errorf("not found: %s", file)
			}
			mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
				execCalled.Store(true)
				cmd := exec.CommandContext(ctx, "echo", "mocked")
				cmd.Args = []string{"echo", "mocked"}
				return cmd
			}
			mp.ExitProcessFn = func(int) {}
		})

		cliCtx := NewCLIContext()
		cliCtx.SetConfigDir(tmpDir)
		cliCtx.SetDeps(d)
		rootCmd.SetContext(WithCLIContext(context.Background(), cliCtx))

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

		originalConfigDir := testCLI.ConfigDir()
		testCLI.SetConfigDir(tmpDir)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		rootCmd.SetContext(WithCLIContext(context.Background(), testCLI))
		rootCmd.Run(rootCmd, []string{"nonexistent"})

		result := output.String()
		if !containsString(result, "not configured") {
			t.Errorf("Expected 'not configured' error, got: %s", result)
		}
	})

	t.Run("double dash uses default provider", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			DefaultProvider: "anthropic",
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic", BaseURL: "https://api.anthropic.com", Model: "claude-sonnet"},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		originalConfigDir := testCLI.ConfigDir()
		testCLI.SetConfigDir(tmpDir)
		defer func() {
			testCLI.SetConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		var execCalled atomic.Bool
		d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
			mp.LookPathFn = func(file string) (string, error) {
				if file == "claude" {
					return "/usr/bin/claude", nil
				}
				return "", fmt.Errorf("not found: %s", file)
			}
			mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
				execCalled.Store(true)
				return exec.CommandContext(ctx, "echo", "mocked")
			}
			mp.ExitProcessFn = func(int) {}
		})

		cliCtx := NewCLIContext()
		cliCtx.SetConfigDir(tmpDir)
		cliCtx.SetDeps(d)
		cliCtx.SetDefaultProviderExplicit(true)
		rootCmd.SetContext(WithCLIContext(context.Background(), cliCtx))

		rootCmd.Run(rootCmd, []string{"hello"})

		if !execCalled.Load() {
			t.Error("Expected harness execution with default provider")
		}
	})
}

func TestRootCmdGetConfigDir(t *testing.T) {
	t.Run("returns flag value when set", func(t *testing.T) {
		originalDir := testCLI.ConfigDir()
		testCLI.SetConfigDir("/custom/config/dir")
		defer func() { testCLI.SetConfigDir(originalDir) }()

		result := testCLI.ConfigDir()
		if result != "/custom/config/dir" {
			t.Errorf("testCLI.ConfigDir() = %q, want %q", result, "/custom/config/dir")
		}
	})

	t.Run("returns resolver default when flag is empty", func(t *testing.T) {
		cliCtx := NewCLIContext()
		cliCtx.SetConfigDirResolver(func() (string, error) {
			return "/from/resolver", nil
		})

		// Override the global context to use our injected resolver
		prevCtx := testCLI
		testCLI = cliCtx
		testCLI.SetConfigDir("")
		defer func() {
			testCLI = prevCtx
			testCLI.SetConfigDir("")
		}()

		result := testCLI.ConfigDir()
		if result != "/from/resolver" {
			t.Errorf("testCLI.ConfigDir() = %q, want %q", result, "/from/resolver")
		}
	})

	t.Run("empty resolver returns empty string", func(t *testing.T) {
		cliCtx := NewCLIContext()
		cliCtx.SetConfigDirResolver(func() (string, error) {
			return "", nil
		})

		prevCtx := testCLI
		testCLI = cliCtx
		testCLI.SetConfigDir("")
		defer func() {
			testCLI = prevCtx
			testCLI.SetConfigDir("")
		}()

		result := testCLI.ConfigDir()
		if result != "" {
			t.Errorf("testCLI.ConfigDir() = %q, want empty with nil resolver", result)
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

func TestLoadRootConfigConfigDirEmpty(t *testing.T) {
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDirResolver(func() (string, error) {
		return "", nil
	})

	got, ok := loadRootConfig(&cobra.Command{}, cliCtx)
	if ok || got != nil {
		t.Error("Expected failure when config dir is empty")
	}
}

func TestIsKnownProvider(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"my-custom": {Name: "My"},
		},
	}

	if !isKnownProvider("my-custom", cfg) {
		t.Error("Expected configured provider to be known")
	}

	if !isKnownProvider("anthropic", cfg) {
		t.Error("Expected built-in provider to be known even if not configured")
	}

	if isKnownProvider("nonexistent", cfg) {
		t.Error("Expected unknown provider to not be known")
	}
}
