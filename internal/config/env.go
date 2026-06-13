package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/errors"
)

// DefaultConfigDir resolves the platform-specific default configuration directory.
func DefaultConfigDir() (string, error) {
	return ConfigDir()
}

// ConfigDir returns the platform-specific default kairo configuration directory.
// KAIRO_CONFIG_DIR environment variable overrides the platform default.
func ConfigDir() (string, error) {
	if dir := os.Getenv("KAIRO_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WrapError(errors.ConfigError,
			"cannot determine home directory", err)
	}

	if runtime.GOOS == constants.WindowsGOOS {
		return filepath.Join(home, "AppData", "Roaming", "kairo"), nil
	}

	return filepath.Join(home, ".config", "kairo"), nil
}
