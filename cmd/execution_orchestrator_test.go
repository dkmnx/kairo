package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestSplitArgsOrchestrator(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
		wantRest []string
	}{
		{
			name:     "no separator",
			args:     []string{"hello", "world"},
			wantArgs: []string{"hello", "world"},
			wantRest: nil,
		},
		{
			name:     "with separator",
			args:     []string{"hello", "--", "world"},
			wantArgs: []string{"hello"},
			wantRest: []string{"world"},
		},
		{
			name:     "separator at start",
			args:     []string{"--", "hello"},
			wantArgs: []string{},
			wantRest: []string{"hello"},
		},
		{
			name:     "empty args",
			args:     []string{},
			wantArgs: []string{},
			wantRest: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotRest := splitArgs(tt.args)
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("splitArgs() args = %v, want %v", gotArgs, tt.wantArgs)
			}
			if len(gotRest) != len(tt.wantRest) {
				t.Errorf("splitArgs() rest = %v, want %v", gotRest, tt.wantRest)
			}
		})
	}
}

func TestHasLeadingArgsSeparatorOrchestrator(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"no separator", []string{"hello", "world"}, false},
		{"separator after flags", []string{"-v", "--", "hello"}, true},
		{"separator at start", []string{"--", "hello"}, true},
		{"empty args", []string{}, false},
		{"non-flag before separator", []string{"hello", "--", "world"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasLeadingArgsSeparator(tt.args); got != tt.want {
				t.Errorf("hasLeadingArgsSeparator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsKnownProviderOrchestrator(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"custom": {Name: "Custom"},
		},
	}

	if !isKnownProvider("custom", cfg) {
		t.Error("isKnownProvider() should return true for configured provider")
	}

	if !isKnownProvider("anthropic", cfg) {
		t.Error("isKnownProvider() should return true for built-in provider")
	}

	if isKnownProvider("nonexistent", cfg) {
		t.Error("isKnownProvider() should return false for unknown provider")
	}
}

func TestLookupProvider(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		},
	}

	cmd := testCmd()

	provider, ok := lookupProvider(cmd, cfg, "anthropic")
	if !ok {
		t.Error("lookupProvider() should return true for existing provider")
	}
	if provider.Name != "Anthropic" {
		t.Errorf("lookupProvider() returned provider with name %q, want %q", provider.Name, "Anthropic")
	}

	_, ok = lookupProvider(cmd, cfg, "nonexistent")
	if ok {
		t.Error("lookupProvider() should return false for non-existent provider")
	}
}

func TestResolveProviderAndArgs_DefaultProvider(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		},
	}

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	cliCtx := NewCLIContext()
	cliCtx.SetDefaultProviderExplicit(false)

	_, providerName := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{})
	if providerName != "anthropic" {
		t.Errorf("resolveProviderAndArgs() returned provider %q, want %q", providerName, "anthropic")
	}
}

func TestResolveProviderAndArgs_NoDefaultProvider(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	cliCtx := NewCLIContext()
	cliCtx.SetDefaultProviderExplicit(true)

	_, providerName := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{})
	if providerName != "" {
		t.Errorf("resolveProviderAndArgs() should return empty provider when no default, got %q", providerName)
	}

	result := output.String()
	if !containsString(result, "No default provider set") {
		t.Errorf("Expected 'No default provider set' message, got: %s", result)
	}
	if !containsString(result, "kairo setup") {
		t.Errorf("Expected usage hint with 'kairo setup', got: %s", result)
	}
}

func TestOrchestrateExecution_NoCLIContext(t *testing.T) {
	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	// Don't set CLIContext - should print error and return
	OrchestrateExecution(cmd, []string{})

	result := output.String()
	if !containsString(result, "no CLI context available") {
		t.Errorf("Expected 'no CLI context available' message, got: %s", result)
	}
}

func TestOrchestrateExecution_EmptyConfigDir(t *testing.T) {
	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDirResolver(func() (string, error) {
		return "", nil
	})
	ctx := context.Background()
	cmd.SetContext(WithCLIContext(ctx, cliCtx))

	OrchestrateExecution(cmd, []string{})

	result := output.String()
	if !containsString(result, "config directory not found") {
		t.Errorf("Expected 'config directory not found' message, got: %s", result)
	}
}

func TestOrchestrateExecution_NoProvidersConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}
	createConfigFile(t, tmpDir, cfg)

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	cmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	OrchestrateExecution(cmd, []string{})

	result := output.String()
	if !containsString(result, "No providers configured") {
		t.Errorf("Expected 'No providers configured' message, got: %s", result)
	}
}

func TestResolveProviderAndArgs_HarnessFallback(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		},
	}

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	// Set harness flag - this triggers the fallback path
	originalHarnessFlag := harnessFlag
	harnessFlag = "claude"
	defer func() { harnessFlag = originalHarnessFlag }()

	cliCtx := NewCLIContext()
	cliCtx.SetDefaultProviderExplicit(false)

	// First arg "unknown" is not a known provider, so it should fall back to default
	harnessArgs, providerName := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{"unknown", "hello"})

	if providerName != "anthropic" {
		t.Errorf("resolveProviderAndArgs() should return default provider, got %q", providerName)
	}
	if len(harnessArgs) != 2 {
		t.Errorf("resolveProviderAndArgs() should return all args as harness args, got %v", harnessArgs)
	}
}

func TestResolveProviderAndArgs_DoubleDashSeparator(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		},
	}

	cmd := testCmd()

	cliCtx := NewCLIContext()
	cliCtx.SetDefaultProviderExplicit(false)

	// With -- separator, provider comes first, then harness args
	harnessArgs, providerName := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{"anthropic", "--", "hello"})

	if providerName != "anthropic" {
		t.Errorf("resolveProviderAndArgs() should return 'anthropic', got %q", providerName)
	}
	if len(harnessArgs) != 1 || harnessArgs[0] != "hello" {
		t.Errorf("resolveProviderAndArgs() should return harness args ['hello'], got %v", harnessArgs)
	}
}
