package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/dkmnx/kairo/internal/constants"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

var (
	overriddenConfigDir   string
	overriddenConfigDirMu sync.RWMutex
)

// ConfigDir returns the kairo configuration directory. It uses the overridden
// value if set, otherwise derives the platform-specific default.
func ConfigDir() (string, error) {
	overriddenConfigDirMu.RLock()
	dir := overriddenConfigDir
	overriddenConfigDirMu.RUnlock()

	if dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.ConfigError,
			"cannot determine home directory", err)
	}

	var defaultPath string
	if runtime.GOOS == constants.WindowsGOOS {
		defaultPath = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		defaultPath = filepath.Join(home, ".config", "kairo")
	}

	return defaultPath, nil
}

// SetConfigDir overrides the default configuration directory.
func SetConfigDir(dir string) {
	overriddenConfigDirMu.Lock()
	overriddenConfigDir = dir
	overriddenConfigDirMu.Unlock()
}
