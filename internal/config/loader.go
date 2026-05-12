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

func checkCtx(ctx context.Context) error {
	return kairoerrors.CheckContext(ctx)
}

func migrateConfigFile(ctx context.Context, configDir string) (bool, error) {
	if err := checkCtx(ctx); err != nil {
		return false, err
	}

	oldConfigPath := filepath.Join(configDir, "config")
	newConfigPath := filepath.Join(configDir, "config.yaml")

	oldInfo, err := statOldConfig(oldConfigPath)
	if err != nil || oldInfo == nil {
		return false, err
	}

	if newExists, err := checkNewConfig(newConfigPath); err != nil || newExists {
		return false, err
	}

	data, err := readAndValidateConfig(oldConfigPath)
	if err != nil {
		return false, err
	}

	if err := writeMigratedConfig(ctx, newConfigPath, data, oldInfo.Mode()); err != nil {
		return false, err
	}

	if err := finalizeMigration(ctx, oldConfigPath, newConfigPath); err != nil {
		return false, err
	}

	return true, nil
}

func statOldConfig(oldConfigPath string) (os.FileInfo, error) {
	oldInfo, err := os.Stat(oldConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to check old config file", err)
	}

	return oldInfo, nil
}

func checkNewConfig(newConfigPath string) (bool, error) {
	if _, err := os.Stat(newConfigPath); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to check new config file", err)
	}

	return false, nil
}

func readAndValidateConfig(oldConfigPath string) ([]byte, error) {
	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to read old config file", err)
	}

	var tempCfg Config
	if err := yaml.Unmarshal(data, &tempCfg); err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
			"old config file is not valid YAML, cannot migrate", err)
	}

	return data, nil
}

func writeMigratedConfig(ctx context.Context, newConfigPath string, data []byte, mode os.FileMode) error {
	if err := checkCtx(ctx); err != nil {
		return err
	}

	return os.WriteFile(newConfigPath, data, mode)
}

func finalizeMigration(ctx context.Context, oldConfigPath, newConfigPath string) error {
	if err := checkCtx(ctx); err != nil {
		os.Remove(newConfigPath)

		return err
	}

	backupPath := oldConfigPath + ".backup"
	if err := os.Rename(oldConfigPath, backupPath); err != nil {
		os.Remove(newConfigPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to backup old config file", err)
	}

	return nil
}

func LoadConfig(ctx context.Context, configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.yaml")

	if err := kairoerrors.CheckContext(ctx); err != nil {
		return nil, err
	}

	_, migrateErr := migrateConfigFile(ctx, configDir)
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

	cfg.validate()

	return &cfg, nil
}

func (c *Config) validate() {
	if c.DefaultProvider == "" {
		return
	}

	if _, exists := c.Providers[c.DefaultProvider]; !exists {
		c.DefaultProvider = ""
	}
}

func SaveConfig(ctx context.Context, configDir string, cfg *Config) error {
	if err := kairoerrors.CheckContext(ctx); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to marshal configuration to YAML", err).
			WithContext("path", configPath)
	}

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
