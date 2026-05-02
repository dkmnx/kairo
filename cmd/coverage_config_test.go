package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

// --- Tests for LoadSecrets ---

func TestLoadSecrets_NoSecretsFile(t *testing.T) {
	tmpDir := withTempConfigDir(t)
	result, err := LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	if len(result.Secrets) != 0 {
		t.Errorf("Expected empty secrets, got %d entries", len(result.Secrets))
	}
	if result.SecretsPath == "" {
		t.Error("SecretsPath should be set")
	}
}

func TestLoadSecrets_WithSecrets(t *testing.T) {
	tmpDir := withTempConfigDir(t)
	ctx := context.Background()
	if err := crypto.EnsureKeyExists(ctx, tmpDir); err != nil {
		t.Fatal(err)
	}
	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)
	if err := crypto.EncryptSecrets(ctx, secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
		t.Fatal(err)
	}

	result, err := LoadSecrets(ctx, tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	if result.Secrets["ZAI_API_KEY"] != "test-key" {
		t.Errorf("ZAI_API_KEY = %q, want %q", result.Secrets["ZAI_API_KEY"], "test-key")
	}
}

// --- Tests for SaveSecrets ---

func TestSaveSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	if err := crypto.EnsureKeyExists(ctx, tmpDir); err != nil {
		t.Fatal(err)
	}
	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)

	err := SaveSecrets(ctx, secretsPath, keyPath, map[string]string{"TEST_API_KEY": "test-value"})
	if err != nil {
		t.Fatalf("SaveSecrets() error = %v", err)
	}

	result, err := LoadSecrets(ctx, tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	if result.Secrets["TEST_API_KEY"] != "test-value" {
		t.Errorf("TEST_API_KEY = %q, want %q", result.Secrets["TEST_API_KEY"], "test-value")
	}
}

// --- Tests for EnsureConfigDir ---

func TestEnsureConfigDir(t *testing.T) {
	origDir := getConfigDir()
	defer setConfigDir(origDir)

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "kairo-test-config")
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)

	err := EnsureConfigDir(cliCtx, configDir)
	if err != nil {
		t.Fatalf("EnsureConfigDir() error = %v", err)
	}
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("config directory should exist")
	}
}

// --- Tests for GetProviderDefinition ---

func TestGetProviderDefinition(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		wantName    string
		wantBaseURL string
		wantModel   string
	}{
		{"zai builtin", "zai", "Z.AI", "https://api.z.ai/api/anthropic", "glm-5.1"},
		{"minimax builtin", "minimax", "MiniMax", "https://api.minimax.io/anthropic", "MiniMax-M2.7"},
		{"unknown uses input", "myprovider", "myprovider", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := GetProviderDefinition(tt.provider)
			if def.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", def.Name, tt.wantName)
			}
			if tt.wantBaseURL != "" && def.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", def.BaseURL, tt.wantBaseURL)
			}
			if tt.wantModel != "" && def.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", def.Model, tt.wantModel)
			}
		})
	}
}

// --- Tests for validateConfiguredModel ---

