package config

import (
	"os"
	"path/filepath"
	"time"
)

const defaultPollInterval = 500 * time.Millisecond

type FileWatcher struct {
	watchDir     string
	cache        *ConfigCache
	stopCh       chan struct{}
	pollInterval time.Duration
}

func NewFileWatcher(configDir string, cache *ConfigCache, pollInterval time.Duration) *FileWatcher {
	if pollInterval == 0 {
		pollInterval = defaultPollInterval
	}
	fw := &FileWatcher{
		watchDir:     configDir,
		cache:        cache,
		stopCh:       make(chan struct{}),
		pollInterval: pollInterval,
	}
	go fw.watch()
	return fw
}

func (fw *FileWatcher) watch() {
	ticker := time.NewTicker(fw.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.stopCh:
			return
		case <-ticker.C:
			fw.checkForChanges()
		}
	}
}

func (fw *FileWatcher) checkForChanges() {
	configPath := filepath.Join(fw.watchDir, "config.yaml")

	info, err := os.Stat(configPath)
	if err != nil {
		return // File doesn't exist, nothing to watch
	}

	fw.cache.mu.RLock()
	entry := fw.cache.entries[fw.watchDir]
	fw.cache.mu.RUnlock()

	if entry != nil {
		// Check if file was modified since we cached it
		if info.ModTime().After(entry.loadedAt) {
			fw.cache.Invalidate(fw.watchDir)
		}
	}
}

func (fw *FileWatcher) Close() {
	close(fw.stopCh)
}
