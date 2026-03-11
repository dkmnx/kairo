package config

import (
	"context"
	"path/filepath"
	"sync"
	"time"
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
	total := m.Hits + m.Misses
	if total == 0 {
		return 0
	}
	return float64(m.Hits) / float64(total)
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

func (c *ConfigCache) Get(ctx context.Context, configDir string) (*Config, error) {
	c.mu.RLock()
	entry, exists := c.entries[configDir]
	if exists && time.Since(entry.loadedAt) < c.ttl {
		cfg := entry.config
		c.metrics.Hits++
		c.mu.RUnlock()

		return cfg, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	c.metrics.Misses++
	c.mu.Unlock()

	cfg, err := LoadConfig(ctx, configDir)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[configDir] = &cachedConfig{
		config:     cfg,
		loadedAt:   time.Now(),
		configPath: filepath.Join(configDir, "config.yaml"),
	}
	c.mu.Unlock()

	return cfg, nil
}

func (c *ConfigCache) Invalidate(configDir string) {
	c.mu.Lock()
	if _, exists := c.entries[configDir]; exists {
		c.metrics.Evictions++
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
			c.metrics.Evictions++
			delete(c.entries, configDir)
		}
	}
}

func (c *ConfigCache) GetMetrics() CacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}
