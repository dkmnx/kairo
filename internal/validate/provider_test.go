package validate

import (
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestValidateCrossProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string // substring to check in error message
	}{
		{
			name: "empty providers",
			cfg: &config.Config{
				Providers: map[string]config.Provider{},
			},
			wantErr: false,
		},
		{
			name: "single provider",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"test": {
						EnvVars: []string{"KEY=value"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple providers same env var same value",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"SHARED_VAR=value1"},
					},
					"provider2": {
						EnvVars: []string{"SHARED_VAR=value1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple providers same env var different values",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"SHARED_VAR=value1"},
					},
					"provider2": {
						EnvVars: []string{"SHARED_VAR=value2"},
					},
				},
			},
			wantErr: true,
			errMsg:  "environment variable collision",
		},
		{
			name: "multiple providers different env vars",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"VAR1=value1", "VAR2=value2"},
					},
					"provider2": {
						EnvVars: []string{"VAR3=value3"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "three providers collision",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
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
			},
			wantErr: true,
			errMsg:  "environment variable collision",
		},
		{
			name: "env var with equals in value",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"JSON_DATA={\"key\":\"value\"}"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "malformed env var (no equals)",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"INVALID_VAR"},
					},
				},
			},
			wantErr: false, // malformed vars are skipped
		},
		{
			name: "malformed env var (empty key)",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"=value"},
					},
				},
			},
			wantErr: false, // malformed vars are skipped
		},
		{
			name: "whitespace in key and value - same after trim",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"  KEY  =  value  "},
					},
					"provider2": {
						EnvVars: []string{"KEY=value"},
					},
				},
			},
			wantErr: false, // Keys and values match after trimming - no collision
		},
		{
			name: "whitespace in key - different values after trim",
			cfg: &config.Config{
				Providers: map[string]config.Provider{
					"provider1": {
						EnvVars: []string{"  KEY  =  value1  "},
					},
					"provider2": {
						EnvVars: []string{"KEY=value2"},
					},
				},
			},
			wantErr: true, // Keys match after trim, but values differ -> collision
			errMsg:  "environment variable collision",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCrossProviderConfig(tt.cfg)
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

		// Invalid model names
		{
			name:        "model too long",
			provider:    "zai",
			model:       strings.Repeat("a", 101),
			wantErr:     true,
			errContains: "too long",
		},
		{
			name:        "model with invalid character @",
			provider:    "zai",
			model:       "invalid@model",
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

		// Non-built-in providers (should skip validation)
		{
			name:     "non-built-in provider with any model",
			provider: "custom",
			model:    "any-model-name-@#$%",
			wantErr:  false,
		},
		{
			name:     "non-built-in provider with empty model",
			provider: "unknown",
			model:    "",
			wantErr:  false,
		},

		// Built-in providers without default models
		{
			name:     "anthropic (no default model) valid",
			provider: "anthropic",
			model:    "any-model",
			wantErr:  false,
		},
		{
			name:     "custom (no default model) valid",
			provider: "custom",
			model:    "model@#$",
			wantErr:  false, // no validation for providers without default model
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
		{'@', false},
		{'#', false},
		{'!', false},
		{' ', false},
		{'/', false},
		{':', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.rune), func(t *testing.T) {
			if got := isValidModelRune(tt.rune); got != tt.valid {
				t.Errorf("isValidModelRune(%q) = %v, want %v", tt.rune, got, tt.valid)
			}
		})
	}
}
