package cmd

import (
	"context"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

// withTempConfigDir creates a temporary config directory, sets it as the active
// config dir, and defers restoration. This replaces the 44+ repeated
// originalConfigDir/defer setConfigDir patterns across test files.
func withTempConfigDir(t *testing.T) string {
	t.Helper()
	originalConfigDir := getConfigDir()
	tmpDir := t.TempDir()
	setConfigDir(tmpDir)
	t.Cleanup(func() { setConfigDir(originalConfigDir) })
	return tmpDir
}

// newCommandWithContext creates a cobra.Command with a CLIContext attached.
func newCommandWithContext(cliCtx *CLIContext) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(WithCLIContext(context.Background(), cliCtx))
	return cmd
}

// saveConfig creates and saves a config in the given temp dir for testing.
func saveConfig(t *testing.T, dir string, cfg *config.Config) {
	t.Helper()
	if err := config.SaveConfig(context.Background(), dir, cfg); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}
}

// mustCreateConfig creates a minimal valid config in the given directory.
func mustCreateConfig(t *testing.T, dir string, cfg *config.Config) {
	t.Helper()
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.Provider)
	}
	if cfg.DefaultModels == nil {
		cfg.DefaultModels = make(map[string]string)
	}
	saveConfig(t, dir, cfg)
}

// mustLoadConfig loads config from the given directory or fails the test.
func mustLoadConfig(t *testing.T, dir string) *config.Config {
	t.Helper()
	cfg, err := config.LoadConfig(context.Background(), dir)
	if err != nil {
		t.Fatalf("mustLoadConfig: %v", err)
	}
	return cfg
}
