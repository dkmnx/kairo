package env

import (
	"os"
	"path/filepath"
	"runtime"
)

var (
	configDir string
)

// GetConfigDir returns the configuration directory path.
// If configDir is set (for testing), it returns that value.
// Otherwise, it returns the platform-specific default path:
//   - Unix: ~/.config/kairo
//   - Windows: %APPDATA%\kairo (AppData\Roaming\kairo)
func GetConfigDir() string {
	if configDir != "" {
		return configDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Roaming", "kairo")
	}
	return filepath.Join(home, ".config", "kairo")
}
