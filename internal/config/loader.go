// Package config loads, validates, and caches the kairo YAML configuration.
package config

import (
	"bytes"
	"context"
	stderrors "errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/fsutil"
	"github.com/dkmnx/kairo/internal/providers"
	"gopkg.in/yaml.v3"
)

// Config represents the top-level kairo configuration file.
type Config struct {
	DefaultProvider string                                        `yaml:"default_provider"`
	Providers       map[string]Provider                           `yaml:"providers"`
	DefaultModels   map[string]string                             `yaml:"default_models"`
	DefaultHarness  string                                        `yaml:"default_harness,omitempty"`
	CustomProviders map[string]providers.CustomProviderDefinition `yaml:"custom_providers"`
}

// Provider represents a single provider's configuration entry.
type Provider struct {
	Name    string   `yaml:"name"`
	BaseURL string   `yaml:"base_url"`
	Model   string   `yaml:"model"`
	EnvVars []string `yaml:"env_vars"`
	EnvKey  string   `yaml:"env_key,omitempty"`
}

func migrateConfigFile(ctx context.Context, configDir string) (bool, error) {
	oldConfigPath := filepath.Join(configDir, "config")
	newConfigPath := filepath.Join(configDir, "config.yaml")

	if err := errors.CheckContext(ctx); err != nil {
		return false, err
	}

	oldInfo, err := os.Stat(oldConfigPath)
	if err != nil {
		if stderrors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, errors.WrapError(errors.FileSystemError,
			"failed to check old config file", err)
	}

	if _, err := os.Stat(newConfigPath); err == nil {
		return false, nil
	} else if !stderrors.Is(err, fs.ErrNotExist) {
		return false, errors.WrapError(errors.FileSystemError,
			"failed to check new config file", err)
	}

	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return false, errors.WrapError(errors.FileSystemError,
			"failed to read old config file", err)
	}

	var tempCfg Config
	if err := yaml.Unmarshal(data, &tempCfg); err != nil {
		return false, errors.WrapError(errors.ConfigError,
			"old config file is not valid YAML, cannot migrate", err)
	}

	if err := os.WriteFile(newConfigPath, data, oldInfo.Mode()); err != nil {
		return false, errors.WrapError(errors.FileSystemError,
			"failed to write migrated config file", err)
	}

	backupPath := oldConfigPath + ".backup"
	if err := os.Rename(oldConfigPath, backupPath); err != nil {
		os.Remove(newConfigPath)

		return false, errors.WrapError(errors.FileSystemError,
			"failed to backup old config file", err)
	}

	return true, nil
}

// LoadConfig reads and parses the configuration file from configDir.
func LoadConfig(ctx context.Context, configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.yaml")

	if err := errors.CheckContext(ctx); err != nil {
		return nil, err
	}

	_, migrateErr := migrateConfigFile(ctx, configDir)
	if migrateErr != nil {
		return nil, errors.WrapError(errors.ConfigError,
			"failed to migrate configuration file", migrateErr).
			WithContext("old_path", filepath.Join(configDir, "config")).
			WithContext("new_path", configPath).
			WithContext("hint", "ensure you have write permissions in the config directory")
	}

	if err := errors.CheckContext(ctx); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if stderrors.Is(err, fs.ErrNotExist) {
			return nil, errors.ErrConfigNotFound
		}

		return nil, errors.WrapError(errors.FileSystemError,
			"failed to read configuration file", err).
			WithContext("path", configPath)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		if isUnknownFieldError(err) {
			return nil, errors.WrapError(errors.ConfigError,
				"configuration file contains field(s) not recognized by this version of kairo", err).
				WithContext("path", configPath).
				WithContext("hint", "your installed kairo binary is outdated, please upgrade")
		}

		return nil, errors.WrapError(errors.ConfigError,
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

	// Reconcile DefaultModels with the authoritative source: the model
	// recorded on each Provider. We treat DefaultModels as a derived
	// index so the two maps cannot drift.
	cfg.reconcileDefaultModels()

	cfg.validate()

	return &cfg, nil
}

// reconcileDefaultModels rebuilds DefaultModels from Providers[].Model.
// Entries already present in DefaultModels are left untouched (so external
// overrides survive) and new entries are populated from the built-in
// provider registry for known providers. This keeps a single source of
// truth: each provider's model lives on the Provider struct.
func (c *Config) reconcileDefaultModels() {
	for name, p := range c.Providers {
		if _, ok := c.DefaultModels[name]; ok {
			continue
		}
		c.DefaultModels[name] = p.Model
	}
}

// isUnknownFieldError reports whether the error is a YAML type error caused by
// unknown fields, indicating the binary is outdated relative to the config file.
func isUnknownFieldError(err error) bool {
	var typeErr *yaml.TypeError

	return stderrors.As(err, &typeErr)
}

func (c *Config) validate() {
	if c.DefaultProvider == "" {
		return
	}

	if _, exists := c.Providers[c.DefaultProvider]; !exists {
		c.DefaultProvider = ""
	}
}

// SaveConfig writes the configuration to configDir atomically.
func SaveConfig(ctx context.Context, configDir string, cfg *Config) error {
	if err := errors.CheckContext(ctx); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.WrapError(errors.ConfigError,
			"failed to marshal configuration to YAML", err).
			WithContext("path", configPath)
	}

	if err := fsutil.WriteAtomic(configPath, func(f *os.File) error {
		_, writeErr := f.Write(data)

		return writeErr
	}); err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to save configuration file", err).
			WithContext("path", configPath)
	}

	if err := os.Chmod(configPath, 0o600); err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to set permissions on configuration file", err).
			WithContext("path", configPath)
	}

	return nil
}
