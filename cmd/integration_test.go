package cmd

import (
	"context"

	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	secretspkg "github.com/dkmnx/kairo/internal/secrets"
)

func TestFullProviderConfigurationWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: make(map[string]config.Provider),
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	providersToTest := []struct {
		name    string
		envVars map[string]string
	}{
		{"zai", map[string]string{"ZAI_API_KEY": "sk-zai-key"}},
		{"minimax", map[string]string{"MINIMAX_API_KEY": "sk-minimax-key"}},
		{"deepseek", map[string]string{"DEEPSEEK_API_KEY": "sk-deepseek-key"}},
	}

	var secretsBuilder string
	for _, p := range providersToTest {
		def, _ := providers.GetBuiltInProvider(p.name)
		newProvider := config.Provider{
			Name:    def.Name,
			BaseURL: def.BaseURL,
			Model:   def.Model,
		}
		if len(def.EnvVars) > 0 {
			newProvider.EnvVars = def.EnvVars
		}
		cfg.Providers[p.name] = newProvider

		for k, v := range p.envVars {
			secretsBuilder += k + "=" + v + "\n"
		}
	}
	cfg.DefaultProvider = "zai"

	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsBuilder); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if len(loadedCfg.Providers) != len(providersToTest) {
		t.Errorf("loaded %d providers, want %d", len(loadedCfg.Providers), len(providersToTest))
	}

	decryptedContent, err := crypto.DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecretsBytes() error = %v", err)
	}
	defer crypto.ClearMemory(decryptedContent)

	parsedSecrets := secretspkg.Parse(string(decryptedContent))
	for _, p := range providersToTest {
		for k := range p.envVars {
			if _, ok := parsedSecrets[k]; !ok {
				t.Errorf("secret %q not found in decrypted secrets", k)
			}
		}
	}
}

func TestCustomProviderConfigPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	providerName := "mycustom"
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			providerName: {Name: "My Custom", BaseURL: "https://api.custom.com", Model: "model-1"},
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

	customKey := "MYCUSTOM_API_KEY=sk-custom-key"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, customKey+"\n"); err != nil {
		t.Fatal(err)
	}

	// Test config persistence without rotation
	decrypted, err := crypto.DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecretsBytes() error = %v", err)
	}
	defer crypto.ClearMemory(decrypted)

	if !strings.Contains(string(decrypted), customKey) {
		t.Errorf("decrypted secrets should contain %q, got: %q", customKey, decrypted)
	}
}

func TestE2ECompleteWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	originalConfigDir := getConfigDir()
	originalLookPath := lookPath
	originalExecCommand := execCommand
	defer func() {
		setConfigDir(originalConfigDir)
		lookPath = originalLookPath
		execCommand = originalExecCommand
	}()

	setConfigDir(tmpDir)

	lookPath = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/bin/claude", nil
		}
		return originalLookPath(file)
	}

	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {
				Name:    "Native Anthropic",
				BaseURL: "",
				Model:   "",
			},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if loadedCfg.DefaultProvider != "anthropic" {
		t.Errorf("Default provider = %q, want 'anthropic'", loadedCfg.DefaultProvider)
	}

	// (In real workflow, user would run 'kairo setup')
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secretsContent := "ZAI_API_KEY=sk-zai-test-key-12345\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	def, _ := providers.GetBuiltInProvider("zai")
	cfg.Providers["zai"] = config.Provider{
		Name:    def.Name,
		BaseURL: def.BaseURL,
		Model:   def.Model,
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err = config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error after adding zai = %v", err)
	}

	if _, exists := loadedCfg.Providers["zai"]; !exists {
		t.Error("zai provider should exist in config")
	}

	decrypted, err := crypto.DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecretsBytes() error = %v", err)
	}
	defer crypto.ClearMemory(decrypted)

	if !strings.Contains(string(decrypted), "ZAI_API_KEY") {
		t.Error("secrets should contain ZAI_API_KEY")
	}

	if !strings.Contains(string(decrypted), "sk-zai-test-key-12345") {
		t.Error("secrets should contain the API key")
	}

	// (In real workflow, running 'kairo zai' would execute with zai provider)
	for _, provider := range []string{"anthropic", "zai"} {
		if _, exists := loadedCfg.Providers[provider]; !exists {
			t.Errorf("Provider %q should be configured", provider)
		}
	}

	cfg.DefaultProvider = "zai"
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err = config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error after changing default = %v", err)
	}

	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("Default provider = %q, want 'zai'", loadedCfg.DefaultProvider)
	}
}
