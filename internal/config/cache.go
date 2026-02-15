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
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[configDir]

	if exists {
		// Check TTL
		if time.Since(entry.loadedAt) < c.ttl {
			return entry.config, nil
		}
		// Entry expired, remove it
		delete(c.entries, configDir)
	}

	// Load fresh
	cfg, err := LoadConfig(configDir)
	if err != nil {
		return nil, err
	}

	// Cache it
	c.entries[configDir] = &cachedConfig{
		config:     cfg,
		loadedAt:   time.Now(),
		configPath: filepath.Join(configDir, "config.yaml"),
	}

	return cfg, nil
}

func (c *ConfigCache) Invalidate(configDir string) {
	c.mu.Lock()
	delete(c.entries, configDir)
	c.mu.Unlock()
}

// Cleanup removes all expired entries from the cache.
// This can be called periodically to prevent memory growth in long-running processes.
func (c *ConfigCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for configDir, entry := range c.entries {
		if now.Sub(entry.loadedAt) >= c.ttl {
			delete(c.entries, configDir)
		}
	}
}
