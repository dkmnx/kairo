package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <provider> [args]",
	Short: "Switch to a provider and execute Claude",
	Long:  "Switch to the specified provider and execute Claude Code with optional arguments",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		providerName := args[0]

		dir := getConfigDir()
		if dir == "" {
			cmd.Println("Error: config directory not found")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		provider, ok := cfg.Providers[providerName]
		if !ok {
			cmd.Printf("Error: provider '%s' not configured\n", providerName)
			return
		}

		providerEnv := os.Environ()
		providerEnv = append(providerEnv, fmt.Sprintf("ANTHROPIC_BASE_URL=%s", provider.BaseURL))
		providerEnv = append(providerEnv, fmt.Sprintf("ANTHROPIC_MODEL=%s", provider.Model))

		providerEnv = append(providerEnv, fmt.Sprintf("ANTHROPIC_DEFAULT_HAIKU_MODEL=%s", provider.Model))
		providerEnv = append(providerEnv, fmt.Sprintf("ANTHROPIC_DEFAULT_SONNET_MODEL=%s", provider.Model))
		providerEnv = append(providerEnv, fmt.Sprintf("ANTHROPIC_DEFAULT_OPUS_MODEL=%s", provider.Model))
		providerEnv = append(providerEnv, fmt.Sprintf("ANTHROPIC_SMALL_FAST_MODEL=%s", provider.Model))

		providerEnv = append(providerEnv, provider.EnvVars...)

		secretsPath := filepath.Join(dir, "secrets.age")
		keyPath := filepath.Join(dir, "age.key")
		secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			if verbose {
				ui.PrintInfo(fmt.Sprintf("Warning: Could not decrypt secrets: %v", err))
			}
		} else {
			secrets := config.ParseSecrets(secretsContent)
			for key, value := range secrets {
				providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", key, value))
			}
			if apiKey, ok := secrets[fmt.Sprintf("%s_API_KEY", providerName)]; ok {
				providerEnv = append(providerEnv, fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", apiKey))
			}
		}

		claudeArgs := args[1:]

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			cmd.Println("Error: 'claude' command not found in PATH")
			return
		}

		ui.PrintBanner(provider.Name)

		execCmd := exec.Command(claudePath, claudeArgs...)
		execCmd.Env = providerEnv
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			cmd.Printf("Error running Claude: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
