package config

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultProvider string              `yaml:"default_provider"`
	Providers       map[string]Provider `yaml:"providers"`
	DefaultModels   map[string]string   `yaml:"default_models"`
	DefaultHarness  string              `yaml:"default_harness,omitempty"`
}

type Provider struct {
	Name    string   `yaml:"name"`
	BaseURL string   `yaml:"base_url"`
	Model   string   `yaml:"model"`
	EnvVars []string `yaml:"env_vars"`
}

func migrateConfigFile(ctx context.Context, configDir string) (bool, error) {
	oldConfigPath := filepath.Join(configDir, "config")
	newConfigPath := filepath.Join(configDir, "config.yaml")

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	oldInfo, err := os.Stat(oldConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to check old config file", err)
	}

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	if _, err := os.Stat(newConfigPath); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to check new config file", err)
	}

	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to read old config file", err)
	}

	var tempCfg Config
	if err := yaml.Unmarshal(data, &tempCfg); err != nil {
		return false, kairoerrors.WrapError(kairoerrors.ConfigError,
			"old config file is not valid YAML, cannot migrate", err)
	}

	if err := os.WriteFile(newConfigPath, data, oldInfo.Mode()); err != nil {
		return false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write migrated config file", err)
	}

	backupPath := oldConfigPath + ".backup"
	if err := os.Rename(oldConfigPath, backupPath); err != nil {
		os.Remove(newConfigPath)

		return false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to backup old config file", err)
	}

	return true, nil
}

func LoadConfig(ctx context.Context, configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.yaml")

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	_, migrateErr := migrateConfigFile(ctx, configDir)
	if migrateErr != nil {
		return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to migrate configuration file", migrateErr).
			WithContext("old_path", filepath.Join(configDir, "config")).
			WithContext("new_path", configPath).
			WithContext("hint", "ensure you have write permissions in the config directory")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
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

func SaveConfig(ctx context.Context, configDir string, cfg *Config) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to marshal configuration to YAML", err).
			WithContext("path", configPath)
	}

	// Use atomic write (temp file + rename) to prevent partial writes on interruption.
	// This mirrors the safe write pattern used in internal/crypto for secrets and keys.
	tempPath := configPath + ".tmp"

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create temporary config file", err).
			WithContext("path", tempPath)
	}

	_, err = file.Write(data)
	if err != nil {
		file.Close()
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write config data", err).
			WithContext("path", tempPath)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close temporary config file", err).
			WithContext("path", tempPath)
	}

	if err := os.Rename(tempPath, configPath); err != nil {
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to rename temporary config file", err).
			WithContext("temp_path", tempPath).
			WithContext("config_path", configPath)
	}

	return nil
}
