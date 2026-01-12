package env

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	configDir     string
	configDirOnce sync.Once
	configDirMu   sync.RWMutex
)

// GetConfigDir returns the configuration directory path.
// If configDir is set (for testing), it returns that value.
// Otherwise, it returns the platform-specific default path:
//   - Unix: ~/.config/kairo
//   - Windows: %APPDATA%\kairo (AppData\Roaming\kairo)
//
// This function is thread-safe and can be called concurrently.
func GetConfigDir() string {
	configDirMu.RLock()
	dir := configDir
	configDirMu.RUnlock()

	if dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	var defaultPath string
	if runtime.GOOS == "windows" {
		defaultPath = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		defaultPath = filepath.Join(home, ".config", "kairo")
	}

	return defaultPath
}

// SetConfigDir sets the configuration directory path.
// This is primarily used for testing to override the default location.
//
// This function is thread-safe and can be called concurrently.
func SetConfigDir(dir string) {
	configDirMu.Lock()
	configDir = dir
	configDirMu.Unlock()
}
