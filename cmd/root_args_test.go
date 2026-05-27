package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync/atomic"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

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

			provider, args := providerFromArgs(cmd, cfg, tt.args)

			if provider != tt.wantProvider {
				t.Errorf("providerFromArgs() provider = %q, want %q", provider, tt.wantProvider)
			}

			if len(args) != len(tt.wantArgs) {
				t.Errorf("providerFromArgs() args length = %d, want %d", len(args), len(tt.wantArgs))
				return
			}

			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("providerFromArgs() args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestHasArgsSeparator(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"bare double dash", []string{"--", "hello"}, true},
		{"flag then double dash", []string{"-v", "--", "hello"}, true},
		{"long flag with value then double dash", []string{"--harness", "pi", "--", "hello"}, true},
		{"long flag with equals then double dash", []string{"--harness=pi", "--", "hello"}, true},
		{"provider then double dash", []string{"anthropic", "--", "hello"}, false},
		{"provider without double dash", []string{"anthropic", "hello"}, false},
		{"no args", []string{}, false},
		{"flags only no double dash", []string{"-v", "--harness", "pi"}, false},
		{"single dash positional", []string{"-", "hello"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasArgsSeparator(tt.args)
			if got != tt.want {
				t.Errorf("hasArgsSeparator(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	t.Run("returns provider-specific key", func(t *testing.T) {
		secrets := map[string]string{
			"ANTHROPIC_API_KEY": "sk-ant-xxx",
		}
		key, ok := resolveAPIKey(secrets, "anthropic")
		if !ok {
			t.Error("Expected key to be found")
		}
		if key != "sk-ant-xxx" {
			t.Errorf("Expected 'sk-ant-xxx', got %q", key)
		}
	})

	t.Run("falls back to custom provider key", func(t *testing.T) {
		secrets := map[string]string{
			"CUSTOM_API_KEY": "sk-custom-xxx",
		}
		key, ok := resolveAPIKey(secrets, "anthropic")
		if !ok {
			t.Error("Expected key to be found via custom fallback")
		}
		if key != "sk-custom-xxx" {
			t.Errorf("Expected 'sk-custom-xxx', got %q", key)
		}
	})

	t.Run("returns false when no key found", func(t *testing.T) {
		secrets := map[string]string{}
		_, ok := resolveAPIKey(secrets, "anthropic")
		if ok {
			t.Error("Expected no key to be found")
		}
	})
}

func TestHarnessFlagUsesDefaultProvider(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic", BaseURL: "https://api.anthropic.com", Model: "claude-sonnet"},
		},
	}
	createConfigFile(t, tmpDir, cfg)

	originalConfigDir := configDir()
	originalHarnessFlag := harnessFlag
	setConfigDir(tmpDir)
	harnessFlag = "claude"
	defer func() {
		setConfigDir(originalConfigDir)
		harnessFlag = originalHarnessFlag
	}()

	originalOut := rootCmd.OutOrStdout()
	originalErr := rootCmd.ErrOrStderr()
	originalCtx := rootCmd.Context()
	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)
	defer func() {
		rootCmd.SetOut(originalOut)
		rootCmd.SetErr(originalErr)
		rootCmd.SetContext(originalCtx)
	}()

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
	rootCmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	rootCmd.Run(rootCmd, []string{"suggest one greek god"})

	if !execCalled.Load() {
		t.Error("Expected harness execution with --harness flag and default provider")
	}
}
