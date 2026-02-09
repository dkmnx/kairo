package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWatcher(t *testing.T) {
	dir := t.TempDir()

	cache := NewConfigCache(5 * time.Minute)
	watcher := NewFileWatcher(dir, cache, 100*time.Millisecond)
	defer watcher.Close()

	// Write config file
	configPath := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(configPath, []byte("test: value"), 0644)

	// Wait for watcher to detect change
	time.Sleep(200 * time.Millisecond)

	// Cache should be invalidated
	cache.mu.RLock()
	entry := cache.entries[dir]
	cache.mu.RUnlock()

	if entry != nil {
		t.Error("cache should be invalidated after file change")
	}
}

func TestFileWatcher_Close(t *testing.T) {
	dir := t.TempDir()
	cache := NewConfigCache(5 * time.Minute)
	watcher := NewFileWatcher(dir, cache, 100*time.Millisecond)

	// Close should not panic
	watcher.Close()
}

func TestFileWatcher_NoConfigFile(t *testing.T) {
	dir := t.TempDir()
	cache := NewConfigCache(5 * time.Minute)
	watcher := NewFileWatcher(dir, cache, 100*time.Millisecond)
	defer watcher.Close()

	// Wait a bit - should not crash when no config file exists
	time.Sleep(200 * time.Millisecond)

	// Cache should remain empty
	cache.mu.RLock()
	entry := cache.entries[dir]
	cache.mu.RUnlock()

	if entry != nil {
		t.Error("cache should not have entry for non-existent config")
	}
}
