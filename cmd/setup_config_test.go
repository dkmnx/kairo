package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/constants"
)

func TestEnsureConfigDir(t *testing.T) {
	t.Run("creates directory and key", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, "kairo-config")

		cliCtx := NewCLIContext()
		err := EnsureConfigDir(cliCtx, configDir)
		if err != nil {
			t.Fatalf("EnsureConfigDir() error = %v", err)
		}

		info, err := os.Stat(configDir)
		if err != nil {
			t.Fatalf("config dir should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("config dir should be a directory")
		}

		keyPath := filepath.Join(configDir, constants.KeyFileName)
		if _, err := os.Stat(keyPath); err != nil {
			t.Errorf("encryption key should exist: %v", err)
		}
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		cliCtx := NewCLIContext()
		err := EnsureConfigDir(cliCtx, tmpDir)
		if err != nil {
			t.Fatalf("EnsureConfigDir() should succeed with existing dir, got: %v", err)
		}
	})
}
