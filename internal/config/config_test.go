package config

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}

	provider, ok := cfg.Providers["zai"]
	if !ok {
		t.Fatal("providers['zai'] not found")
	}
	if provider.Name != "Z.AI" {
		t.Errorf("providers['zai'].Name = %q, want %q", provider.Name, "Z.AI")
	}
	if provider.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("providers['zai'].BaseURL = %q, want %q", provider.BaseURL, "https://api.z.ai/api/anthropic")
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) should error when file not found")
	}
	if !errors.Is(err, kairoerrors.ErrConfigNotFound) {
		t.Errorf("LoadConfig(context.Background(), ) error = %v, want %v", err, kairoerrors.ErrConfigNotFound)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `invalid: yaml: content: [`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) should error on invalid YAML")
	}
}

func TestLoadConfigUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test that unknown fields are rejected with clear error message
	// This helps users catch configuration typos and ensures forward compatibility
	// (older binaries will fail gracefully when reading newer config formats)
	configWithUnknownField := `default_provider: zai
unknown_field: some_value
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
	if err := os.WriteFile(configPath, []byte(configWithUnknownField), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) should reject unknown fields")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error message should mention 'unknown' field, got: %v", err)
	}
}

func TestLoadConfigEmptyProviders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: zai
providers: {}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if cfg.Providers == nil {
		t.Error("Providers should not be nil")
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("Providers count = %d, want 0", len(cfg.Providers))
	}
}

func TestLoadConfigNoProviders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: zai
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	if cfg.Providers == nil {
		t.Error("Providers should not be nil when omitted")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		DefaultProvider: "anthropic",
		Providers: map[string]Provider{
			"anthropic": {
				Name:    "Native Anthropic",
				BaseURL: "",
				Model:   "",
			},
		},
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("config file is empty")
	}
}

func TestSaveConfigCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		DefaultProvider: "test",
		Providers:       make(map[string]Provider),
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestSaveConfigPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		DefaultProvider: "test",
		Providers:       make(map[string]Provider),
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Skip strict permission check on Windows (doesn't support Unix-style 0600)
	if runtime.GOOS != "windows" {
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("File permissions = %o, want 0600", perm)
		}
	}
}

func TestProviderStruct(t *testing.T) {
	p := Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		Model:   "test-model",
		EnvVars: []string{"VAR1=value1", "VAR2=value2"},
	}

	if p.Name != "Test Provider" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Provider")
	}
	if p.BaseURL != "https://api.test.com" {
		t.Errorf("BaseURL = %q, want %q", p.BaseURL, "https://api.test.com")
	}
	if p.Model != "test-model" {
		t.Errorf("Model = %q, want %q", p.Model, "test-model")
	}
	if len(p.EnvVars) != 2 {
		t.Errorf("EnvVars count = %d, want 2", len(p.EnvVars))
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DefaultProvider: "test-provider",
		Providers: map[string]Provider{
			"test": {
				Name:    "Test",
				BaseURL: "https://test.com",
				Model:   "model",
			},
		},
	}

	if cfg.DefaultProvider != "test-provider" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "test-provider")
	}
	if len(cfg.Providers) != 1 {
		t.Errorf("Providers count = %d, want 1", len(cfg.Providers))
	}
}

func TestParseSecrets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single key-value",
			input:    "KEY=value",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "multiple key-values",
			input:    "KEY1=value1\nKEY2=value2\nKEY3=value3",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2", "KEY3": "value3"},
		},
		{
			name:     "value with equals sign",
			input:    "KEY=a=b=c",
			expected: map[string]string{"KEY": "a=b=c"},
		},
		{
			name:     "empty lines ignored",
			input:    "\n\nKEY1=value1\n\nKEY2=value2\n\n",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:     "lines without equals ignored",
			input:    "KEY1=value1\nnoequals\nKEY2=value2",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:     "trailing newline",
			input:    "KEY=value\n",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "real world secrets format",
			input:    "ZAI_API_KEY=sk-test-key123\nMINIMAX_API_KEY=sk-another-key456\n",
			expected: map[string]string{"ZAI_API_KEY": "sk-test-key123", "MINIMAX_API_KEY": "sk-another-key456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSecrets(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseSecrets() returned %d entries, want %d", len(result), len(tt.expected))
				return
			}
			for key, value := range tt.expected {
				if result[key] != value {
					t.Errorf("ParseSecrets()[%q] = %q, want %q", key, result[key], value)
				}
			}
		})
	}
}

func TestMigrateConfigFile(t *testing.T) {
	t.Run("NoMigrationWhenNoOldConfig", func(t *testing.T) {
		tmpDir := t.TempDir()

		migrated, err := migrateConfigFile(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile(context.Background(), ) error = %v", err)
		}
		if migrated {
			t.Error("Expected no migration when old config doesn't exist")
		}

		oldPath := filepath.Join(tmpDir, "config")
		newPath := filepath.Join(tmpDir, "config.yaml")
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Error("Old config file should not exist")
		}
		if _, err := os.Stat(newPath); !os.IsNotExist(err) {
			t.Error("New config file should not exist")
		}
	})

	t.Run("SuccessfulMigration", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
		if err := os.WriteFile(oldConfigPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile(context.Background(), ) error = %v", err)
		}
		if !migrated {
			t.Error("Expected migration to occur")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		data, err := os.ReadFile(newConfigPath)
		if err != nil {
			t.Fatalf("Failed to read new config file: %v", err)
		}

		if string(data) != configContent {
			t.Errorf("Migrated content mismatch.\nGot:\n%s\nWant:\n%s", string(data), configContent)
		}

		backupPath := oldConfigPath + ".backup"
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("Old config file should be backed up, not deleted")
		}

		if _, err := os.Stat(oldConfigPath); !os.IsNotExist(err) {
			t.Error("Old config file should be renamed to backup")
		}
	})

	t.Run("NoMigrationWhenNewConfigExists", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")
		newConfigPath := filepath.Join(tmpDir, "config.yaml")

		oldContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
`
		newContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
