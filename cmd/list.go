package cmd

import (
	"fmt"
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
		cfg, err := loadConfigOrExit(cmd)
		if err != nil || cfg == nil {
			return
		}

		if len(cfg.Providers) == 0 {
			printNoProvidersMessage()

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
				def, _ := providers.BuiltInProvider(name)
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

func sortProviderNames(provs map[string]config.Provider, defaultProvider string) []string {
	names := make([]string, 0, len(provs))
	for name := range provs {
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
