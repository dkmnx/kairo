package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/errors"
)

// ConfigDir returns the platform-specific default kairo configuration directory.
func ConfigDir() (string, error) {
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
