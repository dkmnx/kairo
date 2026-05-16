package config

import (
	"context"
	"maps"
	"path/filepath"
	"sync"
	"time"

	"github.com/dkmnx/kairo/internal/errors"
)

// cachedConfig holds a single cached configuration entry.
type cachedConfig struct {
	config     *Config
	loadedAt   time.Time
	configPath string
}

// ConfigCache provides a TTL-based cache for loaded configurations.
type ConfigCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedConfig
	ttl     time.Duration
}

// NewConfigCache creates a ConfigCache with the given TTL.
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
			EnvKey:  v.EnvKey,
			EnvVars: append([]string{}, v.EnvVars...),
		}
	}
	defaultModels := make(map[string]string, len(cfg.DefaultModels))
	maps.Copy(defaultModels, cfg.DefaultModels)

	return &Config{
		DefaultProvider: cfg.DefaultProvider,
		Providers:       providers,
		DefaultModels:   defaultModels,
		DefaultHarness:  cfg.DefaultHarness,
	}
}

// Get returns the cached config for configDir, loading it fresh if the entry
// is missing or expired.
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
		return nil, errors.WrapError(errors.ConfigError,
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

// Invalidate removes the cached entry for configDir, forcing a reload on next Get.
func (c *ConfigCache) Invalidate(configDir string) {
	c.mu.Lock()
	delete(c.entries, configDir)
	c.mu.Unlock()
}
