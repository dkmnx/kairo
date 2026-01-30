package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	Long:  "Display all configured providers and their status",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			ui.PrintInfo("Run 'kairo setup' to configure providers")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintWarn("No providers configured")
				ui.PrintInfo("Run 'kairo setup' to get started")
				return
			}
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		if len(cfg.Providers) == 0 {
			ui.PrintWarn("No providers configured")
			ui.PrintInfo("Run 'kairo setup' to get started")
			return
		}

		fmt.Println()
		ui.PrintWhite("Configured providers:")
		fmt.Println()

		names := sortProviderNames(cfg.Providers, cfg.DefaultProvider)

		for _, name := range names {
			p := cfg.Providers[name]
			isDefault := (name == cfg.DefaultProvider)

			if isDefault {
				fmt.Printf("%s  ❯ %s %s(default)%s\n", ui.White, name, ui.Gray, ui.Reset)
			} else {
				ui.PrintWhite(fmt.Sprintf("  ❯ %s", name))
			}

			if !providers.RequiresAPIKey(name) {
				def, _ := providers.GetBuiltInProvider(name)
				ui.PrintWhite(fmt.Sprintf("    %s", def.Name))
				ui.PrintWhite("    Native Anthropic (no API key required)")
			} else {
				if p.BaseURL != "" {
					ui.PrintWhite(fmt.Sprintf("    URL   : %s", p.BaseURL))
				}
				if p.Model != "" {
					ui.PrintWhite(fmt.Sprintf("    Model : %s", p.Model))
				}
			}
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

// sortProviderNames sorts provider names with default provider first.
//
// This function extracts provider names and sorts them alphabetically, except
// the default provider which is always placed at the beginning of the list.
// This ensures the default provider is prominently displayed in list output.
//
// Parameters:
//   - providers: Map of provider names to provider configurations
//   - defaultProvider: Name of the default provider (will be sorted first)
//
// Returns:
//   - []string: Sorted slice of provider names, with default provider first
//
// Error conditions: None (no error returns)
//
// Thread Safety: Thread-safe (no shared state, read-only access to parameters)
// Performance Notes: Uses sort.Slice which has O(n log n) complexity
func sortProviderNames(providers map[string]config.Provider, defaultProvider string) []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == defaultProvider {
			return true
		}
		if names[j] == defaultProvider {
			return false
		}
		return names[i] < names[j]
	})
	return names
}