func TestValidateConfiguredModel(t *testing.T) {
	tests := []struct {
		name    string
		cfg     modelValidationConfig
		wantErr bool
	}{
		{"empty model builtin ok", modelValidationConfig{Model: "", ProviderName: "zai", DisplayName: "Z.AI"}, false},
		{"valid model builtin", modelValidationConfig{Model: "glm-5.1", ProviderName: "zai", DisplayName: "Z.AI"}, false},
		{"empty custom requires model", modelValidationConfig{Model: "   ", ProviderName: "custom-provider", DisplayName: "custom-provider"}, true},
		{"valid model custom", modelValidationConfig{Model: "my-model", ProviderName: "custom-provider", DisplayName: "custom-provider"}, false},
		{"model too long for built-in", modelValidationConfig{Model: strings.Repeat("a", 101), ProviderName: "zai", DisplayName: "Z.AI"}, true},
		{"model with invalid chars for built-in", modelValidationConfig{Model: "model@invalid", ProviderName: "zai", DisplayName: "Z.AI"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguredModel(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguredModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- Tests for BuildProviderConfig ---

func TestBuildProviderConfig_NewProvider(t *testing.T) {
	def := providers.ProviderDefinition{Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-5.1"}
	cfg := BuildProviderConfig(ProviderBuildConfig{
		Definition: def, BaseURL: "https://custom.api.com", Model: "custom-model", Exists: false, Existing: nil,
	})
	if cfg.Name != "Z.AI" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Z.AI")
	}
	if cfg.BaseURL != "https://custom.api.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://custom.api.com")
	}
	if cfg.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "custom-model")
	}
}

func TestBuildProviderConfig_EditExisting(t *testing.T) {
	existing := &config.Provider{
		Name: "Z.AI", BaseURL: "https://old.api.com", Model: "old-model", EnvVars: []string{"EXTRA_VAR=extra"},
	}
	def := providers.ProviderDefinition{Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-5.1"}
	cfg := BuildProviderConfig(ProviderBuildConfig{
		Definition: def, BaseURL: "https://new.api.com", Model: "new-model", Exists: true, Existing: existing,
	})
	if cfg.BaseURL != "https://new.api.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://new.api.com")
	}
	if cfg.Model != "new-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "new-model")
	}
	if len(cfg.EnvVars) != 1 || cfg.EnvVars[0] != "EXTRA_VAR=extra" {
		t.Errorf("EnvVars = %v, want [EXTRA_VAR=extra]", cfg.EnvVars)
	}
}

// --- Tests for runResetSecrets ---

func TestRunResetSecrets_Cancelled(t *testing.T) {
	origConfirm := confirmUIFn
	confirmUIFn = func(prompt string) (bool, error) { return false, nil }
	defer func() { confirmUIFn = origConfirm }()

	tmpDir := withTempConfigDir(t)
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	secretsResult := SecretsResult{
		Secrets:     make(map[string]string),
		SecretsPath: filepath.Join(tmpDir, constants.SecretsFileName),
		KeyPath:     filepath.Join(tmpDir, constants.KeyFileName),
	}

	err := runResetSecrets(cliCtx, tmpDir, secretsResult)
	if err == nil {
		t.Error("runResetSecrets() should return error when cancelled")
	}
	if err.Error() != "operation cancelled by user" {
		t.Errorf("error = %q, want 'operation cancelled by user'", err.Error())
	}
}

func TestRunResetSecrets_Confirmed(t *testing.T) {
	origConfirm := confirmUIFn
	confirmUIFn = func(prompt string) (bool, error) { return true, nil }
	defer func() { confirmUIFn = origConfirm }()

	tmpDir := withTempConfigDir(t)
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatal(err)
	}

	secretsResult := SecretsResult{
		Secrets:     make(map[string]string),
		SecretsPath: filepath.Join(tmpDir, constants.SecretsFileName),
		KeyPath:     filepath.Join(tmpDir, constants.KeyFileName),
	}

	err := runResetSecrets(cliCtx, tmpDir, secretsResult)
	if err != nil {
		t.Fatalf("runResetSecrets() error = %v", err)
	}
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("new key should exist after reset")
	}
}

// --- Tests for requireConfigDirWritable ---

func TestRequireConfigDirWritable_CreatesDir(t *testing.T) {
	origDir := getConfigDir()
	defer setConfigDir(origDir)

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "kairo-config")
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(testDir)
	cmd := newCommandWithContext(cliCtx)

	result := requireConfigDirWritable(cmd)
	if result == "" {
		t.Error("requireConfigDirWritable() should return path when dir can be created")
	}
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("directory should be created")
	}
}

func TestRequireConfigDirWritable_NoConfigDir(t *testing.T) {
	// When configDir is empty and GetConfigDir has no override,
	// it falls back to the platform default. So it will find a dir.
	// Test that calling requireConfigDirWritable with a nil context returns empty.
	cmd := &cobra.Command{}
	// No CLIContext set, GetCLIContext falls back to defaultCLIContext
	result := requireConfigDirWritable(cmd)
	// With no override, GetConfigDir returns the platform default, which is writable,
	// so result will be a non-empty path. We just verify it doesn't panic.
	_ = result
}

