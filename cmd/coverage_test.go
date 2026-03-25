package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestHandleSecretsError(t *testing.T) {
	testErr := os.ErrNotExist
	handleSecretsError(testErr)
}

func TestBuildProviderListOptions(t *testing.T) {
	providerList := []string{"anthropic", "zai", "minimax"}
	options := buildProviderListOptions(providerList)

	if len(options) != 3 {
		t.Errorf("expected 3 options, got %d", len(options))
	}

	expectedProviders := map[string]bool{
		"anthropic": true,
		"zai":       true,
		"minimax":   true,
	}

	for _, opt := range options {
		if !expectedProviders[opt.Value] {
			t.Errorf("unexpected provider: %s", opt.Value)
		}
		if opt.Label != opt.Value {
			t.Errorf("label should match value for %s", opt.Value)
		}
	}
}

func TestBuildProviderConfigFromInput(t *testing.T) {
	t.Run("new provider", func(t *testing.T) {
		def := providers.ProviderDefinition{
			Name:    "test",
			BaseURL: "https://api.test.com",
			Model:   "test-model",
		}
		got := BuildProviderConfigFromInput(def, "https://api.test.com", "test-model", false, config.Provider{})
		if got.Name != "test" {
			t.Errorf("Name = %v, want test", got.Name)
		}
		if got.BaseURL != "https://api.test.com" {
			t.Errorf("BaseURL = %v, want https://api.test.com", got.BaseURL)
		}
		if got.Model != "test-model" {
			t.Errorf("Model = %v, want test-model", got.Model)
		}
	})

	t.Run("existing provider", func(t *testing.T) {
		existing := config.Provider{
			Name:    "existing",
			BaseURL: "https://old.com",
			Model:   "old-model",
		}
		got := BuildProviderConfigFromInput(providers.ProviderDefinition{}, "https://new.com", "new-model", true, existing)
		if got.Name != "existing" {
			t.Errorf("Name = %v, want existing", got.Name)
		}
		if got.BaseURL != "https://new.com" {
			t.Errorf("BaseURL = %v, want https://new.com", got.BaseURL)
		}
		if got.Model != "new-model" {
			t.Errorf("Model = %v, want new-model", got.Model)
		}
	})
}

func TestBuildSecretsEnvVars(t *testing.T) {
	secrets := map[string]string{
		"ANTHROPIC_API_KEY": "test-key-123",
		"ZAI_API_KEY":       "zai-key-456",
	}

	envVars := BuildSecretsEnvVars(secrets)

	if len(envVars) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(envVars))
	}

	expectedVars := map[string]bool{
		"ANTHROPIC_API_KEY=test-key-123": true,
		"ZAI_API_KEY=zai-key-456":        true,
	}

	for _, envVar := range envVars {
		if !expectedVars[envVar] {
			t.Errorf("unexpected env var: %s", envVar)
		}
	}
}

func TestBuildBuiltInEnvVars(t *testing.T) {
	provider := EnvProvider{
		BaseURL: "https://api.test.com",
		Model:   "test-model",
	}

	envVars := BuildBuiltInEnvVars(provider)

	expectedKeys := []string{
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_SMALL_FAST_MODEL",
	}

	envMap := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	for _, key := range expectedKeys {
		if _, exists := envMap[key]; !exists {
			t.Errorf("BuildBuiltInEnvVars() missing expected key %s", key)
		}
	}

	if envMap["ANTHROPIC_BASE_URL"] != provider.BaseURL {
		t.Errorf("ANTHROPIC_BASE_URL = %s, want %s", envMap["ANTHROPIC_BASE_URL"], provider.BaseURL)
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		wantKairo   []string
		wantHarness []string
	}{
		{
			name:        "no separator",
			input:       []string{"arg1", "arg2"},
			wantKairo:   []string{"arg1", "arg2"},
			wantHarness: nil,
		},
		{
			name:        "with separator",
			input:       []string{"arg1", "--", "arg2", "arg3"},
			wantKairo:   []string{"arg1"},
			wantHarness: []string{"arg2", "arg3"},
		},
		{
			name:        "empty args",
			input:       []string{},
			wantKairo:   []string{},
			wantHarness: nil,
		},
		{
			name:        "separator at start",
			input:       []string{"--", "arg1"},
			wantKairo:   []string{},
			wantHarness: []string{"arg1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKairo, gotHarness := splitArgs(tt.input)
			if len(gotKairo) != len(tt.wantKairo) {
				t.Errorf("kairo args length = %v, want %v", len(gotKairo), len(tt.wantKairo))
			}
			if len(gotHarness) != len(tt.wantHarness) {
				t.Errorf("harness args length = %v, want %v", len(gotHarness), len(tt.wantHarness))
			}
		})
	}
}

func TestAPIKeyEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"anthropic", "anthropic", "ANTHROPIC_API_KEY"},
		{"zai", "zai", "ZAI_API_KEY"},
		{"minimax", "minimax", "MINIMAX_API_KEY"},
		{"UPPERCASE", "UPPERCASE", "UPPERCASE_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := APIKeyEnvVarName(tt.input)
			if got != tt.expected {
				t.Errorf("APIKeyEnvVarName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveProviderName(t *testing.T) {
	name, err := ResolveProviderName("anthropic")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if name != "anthropic" {
		t.Errorf("expected 'anthropic', got %q", name)
	}
}

func TestGetProviderDefinition(t *testing.T) {
	def := GetProviderDefinition("anthropic")
	if def.Name == "" {
		t.Error("expected non-empty provider definition")
	}

	def = GetProviderDefinition("custom-provider")
	if def.Name != "custom-provider" {
		t.Errorf("expected 'custom-provider', got %q", def.Name)
	}
}
