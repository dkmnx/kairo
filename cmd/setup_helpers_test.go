package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestValidateCustomProviderName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid custom name",
			provider: "myprovider",
			wantErr:  false,
		},
		{
			name:     "valid name with hyphen",
			provider: "my-provider",
			wantErr:  false,
		},
		{
			name:     "valid name with underscore",
			provider: "my_provider",
			wantErr:  false,
		},
		{
			name:     "empty name",
			provider: "",
			wantErr:  true,
			errMsg:   "provider name is required",
		},
		{
			name:     "starts with number",
			provider: "123provider",
			wantErr:  true,
			errMsg:   "start with a letter",
		},
		{
			name:     "contains special characters",
			provider: "my@provider",
			wantErr:  true,
			errMsg:   "alphanumeric characters",
		},
		{
			name:     "reserved builtin name",
			provider: "zai",
			wantErr:  true,
			errMsg:   "reserved provider name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerName, err := validateCustomProviderName(tt.provider)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message should contain %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if providerName != tt.provider {
					t.Errorf("Provider name = %q, want %q", providerName, tt.provider)
				}
			}
		})
	}
}

func TestBuildProviderConfig(t *testing.T) {
	t.Run("builds config from definition", func(t *testing.T) {
		def := providers.ProviderDefinition{
			Name:           "Test Provider",
			BaseURL:        "https://api.test.com",
			Model:          "test-model",
			EnvVars:        []string{"VAR1=value1", "VAR2=value2"},
			RequiresAPIKey: true,
		}

		baseURL := "https://custom.url"
		model := "custom-model"

		provider := buildProviderConfig(def, baseURL, model)

		if provider.Name != def.Name {
			t.Errorf("Name = %q, want %q", provider.Name, def.Name)
		}
		if provider.BaseURL != baseURL {
			t.Errorf("BaseURL = %q, want %q", provider.BaseURL, baseURL)
		}
		if provider.Model != model {
			t.Errorf("Model = %q, want %q", provider.Model, model)
		}
		if len(provider.EnvVars) != 2 {
			t.Errorf("EnvVars length = %d, want 2", len(provider.EnvVars))
		}
	})

	t.Run("handles empty env vars", func(t *testing.T) {
		def := providers.ProviderDefinition{
			Name:           "Test Provider",
			BaseURL:        "https://api.test.com",
			Model:          "test-model",
			EnvVars:        nil,
			RequiresAPIKey: true,
		}

		provider := buildProviderConfig(def, "https://test.com", "model")

		if provider.EnvVars != nil {
			t.Errorf("EnvVars should be nil, got %v", provider.EnvVars)
		}
	})
}

func TestFormatSecrets(t *testing.T) {
	t.Run("formats secrets with sorting", func(t *testing.T) {
		secrets := map[string]string{
			"Z_KEY": "value1",
			"A_KEY": "value2",
			"M_KEY": "value3",
		}

		content := config.FormatSecrets(secrets)

		lines := strings.Split(strings.TrimSpace(content), "\n")
		if len(lines) != 3 {
			t.Errorf("Expected 3 lines, got %d", len(lines))
		}

		// Check that keys are sorted
		if !strings.HasPrefix(lines[0], "A_KEY=") {
			t.Errorf("First line should start with A_KEY, got: %s", lines[0])
		}
		if !strings.HasPrefix(lines[1], "M_KEY=") {
			t.Errorf("Second line should start with M_KEY, got: %s", lines[1])
		}
		if !strings.HasPrefix(lines[2], "Z_KEY=") {
			t.Errorf("Third line should start with Z_KEY, got: %s", lines[2])
		}
	})

	t.Run("handles empty secrets", func(t *testing.T) {
		secrets := map[string]string{}

		content := config.FormatSecrets(secrets)

		if content != "" {
			t.Errorf("Expected empty string, got: %q", content)
		}
	})
}

func TestSaveProviderConfigFile(t *testing.T) {
	t.Run("saves provider to config", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			DefaultProvider: "",
			Providers:       make(map[string]config.Provider),
		}

		provider := config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		}

		err := saveProviderConfigFile(tmpDir, cfg, "testprovider", provider, true)
		if err != nil {
			t.Fatalf("saveProviderConfigFile() error = %v", err)
		}

		if cfg.DefaultProvider != "testprovider" {
			t.Errorf("DefaultProvider = %q, want 'testprovider'", cfg.DefaultProvider)
		}

		savedProvider, ok := cfg.Providers["testprovider"]
		if !ok {
			t.Error("Provider not saved to config")
		}

		if savedProvider.Name != "Test Provider" {
			t.Errorf("Provider name = %q, want 'Test Provider'", savedProvider.Name)
		}
	})
}

func TestValidateAPIKey(t *testing.T) {
	t.Run("delegates to validate.ValidateAPIKey", func(t *testing.T) {
		// This test ensures our wrapper works correctly
		err := validateAPIKey("short", "Test Provider")
		if err == nil {
			t.Error("Should return error for short key")
		}

		// Check it's the right validation error type
		if !strings.Contains(err.Error(), "API key") {
			t.Errorf("Error should mention API key, got: %v", err)
		}
	})
}

func TestValidateBaseURL(t *testing.T) {
	t.Run("delegates to validate.ValidateURL", func(t *testing.T) {
		err := validateBaseURL("http://insecure.com", "Test Provider")
		if err == nil {
			t.Error("Should return error for http URL")
		}

		if !strings.Contains(err.Error(), "HTTPS") {
			t.Errorf("Error should mention HTTPS, got: %v", err)
		}
	})
}

func TestAuditLoggerErrorHandling(t *testing.T) {
	t.Run("logAuditEvent returns error on invalid directory", func(t *testing.T) {
		nonExistentDir := "/this/directory/does/not/exist/xyz123"

		err := logAuditEvent(nonExistentDir, func(l *audit.Logger) error {
			return nil
		})

		if err == nil {
			t.Error("logAuditEvent should return error when directory doesn't exist")
		}
	})

	t.Run("logAuditEvent returns error on logging failure", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := logAuditEvent(tmpDir, func(l *audit.Logger) error {
			return fmt.Errorf("test logging error")
		})

		if err == nil {
			t.Error("logAuditEvent should return error when logFunc returns error")
		}
		if !strings.Contains(err.Error(), "test logging error") {
			t.Errorf("Error should contain original error message, got: %v", err)
		}
	})

	t.Run("logAuditEvent succeeds with valid logger and logFunc", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := logAuditEvent(tmpDir, func(l *audit.Logger) error {
			return l.LogSetup("test-provider")
		})

		if err != nil {
			t.Errorf("logAuditEvent should succeed with valid input, got: %v", err)
		}
	})
}
