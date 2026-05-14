package cmd

import (
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestPromptFieldConfig(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		cfg := promptFieldConfig{
			Label:        "Test Label",
			CurrentValue: "current",
			DefaultValue: "default",
			IsEdit:       true,
			Exists:       true,
		}

		if cfg.Label != "Test Label" {
			t.Errorf("Label = %q, want 'Test Label'", cfg.Label)
		}
		if cfg.CurrentValue != "current" {
			t.Errorf("CurrentValue = %q, want 'current'", cfg.CurrentValue)
		}
		if cfg.DefaultValue != "default" {
			t.Errorf("DefaultValue = %q, want 'default'", cfg.DefaultValue)
		}
		if !cfg.IsEdit {
			t.Error("IsEdit should be true")
		}
		if !cfg.Exists {
			t.Error("Exists should be true")
		}
	})
}

func TestDisplayProviderHeader_NoPanic(t *testing.T) {
	tests := []struct {
		name   string
		config providerPromptConfig
	}{
		{
			name: "edit mode existing provider",
			config: providerPromptConfig{
				ProviderName: "Test Provider",
				Provider: config.Provider{
					Name:    "Test Provider",
					BaseURL: "https://test.com",
					Model:   "test-model",
				},
				IsEdit: true,
				Exists: true,
			},
		},
		{
			name: "new provider mode",
			config: providerPromptConfig{
				ProviderName: "New Provider",
				Provider: config.Provider{
					Name:    "New Provider",
					BaseURL: "https://new.com",
					Model:   "new-model",
				},
				IsEdit: false,
				Exists: false,
			},
		},
		{
			name: "empty provider",
			config: providerPromptConfig{
				Provider: config.Provider{},
				IsEdit:   true,
				Exists:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			displayProviderHeader(tt.config)
		})
	}
}
