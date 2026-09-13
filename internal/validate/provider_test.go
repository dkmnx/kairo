package validate

import (
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestValidateCrossProviderConfig(t *testing.T) {
	tests := []struct {
		name      string
		providers map[string]config.Provider
		wantErr   bool
		errMsg    string // substring to check in error message
	}{
		{
			name:      "empty providers",
			providers: map[string]config.Provider{},
			wantErr:   false,
		},
		{
			name: "single provider",
			providers: map[string]config.Provider{
				"test": {
					EnvVars: []string{"KEY=value"},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple providers same env var same value",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"SHARED_VAR=value1"},
				},
				"provider2": {
					EnvVars: []string{"SHARED_VAR=value1"},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple providers same env var different values",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"SHARED_VAR=value1"},
				},
				"provider2": {
					EnvVars: []string{"SHARED_VAR=value2"},
				},
			},
			wantErr: true,
			errMsg:  "environment variable",
		},
		{
			name: "multiple providers different env vars",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"VAR1=value1", "VAR2=value2"},
				},
				"provider2": {
					EnvVars: []string{"VAR3=value3"},
				},
			},
			wantErr: false,
		},
		{
			name: "three providers collision",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"COMMON=test"},
				},
				"provider2": {
					EnvVars: []string{"COMMON=test"},
				},
				"provider3": {
					EnvVars: []string{"COMMON=different"},
				},
			},
			wantErr: true,
			errMsg:  "environment variable",
		},
		{
			name: "env var with equals in value",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"JSON_DATA={\"key\":\"value\"}"},
				},
			},
			wantErr: false,
		},
		{
			name: "valueless env var (no equals)",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"INVALID_VAR"},
				},
			},
			wantErr: false, // valueless entries bind the key; single provider -> no conflict
		},
		{
			name: "malformed env var (empty key)",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"=value"},
				},
			},
			wantErr: false, // empty keys are skipped
		},
		{
			name: "whitespace in key and value - same after trim",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"  KEY  =  value  "},
				},
				"provider2": {
					EnvVars: []string{"KEY=value"},
				},
			},
			wantErr: false, // Keys and values match after trimming - no collision
		},
		{
			name: "whitespace in key - different values after trim",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"  KEY  =  value1  "},
				},
				"provider2": {
					EnvVars: []string{"KEY=value2"},
				},
			},
			wantErr: true, // Keys match after trim, but values differ -> collision
			errMsg:  "environment variable",
		},
		{
			name: "valueless env var vs valued env var",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"SHARED"},
				},
				"provider2": {
					EnvVars: []string{"SHARED=value"},
				},
			},
			wantErr: true, // valueless entry binds the key with an empty value
			errMsg:  "SHARED",
		},
		{
			name: "two valueless entries same key",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"SHARED"},
				},
				"provider2": {
					EnvVars: []string{"SHARED"},
				},
			},
			wantErr: false, // both empty -> no conflict
		},
		{
			name: "shared api key env var",
			providers: map[string]config.Provider{
				"provider1": {
					EnvKey: "CUSTOM_API_KEY",
				},
				"provider2": {
					EnvKey: "CUSTOM_API_KEY",
				},
			},
			wantErr: true, // two providers bound to the same key variable
			errMsg:  "CUSTOM_API_KEY",
		},
		{
			name: "api key env var vs regular env var",
			providers: map[string]config.Provider{
				"provider1": {
					EnvKey: "FOO",
				},
				"provider2": {
					EnvVars: []string{"FOO=value"},
				},
			},
			wantErr: true, // key binding clashes with a regular env var
			errMsg:  "FOO",
		},
		{
			name: "same provider key env and regular env var",
			providers: map[string]config.Provider{
				"provider1": {
					EnvKey:  "FOO",
					EnvVars: []string{"FOO=value"},
				},
			},
			wantErr: false, // single provider, no cross-provider conflict
		},
		{
			name: "aggregates multiple conflicts",
			providers: map[string]config.Provider{
				"provider1": {
					EnvVars: []string{"A=1", "B=1"},
				},
				"provider2": {
					EnvVars: []string{"A=2", "B=2"},
				},
			},
			wantErr: true,
			errMsg:  "A, B", // both conflicts reported in one error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCrossProviderConfig(tt.providers)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCrossProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCrossProviderConfig() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateProviderModel(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		model       string
		wantErr     bool
		errContains string
	}{
		// Empty model tests
		{
			name:     "empty model",
			provider: "anthropic",
			model:    "",
			wantErr:  false,
		},

		// Valid model names
		{
			name:     "valid model for built-in provider",
			provider: "zai",
			model:    "valid-model-name",
			wantErr:  false,
		},
		{
			name:     "valid model with dots",
			provider: "zai",
			model:    "model.v1.0",
			wantErr:  false,
		},
		{
			name:     "valid model with numbers",
			provider: "zai",
			model:    "claude-3-5-sonnet-20241022",
			wantErr:  false,
		},
		{
			name:     "valid model with context window brackets",
			provider: "zai",
			model:    "deepseek-v4-pro[1m]",
			wantErr:  false,
		},
		{
			name:     "valid model with bracket suffix",
			provider: "zai",
			model:    "model[128k]",
			wantErr:  false,
		},
		{
			name:     "valid model with org slash",
			provider: "zai",
			model:    "org/model",
			wantErr:  false,
		},
		{
			name:     "valid model with deployment colon",
			provider: "zai",
			model:    "azure-deployment:gpt-4o",
			wantErr:  false,
		},
		{
			name:     "valid model with vendor plus",
			provider: "zai",
			model:    "vendor+flavor",
			wantErr:  false,
		},
		{
			name:     "valid model with at-sign path",
			provider: "zai",
			model:    "@cf/meta/llama-3.1-8b-instruct",
			wantErr:  false,
		},
		{
			name:     "valid model with accounts path and parens",
			provider: "zai",
			model:    "accounts/fireworks/models/llama-v3p1-70b(gen)",
			wantErr:  false,
		},

		// Invalid model names
		{
			name:        "model too long",
			provider:    "zai",
			model:       strings.Repeat("a", 101),
			wantErr:     true,
			errContains: "too long",
		},
		{
			name:        "model with invalid character dollar",
			provider:    "zai",
			model:       "invalid$model",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:        "model with invalid character #",
			provider:    "zai",
			model:       "model#name",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:        "model with invalid character space",
			provider:    "zai",
			model:       "invalid model",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:        "model with invalid character !",
			provider:    "zai",
			model:       "model!name",
			wantErr:     true,
			errContains: "invalid characters",
		},

		// Uniform validation: every provider is held to the same charset.
		{
			name:     "built-in without default model valid",
			provider: "anthropic",
			model:    "claude-3-5-sonnet-20241022",
			wantErr:  false,
		},
		{
			name:        "built-in without default model invalid",
			provider:    "anthropic",
			model:       "invalid@model#name",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:     "custom catalog provider valid",
			provider: "custom",
			model:    "my-custom-model",
			wantErr:  false,
		},
		{
			name:        "custom catalog provider invalid",
			provider:    "custom",
			model:       "model@#$",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:     "unknown provider valid",
			provider: "nonexistent",
			model:    "some-model",
			wantErr:  false,
		},
		{
			name:        "unknown provider invalid",
			provider:    "nonexistent",
			model:       "bad!model",
			wantErr:     true,
			errContains: "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderModel(tt.provider, tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProviderModel() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateProviderModel() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestIsValidModelRune(t *testing.T) {
	tests := []struct {
		rune  rune
		valid bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', true},
		{'_', true},
		{'.', true},
		{'[', true},
		{']', true},
		{':', true},
		{'/', true},
		{'+', true},
		{'@', true},
		{'(', true},
		{')', true},
		{'#', false},
		{'!', false},
		{'$', false},
		{'%', false},
		{' ', false},
		{'\t', false},
		{'\n', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.rune), func(t *testing.T) {
			if got := isValidModelRune(tt.rune); got != tt.valid {
				t.Errorf("isValidModelRune(%q) = %v, want %v", tt.rune, got, tt.valid)
			}
		})
	}
}
