package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/app"
	"github.com/dkmnx/kairo/internal/config"
)

func TestPrintResolveError_NoDefaultProvider(t *testing.T) {
	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	printResolveError(cmd, app.ErrNoDefaultProvider)

	result := output.String()
	if !containsString(result, "No default provider set") {
		t.Errorf("expected no-default message, got: %s", result)
	}
	if !containsString(result, "kairo setup") {
		t.Errorf("expected setup hint, got: %s", result)
	}
}

func TestPrintResolveError_ProviderNotConfigured(t *testing.T) {
	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	printResolveError(cmd, &app.ProviderNotConfiguredError{Name: "nope"})

	result := output.String()
	if !containsString(result, "provider 'nope' not configured") {
		t.Errorf("expected provider not configured message, got: %s", result)
	}
}

func TestOrchestrateExecution_NoCLIContext(t *testing.T) {
	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	OrchestrateExecution(cmd, []string{})

	if !containsString(output.String(), "no CLI context available") {
		t.Errorf("Expected 'no CLI context available' message, got: %s", output.String())
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
	cmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	OrchestrateExecution(cmd, []string{})

	if !containsString(output.String(), "config directory not found") {
		t.Errorf("Expected 'config directory not found' message, got: %s", output.String())
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

	if !containsString(output.String(), "No providers configured") {
		t.Errorf("Expected 'No providers configured' message, got: %s", output.String())
	}
}

func TestOrchestrateExecution_NoDefaultProviderMessage(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "",
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI"},
		},
	}
	createConfigFile(t, tmpDir, cfg)

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	cmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	OrchestrateExecution(cmd, []string{})

	if !containsString(output.String(), "No default provider set") {
		t.Errorf("Expected no-default message, got: %s", output.String())
	}
}

func TestOrchestrateExecution_UnconfiguredProviderMessage(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI"},
		},
	}
	createConfigFile(t, tmpDir, cfg)

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	cmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	OrchestrateExecution(cmd, []string{"nope"})

	if !containsString(output.String(), "provider 'nope' not configured") {
		t.Errorf("Expected provider-not-configured message, got: %s", output.String())
	}
}

func TestLoadRootConfig_ConfigError(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte("default_provider: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := testCmd()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	cfg, ok := loadRootConfig(cmd, cliCtx)
	if ok || cfg != nil {
		t.Error("loadRootConfig() should fail for corrupt config")
	}
}
