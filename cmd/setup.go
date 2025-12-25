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

		fmt.Println("Available providers:")
		fmt.Println("  1. Native Anthropic (no API key required)")
		fmt.Println("  2. Z.AI")
		fmt.Println("  3. MiniMax")
		fmt.Println("  4. Kimi")
		fmt.Println("  5. DeepSeek")
		fmt.Println("  6. Custom Provider")
		fmt.Println()

		selection := ui.PromptWithDefault("Select provider to configure", "")
		selection = strings.TrimSpace(selection)

		if selection == "" || selection == "done" {
			return
		}

		num := parseIntOrZero(selection)
		if num < 1 || num > 6 {
			ui.PrintError("Invalid selection. Please enter a number 1-6.")
			return
		}

		providerList := []string{"anthropic", "zai", "minimax", "kimi", "deepseek", "custom"}
		providerName := providerList[num-1]

		if providerName == "anthropic" {
			cfg.Providers["anthropic"] = config.Provider{
				Name:    "Native Anthropic",
				BaseURL: "",
				Model:   "",
			}
			if err := config.SaveConfig(dir, cfg); err != nil {
				ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
				return
			}
			ui.PrintSuccess("Native Anthropic is ready to use!")
			ui.PrintInfo("Run 'kairo anthropic' or just 'kairo' to use it.")
			return
		}

		if providerName == "custom" {
			customName := ui.Prompt("Provider name")
			if customName == "" {
				ui.PrintError("Provider name is required")
				return
			}
			if providers.IsBuiltInProvider(customName) {
				ui.PrintError("This is a reserved provider name")
				return
			}
			providerName = customName
		}

		def, _ := providers.GetBuiltInProvider(providerName)
		if def.Name == "" {
			def.Name = providerName
		}

		fmt.Println()
		ui.PrintHeader(fmt.Sprintf("%s Configuration", def.Name))

		provider := config.Provider{
			Name: def.Name,
		}

		apiKey, err := ui.PromptSecret("API Key")
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error reading API key: %v", err))
			return
		}
		if err := validate.ValidateAPIKey(apiKey); err != nil {
			ui.PrintError(err.Error())
			return
		}

		baseURL := ui.PromptWithDefault("Base URL", def.BaseURL)
		if err := validate.ValidateURL(baseURL); err != nil {
			ui.PrintError(err.Error())
			return
		}
		provider.BaseURL = baseURL

		model := ui.PromptWithDefault("Model", def.Model)
		provider.Model = model

		if len(def.EnvVars) > 0 {
			provider.EnvVars = def.EnvVars
		}

		cfg.Providers[providerName] = provider

		if cfg.DefaultProvider == "" {
			cfg.DefaultProvider = providerName
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

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

		if providerName == "custom" {
			lines[fmt.Sprintf("CUSTOM_%s_API_KEY", providerName)] = apiKey
		} else {
			lines[fmt.Sprintf("%s_API_KEY", providerName)] = apiKey
		}

		var secretsBuilder strings.Builder
		for key, value := range lines {
			if key != "" && value != "" {
				secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder.String()); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving API key: %v", err))
			return
		}

		ui.PrintSuccess(fmt.Sprintf("%s configured successfully", def.Name))
		ui.PrintInfo(fmt.Sprintf("Run 'kairo %s' to use this provider", providerName))
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
