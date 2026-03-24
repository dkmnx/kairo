package integration

import (
	"context"

	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestFullWorkflowSetupConfigAndSwitch(t *testing.T) {
	if testBinary == "" {
		t.Fatal("testBinary not initialized, TestMain may have failed")
	}

	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "anthropic",
		DefaultHarness:  "claude",
		Providers: map[string]config.Provider{
			"anthropic": {
				Name:    "Native Anthropic",
				BaseURL: "",
				Model:   "",
			},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Verify initial state
	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loadedCfg.DefaultProvider != "anthropic" {
		t.Errorf("initial default provider = %q, want 'anthropic'", loadedCfg.DefaultProvider)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secretsContent := "ZAI_API_KEY=TEST-KEY-DO-NOT-USE-zai-001\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	def, _ := providers.GetBuiltInProvider("zai")
	cfg.Providers["zai"] = config.Provider{
		Name:    def.Name,
		BaseURL: def.BaseURL,
		Model:   def.Model,
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config with zai: %v", err)
	}

	// Verify zai provider configured
	loadedCfg, err = config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after zai: %v", err)
	}
	if _, exists := loadedCfg.Providers["zai"]; !exists {
		t.Error("zai provider should exist")
	}

	secretsContent += "MINIMAX_API_KEY=TEST-KEY-DO-NOT-USE-minimax-002\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets with minimax: %v", err)
	}

	def, _ = providers.GetBuiltInProvider("minimax")
	cfg.Providers["minimax"] = config.Provider{
		Name:    def.Name,
		BaseURL: def.BaseURL,
		Model:   def.Model,
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config with minimax: %v", err)
	}

	// Verify both providers exist
	loadedCfg, err = config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after minimax: %v", err)
	}
	if len(loadedCfg.Providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(loadedCfg.Providers))
	}

	cfg.DefaultProvider = "zai"
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to update default provider: %v", err)
	}

	loadedCfg, err = config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after default change: %v", err)
	}
	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("default provider = %q, want 'zai'", loadedCfg.DefaultProvider)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt secrets: %v", err)
	}
	if !strings.Contains(decrypted, "ZAI_API_KEY") {
		t.Error("secrets should contain ZAI_API_KEY")
	}
	if !strings.Contains(decrypted, "MINIMAX_API_KEY") {
		t.Error("secrets should contain MINIMAX_API_KEY")
	}
}

func TestFullWorkflowHarnessSwitching(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup config with default harness
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		DefaultHarness:  "claude",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"zai":       {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Verify initial harness
	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loadedCfg.DefaultHarness != "claude" {
		t.Errorf("initial harness = %q, want 'claude'", loadedCfg.DefaultHarness)
	}

	// Switch to qwen harness
	cfg.DefaultHarness = "qwen"
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to update harness: %v", err)
	}

	loadedCfg, err = config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after harness change: %v", err)
	}
	if loadedCfg.DefaultHarness != "qwen" {
		t.Errorf("harness = %q, want 'qwen'", loadedCfg.DefaultHarness)
	}

	// Switch back to claude
	cfg.DefaultHarness = "claude"
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to switch back to claude: %v", err)
	}

	loadedCfg, err = config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after switching back: %v", err)
	}
	if loadedCfg.DefaultHarness != "claude" {
		t.Errorf("harness = %q, want 'claude'", loadedCfg.DefaultHarness)
	}
}

func TestFullWorkflowListAndStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup multiple providers with varying configuration states
	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"zai":       {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7"},
			"minimax":   {Name: "MiniMax", BaseURL: "https://api.minimax.chat/v1", Model: "minimax-abab6.5"},
			"deepseek":  {Name: "DeepSeek", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Only configure secrets for some providers
	secretsContent := `ZAI_API_KEY=TEST-KEY-DO-NOT-USE-list-zai
DEEPSEEK_API_KEY=TEST-KEY-DO-NOT-USE-list-deepseek
`
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Verify config
	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check all providers are listed
	expectedProviders := []string{"anthropic", "zai", "minimax", "deepseek"}
	for _, provider := range expectedProviders {
		if _, exists := loadedCfg.Providers[provider]; !exists {
			t.Errorf("provider %q should be listed", provider)
		}
	}

	// Verify default provider
	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("default provider = %q, want 'zai'", loadedCfg.DefaultProvider)
	}

	// Verify secrets
	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt secrets: %v", err)
	}

	// Check which providers have API keys
	parsedSecrets := config.ParseSecrets(decrypted)
	if _, exists := parsedSecrets["ZAI_API_KEY"]; !exists {
		t.Error("ZAI_API_KEY should exist")
	}
	if _, exists := parsedSecrets["DEEPSEEK_API_KEY"]; !exists {
		t.Error("DEEPSEEK_API_KEY should exist")
	}
	if _, exists := parsedSecrets["MINIMAX_API_KEY"]; exists {
		t.Error("MINIMAX_API_KEY should not exist")
	}
}

func TestFullWorkflowKeyPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup initial config
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Read the key to verify persistence
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key: %v", err)
	}

	// Verify key file exists and has content
	if len(keyData) == 0 {
		t.Error("key file should not be empty")
	}

	// Verify key format (should contain age secret key prefix)
	if !strings.Contains(string(keyData), "AGE-SECRET-KEY-") {
		t.Error("key should be in age format with AGE-SECRET-KEY prefix")
	}
}

func TestFullWorkflowCustomProvider(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup with a custom provider
	customProviderName := "mycustom"
	cfg := &config.Config{
		DefaultProvider: customProviderName,
		Providers: map[string]config.Provider{
			customProviderName: {
				Name:    "My Custom Provider",
				BaseURL: "https://api.custom.example.com/v1",
				Model:   "custom-model-v1",
				EnvVars: []string{"CUSTOM_API_VERSION=v1"},
			},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	secretsContent := "MYCUSTOM_API_KEY=TEST-KEY-DO-NOT-USE-custom-provider\n"
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Verify custom provider
	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	customProvider, exists := loadedCfg.Providers[customProviderName]
	if !exists {
		t.Fatal("custom provider should exist")
	}

	if customProvider.Name != "My Custom Provider" {
		t.Errorf("custom provider name = %q, want 'My Custom Provider'", customProvider.Name)
	}
	if customProvider.BaseURL != "https://api.custom.example.com/v1" {
		t.Errorf("custom provider baseURL = %q, want 'https://api.custom.example.com/v1'", customProvider.BaseURL)
	}
	if customProvider.Model != "custom-model-v1" {
		t.Errorf("custom provider model = %q, want 'custom-model-v1'", customProvider.Model)
	}

	// Verify secrets
	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt secrets: %v", err)
	}
	if !strings.Contains(decrypted, "MYCUSTOM_API_KEY") {
		t.Error("secrets should contain MYCUSTOM_API_KEY")
	}
	if !strings.Contains(decrypted, "TEST-KEY-DO-NOT-USE-custom-provider") {
		t.Error("secrets should contain the custom API key")
	}
}

func TestFullWorkflowHarnessCLIExecution(t *testing.T) {
	if testBinary == "" {
		t.Fatal("testBinary not initialized, TestMain may have failed")
	}

	tmpDir := t.TempDir()

	// Setup config with qwen as default harness
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		DefaultHarness:  "qwen",
		Providers: map[string]config.Provider{
			"anthropic": {
				Name:    "Native Anthropic",
				BaseURL: "https://api.anthropic.com/v1",
				Model:   "claude-3-5-sonnet",
			},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	claudeCmd := exec.Command(testBinary, "--config", tmpDir, "switch", "--harness", "claude", "anthropic", "--help")
	claudeOutput, _ := claudeCmd.CombinedOutput()
	if strings.Contains(string(claudeOutput), "unknown flag") {
		t.Errorf("harness flag not recognized for claude: %s", string(claudeOutput))
	}

	qwenCmd := exec.Command(testBinary, "--config", tmpDir, "switch", "--harness", "qwen", "anthropic", "--help")
	qwenOutput, _ := qwenCmd.CombinedOutput()
	if strings.Contains(string(qwenOutput), "unknown flag") {
		t.Errorf("harness flag not recognized for qwen: %s", string(qwenOutput))
	}
}
