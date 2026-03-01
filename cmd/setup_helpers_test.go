package cmd

import (
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/validate"
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

		err := addAndSaveProvider(tmpDir, cfg, "testprovider", provider, true)
		if err != nil {
			t.Fatalf("addAndSaveProvider() error = %v", err)
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

	t.Run("saves provider without setting default", func(t *testing.T) {
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

		err := addAndSaveProvider(tmpDir, cfg, "testprovider", provider, false)
		if err != nil {
			t.Fatalf("addAndSaveProvider() error = %v", err)
		}

		if cfg.DefaultProvider != "" {
			t.Errorf("DefaultProvider = %q, want empty", cfg.DefaultProvider)
		}
	})

	t.Run("does not override existing default when setAsDefault is true", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			DefaultProvider: "existing",
			Providers:       make(map[string]config.Provider),
		}

		provider := config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		}

		err := addAndSaveProvider(tmpDir, cfg, "newprovider", provider, true)
		if err != nil {
			t.Fatalf("addAndSaveProvider() error = %v", err)
		}

		if cfg.DefaultProvider != "existing" {
			t.Errorf("DefaultProvider = %q, want 'existing'", cfg.DefaultProvider)
		}
	})
}

// FuzzValidateCustomProviderName fuzzes the validateCustomProviderName function with random inputs.
func FuzzValidateCustomProviderName(f *testing.F) {
	// Seed with some initial values
	f.Add("myprovider")
	f.Add("")
	f.Add("my-provider")
	f.Add("my_provider")
	f.Add("123provider")
	f.Add("my@provider")
	f.Add("zai")
	f.Add("minimax")
	f.Add("kimi")
	f.Add("deepseek")
	f.Add("anthropic")
	f.Add("openai")
	f.Add(strings.Repeat("a", 50))
	f.Add(strings.Repeat("a", 51))

	f.Fuzz(func(t *testing.T, name string) {
		result, err := validateCustomProviderName(name)

		// Verify empty names always fail
		if name == "" && err == nil {
			t.Errorf("validateCustomProviderName() should fail for empty name")
		}

		// Verify names exceeding max length always fail
		if len(name) > validate.MaxProviderNameLength && err == nil {
			t.Errorf("validateCustomProviderName() should fail for name exceeding max length (%d)", validate.MaxProviderNameLength)
		}

		// Verify names starting with numbers always fail (when not empty)
		if name != "" && len(name) > 0 && name[0] >= '0' && name[0] <= '9' && err == nil {
			t.Errorf("validateCustomProviderName() should fail for name starting with number: %s", name)
		}

		// Verify builtin provider names always fail
		if providers.IsBuiltInProvider(strings.ToLower(name)) && err == nil {
			t.Errorf("validateCustomProviderName() should fail for builtin provider name: %s", name)
		}

		// Verify valid names succeed (when not empty, not too long, starts with letter, no special chars, not builtin)
		if name != "" &&
			len(name) <= validate.MaxProviderNameLength &&
			((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) &&
			err == nil {
			// If validation passed, verify all characters are valid
			for _, r := range name {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
					t.Errorf("validateCustomProviderName() should fail for name with invalid character %q in %q", r, name)
				}
			}
		}

		// Verify successful validation returns the original name
		if err == nil && result != name {
			t.Errorf("validateCustomProviderName() should return original name on success, got %q want %q", result, name)
		}
	})
}
