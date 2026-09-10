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

	originalConfigDir := testCLI.ConfigDir()
	testCLI.SetConfigDir(tmpDir)
	defer func() { testCLI.SetConfigDir(originalConfigDir) }()

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

	provider := config.Provider{Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-5"}
	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers:       map[string]config.Provider{"zai": provider},
	}
	createConfigFile(t, tmpDir, cfg)

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
	skipPermissionsFlag = false
	harnessFlag = ""

	runPiProvider(rootCmd, cliCtx, cfg, provider, "zai", "pi", []string{"hello"})

	if !execCalled {
		t.Error("Expected executeWithAuth to be called for Pi harness with API key")
	}
}

// Pi supports multi-provider sessions, so every configured provider key
// present in secrets must be injected — not only the launch provider's.
func TestInjectPiAPIKeysAllConfiguredProviders(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":      {Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-5"},
			"deepseek": {Name: "DeepSeek", BaseURL: "https://api.deepseek.com", Model: "deepseek-chat"},
		},
	}

	envResult := EnvBuildResult{
		ProviderEnv: []string{"PATH=/usr/bin"},
		Secrets: map[string]string{
			"ZAI_API_KEY":      "sk-zai-test",
			"DEEPSEEK_API_KEY": "sk-deepseek-test",
		},
	}

	if !injectPiAPIKeys(&envResult, cfg) {
		t.Fatal("expected injectPiAPIKeys to find at least one key")
	}

	if !hasEnvEntry(envResult.ProviderEnv, "ZAI_API_KEY=sk-zai-test") {
		t.Errorf("expected zai key in env, got: %v", envResult.ProviderEnv)
	}
	if !hasEnvEntry(envResult.ProviderEnv, "DEEPSEEK_API_KEY=sk-deepseek-test") {
		t.Errorf("expected deepseek key in env for multi-provider Pi sessions, got: %v", envResult.ProviderEnv)
	}
}

func hasEnvEntry(env []string, pair string) bool {
	for _, e := range env {
		if e == pair {
			return true
		}
	}

	return false
}

func TestInjectPiAPIKeysMissingSecret(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-5"},
		},
	}

	envResult := EnvBuildResult{
		ProviderEnv: []string{"PATH=/usr/bin"},
		Secrets:     map[string]string{},
	}

	if injectPiAPIKeys(&envResult, cfg) {
		t.Error("expected injectPiAPIKeys to return false when no secret exists")
	}
	if len(envResult.ProviderEnv) != 1 {
		t.Errorf("env should be unchanged without secrets, got: %v", envResult.ProviderEnv)
	}
}

func TestAPIKeyEnvVarNameResolution(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		provider     config.Provider
		want         string
	}{
		{
			name:         "catalog-defined",
			providerName: "zai",
			provider:     config.Provider{},
			want:         "ZAI_API_KEY",
		},
		{
			name:         "configured env key",
			providerName: "myproxy",
			provider:     config.Provider{EnvKey: "MYPROXY_TOKEN"},
			want:         "MYPROXY_TOKEN",
		},
		{
			name:         "conventional fallback",
			providerName: "myproxy",
			provider:     config.Provider{},
			want:         "MYPROXY_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiKeyEnvVarName(tt.providerName, tt.provider)
			if got != tt.want {
				t.Errorf("apiKeyEnvVarName(%q) = %q, want %q", tt.providerName, got, tt.want)
			}
		})
	}
}
