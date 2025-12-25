package cmd

import (
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	Long:  "Display all configured providers and their status",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			cmd.Println("No configured providers")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				cmd.Println("No configured providers")
				return
			}
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		if len(cfg.Providers) == 0 {
			cmd.Println("No configured providers")
			return
		}

		cmd.Println("Configured providers:")
		for name, p := range cfg.Providers {
			marker := " "
			if name == cfg.DefaultProvider {
				marker = "*"
			}
			cmd.Printf("  %s %s (%s)\n", marker, name, p.BaseURL)
		}

		if cfg.DefaultProvider != "" {
			cmd.Printf("\nDefault provider: %s\n", cfg.DefaultProvider)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func providerExists(name string, cfg *config.Config) bool {
	_, ok := cfg.Providers[name]
	return ok
}

func getProvider(name string, cfg *config.Config) (config.Provider, bool) {
	p, ok := cfg.Providers[name]
	return p, ok
}

func setProvider(name string, p config.Provider, cfg *config.Config) {
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.Provider)
	}
	cfg.Providers[name] = p
}
