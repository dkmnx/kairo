package env

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	configDir string
)

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

func SetConfigDir(dir string) {
	configDir = dir
}

func UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func IsSubPath(parent, child string) bool {
	if !strings.HasPrefix(child, parent) {
		return false
	}
	rel := strings.TrimPrefix(child, parent)
	return rel != "" && (rel[0] == '/' || rel[0] == filepath.Separator)
}
