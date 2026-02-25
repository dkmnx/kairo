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
	secretsContent := "ZAI_API_KEY=sk-zai-integration-test-key\n"
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
	secretsContent += "MINIMAX_API_KEY=sk-minimax-integration-test-key\n"
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

// TestFullWorkflowKeyRotation tests key rotation with multiple configured providers.
func TestFullWorkflowKeyRotation(t *testing.T) {
	if testBinary == "" {
		t.Fatal("testBinary not initialized, TestMain may have failed")
	}

	tmpDir := t.TempDir()

	// Setup multiple providers
	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"zai":       {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7"},
			"minimax":   {Name: "MiniMax", BaseURL: "https://api.minimax.chat/v1", Model: "minimax-abab6.5"},
			"kimi":      {Name: "Kimi", BaseURL: "https://api.moonshot.cn/v1", Model: "moonshot-v1-8k"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	secretsContent := `ZAI_API_KEY=sk-zai-key-123
MINIMAX_API_KEY=sk-minimax-key-456
KIMI_API_KEY=sk-kimi-key-789
`
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Verify pre-rotation state
	preRotationSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt pre-rotation secrets: %v", err)
	}
	if preRotationSecrets == "" {
		t.Fatal("pre-rotation secrets should not be empty")
	}

	// Perform key rotation using CLI
	rotateCmd := exec.Command(testBinary, "--config", tmpDir, "rotate")
	rotateOutput, err := rotateCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run rotate command: %v, output: %s", err, string(rotateOutput))
	}

	// Verify post-rotation state
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("age.key should exist after rotation")
	}

	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		t.Error("secrets.age should exist after rotation")
	}

	// Verify secrets can be decrypted with new key
	postRotationSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt post-rotation secrets: %v", err)
	}

	// Verify all secrets preserved
	expectedSecrets := []string{"ZAI_API_KEY", "MINIMAX_API_KEY", "KIMI_API_KEY", "sk-zai-key-123", "sk-minimax-key-456", "sk-kimi-key-789"}
	for _, secret := range expectedSecrets {
		if !strings.Contains(postRotationSecrets, secret) {
			t.Errorf("post-rotation secrets should contain %q", secret)
		}
	}

	// Verify config preserved
	rotatedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after rotation: %v", err)
	}
	if rotatedCfg.DefaultProvider != "zai" {
		t.Errorf("default provider changed after rotation: got %q, want 'zai'", rotatedCfg.DefaultProvider)
	}
	if len(rotatedCfg.Providers) != 4 {
		t.Errorf("providers changed after rotation: got %d, want 4", len(rotatedCfg.Providers))
	}
}

// TestFullWorkflowBackupRestore tests complete backup and restore cycle.
func TestFullWorkflowBackupRestore(t *testing.T) {
	if testBinary == "" {
		t.Fatal("testBinary not initialized, TestMain may have failed")
	}

	tmpDir := t.TempDir()

	// Setup configuration
	cfg := &config.Config{
		DefaultProvider: "minimax",
		DefaultHarness:  "claude",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"zai":       {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7"},
			"minimax":   {Name: "MiniMax", BaseURL: "https://api.minimax.chat/v1", Model: "minimax-abab6.5"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	secretsContent := `ZAI_API_KEY=backup-test-zai-key
MINIMAX_API_KEY=backup-test-minimax-key
`
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Create audit log (NewLogger expects configDir, not full path)
	logger, err := audit.NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}
	if err := logger.LogConfig("zai", "add", nil); err != nil {
		t.Fatalf("failed to write audit log: %v", err)
	}
	auditPath := filepath.Join(tmpDir, "audit.log")

	// Create backup
	backupCmd := exec.Command(testBinary, "--config", tmpDir, "backup")
	backupOutput, err := backupCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run backup command: %v, output: %s", err, string(backupOutput))
	}

	// Find backup file
	backupsDir := filepath.Join(tmpDir, "backups")
	backups, err := os.ReadDir(backupsDir)
	if err != nil || len(backups) == 0 {
		t.Fatalf("no backup file created: %v", err)
	}
	backupPath := filepath.Join(backupsDir, backups[0].Name())

	// Delete original files
	os.Remove(keyPath)
	os.Remove(secretsPath)
	os.Remove(filepath.Join(tmpDir, "config.yaml"))
	os.Remove(auditPath)

	// Restore from backup
	restoreCmd := exec.Command(testBinary, "--config", tmpDir, "restore", backupPath)
	restoreOutput, err := restoreCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run restore command: %v, output: %s", err, string(restoreOutput))
	}

	// Verify all files restored (note: audit.log is not included in backups)
	files := []string{"age.key", "secrets.age", "config.yaml"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("%s not restored", file)
		}
	}

	// Verify config content
	restoredCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load restored config: %v", err)
	}
	if len(restoredCfg.Providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(restoredCfg.Providers))
	}
	if restoredCfg.DefaultProvider != "minimax" {
		t.Errorf("default provider = %q, want 'minimax'", restoredCfg.DefaultProvider)
	}
	if restoredCfg.DefaultHarness != "claude" {
		t.Errorf("default harness = %q, want 'claude'", restoredCfg.DefaultHarness)
	}

	// Verify secrets
	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("failed to decrypt restored secrets: %v", err)
	}
	if !strings.Contains(decrypted, "backup-test-zai-key") {
		t.Error("restored secrets missing zai key")
	}
	if !strings.Contains(decrypted, "backup-test-minimax-key") {
		t.Error("restored secrets missing minimax key")
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

// TestFullWorkflowProviderReset tests removing providers.
func TestFullWorkflowProviderReset(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup multiple providers
	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"zai":       {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7"},
			"minimax":   {Name: "MiniMax", BaseURL: "https://api.minimax.chat/v1", Model: "minimax-abab6.5"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	secretsContent := `ZAI_API_KEY=sk-zai-key
MINIMAX_API_KEY=sk-minimax-key
`
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Verify initial state
	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load initial config: %v", err)
	}
	if len(loadedCfg.Providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(loadedCfg.Providers))
	}

	// Remove minimax provider
	delete(cfg.Providers, "minimax")
	cfg.DefaultProvider = "anthropic"
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config after removal: %v", err)
	}

	// Verify minimax removed
	loadedCfg, err = config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config after removal: %v", err)
	}
	if _, exists := loadedCfg.Providers["minimax"]; exists {
		t.Error("minimax should be removed")
	}
	if len(loadedCfg.Providers) != 2 {
		t.Errorf("expected 2 providers after removal, got %d", len(loadedCfg.Providers))
	}
	if loadedCfg.DefaultProvider != "anthropic" {
		t.Errorf("default provider = %q, want 'anthropic'", loadedCfg.DefaultProvider)
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
	secretsContent := `ZAI_API_KEY=sk-zai-key
DEEPSEEK_API_KEY=sk-deepseek-key
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

	secretsContent := "MYCUSTOM_API_KEY=sk-custom-provider-key\n"
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
	if !strings.Contains(decrypted, "sk-custom-provider-key") {
		t.Error("secrets should contain the custom API key")
	}
}
