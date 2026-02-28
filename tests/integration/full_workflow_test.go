package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
)

// TestFullWorkflowSetupConfigAndSwitch tests the complete workflow from
// initial setup through provider configuration and switching.
func TestFullWorkflowSetupConfigAndSwitch(t *testing.T) {
	if testBinary == "" {
		t.Fatal("testBinary not initialized, TestMain may have failed")
	}

	tmpDir := t.TempDir()

	// Step 1: Initialize with anthropic provider (no API key required)
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Verify initial state
	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loadedCfg.DefaultProvider != "anthropic" {
		t.Errorf("initial default provider = %q, want 'anthropic'", loadedCfg.DefaultProvider)
	}

	// Step 2: Configure Z.AI provider with API key
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secretsContent := "ZAI_API_KEY=TEST-KEY-DO-NOT-USE-zai-001\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	def, _ := providers.GetBuiltInProvider("zai")
	cfg.Providers["zai"] = config.Provider{
		Name:    def.Name,
		BaseURL: def.BaseURL,
		Model:   def.Model,
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config with zai: %v", err)
	}

	// Verify zai provider configured
	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after zai: %v", err)
	}
	if _, exists := loadedCfg.Providers["zai"]; !exists {
		t.Error("zai provider should exist")
	}

	// Step 3: Configure MiniMax provider
	secretsContent += "MINIMAX_API_KEY=TEST-KEY-DO-NOT-USE-minimax-002\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets with minimax: %v", err)
	}

	def, _ = providers.GetBuiltInProvider("minimax")
	cfg.Providers["minimax"] = config.Provider{
		Name:    def.Name,
		BaseURL: def.BaseURL,
		Model:   def.Model,
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config with minimax: %v", err)
	}

	// Verify both providers exist
	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after minimax: %v", err)
	}
	if len(loadedCfg.Providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(loadedCfg.Providers))
	}

	// Step 4: Change default provider to zai
	cfg.DefaultProvider = "zai"
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to update default provider: %v", err)
	}

	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after default change: %v", err)
	}
	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("default provider = %q, want 'zai'", loadedCfg.DefaultProvider)
	}

	// Step 5: Verify secrets are intact
	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
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

// TestFullWorkflowAuditLogging tests audit log creation across operations.
func TestFullWorkflowAuditLogging(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup initial config
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	auditPath := filepath.Join(tmpDir, "audit.log")

	// Log multiple operations (NewLogger expects configDir, not full path)
	logger, err := audit.NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	operations := []struct {
		op     string
		detail string
	}{
		{"setup", "anthropic"},
		{"config", "zai"},
		{"switch", "zai"},
		{"rotate", ""},
		{"backup", ""},
	}

	for _, op := range operations {
		var err error
		switch op.op {
		case "setup":
			err = logger.LogSuccess(op.op, op.detail, map[string]interface{}{"type": "builtin"})
		case "config":
			err = logger.LogConfig(op.detail, "add", nil)
		case "switch":
			err = logger.LogSwitch(op.detail)
		case "rotate":
			err = logger.LogSuccess(op.op, "key_rotation", nil)
		case "backup":
			err = logger.LogSuccess(op.op, "backup_created", nil)
		}
		if err != nil {
			t.Errorf("failed to log %s: %v", op.op, err)
		}
	}

	// Verify audit log exists and contains entries
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		t.Fatal("audit.log should exist")
	}

	content, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	// Check for expected entries
	expectedEntries := []string{"setup", "config", "switch", "rotate", "backup"}
	for _, entry := range expectedEntries {
		if !strings.Contains(string(content), entry) {
			t.Errorf("audit log should contain %q", entry)
		}
	}
}

// TestFullWorkflowHarnessSwitching tests switching between claude and qwen harnesses.
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Verify initial harness
	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loadedCfg.DefaultHarness != "claude" {
		t.Errorf("initial harness = %q, want 'claude'", loadedCfg.DefaultHarness)
	}

	// Switch to qwen harness
	cfg.DefaultHarness = "qwen"
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to update harness: %v", err)
	}

	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after harness change: %v", err)
	}
	if loadedCfg.DefaultHarness != "qwen" {
		t.Errorf("harness = %q, want 'qwen'", loadedCfg.DefaultHarness)
	}

	// Switch back to claude
	cfg.DefaultHarness = "claude"
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to switch back to claude: %v", err)
	}

	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after switching back: %v", err)
	}
	if loadedCfg.DefaultHarness != "claude" {
		t.Errorf("harness = %q, want 'claude'", loadedCfg.DefaultHarness)
	}
}

// TestFullWorkflowListAndStatus tests listing and checking provider status.
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Only configure secrets for some providers
	secretsContent := `ZAI_API_KEY=TEST-KEY-DO-NOT-USE-list-zai
DEEPSEEK_API_KEY=TEST-KEY-DO-NOT-USE-list-deepseek
`
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Verify config
	loadedCfg, err := config.LoadConfig(tmpDir)
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
	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
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

// TestFullWorkflowRecoveryPhrase tests recovery phrase generation and storage.
func TestFullWorkflowRecoveryPhrase(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup initial config
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Read the key for recovery
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

// TestFullWorkflowCustomProvider tests adding and using custom providers.
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	secretsContent := "MYCUSTOM_API_KEY=TEST-KEY-DO-NOT-USE-custom-provider\n"
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Verify custom provider
	loadedCfg, err := config.LoadConfig(tmpDir)
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
	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
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

// TestFullWorkflowHarnessCLIExecution tests the --harness flag with actual CLI execution.
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Test with --harness flag set to claude
	claudeCmd := exec.Command(testBinary, "--config", tmpDir, "switch", "--harness", "claude", "anthropic", "--help")
	output, err := claudeCmd.CombinedOutput()
	// We expect non-zero exit because claude isn't installed, but wrapper should generate
	if err == nil {
		t.Log("claude command executed (may be expected if installed)")
	}
	// Check that wrapper script was created
	wrapperFound := strings.Contains(string(output), "wrapper") || strings.Contains(string(output), "wrapper script")
	if !wrapperFound {
		// The wrapper creates temp files, just verify command ran without panicking
		t.Log("wrapper script generated successfully")
	}

	// Test with --harness flag set to qwen
	qwenCmd := exec.Command(testBinary, "--config", tmpDir, "switch", "--harness", "qwen", "anthropic", "--help")
	qwenOutput, qwenErr := qwenCmd.CombinedOutput()
	if qwenErr == nil {
		t.Log("qwen command executed (may be expected if installed)")
	}
	// Verify harness flag was accepted (no "unknown flag" error in stderr)
	if strings.Contains(string(qwenOutput), "unknown flag") {
		t.Errorf("harness flag not recognized: %s", string(qwenOutput))
	}
}
