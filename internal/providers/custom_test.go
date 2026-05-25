package providers

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCustomProviderDefinition_ToProviderDefinition(t *testing.T) {
	c := CustomProviderDefinition{
		Name:           "my-provider",
		BaseURL:        "https://api.example.com",
		Model:          "test-model",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "MY_PROVIDER_API_KEY",
		MinKeyLength:   32,
		KeyPrefix:      "sk-",
		EnvVars:        []string{"EXTRA_VAR=value"},
	}

	d := c.ToProviderDefinition()
	if d.Name != "my-provider" {
		t.Errorf("Name = %q, want my-provider", d.Name)
	}
	if d.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q", d.BaseURL)
	}
	if d.KeyFormat.MinLength != 32 {
		t.Errorf("MinLength = %d, want 32", d.KeyFormat.MinLength)
	}
	if d.KeyFormat.Prefix != "sk-" {
		t.Errorf("Prefix = %q, want sk-", d.KeyFormat.Prefix)
	}
}

func TestCustomProviderDefinition_DefaultsKeyFormatMinLength(t *testing.T) {
	c := CustomProviderDefinition{
		Name: "no-key-format",
	}
	d := c.ToProviderDefinition()
	if d.KeyFormat.MinLength != DefaultMinKeyLength {
		t.Errorf("MinLength = %d, want %d", d.KeyFormat.MinLength, DefaultMinKeyLength)
	}
}

func TestCustomProviderDefinition_YAMLUnmarshal(t *testing.T) {
	yamlData := `
name: test-provider
base_url: https://api.test.com
model: gpt-4
requires_api_key: true
api_key_env_var: TEST_API_KEY
min_key_length: 32
key_prefix: sk-
env_vars:
  - EXTRA=value
`
	var c CustomProviderDefinition
	if err := yaml.Unmarshal([]byte(yamlData), &c); err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}
	if c.Name != "test-provider" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.BaseURL != "https://api.test.com" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.MinKeyLength != 32 {
		t.Errorf("MinKeyLength = %d", c.MinKeyLength)
	}
	if len(c.EnvVars) != 1 || c.EnvVars[0] != "EXTRA=value" {
		t.Errorf("EnvVars = %v", c.EnvVars)
	}
}

func TestCustomProviderDefinition_UnknownFieldsRejected(t *testing.T) {
	yamlData := `
name: test
unknown_field: bad
`
	var c CustomProviderDefinition
	dec := yaml.NewDecoder(strings.NewReader(yamlData))
	dec.KnownFields(true)
	err := dec.Decode(&c)
	if err == nil {
		t.Error("expected error for unknown field")
	}
}
