package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	overriddenConfigDir   string
	overriddenConfigDirMu sync.RWMutex
)

func GetConfigDir() string {
	overriddenConfigDirMu.RLock()
	dir := overriddenConfigDir
	overriddenConfigDirMu.RUnlock()

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

func SetConfigDir(dir string) {
	overriddenConfigDirMu.Lock()
	overriddenConfigDir = dir
	overriddenConfigDirMu.Unlock()
}
