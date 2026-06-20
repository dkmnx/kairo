package cmd

import (
	"fmt"
	"sort"

	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage provider catalog",
	Long: `List and refresh the provider catalog.

The provider catalog contains definitions for all supported AI providers,
including their base URLs, default models, API key formats, and environment
variable names.`,
}

var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available providers",
	Long:  "Display all providers in the catalog with their source and key info.",
	Run: func(cmd *cobra.Command, args []string) {
		deps := CLIContextFromCmd(cmd).Deps()

		names := deps.Catalog.ProviderList()
		sort.Strings(names)

		fmt.Println()
		ui.PrintWhite("Available providers:")
		fmt.Println()

		for _, name := range names {
			def, ok := deps.Catalog.BuiltInProvider(name)
			if !ok {
				continue
			}

			source := deps.Catalog.ProviderSource(name)

			fmt.Printf("  %s", name)
			if def.Name != "" && def.Name != name {
				fmt.Printf(" (%s)", def.Name)
			}
			fmt.Printf("  [%s]", source)
			fmt.Println()

			if def.BaseURL != "" {
				fmt.Printf("    URL: %s\n", def.BaseURL)
			}
			if def.Model != "" {
				fmt.Printf("    Model: %s\n", def.Model)
			}
			if def.APIKeyEnvVar != "" {
				fmt.Printf("    API Key: $%s\n", def.APIKeyEnvVar)
			}
			fmt.Println()
		}
	},
}

var providersRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh provider catalog from remote source",
	Long: `Fetch the latest provider catalog from the remote source,
verify its integrity via cosign sigstore, and cache it locally.

Use KAIRO_PROVIDER_CATALOG_URL to override the catalog URL.`,
	Run: func(cmd *cobra.Command, args []string) {
		deps := CLIContextFromCmd(cmd).Deps()

		cmd.Println("Fetching and verifying provider catalog...")

		n, err := deps.Catalog.RefreshFromRemote(cmd.Context())
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to refresh provider catalog: %v", err))

			return
		}

		cmd.Printf("Successfully refreshed provider catalog (%d providers)\n", n)
	},
}

func init() {
	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(providersRefreshCmd)
	rootCmd.AddCommand(providersCmd)
}
