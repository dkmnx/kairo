package env

import (
	"os"
	"path/filepath"
)

var (
	configDir string
)

// GetConfigDir returns the configuration directory path.
// If configDir is set (for testing), it returns that value.
// Otherwise, it returns the default path: ~/.config/kairo
func GetConfigDir() string {
	if configDir != "" {
		return configDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "kairo")
}
