package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var harnessGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current harness",
	Long:  "Get the currently configured default harness",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		cfg, err := configCache.Get(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		if cfg.DefaultHarness == "" {
			ui.PrintInfo("No default harness configured (using claude)")
			return
		}

		ui.PrintInfo(fmt.Sprintf("Default harness: %s", cfg.DefaultHarness))
	},
}

var harnessSetCmd = &cobra.Command{
	Use:   "set <harness>",
	Short: "Set default harness",
	Long:  "Set the default CLI harness to use (claude or qwen)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		harnessName := strings.ToLower(args[0])

		if harnessName != "claude" && harnessName != "qwen" {
			ui.PrintError(fmt.Sprintf("Invalid harness: '%s'", args[0]))
			ui.PrintInfo("Valid harnesses: claude, qwen")
			return
		}

		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		if err := os.MkdirAll(dir, 0700); err != nil {
			ui.PrintError(fmt.Sprintf("Error creating config directory: %v", err))
			return
		}

		cfg, err := configCache.Get(dir)
		if err != nil && !errors.Is(err, kairoerrors.ErrConfigNotFound) {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}
		if err != nil {
			cfg = &config.Config{
				Providers:     make(map[string]config.Provider),
				DefaultModels: make(map[string]string),
			}
		}

		cfg.DefaultHarness = harnessName
		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		ui.PrintSuccess(fmt.Sprintf("Default harness set to: %s", harnessName))
	},
}

var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "Manage CLI harness",
	Long:  "Manage the CLI harness (claude or qwen)",
}

func init() {
	harnessCmd.AddCommand(harnessGetCmd)
	harnessCmd.AddCommand(harnessSetCmd)
	rootCmd.AddCommand(harnessCmd)
}