`

		if err := os.WriteFile(oldConfigPath, []byte(oldContent), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newConfigPath, []byte(newContent), 0600); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile(context.Background(), ) error = %v", err)
		}
		if migrated {
			t.Error("Should not migrate when new config already exists")
		}

		data, err := os.ReadFile(newConfigPath)
		if err != nil {
			t.Fatalf("Failed to read new config: %v", err)
		}
		if string(data) != newContent {
			t.Error("New config file should not be overwritten")
		}

		if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
			t.Error("Old config file should still exist")
		}
	})

	t.Run("MigrationFailsWithInvalidYAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		invalidYAML := `invalid: yaml: content: [`
		if err := os.WriteFile(oldConfigPath, []byte(invalidYAML), 0600); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(context.Background(), tmpDir)
		if err == nil {
			t.Error("Expected error when migrating invalid YAML")
		}
		if migrated {
			t.Error("Should not report migration on error")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		if _, err := os.Stat(newConfigPath); !os.IsNotExist(err) {
			t.Error("New config file should not be created when old has invalid YAML")
		}

		if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
			t.Error("Old config file should still exist after failed migration")
		}
	})

	t.Run("MigrationPreservesPermissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
`
		if err := os.WriteFile(oldConfigPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		migrated, err := migrateConfigFile(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("migrateConfigFile(context.Background(), ) error = %v", err)
		}
		if !migrated {
			t.Error("Expected migration to occur")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		info, err := os.Stat(newConfigPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0644 {
			t.Errorf("Permissions not preserved: got %o, want %o", info.Mode().Perm(), 0644)
		}
	})
}

func TestLoadConfigWithMigration(t *testing.T) {
	t.Run("LoadConfigMigratesOldFormat", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := filepath.Join(tmpDir, "config")

		configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
`
		if err := os.WriteFile(oldConfigPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		if cfg.DefaultProvider != "zai" {
			t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
		}

		provider, ok := cfg.Providers["zai"]
		if !ok {
			t.Fatal("zai provider not found")
		}
		if provider.Name != "Z.AI" {
			t.Errorf("Provider name = %q, want %q", provider.Name, "Z.AI")
		}

		newConfigPath := filepath.Join(tmpDir, "config.yaml")
		if _, err := os.Stat(newConfigPath); os.IsNotExist(err) {
			t.Error("New config.yaml should exist after LoadConfig with migration")
		}

		backupPath := oldConfigPath + ".backup"
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("Old config should be backed up")
		}
	})

	t.Run("LoadConfigWorksWhenAlreadyMigrated", func(t *testing.T) {
		tmpDir := t.TempDir()
		newConfigPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
`
		if err := os.WriteFile(newConfigPath, []byte(configContent), 0600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
		}

		if cfg.DefaultProvider != "anthropic" {
			t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "anthropic")
		}
	})
}

