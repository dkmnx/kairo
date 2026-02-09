package config

import (
	"path/filepath"
	"sync"
	"time"
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

func (c *ConfigCache) Get(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.yaml")

	c.mu.RLock()
	entry, exists := c.entries[configDir]
	c.mu.RUnlock()

	if exists {
		// Check TTL
		if time.Since(entry.loadedAt) < c.ttl {
			return entry.config, nil
		}
	}

	// Load fresh
	cfg, err := LoadConfig(configDir)
	if err != nil {
		return nil, err
	}

	// Cache it
	c.mu.Lock()
	c.entries[configDir] = &cachedConfig{
		config:     cfg,
		loadedAt:   time.Now(),
		configPath: configPath,
	}
	c.mu.Unlock()

	return cfg, nil
}

func (c *ConfigCache) Invalidate(configDir string) {
	c.mu.Lock()
	delete(c.entries, configDir)
	c.mu.Unlock()
}
