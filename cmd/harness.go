package cmd

import (
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

const (
	harnessClaude = harness.Claude
	harnessQwen   = harness.Qwen
	harnessPi     = harness.Pi
	harnessCrush  = harness.Crush
)

func isValidHarness(name string) bool {
	return harness.IsValid(name)
}

var harnessGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current harness",
	Long:  "Get the currently configured default harness",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfigOrExit(cmd)
		if err != nil || cfg == nil {
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
	Long:  "Set the default CLI harness to use (claude, qwen, pi, or crush)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		harnessName := strings.ToLower(args[0])

		if !isValidHarness(harnessName) {
			ui.PrintError(fmt.Sprintf("Invalid harness: '%s'", args[0]))
			ui.PrintInfo("Valid harnesses: claude, qwen, pi, crush")

			return
		}

		dir := requireConfigDirWritable(cmd)
		if dir == "" {
			return
		}

		cliCtx := CLIContextFromCmd(cmd)

		cfg := loadConfigOrEmpty(cmd)
		if cfg == nil {
			return
		}

		cfg.DefaultHarness = harnessName
		if err := config.SaveConfig(cliCtx.RootCtx(), dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))

			return
		}

		cliCtx.InvalidateCache(dir)

		ui.PrintSuccess(fmt.Sprintf("Default harness set to: %s", harnessName))
	},
}

var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "Manage CLI harness",
	Long:  "Manage the CLI harness (claude, qwen, pi, or crush)",
}

func init() {
	harnessCmd.AddCommand(harnessGetCmd)
	harnessCmd.AddCommand(harnessSetCmd)
	rootCmd.AddCommand(harnessCmd)
}

func resolveHarness(flagHarness, configHarness string) string {
	h := harness.Resolve(flagHarness, configHarness)
	if h != flagHarness && h != configHarness && h == harnessClaude && (flagHarness != "" || configHarness != "") {
		ui.PrintWarn(fmt.Sprintf("Unknown harness '%s', using 'claude'", flagHarness))
	}

	return h
}

func harnessBinary(harnessName string) string {
	return harnessName
}
