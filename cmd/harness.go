package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

const (
	harnessClaude = "claude"
	harnessQwen   = "qwen"
)

func isValidHarness(name string) bool {
	return name == harnessClaude || name == harnessQwen
}

var harnessGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current harness",
	Long:  "Get the currently configured default harness",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := loadConfigOrExit(cmd)
		if cfg == nil {
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

		if !isValidHarness(harnessName) {
			ui.PrintError(fmt.Sprintf("Invalid harness: '%s'", args[0]))
			ui.PrintInfo("Valid harnesses: claude, qwen")
			return
		}

		dir := requireConfigDirWritable()
		if dir == "" {
			return
		}

		cfg, err := configCache.Get(getRootCtx(), dir)
		if err != nil && !errors.Is(err, kairoerrors.ErrConfigNotFound) {
			handleConfigError(cmd, err)
			return
		}
		if err != nil {
			cfg = &config.Config{
				Providers:     make(map[string]config.Provider),
				DefaultModels: make(map[string]string),
			}
		}

		cfg.DefaultHarness = harnessName
		if err := config.SaveConfig(getRootCtx(), dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		configCache.Invalidate(dir)

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

// getHarness returns the harness to use, checking flag then config then defaulting to claude.
func getHarness(flagHarness, configHarness string) string {
	harness := flagHarness
	if harness == "" {
		harness = configHarness
	}
	if harness == "" {
		return harnessClaude
	}
	if !isValidHarness(harness) {
		ui.PrintWarn(fmt.Sprintf("Unknown harness '%s', using 'claude'", harness))
		return harnessClaude
	}
	return harness
}

// getHarnessBinary returns the CLI binary name for a given harness.
func getHarnessBinary(harness string) string {
	switch harness {
	case harnessQwen:
		return harnessQwen
	case harnessClaude:
		return harnessClaude
	default:
		return harnessClaude
	}
}
