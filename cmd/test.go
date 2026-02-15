package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/ui"
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
			ui.PrintError("Config directory not found")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintError(fmt.Sprintf("Provider '%s' not configured", providerName))
				ui.PrintInfo("Run 'kairo config " + providerName + "' to configure")
				return
			}
			handleConfigError(cmd, err)
			return
		}

		provider, ok := cfg.Providers[providerName]
		if !ok {
			ui.PrintError(fmt.Sprintf("Provider '%s' not configured", providerName))
			ui.PrintInfo("Run 'kairo config " + providerName + "' to configure")
			return
		}

		if provider.BaseURL == "" {
			ui.PrintWarn(fmt.Sprintf("Skipping %s: no base URL configured", providerName))
			return
		}

		ui.PrintInfo(fmt.Sprintf("Testing %s (%s)...", providerName, provider.BaseURL))

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		req, err := http.NewRequest("GET", provider.BaseURL+"/models", nil)
		if err != nil {
			ui.PrintError(err.Error())
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed: %v", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
			ui.PrintSuccess(fmt.Sprintf("OK: HTTP %d", resp.StatusCode))
		} else {
			ui.PrintWarn(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
