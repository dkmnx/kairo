package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}

	provider, ok := cfg.Providers["zai"]
	if !ok {
		t.Fatal("providers['zai'] not found")
	}
	if provider.Name != "Z.AI" {
		t.Errorf("providers['zai'].Name = %q, want %q", provider.Name, "Z.AI")
	}
	if provider.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("providers['zai'].BaseURL = %q, want %q", provider.BaseURL, "https://api.z.ai/api/anthropic")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	cfg := &Config{
		DefaultProvider: "anthropic",
		Providers: map[string]Provider{
			"anthropic": {
				Name:    "Native Anthropic",
				BaseURL: "",
				Model:   "",
			},
		},
	}

	err := SaveConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("config file is empty")
	}
}

func TestProviderStruct(t *testing.T) {
	p := Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		Model:   "test-model",
	}

	if p.Name != "Test Provider" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Provider")
	}
	if p.BaseURL != "https://api.test.com" {
		t.Errorf("BaseURL = %q, want %q", p.BaseURL, "https://api.test.com")
	}
	if p.Model != "test-model" {
		t.Errorf("Model = %q, want %q", p.Model, "test-model")
	}
}

func TestParseSecrets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single key-value",
			input:    "KEY=value",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "multiple key-values",
			input:    "KEY1=value1\nKEY2=value2\nKEY3=value3",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2", "KEY3": "value3"},
		},
		{
			name:     "value with equals sign",
			input:    "KEY=a=b=c",
			expected: map[string]string{"KEY": "a=b=c"},
		},
		{
			name:     "empty lines ignored",
			input:    "\n\nKEY1=value1\n\nKEY2=value2\n\n",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:     "lines without equals ignored",
			input:    "KEY1=value1\nnoequals\nKEY2=value2",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:     "trailing newline",
			input:    "KEY=value\n",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "real world secrets format",
			input:    "ZAI_API_KEY=sk-test-key123\nMINIMAX_API_KEY=sk-another-key456\n",
			expected: map[string]string{"ZAI_API_KEY": "sk-test-key123", "MINIMAX_API_KEY": "sk-another-key456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSecrets(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseSecrets() returned %d entries, want %d", len(result), len(tt.expected))
				return
			}
			for key, value := range tt.expected {
				if result[key] != value {
					t.Errorf("ParseSecrets()[%q] = %q, want %q", key, result[key], value)
				}
			}
		})
	}
}

func TestParseSecretsEmptyKey(t *testing.T) {
	result := ParseSecrets("=value")
	if value, ok := result[""]; !ok || value != "value" {
		t.Errorf("ParseSecrets()[empty key] = %q, want %q", value, "value")
	}
}

func TestParseSecretsEmptyValue(t *testing.T) {
	result := ParseSecrets("KEY=")
	if value, ok := result["KEY"]; !ok || value != "" {
		t.Errorf("ParseSecrets()[%q] = %q, want empty string", "KEY", value)
	}
}
