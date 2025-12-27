package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	DefaultProvider string              `yaml:"default_provider"`
	Providers       map[string]Provider `yaml:"providers"`
}

// Provider represents a configured API provider.
type Provider struct {
	Name    string   `yaml:"name"`
	BaseURL string   `yaml:"base_url"`
	Model   string   `yaml:"model"`
	EnvVars []string `yaml:"env_vars"`
}

// LoadConfig reads and parses the configuration file from the specified directory.
// Returns ErrConfigNotFound if the file doesn't exist.
func LoadConfig(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, ErrConfigNotFound
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]Provider)
	}

	return &cfg, nil
}

// SaveConfig writes the configuration to the specified directory.
func SaveConfig(configDir string, cfg *Config) error {
	configPath := filepath.Join(configDir, "config")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}
