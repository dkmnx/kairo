package cmd

import (
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestFullProviderConfigurationWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: make(map[string]config.Provider),
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
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

	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(loadedCfg.Providers) != len(providersToTest) {
		t.Errorf("loaded %d providers, want %d", len(loadedCfg.Providers), len(providersToTest))
	}

	secrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	parsedSecrets := config.ParseSecrets(secrets)
	for _, p := range providersToTest {
		for k := range p.envVars {
			if _, ok := parsedSecrets[k]; !ok {
				t.Errorf("secret %q not found in decrypted secrets", k)
			}
		}
	}
}

func TestKeyRotationWithMultipleProviders(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "minimax",
		Providers: map[string]config.Provider{
			"zai":     {Name: "Z.AI"},
			"minimax": {Name: "MiniMax"},
			"kimi":    {Name: "Kimi"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	secrets := `ZAI_API_KEY=zai-secret
MINIMAX_API_KEY=minimax-secret
KIMI_API_KEY=kimi-secret
`
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secrets); err != nil {
		t.Fatal(err)
	}

	if err := crypto.RotateKey(tmpDir); err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() after rotation error = %v", err)
	}

	if decrypted != secrets {
		t.Errorf("decrypted = %q, want %q", decrypted, secrets)
	}

	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() after rotation error = %v", err)
	}

	if loadedCfg.DefaultProvider != "minimax" {
		t.Errorf("DefaultProvider = %q, want %q", loadedCfg.DefaultProvider, "minimax")
	}

	if len(loadedCfg.Providers) != 3 {
		t.Errorf("loaded %d providers, want 3", len(loadedCfg.Providers))
	}
}

func TestProviderStatusAfterKeyRotation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"zai":       {Name: "Z.AI"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	secrets := "ZAI_API_KEY=zai-key\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secrets); err != nil {
		t.Fatal(err)
	}

	secretsMap, _ := crypto.DecryptSecrets(secretsPath, keyPath)
	parsedSecrets := config.ParseSecrets(secretsMap)

	if !isProviderConfiguredForTest(cfg, parsedSecrets, "zai") {
		t.Error("zai should be configured before rotation")
	}

	if !isProviderConfiguredForTest(cfg, parsedSecrets, "anthropic") {
		t.Error("anthropic should be configured before rotation")
	}

	if err := crypto.RotateKey(tmpDir); err != nil {
		t.Fatal(err)
	}

	decrypted, _ := crypto.DecryptSecrets(secretsPath, keyPath)
	rotatedSecrets := config.ParseSecrets(decrypted)

	if !isProviderConfiguredForTest(cfg, rotatedSecrets, "zai") {
		t.Error("zai should still be configured after rotation")
	}

	if !isProviderConfiguredForTest(cfg, rotatedSecrets, "anthropic") {
		t.Error("anthropic should still be configured after rotation")
	}
}

func TestCustomProviderSecretsAfterRotation(t *testing.T) {
	tmpDir := t.TempDir()

	providerName := "mycustom"
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			providerName: {Name: "My Custom", BaseURL: "https://api.custom.com", Model: "model-1"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	customKey := "MYCUSTOM_API_KEY=sk-custom-key"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, customKey+"\n"); err != nil {
		t.Fatal(err)
	}

	if err := crypto.RotateKey(tmpDir); err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	if !contains(decrypted, customKey) {
		t.Errorf("decrypted secrets should contain %q, got: %q", customKey, decrypted)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

// TestE2ESetupToSwitchWorkflow tests the complete end-to-end workflow
// from initial setup through provider switching.
func TestE2ESetupToSwitchWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up mock functions for testing
	originalConfigDir := getConfigDir()
	originalLookPath := lookPath
	originalExecCommand := execCommand
	defer func() {
		setConfigDir(originalConfigDir)
		lookPath = originalLookPath
		execCommand = originalExecCommand
	}()

	setConfigDir(tmpDir)

	// Mock lookPath to return a fake claude path
	lookPath = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/bin/claude", nil
		}
		return originalLookPath(file)
	}

	// Create initial config with anthropic provider (no API key needed)
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create key file
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	// Verify initial state: anthropic is default provider
	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loadedCfg.DefaultProvider != "anthropic" {
		t.Errorf("Default provider = %q, want 'anthropic'", loadedCfg.DefaultProvider)
	}

	// Test: Add zai provider with API key via direct config manipulation
	// (In real workflow, user would run 'kairo config zai')
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secretsContent := "ZAI_API_KEY=sk-zai-test-key-12345\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	def, _ := providers.GetBuiltInProvider("zai")
	cfg.Providers["zai"] = config.Provider{
		Name:    def.Name,
		BaseURL: def.BaseURL,
		Model:   def.Model,
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify zai provider and secrets were saved
	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error after adding zai = %v", err)
	}

	if _, exists := loadedCfg.Providers["zai"]; !exists {
		t.Error("zai provider should exist in config")
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	if !contains(decrypted, "ZAI_API_KEY") {
		t.Error("secrets should contain ZAI_API_KEY")
	}

	if !contains(decrypted, "sk-zai-test-key-12345") {
		t.Error("secrets should contain the API key")
	}

	// Test: Verify switch workflow can access both providers
	// (In real workflow, running 'kairo zai' would switch to zai provider)
	for _, provider := range []string{"anthropic", "zai"} {
		if _, exists := loadedCfg.Providers[provider]; !exists {
			t.Errorf("Provider %q should be configured", provider)
		}
	}

	// Test: Default provider switching
	cfg.DefaultProvider = "zai"
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error after changing default = %v", err)
	}

	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("Default provider = %q, want 'zai'", loadedCfg.DefaultProvider)
	}
}
