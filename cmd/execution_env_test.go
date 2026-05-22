package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

func TestRequiresAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"built-in provider with key", "zai", true},
		{"built-in anthropic", "anthropic", true},
		{"unknown provider defaults to true", "unknown-provider", true},
		{"empty provider defaults to true", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequiresAPIKey(tt.provider)
			if got != tt.want {
				t.Errorf("RequiresAPIKey(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestBuildProviderEnvironment_Success(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		BaseURL: "https://api.test.com",
		Model:   "test-model",
		EnvVars: []string{"CUSTOM_VAR=custom_value"},
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	result, err := BuildProviderEnv(cliCtx, tmpDir, config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "test-provider")
	if err != nil {
		t.Fatalf("BuildProviderEnv() should succeed with no secrets file, got: %v", err)
	}

	if result.Secrets == nil {
		t.Error("BuildProviderEnv() should return empty secrets map, not nil")
	}

	if len(result.ProviderEnv) == 0 {
		t.Error("BuildProviderEnv() should return provider environment variables")
	}
}

func TestApiKeyEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"lowercase provider", "anthropic", "ANTHROPIC_API_KEY"},
		{"uppercase provider", "ANTHROPIC", "ANTHROPIC_API_KEY"},
		{"mixed case provider", "MiniMax", "MINIMAX_API_KEY"},
		{"provider with hyphen", "my-provider", "MY-PROVIDER_API_KEY"},
		{"provider with underscore", "my_provider", "MY_PROVIDER_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := APIKeyEnvVarName(tt.provider)
			if result != tt.expected {
				t.Errorf("APIKeyEnvVarName(%q) = %q, want %q", tt.provider, result, tt.expected)
			}
		})
	}
}

func TestBuildProviderEnvironment_NoAPIKeyRequired(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"CUSTOM_VAR=value"},
	}

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	result, err := BuildProviderEnv(cliCtx, tmpDir, config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "ollama")
	if err != nil {
		t.Fatalf("BuildProviderEnv() for provider without API key should not error, got: %v", err)
	}
	if result.ProviderEnv == nil {
		t.Error("BuildProviderEnv() returned nil env for provider without API key")
	}
	if result.Secrets == nil {
		t.Error("BuildProviderEnv() returned nil secrets map")
	}
}

func TestBuildProviderEnvironment_WithProviderEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"PROVIDER_VAR=provider_value", "ANOTHER_VAR=another_value"},
	}

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	result, err := BuildProviderEnv(cliCtx, tmpDir, config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "ollama")
	if err != nil {
		t.Fatalf("BuildProviderEnv() error = %v", err)
	}
	if len(result.ProviderEnv) == 0 {
		t.Error("BuildProviderEnv() should include provider EnvVars")
	}
	if len(result.ProviderEnv) < len(provider.EnvVars) {
		t.Error("BuildProviderEnv() should include all provider EnvVars")
	}
	envStr := strings.Join(result.ProviderEnv, "|")
	if !strings.Contains(envStr, "PROVIDER_VAR=provider_value") {
		t.Error("buildProviderEnvironment() should include provider EnvVars")
	}
	if !strings.Contains(envStr, "ANOTHER_VAR=another_value") {
		t.Error("buildProviderEnvironment() should include all provider EnvVars")
	}
}

func TestBuildBuiltInEnvVars_Extended(t *testing.T) {
	t.Run("provider with special characters in values", func(t *testing.T) {
		provider := config.Provider{
			BaseURL: "https://api.test.com/path?query=value",
			Model:   "test-model-v1.0-beta",
		}

		envVars := BuildBuiltInEnvVars(config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars})
		if len(envVars) == 0 {
			t.Error("buildBuiltInEnvVars() returned empty slice")
		}

		hasBaseURL := false
		hasModel := false
		for _, v := range envVars {
			if strings.HasPrefix(v, "ANTHROPIC_BASE_URL=") {
				hasBaseURL = true
			}
			if strings.HasPrefix(v, "ANTHROPIC_MODEL=") {
				hasModel = true
			}
		}

		if !hasBaseURL {
			t.Error("missing ANTHROPIC_BASE_URL")
		}
		if !hasModel {
			t.Error("missing ANTHROPIC_MODEL")
		}
	})
}

func TestBuildPiEnvVars(t *testing.T) {
	provider := config.Provider{
		BaseURL: "https://api.z.ai/api/anthropic",
		Model:   "glm-5.1",
	}
	envVars := BuildPiEnvVars(provider, "zai")

	hasProvider := false
	hasModel := false
	for _, v := range envVars {
		if v == "PI_PROVIDER=zai" {
			hasProvider = true
		}
		if v == "PI_MODEL=glm-5.1" {
			hasModel = true
		}
	}

	if !hasProvider {
		t.Error("missing PI_PROVIDER")
	}
	if !hasModel {
		t.Error("missing PI_MODEL")
	}
}

func TestPiAPIKeyEnvVarMapping(t *testing.T) {
	tests := []struct {
		provider string
		envVar   string
		ok       bool
	}{
		{"zai", "ZAI_API_KEY", true},
		{"minimax", "MINIMAX_API_KEY", true},
		{"deepseek", "DEEPSEEK_API_KEY", true},
		{"kimi", "KIMI_API_KEY", true},
		{"unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			envVar, ok := PiAPIKeyEnvVar(tt.provider)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if envVar != tt.envVar {
				t.Errorf("envVar = %q, want %q", envVar, tt.envVar)
			}
		})
	}
}