func TestParseSecretsEmptyKey(t *testing.T) {
	result := ParseSecrets("=value")
	if _, ok := result[""]; ok {
		t.Errorf("ParseSecrets() should skip entries with empty keys, got entry for empty key")
	}
	if len(result) != 0 {
		t.Errorf("ParseSecrets() length = %d, want 0 (empty keys should be skipped)", len(result))
	}
}

func TestParseSecretsEmptyValue(t *testing.T) {
	result := ParseSecrets("KEY=")
	// Empty values should be skipped
	if _, ok := result["KEY"]; ok {
		t.Errorf("ParseSecrets() should skip entries with empty values, got entry for KEY")
	}
	if len(result) != 0 {
		t.Errorf("ParseSecrets() length = %d, want 0 (empty values should be skipped)", len(result))
	}
}

func TestParseSecretsNewlines(t *testing.T) {
	result := ParseSecrets("KEY1=value1\nKEY2=value\nwith\nnewline\nKEY3=value3")

	// Valid entries should be parsed correctly
	if result["KEY1"] != "value1" {
		t.Errorf("ParseSecrets()[KEY1] = %q, want %q", result["KEY1"], "value1")
	}
	if result["KEY3"] != "value3" {
		t.Errorf("ParseSecrets()[KEY3] = %q, want %q", result["KEY3"], "value3")
	}

	// KEY2 value gets split by newline during parsing, so "value" is stored
	if result["KEY2"] != "value" {
		t.Errorf("ParseSecrets()[KEY2] = %q, want %q", result["KEY2"], "value")
	}

	// Lines without = are skipped
	if _, exists := result["with"]; exists {
		t.Error("ParseSecrets() should skip line 'with' (no =)")
	}
	if _, exists := result["newline"]; exists {
		t.Error("ParseSecrets() should skip line 'newline' (no =)")
	}
}

func TestLoadConfigEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Empty file should return error (not valid YAML)
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) on empty file should error")
	}
}

func TestLoadConfigWhitespaceOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte("   \n\n   \n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Whitespace-only file should return error (not valid YAML)
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) on whitespace-only file should error")
	}
}

func TestLoadConfigCommentOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	commentContent := `# This is a comment
# Another comment
`
	if err := os.WriteFile(configPath, []byte(commentContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Comment-only file returns error (YAML parser requires content)
	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) on comment-only file should error")
	}
}

