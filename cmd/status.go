package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Test all configured providers",
	Long:  "Test connectivity for all configured providers",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintWarn("No providers configured")
			ui.PrintInfo("Run 'kairo setup' to get started")
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

		ui.PrintHeader("Provider Status")
		fmt.Println()

		var wg sync.WaitGroup
		results := make(map[string]string)
		var mu sync.Mutex

		for name, provider := range cfg.Providers {
			wg.Add(1)
			go func(n string, p config.Provider) {
				defer wg.Done()
				ok, status := testProvider(p)
				mu.Lock()
				results[n] = fmt.Sprintf("%s %s", statusMarker(ok), status)
				mu.Unlock()
			}(name, provider)
		}

		wg.Wait()

		for name, status := range results {
			fmt.Printf("  %s - %s\n", name, status)
		}

		if cfg.DefaultProvider != "" {
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Default provider: %s", cfg.DefaultProvider))
		}
	},
}

func statusMarker(ok bool) string {
	if ok {
		return "✓ OK"
	}
	return "✗ Failed"
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
