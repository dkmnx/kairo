package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

func TestLoadRootConfigEmptyProviders(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}
	createConfigFile(t, tmpDir, cfg)

	originalConfigDir := getConfigDir()
	setConfigDir(tmpDir)
	defer func() { setConfigDir(originalConfigDir) }()

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	_, ok := loadRootConfig(rootCmd, cliCtx)
	if ok {
		t.Error("loadRootConfig() should return false for empty providers")
	}

	result := output.String()
	if !containsString(result, "No providers configured") {
		t.Errorf("Expected 'No providers configured' message, got: %s", result)
	}
}

func TestRunStandardProviderBuildEnvError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic", BaseURL: "https://api.anthropic.com", Model: "claude-sonnet"},
		},
	}
	createConfigFile(t, tmpDir, cfg)

	// Create corrupted crypto files so LoadSecrets fails
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "secrets.age"), []byte("corrupted"), 0o600); err != nil {
		t.Fatalf("Failed to create corrupted secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "age.key"), []byte("corrupted"), 0o600); err != nil {
		t.Fatalf("Failed to create corrupted key: %v", err)
	}

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	runStandardProvider(rootCmd, cliCtx, cfg.Providers["anthropic"], "anthropic", "claude", []string{"hello"})
}

func TestRunPiProviderWithAuth(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-5"},
		},
	}
	createConfigFile(t, tmpDir, cfg)

	// Set up real crypto keys and encrypted secrets
	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "ZAI_API_KEY=sk-zai-test\n"); err != nil {
		t.Fatalf("EncryptSecrets: %v", err)
	}

	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	var execCalled bool
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/pi", nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			execCalled = true
			return exec.CommandContext(ctx, "echo", "mocked")
		}
	})
	cliCtx.SetDeps(d)
	yoloFlag = false
	harnessFlag = ""

	runPiProvider(rootCmd, cliCtx, cfg, cfg.Providers["zai"], "zai", "pi", []string{"hello"})

	if !execCalled {
		t.Error("Expected executeWithAuth to be called for Pi harness with API key")
	}
}
