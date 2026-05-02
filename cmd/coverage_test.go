package cmd

import (
	"context"
	"os"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestHandleSecretsError(t *testing.T) {
	testErr := os.ErrNotExist
	handleSecretsError(testErr)
}

// TestBuildProviderListOptions is now in setup_prompts_test.go with table-driven tests.

func TestBuildProviderConfig(t *testing.T) {
	t.Run("new provider", func(t *testing.T) {
		def := providers.ProviderDefinition{
			Name:    "test",
			BaseURL: "https://api.test.com",
			Model:   "test-model",
		}
		got := BuildProviderConfig(ProviderBuildConfig{
			Definition: def,
			BaseURL:    "https://api.test.com",
			Model:      "test-model",
		})
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
		got := BuildProviderConfig(ProviderBuildConfig{
			Definition: providers.ProviderDefinition{},
			BaseURL:    "https://new.com",
			Model:      "new-model",
			Exists:     true,
			Existing:   &existing,
		})
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

// TestBuildSecretsEnvVars, TestBuildBuiltInEnvVars, and TestAPIKeyEnvVarName
// are now in coverage_env_test.go with improved coverage.

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

// TestAPIKeyEnvVarName is now in coverage_env_test.go with additional test cases.

func TestResolveProviderName(t *testing.T) {
	name, err := ResolveProviderName(context.Background(), "anthropic")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if name != "anthropic" {
		t.Errorf("expected 'anthropic', got %q", name)
	}
}

// TestGetProviderDefinition is now in coverage_config_test.go with table-driven tests.
