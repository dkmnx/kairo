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

func TestProviderFromArgs_FirstArgIsProvider(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
			"openai":    {Name: "OpenAI"},
		},
	}

	tests := []struct {
		name             string
		args             []string
		wantProvider     string
		wantHarnessCount int
	}{
		{"first positional is configured provider", []string{"openai", "--prompt", "hi"}, "openai", 2},
		{"first positional is built-in provider", []string{"zai", "--help"}, "zai", 1},
		{"unknown is returned as-is (fallback in resolveProviderAndArgs)", []string{"unknown", "arg"}, "unknown", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := testCmd()
			gotProvider, gotArgs := providerFromArgs(cmd, cfg, tt.args)
			if gotProvider != tt.wantProvider {
				t.Errorf("providerFromArgs() provider = %q, want %q", gotProvider, tt.wantProvider)
			}
			if len(gotArgs) != tt.wantHarnessCount {
				t.Errorf("providerFromArgs() harness args count = %d, want %d: %v", len(gotArgs), tt.wantHarnessCount, gotArgs)
			}
		})
	}
}

func TestProviderFromArgs_FirstArgIsFlag(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		},
	}

	tests := []struct {
		name         string
		args         []string
		wantProvider string
		wantArgs     []string
	}{
		{"flag arg uses default", []string{"--prompt", "hi"}, "anthropic", []string{"--prompt", "hi"}},
		{"dash arg uses default", []string{"-v"}, "anthropic", []string{"-v"}},
		{"flag with equals", []string{"--x=y"}, "anthropic", []string{"--x=y"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := testCmd()
			gotProvider, gotArgs := providerFromArgs(cmd, cfg, tt.args)
			if gotProvider != tt.wantProvider {
				t.Errorf("providerFromArgs() provider = %q, want %q", gotProvider, tt.wantProvider)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("providerFromArgs() args = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestProviderFromArgs_NoDefaultWithFlag(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	provider, args := providerFromArgs(cmd, cfg, []string{"--flag"})
	if provider != "" {
		t.Errorf("expected empty provider, got %q", provider)
	}
	if args != nil {
		t.Errorf("expected nil args, got %v", args)
	}
	if !containsString(output.String(), "No default provider set") {
		t.Error("expected error message about no default provider")
	}
}

func TestProviderFromArgs_WithDoubleDash(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		},
	}

	tests := []struct {
		name         string
		args         []string
		wantProvider string
		wantHarness  []string
	}{
		{"double dash after provider", []string{"anthropic", "--", "--flag"}, "anthropic", []string{"--flag"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := testCmd()
			gotProvider, gotArgs := providerFromArgs(cmd, cfg, tt.args)
			if gotProvider != tt.wantProvider {
				t.Errorf("providerFromArgs() provider = %q, want %q", gotProvider, tt.wantProvider)
			}
			if len(gotArgs) != len(tt.wantHarness) {
				t.Errorf("providerFromArgs() args = %v, want %v", gotArgs, tt.wantHarness)
			}
		})
	}
}

func TestResolveProviderAndArgs_EmptyArgs(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	t.Run("empty args no default", func(t *testing.T) {
		cmd := testCmd()
		output := &bytes.Buffer{}
		cmd.SetOut(output)

		cliCtx := NewCLIContext()
		cliCtx.SetDefaultProviderExplicit(false)

		_, provider := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{})
		if provider != "" {
			t.Errorf("expected empty provider, got %q", provider)
		}
		if !containsString(output.String(), "No default provider set") {
			t.Error("expected no default provider message")
		}
	})

	t.Run("empty args with default", func(t *testing.T) {
		cfg.DefaultProvider = "anthropic"
		cfg.Providers = map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		}

		cmd := testCmd()
		cliCtx := NewCLIContext()
		cliCtx.SetDefaultProviderExplicit(false)

		args, provider := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{})
		if provider != "anthropic" {
			t.Errorf("expected 'anthropic', got %q", provider)
		}
		if len(args) != 0 {
			t.Errorf("expected empty args, got %v", args)
		}
	})
}

func TestResolveProviderAndArgs_HarnessOverrideBranch(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Anthropic"},
		},
	}

	originalHarnessFlag := harnessFlag
	defer func() { harnessFlag = originalHarnessFlag }()

	t.Run("harness set with unknown provider falls back to default", func(t *testing.T) {
		harnessFlag = "claude"

		cmd := testCmd()
		cliCtx := NewCLIContext()
		cliCtx.SetDefaultProviderExplicit(false)

		harnessArgs, provider := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{"nonexistent", "--prompt", "hi"})
		if provider != "anthropic" {
			t.Errorf("expected 'anthropic', got %q", provider)
		}
		if len(harnessArgs) != 3 {
			t.Errorf("expected all 3 args to be harness args, got %v", harnessArgs)
		}
	})

	t.Run("harness set with known provider still uses that provider", func(t *testing.T) {
		harnessFlag = "claude"

		cmd := testCmd()
		cliCtx := NewCLIContext()
		cliCtx.SetDefaultProviderExplicit(false)

		_, provider := resolveProviderAndArgs(cmd, cliCtx, cfg, []string{"anthropic"})
		if provider != "anthropic" {
			t.Errorf("expected 'anthropic', got %q", provider)
		}
	})
}

func TestHasLeadingArgsSeparator_Extended(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"flag with value before separator", []string{"--harness", "pi", "--", "hello"}, true},
		{"flag with equals before separator", []string{"--harness=pi", "--", "hello"}, true},
		{"short flag before separator", []string{"-v", "--", "hello"}, true},
		{"positional before separator is false", []string{"hello", "--", "world"}, false},
		{"empty", []string{}, false},
		{"just separator", []string{"--"}, true},
		{"multiple flags before separator", []string{"-v", "--harness", "pi", "--", "arg"}, true},
		{"dash dash at start is always separator", []string{"--", "hello", "--"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasLeadingArgsSeparator(tt.args); got != tt.want {
				t.Errorf("hasLeadingArgsSeparator(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestSplitArgs_Extended(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
		wantRest []string
	}{
		{"two dashes after args", []string{"--harness", "pi", "--", "rest"}, []string{"--harness", "pi"}, []string{"rest"}},
		{"multiple dashes split at first", []string{"a", "--", "b", "--", "c"}, []string{"a"}, []string{"b", "--", "c"}},
		{"flag with value", []string{"--harness", "pi", "--", "a", "b"}, []string{"--harness", "pi"}, []string{"a", "b"}},
		{"flag with equals before dash", []string{"--x=y", "--", "z"}, []string{"--x=y"}, []string{"z"}},
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
