package config

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

type cachedConfig struct {
	config     *Config
	loadedAt   time.Time
	configPath string
}

type ConfigCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedConfig
	ttl     time.Duration
}

func NewConfigCache(ttl time.Duration) *ConfigCache {
	return &ConfigCache{
		entries: make(map[string]*cachedConfig),
		ttl:     ttl,
	}
}

func deepCopyConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}
	providers := make(map[string]Provider, len(cfg.Providers))
	for k, v := range cfg.Providers {
		providers[k] = Provider{
			Name:    v.Name,
			BaseURL: v.BaseURL,
			Model:   v.Model,
			EnvVars: append([]string{}, v.EnvVars...),
		}
	}
	defaultModels := make(map[string]string, len(cfg.DefaultModels))
	for k, v := range cfg.DefaultModels {
		defaultModels[k] = v
	}

	return &Config{
		DefaultProvider: cfg.DefaultProvider,
		Providers:       providers,
		DefaultModels:   defaultModels,
		DefaultHarness:  cfg.DefaultHarness,
	}
}

func (c *ConfigCache) Get(ctx context.Context, configDir string) (*Config, error) {
	c.mu.RLock()
	entry, exists := c.entries[configDir]
	if exists && time.Since(entry.loadedAt) < c.ttl {
		cfg := deepCopyConfig(entry.config)
		c.mu.RUnlock()

		return cfg, nil
	}
	c.mu.RUnlock()

	cfg, err := LoadConfig(ctx, configDir)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to load config from cache", err).
			WithContext("config_dir", configDir)
	}

	c.mu.Lock()
	c.entries[configDir] = &cachedConfig{
		config:     cfg,
		loadedAt:   time.Now(),
		configPath: filepath.Join(configDir, "config.yaml"),
	}
	c.mu.Unlock()

	return deepCopyConfig(cfg), nil
}

func (c *ConfigCache) Invalidate(configDir string) {
	c.mu.Lock()
	delete(c.entries, configDir)
	c.mu.Unlock()
}
