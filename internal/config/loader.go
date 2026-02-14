package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	DefaultProvider string              `yaml:"default_provider"`
	Providers       map[string]Provider `yaml:"providers"`
	DefaultModels   map[string]string   `yaml:"default_models"`
	// Version is deprecated and kept for backward compatibility with existing configs.
	// It is no longer used but cannot be removed without breaking existing configs.
	Version string `yaml:"version,omitempty"`
	// DefaultHarness specifies the default CLI harness to use (claude or qwen).
	DefaultHarness string `yaml:"default_harness,omitempty"`
}

// Provider represents a configured API provider.
type Provider struct {
	Name    string   `yaml:"name"`
	BaseURL string   `yaml:"base_url"`
	Model   string   `yaml:"model"`
	EnvVars []string `yaml:"env_vars"`
}

// migrateConfigFile migrates an old config file to the new config.yaml format.
// Returns true if migration was performed, false if not needed.
// Preserves the original file permissions and only migrates if:
// - Old config file exists
// - New config.yaml does not exist
// - Migration succeeds
func migrateConfigFile(configDir string) (bool, error) {
	oldConfigPath := filepath.Join(configDir, "config")
	newConfigPath := filepath.Join(configDir, "config.yaml")

	// Check if old config exists
	oldInfo, err := os.Stat(oldConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No old config to migrate
			return false, nil
		}
		return false, fmt.Errorf("failed to check old config file: %w", err)
	}

	// Check if new config already exists
	if _, err := os.Stat(newConfigPath); err == nil {
		// New config exists, don't overwrite - keep both for safety
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check new config file: %w", err)
	}

	// Read the old config file to verify it's valid YAML before migrating
	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return false, fmt.Errorf("failed to read old config file: %w", err)
	}

	// Verify it's valid YAML
	var testCfg Config
	if err := yaml.Unmarshal(data, &testCfg); err != nil {
		return false, fmt.Errorf("old config file is not valid YAML, cannot migrate: %w", err)
	}

	// Write to new location with same permissions
	if err := os.WriteFile(newConfigPath, data, oldInfo.Mode()); err != nil {
		return false, fmt.Errorf("failed to write migrated config file: %w", err)
	}

	// Rename old file to .backup instead of deleting
	backupPath := oldConfigPath + ".backup"
	if err := os.Rename(oldConfigPath, backupPath); err != nil {
		// If rename fails, try to remove the new file and report error
		os.Remove(newConfigPath)
		return false, fmt.Errorf("failed to backup old config file: %w", err)
	}

	return true, nil
}

// LoadConfig reads and parses the configuration file from the specified directory.
// Returns ErrConfigNotFound if the file doesn't exist.
// Automatically migrates old "config" file to "config.yaml" if needed.
func LoadConfig(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.yaml")

	// Attempt migration if old config exists
	_, migrateErr := migrateConfigFile(configDir)
	if migrateErr != nil {
		return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to migrate configuration file", migrateErr).
			WithContext("old_path", filepath.Join(configDir, "config")).
			WithContext("new_path", configPath).
			WithContext("hint", "ensure you have write permissions in the config directory")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, kairoerrors.ErrConfigNotFound
		}
		return nil, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to read configuration file", err).
			WithContext("path", configPath)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		// Check for unknown field errors and provide helpful guidance
		errStr := err.Error()
		if containsUnknownField(errStr) {
			return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
				"configuration file contains field(s) not recognized by this kairo version", err).
				WithContext("path", configPath).
				WithContext("hint", "your installed kairo binary is outdated - rebuild and reinstall from source")
		}
		return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to parse configuration file (invalid YAML)", err).
			WithContext("path", configPath).
			WithContext("hint", "check YAML syntax and indentation")
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]Provider)
	}

	if cfg.DefaultModels == nil {
		cfg.DefaultModels = make(map[string]string)
	}

	return &cfg, nil
}

// SaveConfig writes the configuration to the specified directory.
func SaveConfig(configDir string, cfg *Config) error {
	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to marshal configuration to YAML", err).
			WithContext("path", configPath)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write configuration file", err).
			WithContext("path", configPath).
			WithContext("permissions", "0600")
	}

	return nil
}

// containsUnknownField checks if the error message indicates an unknown YAML field.
// This pattern appears when the config file contains fields that don't exist
// in the current Config struct, typically due to an outdated binary.
func containsUnknownField(errStr string) bool {
	return containsSubstring(errStr, "field") &&
		containsSubstring(errStr, "not found in type")
}

// containsSubstring is a simple substring checker to avoid importing strings package.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
