package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestProviderListConstant(t *testing.T) {
	providerList := providers.ProviderList()

	if len(providerList) < 4 {
		t.Errorf("providerList has %d entries, want at least 4", len(providerList))
	}

	for _, p := range providerList {
		if !providers.IsBuiltInProvider(p) {
			t.Errorf("providerList contains %q which is not a built-in provider", p)
		}
	}
}

func TestProviderEnvVarSetup(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantEnvCount int
	}{
		{"zai has env vars", "zai", 1},
		{"minimax has env vars", "minimax", 2},
		{"kimi has env vars", "kimi", 2},
		{"deepseek has env vars", "deepseek", 2},
		{"custom has no env vars", "custom", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := providers.BuiltInProvider(tt.provider)
			if !ok {
				t.Fatalf("GetBuiltInProvider(%q) failed", tt.provider)
			}

			if tt.wantEnvCount > 0 && len(def.EnvVars) == 0 {
				t.Errorf("Provider %q has 0 env vars, want at least %d", tt.provider, tt.wantEnvCount)
			}

			if tt.wantEnvCount == 0 && len(def.EnvVars) > 0 {
				t.Errorf("Provider %q has %d env vars, want 0", tt.provider, len(def.EnvVars))
			}
		})
	}
}

func TestSwitchCmdProviderNotFound(t *testing.T) {
	originalConfigDir := configDir()
	defer setConfigDir(originalConfigDir)

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"minimax": {Name: "MiniMax", BaseURL: "https://api.minimax.io", Model: "test"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "MINIMAX_API_KEY=test-key\n"); err != nil {
		t.Fatal(err)
	}

	dir := configDir()
	if dir != tmpDir {
		t.Errorf("configDir() = %q, want %q", dir, tmpDir)
	}
}

func TestCustomProviderKeyFormat(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"myprovider": {Name: "My Provider", BaseURL: "https://api.myprovider.com", Model: "model-1"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	customName := "myprovider"
	apiKey := "sk-test-key-12345"
	secrets := map[string]string{
		fmt.Sprintf("%s_API_KEY", customName): apiKey,
	}

	var secretsBuilder strings.Builder
	for key, value := range secrets {
		if key != "" && value != "" {
			secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}

	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsBuilder.String()); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
	}

	expectedKey := fmt.Sprintf("%s_API_KEY=", customName)
	if !strings.Contains(decrypted, expectedKey) {
		t.Errorf("Decrypted secrets should contain %q, got: %q", expectedKey, decrypted)
	}

	if !strings.Contains(decrypted, "myprovider_API_KEY=sk-test-key-12345") {
		t.Errorf("Decrypted secrets should contain 'myprovider_API_KEY=sk-test-key-12345', got: %q", decrypted)
	}

	for _, line := range strings.Split(decrypted, "\n") {
		if strings.HasPrefix(line, expectedKey) {
			if strings.HasPrefix(line, "CUSTOM_") {
				t.Errorf("Custom provider key should NOT have CUSTOM_ prefix, got: %q", line)
			}
			return
		}
	}

	t.Errorf("Expected key %q not found in decrypted secrets", expectedKey)
}

func TestCustomProviderKeyLookupInSwitch(t *testing.T) {
	tmpDir := t.TempDir()

	providerName := "mycustomprovider"
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			providerName: {Name: "My Custom Provider", BaseURL: "https://api.example.com", Model: "test"},
		},
		DefaultProvider: providerName,
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	apiKey := "sk-custom-key-abcdef"
	secretsContent := fmt.Sprintf("%s_API_KEY=%s\n", providerName, apiKey)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
	}

	prefix := fmt.Sprintf("%s_API_KEY=", providerName)
	if !strings.HasPrefix(decrypted, prefix) {
		t.Errorf("Secrets should start with %q, got: %q", prefix, decrypted)
	}

	for _, line := range strings.Split(decrypted, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				t.Errorf("Expected key=value format, got: %q", line)
				continue
			}
			if parts[1] != apiKey {
				t.Errorf("API key = %q, want %q", parts[1], apiKey)
			}
			if strings.HasPrefix(line, "CUSTOM_") {
				t.Errorf("Key should NOT have CUSTOM_ prefix for custom provider")
			}
			return
		}
	}

	t.Errorf("Expected to find %q in secrets", prefix)
}
