package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
)

// --- Tests for RequiresAPIKey ---

func TestRequiresAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"zai requires key", "zai", true},
		{"minimax requires key", "minimax", true},
		{"deepseek requires key", "deepseek", true},
		{"kimi requires key", "kimi", true},
		{"custom requires key", "custom", true},
		{"unknown requires key", "unknown-provider", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := RequiresAPIKey(tt.provider); result != tt.want {
				t.Errorf("RequiresAPIKey(%q) = %v, want %v", tt.provider, result, tt.want)
			}
		})
	}
}

// --- Tests for BuildProviderEnv ---

func TestBuildProviderEnv_BasicEnv(t *testing.T) {
	tmpDir := withTempConfigDir(t)
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	provider := EnvProvider{BaseURL: "https://api.test.com", Model: "test-model", EnvVars: []string{"TEST_VAR=value123"}}
	result, err := BuildProviderEnv(cliCtx, tmpDir, provider, "test")
	if err != nil {
		t.Fatalf("BuildProviderEnv() error = %v", err)
	}

	envMap := make(map[string]string)
	for _, env := range result.ProviderEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if envMap["ANTHROPIC_BASE_URL"] != "https://api.test.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", envMap["ANTHROPIC_BASE_URL"], "https://api.test.com")
	}
	if envMap["ANTHROPIC_MODEL"] != "test-model" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q", envMap["ANTHROPIC_MODEL"], "test-model")
	}
	if envMap["TEST_VAR"] != "value123" {
		t.Errorf("TEST_VAR = %q, want %q", envMap["TEST_VAR"], "value123")
	}
}

func TestBuildProviderEnv_SecretsLoadFallback(t *testing.T) {
	tmpDir := withTempConfigDir(t)
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	// When no secrets file exists, LoadSecrets returns empty map, no error.
	// So BuildProviderEnv succeeds with empty secrets.
	provider := EnvProvider{BaseURL: "https://api.test.com", Model: "test-model"}
	result, err := BuildProviderEnv(cliCtx, tmpDir, provider, "zai")
	if err != nil {
		t.Fatalf("BuildProviderEnv() should not error when secrets file doesn't exist, got: %v", err)
	}
	if len(result.Secrets) != 0 {
		t.Errorf("Expected empty secrets, got %d entries", len(result.Secrets))
	}
}

// --- Tests for BuildBuiltInEnvVars ---

func TestBuildBuiltInEnvVars(t *testing.T) {
	envVars := BuildBuiltInEnvVars(EnvProvider{BaseURL: "https://api.example.com", Model: "gpt-4"})
	expectedKeys := []string{
		"ANTHROPIC_BASE_URL", "ANTHROPIC_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_SMALL_FAST_MODEL", "NODE_OPTIONS",
	}
	envMap := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for _, key := range expectedKeys {
		if _, ok := envMap[key]; !ok {
			t.Errorf("Expected env var %q not found", key)
		}
	}
	if envMap["NODE_OPTIONS"] != "--no-deprecation" {
		t.Errorf("NODE_OPTIONS = %q, want %q", envMap["NODE_OPTIONS"], "--no-deprecation")
	}
}

// --- Tests for APIKeyEnvVarName ---

func TestAPIKeyEnvVarName(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"zai", "ZAI_API_KEY"},
		{"minimax", "MINIMAX_API_KEY"},
		{"deepseek", "DEEPSEEK_API_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if result := APIKeyEnvVarName(tt.provider); result != tt.want {
				t.Errorf("APIKeyEnvVarName(%q) = %q, want %q", tt.provider, result, tt.want)
			}
		})
	}
}

// --- Tests for BuildSecretsEnvVars ---

func TestBuildSecretsEnvVars(t *testing.T) {
	secrets := map[string]string{"ZAI_API_KEY": "sk-abc123", "EXTRA_VAR": "extra-value"}
	envVars := BuildSecretsEnvVars(secrets)
	if len(envVars) != 2 {
		t.Errorf("returned %d vars, want 2", len(envVars))
	}
	foundKeys := make(map[string]bool)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			foundKeys[parts[0]] = true
		}
	}
	if !foundKeys["ZAI_API_KEY"] || !foundKeys["EXTRA_VAR"] {
		t.Error("Missing expected keys in env vars")
	}
}

// --- Tests for ResetSecretsFiles ---

func TestResetSecretsFiles(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)
	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

	if err := crypto.GenerateKey(ctx, keyPath); err != nil {
		t.Fatal(err)
	}
	if err := crypto.EncryptSecrets(ctx, secretsPath, keyPath, "TEST_API_KEY=test-secret\n"); err != nil {
		t.Fatal(err)
	}

	err := ResetSecretsFiles(ctx, tmpDir, secretsPath, keyPath)
	if err != nil {
		t.Fatalf("ResetSecretsFiles() error = %v", err)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("new key file should exist after reset")
	}
	if _, err := os.Stat(secretsPath); !os.IsNotExist(err) {
		t.Error("old secrets file should be removed after reset")
	}
}

func TestResetSecretsFiles_NoExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)
	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

	err := ResetSecretsFiles(ctx, tmpDir, secretsPath, keyPath)
	if err != nil {
		t.Fatalf("ResetSecretsFiles() error = %v", err)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("new key file should exist after reset")
	}
}