func TestFormatSecrets(t *testing.T) {
	tests := []struct {
		name     string
		secrets  map[string]string
		expected string
	}{
		{
			name:     "empty map",
			secrets:  map[string]string{},
			expected: "",
		},
		{
			name:     "single entry",
			secrets:  map[string]string{"KEY": "value"},
			expected: "KEY=value\n",
		},
		{
			name:     "multiple entries sorted",
			secrets:  map[string]string{"Z_KEY": "val3", "A_KEY": "val1", "M_KEY": "val2"},
			expected: "A_KEY=val1\nM_KEY=val2\nZ_KEY=val3\n",
		},
		{
			name:     "skips empty key",
			secrets:  map[string]string{"": "value", "KEY": "value"},
			expected: "KEY=value\n",
		},
		{
			name:     "skips empty value",
			secrets:  map[string]string{"KEY": "", "OTHER": "value"},
			expected: "OTHER=value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSecrets(tt.secrets)
			if result != tt.expected {
				t.Errorf("FormatSecrets() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatSecretsRoundTrip(t *testing.T) {
	original := map[string]string{
		"ZAI_API_KEY":   "sk-test-123",
		"ANTHROPIC_KEY": "sk-ant-456",
		"DEEPSEEK_KEY":  "sk-deep-789",
	}

	formatted := FormatSecrets(original)
	parsed := ParseSecrets(formatted)

	if len(parsed) != len(original) {
		t.Errorf("Round trip: got %d entries, want %d", len(parsed), len(original))
	}

	for key, value := range original {
		if parsed[key] != value {
			t.Errorf("Round trip: parsed[%q] = %q, want %q", key, parsed[key], value)
		}
	}
}

func TestParseSecretsDoesNotLogRawValues(t *testing.T) {
	// SECURITY TEST: Verify that malformed secret entries do not log raw secret values.
	// This test captures log output and asserts that sensitive data is never exposed.
	//
	// The following cases should NOT log the secret value:
	// 1. Empty key (value contains secret)
	// 2. Empty value (key exists but no secret)
	// 3. Key with newline (malformed entry)
	// 4. Value with newline (malformed entry)
	//
	// The log should only contain:
	// - Line number
	// - Type of issue (e.g., "empty key", "empty value")
	// - NOT the actual key or value

	tests := []struct {
		name          string
		input         string
		secretValue   string // A value that should NEVER appear in logs
		expectParsed  bool   // Whether any valid entries should be parsed
		expectedKey   string // Expected valid key if any
		expectedValue string // Expected valid value if any
	}{
		{
			name:          "malformed entry with secret value should not log secret",
			input:         "=my-super-secret-key-12345\nVALID_KEY=valid_value",
			secretValue:   "my-super-secret-key-12345",
			expectParsed:  true,
			expectedKey:   "VALID_KEY",
			expectedValue: "valid_value",
		},
		{
			name:          "secret in value with newline should not be logged",
			input:         "SECRET_KEY=secret-with\nnewline\nOTHER_KEY=value",
			secretValue:   "secret-with",
			expectParsed:  true,
			expectedKey:   "OTHER_KEY",
			expectedValue: "value",
		},
		{
			name:          "empty value with key should not log key",
			input:         "MY_SECRET_KEY=\nOTHER_KEY=valid",
			secretValue:   "MY_SECRET_KEY",
			expectParsed:  true,
			expectedKey:   "OTHER_KEY",
			expectedValue: "valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf strings.Builder
			originalFlags := log.Flags()
			log.SetOutput(&logBuf)
			log.SetFlags(0) // Remove timestamp for easier testing
			defer func() {
				log.SetOutput(os.Stderr)
				log.SetFlags(originalFlags)
			}()

			// Parse the secrets
			result := ParseSecrets(tt.input)

			// Get the log output
			logOutput := logBuf.String()

			// SECURITY ASSERTION: The secret value must NEVER appear in logs
			if strings.Contains(logOutput, tt.secretValue) {
				t.Errorf("SECURITY VIOLATION: log output contains secret value %q:\n%s", tt.secretValue, logOutput)
			}

			if tt.expectParsed {
				if result[tt.expectedKey] != tt.expectedValue {
					t.Errorf("ParseSecrets()[%q] = %q, want %q", tt.expectedKey, result[tt.expectedKey], tt.expectedValue)
				}
			}
		})
	}
}

func TestParseSecretsLogOutputContainsUsefulDiagnostics(t *testing.T) {
	// without exposing secret values.

	var logBuf strings.Builder
	originalFlags := log.Flags()
	log.SetOutput(&logBuf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(originalFlags)
	}()

	// Parse secrets with various malformed entries
	_ = ParseSecrets("=secret_value\nKEY=\n\nVALID=valid")

	logOutput := logBuf.String()

	if !strings.Contains(logOutput, "line") {
		t.Errorf("Log output should mention 'line' for diagnostics, got: %s", logOutput)
	}

	if strings.Contains(logOutput, "secret_value") {
		t.Errorf("Log output should not contain secret value, got: %s", logOutput)
	}
}
