package cmd

import (
	stderrors "errors"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/envutil"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

func requireConfigDir(cmd *cobra.Command) string {
	cliCtx := CLIContextFromCmd(cmd)
	if cliCtx == nil {
		ui.PrintError("CLI context not available")

		return ""
	}

	dir := cliCtx.ConfigDir()
	if dir == "" {
		ui.PrintError("Config directory not found")
	}

	return dir
}

func requireConfigDirWritable(cmd *cobra.Command) string {
	dir := requireConfigDir(cmd)
	if dir == "" {
		return ""
	}
	if err := os.MkdirAll(dir, constants.DirPermSecure); err != nil {
		ui.PrintError("Error creating config directory: " + err.Error())

		return ""
	}

	return dir
}

func loadConfigOrExit(cmd *cobra.Command) (*config.Config, error) {
	dir := requireConfigDir(cmd)
	if dir == "" {
		return nil, stderrors.New("config directory not found")
	}

	cliCtx := CLIContextFromCmd(cmd)
	if cliCtx == nil {
		return nil, stderrors.New("CLI context not available")
	}

	cfg, err := cliCtx.ConfigCache().Get(cliCtx.RootCtx(), dir)
	if err != nil {
		if stderrors.Is(err, kairoerrors.ErrConfigNotFound) {
			printNoProvidersMessage()

			return nil, nil
		}

		handleConfigError(cmd, err)

		return nil, err
	}

	return cfg, nil
}

func loadConfigOrEmpty(cmd *cobra.Command) (*config.Config, error) {
	cfg, err := loadConfigOrExit(cmd)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &config.Config{
			Providers:     make(map[string]config.Provider),
			DefaultModels: make(map[string]string),
		}, nil
	}

	return cfg, nil
}

// printNoProvidersMessage prints a standard message indicating no providers
// are configured and directs the user to run setup.
func printNoProvidersMessage() {
	ui.PrintWarn("No providers configured")
	ui.PrintInfo("Run 'kairo setup' to get started")
}

func printSecretsRecoveryHelp() {
	ui.PrintInfo("Restore 'age.key' and 'secrets.age' from backup,")
	ui.PrintInfo("or remove both files and run 'kairo setup --reset-secrets' to re-enter API keys.")
	ui.PrintInfo("Use --verbose for more details.")
}

func runningWithRaceDetector() bool {
	return strings.Contains(os.Getenv("GOFLAGS"), "-race")
}

// mergeEnvVars combines multiple environment variable slices, deduplicating
// by key name. When duplicates exist, the value from the later slice wins.
func mergeEnvVars(envs ...[]string) []string {
	return envutil.Merge(envs...)
}
