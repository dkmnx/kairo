package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/validate"
)

func TestSetup_ProviderNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "simple",
			wantErr: false,
		},
		{
			name:    "provider123",
			wantErr: false,
		},
		{
			name:    "my_provider",
			wantErr: false, // Should allow underscores
		},
		{
			name:    "custom-provider",
			wantErr: false, // Should allow hyphens
		},
		{
			name:    "provider_with_underscores",
			wantErr: false, // Should allow underscores
		},
		{
			name:    "provider-with-hyphens",
			wantErr: false, // Should allow hyphens
		},
		{
			name:    "",
			wantErr: true,
		},
		{
			name:    "123invalid",
			wantErr: true, // Must start with letter
		},
		{
			name:    "_invalid",
			wantErr: true, // Must start with letter
		},
		{
			name:    "-invalid",
			wantErr: true, // Must start with letter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestSetup_ProviderNameLength(t *testing.T) {
	maxValidName := strings.Repeat("a", 50) // Exactly 50 characters
	invalidName := strings.Repeat("b", 51)  // 51 characters - exceeds max

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "a",
			wantErr: false, // Minimum 1 character
		},
		{
			name:    "valid",
			wantErr: false,
		},
		{
			name:    maxValidName,
			wantErr: false, // Exactly 50 characters - max allowed
		},
		{
			name:    invalidName,
			wantErr: true, // 51 characters - exceeds max length
		},
		{
			name:    "this_provider_name_is_way_too_long_and_exceeds_the_maximum_allowed_length_of_fifty_characters",
			wantErr: true, // Much longer than 50 characters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestSetup_ProviderNameReservedWords(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "zai",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "minimax",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "deepseek",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "kimi",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "custom",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "ZAI",
			wantErr: true, // Reserved - case-insensitive
		},
		{
			name:    "mycustom",
			wantErr: false, // Not reserved
		},
		{
			name:    "provider",
			wantErr: false, // Not reserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestSetup_ValidateBaseURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		providerName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid https url",
			url:          "https://api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      false,
		},
		{
			name:         "valid https url with path",
			url:          "https://api.example.com/v1/anthropic",
			providerName: "test-provider",
			wantErr:      false,
		},
		{
			name:         "empty url",
			url:          "",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "cannot be empty",
		},
		{
			name:         "whitespace only url",
			url:          "   ",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "non-https url",
			url:          "http://api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "ftp url",
			url:          "ftp://api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "localhost url",
			url:          "https://localhost/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "127.0.0.1 url",
			url:          "https://127.0.0.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 10.x.x.x",
			url:          "https://10.0.0.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 172.16-31.x.x",
			url:          "https://172.16.0.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 192.168.x.x",
			url:          "https://192.168.1.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 169.254.x.x",
			url:          "https://169.254.1.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "invalid url format",
			url:          "not-a-url",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "url without scheme",
			url:          "api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "url with only scheme",
			url:          "https://",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.ValidateURL(tt.url, tt.providerName)

			if (err != nil) != tt.wantErr {
				t.Errorf("validate.ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("validate.ValidateURL(%q) expected error containing %q, got nil", tt.url, tt.errContains)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("validate.ValidateURL(%q) error = %q, want error containing %q", tt.url, err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestSetup_ValidateModel(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		displayName string
		model       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty model for custom provider",
			provider:    "custom-provider",
			displayName: "custom-provider",
			model:       "",
			wantErr:     true,
			errContains: "model name is required",
		},
		{
			name:        "whitespace only model for custom provider",
			provider:    "custom-provider",
			displayName: "custom-provider",
			model:       "   ",
			wantErr:     true,
			errContains: "model name is required",
		},
		{
			name:        "valid model for custom provider",
			provider:    "custom-provider",
			displayName: "custom-provider",
			model:       "gpt-4-turbo",
			wantErr:     false,
		},
		{
			name:        "empty model for built-in provider",
			provider:    "zai",
			displayName: "Z.AI",
			model:       "",
			wantErr:     false,
		},
		{
			name:        "valid model for built-in provider",
			provider:    "zai",
			displayName: "Z.AI",
			model:       "glm-4.7-flash",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguredModel(modelValidationConfig{
				Model:        tt.model,
				ProviderName: tt.provider,
				DisplayName:  tt.displayName,
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateConfiguredModel() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errContains != "" && (err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains))) {
				t.Fatalf("validateConfiguredModel() error = %v, want substring %q", err, tt.errContains)
			}
		})
	}
}

func TestResolveProviderName_NonCustom(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		want         string
		wantErr      bool
	}{
		{
			name:         "builtin provider zai",
			providerName: "zai",
			want:         "zai",
			wantErr:      false,
		},
		{
			name:         "builtin provider anthropic",
			providerName: "anthropic",
			want:         "anthropic",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProviderName(tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveProviderName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveProviderName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveProviderConfiguration_ValidationErrors(t *testing.T) {
	t.Run("missing config directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}

		err := AddAndSaveProvider(AddProviderParams{
			CLIContext:   NewCLIContext(),
			ConfigDir:    "/nonexistent/path/that/cannot/be/created",
			Cfg:          cfg,
			ProviderName: "testprovider",
			Provider: config.Provider{
				Name:    "Test Provider",
				BaseURL: "https://test.com",
				Model:   "test-model",
			},
			SetAsDefault: true,
		})
		if err == nil {
			t.Error("expected error for invalid config directory")
		}
	})
}
