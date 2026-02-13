package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestProviderDefaults(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		wantDefaultURL   bool
		wantURL          string
		wantDefaultModel bool
		wantModel        string
	}{
		{
			name:             "anthropic has no defaults",
			provider:         "anthropic",
			wantDefaultURL:   false,
			wantDefaultModel: false,
		},
		{
			name:             "zai has default URL and model",
			provider:         "zai",
			wantDefaultURL:   true,
			wantURL:          "https://api.z.ai/api/anthropic",
			wantDefaultModel: true,
			wantModel:        "glm-4.7",
		},
		{
			name:             "minimax has default URL and model",
			provider:         "minimax",
			wantDefaultURL:   true,
			wantURL:          "https://api.minimax.io/anthropic",
			wantDefaultModel: true,
			wantModel:        "MiniMax-M2.5",
		},
		{
			name:             "kimi has default URL and model",
			provider:         "kimi",
			wantDefaultURL:   true,
			wantURL:          "https://api.kimi.com/coding/",
			wantDefaultModel: true,
			wantModel:        "kimi-for-coding",
		},
		{
			name:             "deepseek has default URL and model",
			provider:         "deepseek",
			wantDefaultURL:   true,
			wantURL:          "https://api.deepseek.com/anthropic",
			wantDefaultModel: true,
			wantModel:        "deepseek-chat",
		},
		{
			name:             "custom has no defaults",
			provider:         "custom",
			wantDefaultURL:   false,
			wantDefaultModel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := providers.GetBuiltInProvider(tt.provider)
			if !ok {
				t.Fatalf("GetBuiltInProvider(%q) = false, want true", tt.provider)
			}

			if tt.wantDefaultURL {
				if def.BaseURL == "" {
					t.Errorf("GetBuiltInProvider(%q).BaseURL = empty, want %q", tt.provider, tt.wantURL)
				}
			} else {
				if def.BaseURL != "" {
					t.Errorf("GetBuiltInProvider(%q).BaseURL = %q, want empty", tt.provider, def.BaseURL)
				}
			}

			if tt.wantDefaultModel {
				if def.Model == "" {
					t.Errorf("GetBuiltInProvider(%q).Model = empty, want %q", tt.provider, tt.wantModel)
				}
			} else {
				if def.Model != "" {
					t.Errorf("GetBuiltInProvider(%q).Model = %q, want empty", tt.provider, def.Model)
				}
			}
		})
	}
}

func TestProviderEnvVars(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantEnvVars  bool
		envVarPrefix string
	}{
		{
			name:        "anthropic has no env vars",
			provider:    "anthropic",
			wantEnvVars: false,
		},
		{
			name:         "zai has env vars",
			provider:     "zai",
			wantEnvVars:  true,
			envVarPrefix: "ANTHROPIC_DEFAULT_HAIKU_MODEL",
		},
		{
			name:         "minimax has env vars",
			provider:     "minimax",
			wantEnvVars:  true,
			envVarPrefix: "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT",
		},
		{
			name:         "kimi has env vars",
			provider:     "kimi",
			wantEnvVars:  true,
			envVarPrefix: "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT",
		},
		{
			name:         "deepseek has env vars",
			provider:     "deepseek",
			wantEnvVars:  true,
			envVarPrefix: "API_TIMEOUT_MS",
		},
		{
			name:        "custom has no env vars",
			provider:    "custom",
			wantEnvVars: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := providers.GetBuiltInProvider(tt.provider)
			if !ok {
				t.Fatalf("GetBuiltInProvider(%q) = false, want true", tt.provider)
			}

			if tt.wantEnvVars {
				if len(def.EnvVars) == 0 {
					t.Errorf("GetBuiltInProvider(%q).EnvVars = empty, want non-empty", tt.provider)
				}
				if tt.envVarPrefix != "" {
					found := false
					for _, env := range def.EnvVars {
						if len(env) >= len(tt.envVarPrefix) && env[:len(tt.envVarPrefix)] == tt.envVarPrefix {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("GetBuiltInProvider(%q).EnvVars = %v, want env var with prefix %q", tt.provider, def.EnvVars, tt.envVarPrefix)
					}
				}
			} else {
				if len(def.EnvVars) > 0 {
					t.Errorf("GetBuiltInProvider(%q).EnvVars = %v, want empty", tt.provider, def.EnvVars)
				}
			}
		})
	}
}

func TestIsBuiltInProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"anthropic is builtin", "anthropic", true},
		{"zai is builtin", "zai", true},
		{"minimax is builtin", "minimax", true},
		{"kimi is builtin", "kimi", true},
		{"deepseek is builtin", "deepseek", true},
		{"custom is builtin", "custom", true},
		{"unknown provider", "unknown", false},
		{"empty provider", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providers.IsBuiltInProvider(tt.provider)
			if got != tt.want {
				t.Errorf("IsBuiltInProvider(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestProviderConfigSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {
				Name:    "Z.AI",
				BaseURL: "https://api.z.ai/api/anthropic",
				Model:   "glm-4.7",
				EnvVars: []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash"},
			},
			"minimax": {
				Name:    "MiniMax",
				BaseURL: "https://api.minimax.io/anthropic",
				Model:   "MiniMax-M2.5",
			},
		},
		DefaultProvider: "zai",
	}

	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	zaiProvider, ok := loadedCfg.Providers["zai"]
	if !ok {
		t.Fatal("zai provider not found in loaded config")
	}
	if zaiProvider.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("zai BaseURL = %q, want %q", zaiProvider.BaseURL, "https://api.z.ai/api/anthropic")
	}
	if zaiProvider.Model != "glm-4.7" {
		t.Errorf("zai Model = %q, want %q", zaiProvider.Model, "glm-4.7")
	}

	minimaxProvider, ok := loadedCfg.Providers["minimax"]
	if !ok {
		t.Fatal("minimax provider not found in loaded config")
	}
	if minimaxProvider.BaseURL != "https://api.minimax.io/anthropic" {
		t.Errorf("minimax BaseURL = %q, want %q", minimaxProvider.BaseURL, "https://api.minimax.io/anthropic")
	}

	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", loadedCfg.DefaultProvider, "zai")
	}
}

func TestGetConfigDir(t *testing.T) {
	// Reset configDir to avoid pollution from other tests
	configDir = ""

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot find home directory")
	}

	var expectedDir string
	if runtime.GOOS == "windows" {
		expectedDir = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		expectedDir = filepath.Join(home, ".config", "kairo")
	}
	dir := getConfigDir()
	if dir != expectedDir {
		t.Errorf("getConfigDir() = %q, want %q", dir, expectedDir)
	}
}

func TestConfig_RollbackOnFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {
				Name:    "Z.AI",
				BaseURL: "https://api.z.ai/api/anthropic",
				Model:   "glm-4.7",
			},
		},
		DefaultProvider: "zai",
	}

	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Create backup of config
	backupPath, err := createConfigBackup(tmpDir)
	if err != nil {
		t.Fatalf("createConfigBackup() error = %v", err)
	}
	if backupPath == "" {
		t.Fatal("createConfigBackup() returned empty path")
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file was not created at %s", backupPath)
	}

	// Modify the config (simulating a failed operation)
	modifiedCfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {
				Name:    "Z.AI",
				BaseURL: "https://modified.example.com/api",
				Model:   "modified-model",
			},
		},
		DefaultProvider: "zai",
	}

	if err := config.SaveConfig(tmpDir, modifiedCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify config was modified
	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loadedCfg.Providers["zai"].BaseURL != "https://modified.example.com/api" {
		t.Error("Config was not modified as expected")
	}

	// Rollback to backup
	if err := rollbackConfig(tmpDir, backupPath); err != nil {
		t.Fatalf("rollbackConfig() error = %v", err)
	}

	// Verify config was restored to original state
	restoredCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() after rollback error = %v", err)
	}
	if restoredCfg.Providers["zai"].BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("After rollback, BaseURL = %q, want %q", restoredCfg.Providers["zai"].BaseURL, "https://api.z.ai/api/anthropic")
	}
	if restoredCfg.Providers["zai"].Model != "glm-4.7" {
		t.Errorf("After rollback, Model = %q, want %q", restoredCfg.Providers["zai"].Model, "glm-4.7")
	}
	if restoredCfg.DefaultProvider != "zai" {
		t.Errorf("After rollback, DefaultProvider = %q, want %q", restoredCfg.DefaultProvider, "zai")
	}
}

