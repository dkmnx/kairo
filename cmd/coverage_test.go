// Package cmd tests for improving code coverage
package cmd

import (
	"os"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

// TestHandleSecretsError tests handleSecretsError function
func TestHandleSecretsError(t *testing.T) {
	// This function only prints to UI, so we just verify it doesn't panic
	testErr := os.ErrNotExist
	handleSecretsError(testErr)
}

// TestBuildProviderListOptions tests building provider list options
func TestBuildProviderListOptions(t *testing.T) {
	providers := []string{"anthropic", "zai", "minimax"}
	options := buildProviderListOptions(providers)

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

// TestBuildProviderConfigFromInput tests building provider config from input
func TestBuildProviderConfigFromInput(t *testing.T) {
	tests := []struct {
		name        string
		exists      bool
		input       BuildProviderConfigParams
		wantName    string
		wantBaseURL string
		wantModel   string
	}{
		{
			name:   "new provider",
			exists: false,
			input: BuildProviderConfigParams{
				Definition: providers.ProviderDefinition{
					Name:    "test",
					BaseURL: "https://api.test.com",
					Model:   "test-model",
				},
				BaseURL: "https://api.test.com",
				Model:   "test-model",
			},
			wantName:    "test",
			wantBaseURL: "https://api.test.com",
			wantModel:   "test-model",
		},
		{
			name:   "existing provider",
			exists: true,
			input: BuildProviderConfigParams{
				Existing: config.Provider{
					Name:    "existing",
					BaseURL: "https://old.com",
					Model:   "old-model",
				},
				BaseURL: "https://new.com",
				Model:   "new-model",
			},
			wantName:    "existing",
			wantBaseURL: "https://new.com",
			wantModel:   "new-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildProviderConfigFromInput(tt.input)
			if got.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", got.Name, tt.wantName)
			}
			if got.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %v, want %v", got.BaseURL, tt.wantBaseURL)
			}
			if got.Model != tt.wantModel {
				t.Errorf("Model = %v, want %v", got.Model, tt.wantModel)
			}
		})
	}
}

// TestBuildSecretsEnvVars tests building secrets env vars
func TestBuildSecretsEnvVars(t *testing.T) {
	secrets := map[string]string{
		"ANTHROPIC_API_KEY": "test-key-123",
		"ZAI_API_KEY":       "zai-key-456",
	}

	envVars := buildSecretsEnvVars(secrets)

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

// TestSplitArgs tests splitArgs function
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

// TestAPIKeyEnvVarName tests API key env var name
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
			got := apiKeyEnvVarName(tt.input)
			if got != tt.expected {
				t.Errorf("apiKeyEnvVarName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestResolveProviderName tests resolving provider name
func TestResolveProviderName(t *testing.T) {
	// Test with non-custom provider name
	name, err := resolveProviderName("anthropic")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if name != "anthropic" {
		t.Errorf("expected 'anthropic', got %q", name)
	}

	// Test with custom - this would normally prompt for input
	// We skip this in automated tests as it requires UI interaction
}

// TestGetProviderDefinition tests getting provider definition
func TestGetProviderDefinition(t *testing.T) {
	def := getProviderDefinition("anthropic")
	if def.Name == "" {
		t.Error("expected non-empty provider definition")
	}

	// Test with custom provider name
	def = getProviderDefinition("custom-provider")
	if def.Name != "custom-provider" {
		t.Errorf("expected 'custom-provider', got %q", def.Name)
	}
}
