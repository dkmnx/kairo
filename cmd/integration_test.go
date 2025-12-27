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

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
