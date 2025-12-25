package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Test a specific provider",
	Long:  "Test connectivity to a specific provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		providerName := args[0]

		dir := getConfigDir()
		if dir == "" {
			cmd.Println("Error: config directory not found")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				cmd.Printf("Error: provider '%s' not configured\n", providerName)
				return
			}
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		provider, ok := cfg.Providers[providerName]
		if !ok {
			cmd.Printf("Error: provider '%s' not configured\n", providerName)
			return
		}

		if provider.BaseURL == "" {
			cmd.Printf("Skipping %s: no base URL configured\n", providerName)
			return
		}

		cmd.Printf("Testing %s (%s)...\n", providerName, provider.BaseURL)

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		req, err := http.NewRequest("GET", provider.BaseURL+"/models", nil)
		if err != nil {
			cmd.Printf("  Error: %v\n", err)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			cmd.Printf("  Failed: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
			cmd.Printf("  OK: HTTP %d\n", resp.StatusCode)
		} else {
			cmd.Printf("  Warning: HTTP %d\n", resp.StatusCode)
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func testProvider(provider config.Provider) (bool, string) {
	if provider.BaseURL == "" {
		return false, "no base URL"
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", provider.BaseURL+"/models", nil)
	if err != nil {
		return false, err.Error()
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return true, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}
