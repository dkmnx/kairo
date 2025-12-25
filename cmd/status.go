package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Test all configured providers",
	Long:  "Test connectivity for all configured providers",
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

		cmd.Println("Provider Status")
		cmd.Println("===============")
		cmd.Println()

		var wg sync.WaitGroup
		results := make(map[string]string)
		var mu sync.Mutex

		for name, provider := range cfg.Providers {
			wg.Add(1)
			go func(n string, p config.Provider) {
				defer wg.Done()
				ok, status := testProvider(p)
				mu.Lock()
				results[n] = fmt.Sprintf("%s: %s", statusMarker(ok), status)
				mu.Unlock()
			}(name, provider)
		}

		wg.Wait()

		for name, status := range results {
			cmd.Printf("  %s - %s\n", name, status)
		}

		if cfg.DefaultProvider != "" {
			cmd.Printf("\nDefault provider: %s\n", cfg.DefaultProvider)
		}
	},
}

func statusMarker(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