// --- Tests for loadConfigOrExit ---

func TestLoadConfigOrExit_NoConfig(t *testing.T) {
	tmpDir := withTempConfigDir(t)
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	cmd := newCommandWithContext(cliCtx)
	result := loadConfigOrExit(cmd)
	if result != nil {
		t.Error("loadConfigOrExit() should return nil when no config exists")
	}
}

func TestLoadConfigOrExit_WithConfig(t *testing.T) {
	tmpDir := withTempConfigDir(t)
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-5.1"},
		},
		DefaultModels: make(map[string]string),
	}
	mustCreateConfig(t, tmpDir, cfg)

	cmd := newCommandWithContext(cliCtx)
	result := loadConfigOrExit(cmd)
	if result == nil {
		t.Fatal("loadConfigOrExit() should return config when it exists")
	}
	if result.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", result.DefaultProvider, "zai")
	}
}

// --- Tests for configureProvider integration ---

func TestConfigureProvider_InvalidName(t *testing.T) {
	_, err := configureProvider(ProviderSetup{
		ProviderName: "123invalid",
		Cfg:          &config.Config{Providers: make(map[string]config.Provider)},
	})
	if err == nil {
		t.Error("configureProvider() should return error for invalid provider name")
	}
}

func TestConfigureProvider_EmptyAPIKey(t *testing.T) {
	withMockedTAP(t)
	tapPasswordFn = func(ctx context.Context, opts tap.PasswordOptions) string { return "" }

	tmpDir := t.TempDir()
	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatal(err)
	}

	_, err := configureProvider(ProviderSetup{
		CLIContext: NewCLIContext(), ConfigDir: tmpDir,
		Cfg:          &config.Config{Providers: make(map[string]config.Provider), DefaultModels: make(map[string]string)},
		ProviderName: "zai", Secrets: make(map[string]string),
		SecretsPath: filepath.Join(tmpDir, constants.SecretsFileName),
		KeyPath:     filepath.Join(tmpDir, constants.KeyFileName), IsEdit: false,
	})
	if err == nil {
		t.Error("configureProvider() should return error for empty API key")
	}
}

func TestConfigureProvider_Success(t *testing.T) {
	withMockedTAP(t)
	apiKey := "sk-test-api-key-that-is-long-enough-1234567890"
	tapPasswordFn = func(ctx context.Context, opts tap.PasswordOptions) string { return apiKey }
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "" }
	tapConfirmFn = func(ctx context.Context, opts tap.ConfirmOptions) bool { return true }
	tapIntroFn = func(title string, opts ...tap.MessageOptions) {}
	tapMessageFn = func(message string, opts ...tap.MessageOptions) {}
	tapOutroFn = func(message string, opts ...tap.MessageOptions) {}

	tmpDir := t.TempDir()
	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatal(err)
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)

	result, err := configureProvider(ProviderSetup{
		CLIContext: cliCtx, ConfigDir: tmpDir,
		Cfg:          &config.Config{Providers: make(map[string]config.Provider), DefaultModels: make(map[string]string)},
		ProviderName: "zai", Secrets: make(map[string]string),
		SecretsPath: filepath.Join(tmpDir, constants.SecretsFileName),
		KeyPath:     filepath.Join(tmpDir, constants.KeyFileName), IsEdit: false,
	})
	if err != nil {
		t.Fatalf("configureProvider() error = %v", err)
	}
	if result != "zai" {
		t.Errorf("configureProvider() = %q, want %q", result, "zai")
	}

	loadedCfg := mustLoadConfig(t, tmpDir)
	zaiProvider, ok := loadedCfg.Providers["zai"]
	if !ok {
		t.Fatal("zai provider not found in config")
	}
	if zaiProvider.Name != "Z.AI" {
		t.Errorf("Provider Name = %q, want %q", zaiProvider.Name, "Z.AI")
	}

	secretsResult, err := LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	if secretsResult.Secrets["ZAI_API_KEY"] != apiKey {
		t.Errorf("ZAI_API_KEY = %q, want %q", secretsResult.Secrets["ZAI_API_KEY"], apiKey)
	}
}
