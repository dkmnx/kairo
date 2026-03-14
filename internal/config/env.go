package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

const WindowsGOOS = "windows"

var (
	overriddenConfigDir   string
	overriddenConfigDirMu sync.RWMutex
)

func GetConfigDir() (string, error) {
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
	if runtime.GOOS == WindowsGOOS {
		defaultPath = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		defaultPath = filepath.Join(home, ".config", "kairo")
	}

	return defaultPath, nil
}

func SetConfigDir(dir string) {
	overriddenConfigDirMu.Lock()
	overriddenConfigDir = dir
	overriddenConfigDirMu.Unlock()
}
