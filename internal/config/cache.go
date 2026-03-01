// Package config provides configuration loading, saving, and caching for Kairo.
//
// Cache Behavior:
//   - ConfigCache is designed for CLI usage where processes are short-lived
//   - Cache entries have a TTL (default 5 minutes) to balance freshness vs I/O
//   - Cleanup() is available but typically not needed for CLI tools
//   - For long-running processes, call Cleanup() periodically to prevent memory growth
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
	c.mu.RLock()
	entry, exists := c.entries[configDir]
	if exists && time.Since(entry.loadedAt) < c.ttl {
		cfg := entry.config
		c.mu.RUnlock()
		return cfg, nil
	}
	c.mu.RUnlock()

	// Load config from file
	cfg, err := LoadConfig(configDir)
	if err != nil {
		return nil, err
	}

	// Cache the loaded config
	// Multiple concurrent loads will result in redundant I/O but correct results
	// The last write will win, which is acceptable for this use case
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
	delete(c.entries, configDir)
	c.mu.Unlock()
}

// Cleanup removes all expired entries from the cache.
//
// Note: This method is typically not needed for CLI usage since the process
// exits after each command, releasing all memory. It is provided for
// long-running processes (e.g., daemons, services) that need to prevent
// unbounded memory growth when processing many different config directories.
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