func TestConfig_TransactionBehavior(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {
				Name:    "Z.AI",
				BaseURL: "https://api.z.ai/api/anthropic",
				Model:   "glm-4.7",
			},
		},
		DefaultProvider: "zai",
	}

	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	originalCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Test 1: Successful transaction should commit changes
	err = withConfigTransaction(tmpDir, func(txDir string) error {
		txCfg := &config.Config{
			Providers: map[string]config.Provider{
				"zai": {
					Name:    "Z.AI",
					BaseURL: "https://transaction.example.com/api",
					Model:   "transaction-model",
				},
			},
			DefaultProvider: "zai",
		}
		return config.SaveConfig(txDir, txCfg)
	})
	if err != nil {
		t.Fatalf("Transaction failed unexpectedly: %v", err)
	}

	// Verify changes were applied
	finalCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() after successful transaction error = %v", err)
	}
	if finalCfg.Providers["zai"].BaseURL != "https://transaction.example.com/api" {
		t.Errorf("Expected BaseURL to be updated to %q, got %q", "https://transaction.example.com/api", finalCfg.Providers["zai"].BaseURL)
	}
	if finalCfg.Providers["zai"].Model != "transaction-model" {
		t.Errorf("Expected Model to be updated to %q, got %q", "transaction-model", finalCfg.Providers["zai"].Model)
	}

	// Test 2: Failed transaction should rollback changes
	err = withConfigTransaction(tmpDir, func(txDir string) error {
		// Modify config
		txCfg := &config.Config{
			Providers: map[string]config.Provider{
				"zai": {
					Name:    "Z.AI",
					BaseURL: "https://should-rollback.example.com/api",
					Model:   "rollback-model",
				},
			},
			DefaultProvider: "zai",
		}
		// Save the change
		if saveErr := config.SaveConfig(txDir, txCfg); saveErr != nil {
			return saveErr
		}
		// Return an error to simulate transaction failure
		return fmt.Errorf("simulated transaction failure")
	})
	if err == nil {
		t.Fatal("Expected transaction to fail, but it succeeded")
	}

	// Verify changes were rolled back (config should remain as after Test 1)
	rollbackCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() after failed transaction error = %v", err)
	}
	if rollbackCfg.Providers["zai"].BaseURL != "https://transaction.example.com/api" {
		t.Errorf("Expected BaseURL to remain %q after rollback, got %q", "https://transaction.example.com/api", rollbackCfg.Providers["zai"].BaseURL)
	}
	if rollbackCfg.Providers["zai"].Model != "transaction-model" {
		t.Errorf("Expected Model to remain %q after rollback, got %q", "transaction-model", rollbackCfg.Providers["zai"].Model)
	}

	// Restore original config for cleanup
	if err := config.SaveConfig(tmpDir, originalCfg); err != nil {
		t.Fatalf("Failed to restore original config: %v", err)
	}
}

func TestConfig_CrossProviderValidation(t *testing.T) {
	// Test environment variable collision detection
	t.Run("EnvVarCollision", func(t *testing.T) {
		// Create config with multiple providers that have conflicting env vars
		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"zai": {
					Name:    "Z.AI",
					BaseURL: "https://api.z.ai/api/anthropic",
					Model:   "glm-4.7",
					EnvVars: []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash"},
				},
				"minimax": {
					Name:    "MiniMax",
					BaseURL: "https://api.minimax.io/anthropic",
					Model:   "MiniMax-M2.5",
					EnvVars: []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=different-model"},
				},
			},
			DefaultProvider: "zai",
		}

		// This should detect the collision
		err := validateCrossProviderConfig(cfg)
		if err == nil {
			t.Error("Expected error for env var collision, got nil")
		}
		if !strings.Contains(err.Error(), "ANTHROPIC_DEFAULT_HAIKU_MODEL") {
			t.Errorf("Expected error to mention 'ANTHROPIC_DEFAULT_HAIKU_MODEL', got: %v", err)
		}
	})

	// Test model validation against provider capabilities
	t.Run("ModelValidation", func(t *testing.T) {
		// Test with a provider that has a default model (zai has "glm-4.7")
		// This should validate the model name
		err := validateProviderModel("zai", "invalid@model#name!")
		if err == nil {
			t.Error("Expected error for invalid model with special characters, got nil")
		}
		// Test with a model that's too long
		longModel := strings.Repeat("a", 101)
		err = validateProviderModel("zai", longModel)
		if err == nil {
			t.Error("Expected error for model name that's too long, got nil")
		}
		// Test with a valid model - should not error
		err = validateProviderModel("zai", "valid-model-name.123")
		if err != nil {
			t.Errorf("Expected valid model to pass validation, got error: %v", err)
		}
	})

	// Test successful cross-provider validation
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"zai": {
					Name:    "Z.AI",
					BaseURL: "https://api.z.ai/api/anthropic",
					Model:   "glm-4.7",
					EnvVars: []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash"},
				},
				"deepseek": {
					Name:    "DeepSeek AI",
					BaseURL: "https://api.deepseek.com/anthropic",
					Model:   "deepseek-chat",
					EnvVars: []string{"API_TIMEOUT_MS=600000"},
				},
			},
			DefaultProvider: "zai",
		}

		err := validateCrossProviderConfig(cfg)
		if err != nil {
			t.Errorf("Expected valid config to pass validation, got error: %v", err)
		}
	})
}
