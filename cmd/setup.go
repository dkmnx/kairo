package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	Long:  "Run the interactive setup wizard to configure providers",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				ui.PrintError("Cannot find home directory")
				return
			}
			dir = filepath.Join(home, ".config", "kairo")
		}

		if err := os.MkdirAll(dir, 0700); err != nil {
			ui.PrintError(fmt.Sprintf("Error creating config directory: %v", err))
			return
		}

		if err := crypto.EnsureKeyExists(dir); err != nil {
			ui.PrintError(fmt.Sprintf("Error creating encryption key: %v", err))
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil && !os.IsNotExist(err) {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}
		if err != nil {
			cfg = &config.Config{
				Providers: make(map[string]config.Provider),
			}
		}

		ui.PrintHeader("Kairo Setup Wizard")
		fmt.Println()

		providerList := []string{"anthropic", "zai", "minimax", "kimi", "deepseek", "custom"}

		fmt.Println("Available providers:")
		for i, name := range providerList {
			def, _ := providers.GetBuiltInProvider(name)
			marker := " "
			if _, exists := cfg.Providers[name]; exists {
				marker = "✓"
			}
			fmt.Printf("  %d. %s %s\n", i+1, marker, def.Name)
		}
		fmt.Println()
		fmt.Println("Enter provider numbers to configure (e.g., '1 2 3' or '1-3'),")
		fmt.Println("or 'all' to configure all, 'done' to finish:")

		selection := ui.Prompt("Selection")
		selection = strings.TrimSpace(selection)

		var providersToConfigure []string

		if selection == "done" || selection == "" {
			return
		}

		if selection == "all" {
			providersToConfigure = providerList
		} else if strings.Contains(selection, "-") {
			parts := strings.Split(selection, "-")
			if len(parts) == 2 {
				start := parseIntOrZero(parts[0])
				end := parseIntOrZero(parts[1])
				for i := start; i <= end; i++ {
					if i > 0 && i <= len(providerList) {
						providersToConfigure = append(providersToConfigure, providerList[i-1])
					}
				}
			}
		} else {
			for _, part := range strings.Fields(selection) {
				num := parseIntOrZero(part)
				if num > 0 && num <= len(providerList) {
					providersToConfigure = append(providersToConfigure, providerList[num-1])
				}
			}
		}

		for _, name := range providersToConfigure {
			if name == "anthropic" {
				cfg.Providers["anthropic"] = config.Provider{
					Name:    "Native Anthropic",
					BaseURL: "",
					Model:   "",
				}
				ui.PrintSuccess("Native Anthropic selected (no API key required)")
				continue
			}

			def, _ := providers.GetBuiltInProvider(name)

			fmt.Println()
			ui.PrintHeader(fmt.Sprintf("Configuring %s", def.Name))

			provider := config.Provider{
				Name: def.Name,
			}

			baseURL := ui.PromptWithDefault("Base URL", def.BaseURL)
			if err := validate.ValidateURL(baseURL); err != nil {
				ui.PrintError(err.Error())
				continue
			}
			provider.BaseURL = baseURL

			model := ui.PromptWithDefault("Model", def.Model)
			provider.Model = model

			apiKey, err := ui.PromptSecret("API Key")
			if err != nil {
				ui.PrintError(fmt.Sprintf("Error reading API key: %v", err))
				continue
			}
			if err := validate.ValidateAPIKey(apiKey); err != nil {
				ui.PrintError(err.Error())
				continue
			}

			if len(def.EnvVars) > 0 {
				provider.EnvVars = def.EnvVars
			}

			cfg.Providers[name] = provider

			secretsPath := filepath.Join(dir, "secrets.age")
			keyPath := filepath.Join(dir, "age.key")

			var existingSecrets string
			existingSecrets, err = crypto.DecryptSecrets(secretsPath, keyPath)
			if err != nil {
				existingSecrets = ""
			}

			lines := make(map[string]string)
			for _, line := range strings.Split(existingSecrets, "\n") {
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					lines[parts[0]] = parts[1]
				}
			}

			lines[fmt.Sprintf("%s_API_KEY", name)] = apiKey

			var secretsBuilder strings.Builder
			for key, value := range lines {
				if key != "" && value != "" {
					secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
				}
			}

			if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder.String()); err != nil {
				ui.PrintError(fmt.Sprintf("Error saving API key: %v", err))
				continue
			}

			ui.PrintSuccess(fmt.Sprintf("%s configured", def.Name))
		}

		fmt.Println()
		if len(cfg.Providers) > 0 {
			ui.PrintInfo("Configured providers:")
			for name := range cfg.Providers {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println()

			defaultProvider := ui.PromptWithDefault("Set default provider", "")
			if defaultProvider != "" {
				if _, ok := cfg.Providers[defaultProvider]; ok {
					cfg.DefaultProvider = defaultProvider
				}
			}
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		fmt.Println()
		ui.PrintSuccess("Setup complete!")
		fmt.Println("Run 'kairo list' to see configured providers")
	},
}

func parseIntOrZero(s string) int {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		result = result*10 + int(c-'0')
	}
	return result
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
