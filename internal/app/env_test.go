package app

import (
	"context"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

func TestBuildProviderEnv_Success(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		BaseURL: "https://api.test.com",
		Model:   "test-model",
		EnvVars: []string{"CUSTOM_VAR=custom_value"},
	}

	result, err := BuildProviderEnv(context.Background(), crypto.DefaultService{}, tmpDir, provider, "test-provider")
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

func TestBuildProviderEnv_NoAPIKeyRequired(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"CUSTOM_VAR=value"},
	}

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	result, err := BuildProviderEnv(context.Background(), crypto.DefaultService{}, tmpDir, provider, "ollama")
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

func TestBuildProviderEnv_WithProviderEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"PROVIDER_VAR=provider_value", "ANOTHER_VAR=another_value"},
	}

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	result, err := BuildProviderEnv(context.Background(), crypto.DefaultService{}, tmpDir, provider, "ollama")
	if err != nil {
		t.Fatalf("BuildProviderEnv() error = %v", err)
	}

	envStr := strings.Join(result.ProviderEnv, "|")
	if !strings.Contains(envStr, "PROVIDER_VAR=provider_value") {
		t.Error("BuildProviderEnv() should include provider EnvVars")
	}
	if !strings.Contains(envStr, "ANOTHER_VAR=another_value") {
		t.Error("BuildProviderEnv() should include all provider EnvVars")
	}
}

func TestBuiltInEnvVars(t *testing.T) {
	provider := config.Provider{
		BaseURL: "https://api.test.com",
		Model:   "test-model",
	}

	envVars := BuiltInEnvVars(provider)

	envMap := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["ANTHROPIC_BASE_URL"] != provider.BaseURL {
		t.Errorf("ANTHROPIC_BASE_URL = %s, want %s", envMap["ANTHROPIC_BASE_URL"], provider.BaseURL)
	}
	if envMap["ANTHROPIC_MODEL"] != provider.Model {
		t.Errorf("ANTHROPIC_MODEL = %s, want %s", envMap["ANTHROPIC_MODEL"], provider.Model)
	}

	notExpected := []string{
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_SMALL_FAST_MODEL",
	}
	for _, key := range notExpected {
		if _, exists := envMap[key]; exists {
			t.Errorf("BuiltInEnvVars() should not set %s", key)
		}
	}
}

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

	if !InjectPiAPIKeys(&envResult, cfg) {
		t.Fatal("expected InjectPiAPIKeys to find at least one key")
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

	if InjectPiAPIKeys(&envResult, cfg) {
		t.Error("expected InjectPiAPIKeys to return false when no secret exists")
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
			name:         "catalog non-conventional huggingface",
			providerName: "huggingface",
			provider:     config.Provider{},
			want:         "HF_TOKEN",
		},
		{
			name:         "catalog non-conventional google",
			providerName: "google",
			provider:     config.Provider{},
			want:         "GEMINI_API_KEY",
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
			got := APIKeyEnvVarName(tt.providerName, tt.provider)
			if got != tt.want {
				t.Errorf("APIKeyEnvVarName(%q) = %q, want %q", tt.providerName, got, tt.want)
			}
		})
	}
}

func TestSecretsKeyIsConventionalNotCatalogEnvVar(t *testing.T) {
	// Lookup must use conventional PROVIDER_API_KEY storage keys even when
	// the process env name is catalog-specific (HF_TOKEN).
	secretsMap := map[string]string{
		"HUGGINGFACE_API_KEY": "hf-test",
	}

	key, ok := LookupAPIKeyWithFallback(secretsMap, "huggingface")
	if !ok || key != "hf-test" {
		t.Fatalf("LookupAPIKeyWithFallback() = %q, %v", key, ok)
	}

	if APIKeyEnvVarName("huggingface", config.Provider{}) != "HF_TOKEN" {
		t.Error("process env name should be catalog HF_TOKEN")
	}
}

func TestLookupAPIKeyWithFallback(t *testing.T) {
	t.Run("returns provider-specific key", func(t *testing.T) {
		key, ok := LookupAPIKeyWithFallback(map[string]string{
			"ANTHROPIC_API_KEY": "sk-ant-xxx",
		}, "anthropic")
		if !ok || key != "sk-ant-xxx" {
			t.Errorf("LookupAPIKeyWithFallback() = %q, %v, want sk-ant-xxx, true", key, ok)
		}
	})

	t.Run("falls back to custom provider key", func(t *testing.T) {
		key, ok := LookupAPIKeyWithFallback(map[string]string{
			"CUSTOM_API_KEY": "sk-custom-xxx",
		}, "anthropic")
		if !ok || key != "sk-custom-xxx" {
			t.Errorf("LookupAPIKeyWithFallback() = %q, %v, want sk-custom-xxx, true", key, ok)
		}
	})

	t.Run("returns false when no key found", func(t *testing.T) {
		if _, ok := LookupAPIKeyWithFallback(map[string]string{}, "anthropic"); ok {
			t.Error("Expected no key to be found")
		}
	})
}
