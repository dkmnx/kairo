package config

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

type cachedConfig struct {
	config     *Config
	loadedAt   time.Time
	configPath string
}

type CacheMetrics struct {
	Hits      int64
	Misses    int64
	Evictions int64
}

func (m *CacheMetrics) HitRate() float64 {
	hits := atomic.LoadInt64(&m.Hits)
	misses := atomic.LoadInt64(&m.Misses)
	total := hits + misses
	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total)
}

type ConfigCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedConfig
	ttl     time.Duration
	metrics CacheMetrics
}

func NewConfigCache(ttl time.Duration) *ConfigCache {
	return &ConfigCache{
		entries: make(map[string]*cachedConfig),
		ttl:     ttl,
	}
}

// deepCopyConfig creates a deep copy of a Config to prevent mutation of cached values.
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
		atomic.AddInt64(&c.metrics.Hits, 1)
		c.mu.RUnlock()

		return cfg, nil
	}
	c.mu.RUnlock()

	atomic.AddInt64(&c.metrics.Misses, 1)

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
	if _, exists := c.entries[configDir]; exists {
		atomic.AddInt64(&c.metrics.Evictions, 1)
	}
	delete(c.entries, configDir)
	c.mu.Unlock()
}

func (c *ConfigCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for configDir, entry := range c.entries {
		if now.Sub(entry.loadedAt) >= c.ttl {
			atomic.AddInt64(&c.metrics.Evictions, 1)
			delete(c.entries, configDir)
		}
	}
}

func (c *ConfigCache) GetMetrics() CacheMetrics {
	return CacheMetrics{
		Hits:      atomic.LoadInt64(&c.metrics.Hits),
		Misses:    atomic.LoadInt64(&c.metrics.Misses),
		Evictions: atomic.LoadInt64(&c.metrics.Evictions),
	}
}
