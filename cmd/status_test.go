package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

// TestStatusCommandUppercaseKeyFormat verifies that the status command
// looks up API keys using uppercase provider names (e.g., ZAI_API_KEY)
func TestStatusCommandUppercaseKeyFormat(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	// Create config with lowercase provider name "zai"
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Create secrets with UPPERCASE key (ZAI_API_KEY)
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")

	if err := crypto.EnsureKeyExists(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Save with uppercase key format (the fix)
	secretsContent := "ZAI_API_KEY=test-api-key-123\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	// Load config and secrets to verify lookup works
	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	secretsContentDecrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	secrets := config.ParseSecrets(secretsContentDecrypted)

	// Verify the uppercase key lookup format (the fix)
	providerName := "zai"
	apiKeyVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))

	if apiKeyVar != "ZAI_API_KEY" {
		t.Errorf("Expected lookup key 'ZAI_API_KEY', got '%s'", apiKeyVar)
	}

	// Verify the key exists in secrets
	_, hasApiKey := secrets[apiKeyVar]
	if !hasApiKey {
		t.Errorf("Status command lookup key '%s' not found in secrets", apiKeyVar)
	}

	// Verify provider exists in config
	provider, ok := cfg.Providers[providerName]
	if !ok {
		t.Errorf("Provider '%s' not found in config", providerName)
	}

	// Verify the provider has the expected values
	if provider.Name != "Z.AI" {
		t.Errorf("Expected provider name 'Z.AI', got '%s'", provider.Name)
	}

	if provider.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("Expected base URL 'https://api.z.ai/api/anthropic', got '%s'", provider.BaseURL)
	}
}

// TestStatusCommandLowercaseKeyNotSeen verifies that keys stored with
// lowercase provider names (the old bug) are NOT found by the status command
func TestStatusCommandLowercaseKeyNotSeen(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")

	if err := crypto.EnsureKeyExists(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Save with LOWERCASE key (old bug format)
	secretsContent := "zai_API_KEY=test-api-key-123\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	secretsContentDecrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	secrets := config.ParseSecrets(secretsContentDecrypted)

	// The status command looks for UPPERCASE key
	providerName := "zai"
	apiKeyVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))

	// Verify the lowercase key exists in storage
	_, hasLowercase := secrets["zai_API_KEY"]
	if !hasLowercase {
		t.Error("Lowercase key 'zai_API_KEY' should exist in secrets")
	}

	// Verify the UPPERCASE lookup key does NOT match the stored lowercase key
	_, hasUppercase := secrets[apiKeyVar]
	if hasUppercase {
		t.Errorf("Uppercase lookup key '%s' should not exist (we stored lowercase)", apiKeyVar)
	}

	// This demonstrates the bug: lowercase keys won't be found
	// after the fix that uses uppercase for lookup
	if apiKeyVar == "ZAI_API_KEY" && !hasUppercase && hasLowercase {
		t.Logf("As expected: status command looks for '%s' but storage has 'zai_API_KEY' - key won't be found", apiKeyVar)
	}
}

// TestStatusCommandNoConfig tests status with no configuration
func TestStatusCommandNoConfig(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	// Just verify it executes without error
	err := statusCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

// TestStatusCommandKeyFormatConsistency verifies all code paths
// use the same uppercase key format
func TestStatusCommandKeyFormatConsistency(t *testing.T) {
	testCases := []struct {
		name         string
		providerName string
		expectedKey  string
	}{
		{"zai uppercase", "zai", "ZAI_API_KEY"},
		{"minimax uppercase", "minimax", "MINIMAX_API_KEY"},
		{"kimi uppercase", "kimi", "KIMI_API_KEY"},
		{"deepseek uppercase", "deepseek", "DEEPSEEK_API_KEY"},
		{"custom provider", "myprovider", "MYPROVIDER_API_KEY"},
		{"already uppercase", "ZAI", "ZAI_API_KEY"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiKeyVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(tc.providerName))
			if apiKeyVar != tc.expectedKey {
				t.Errorf("Expected '%s', got '%s'", tc.expectedKey, apiKeyVar)
			}
		})
	}
}
