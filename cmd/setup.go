package cmd

import (
	"fmt"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

var setupResetSecrets bool

func configureProvider(params ProviderSetup) (string, error) {
	validatedName, err := ResolveProviderName(params.ProviderName)
	if err != nil {
		return "", err
	}

	definition := ProviderDefinition(validatedName)
	provider, exists := params.Cfg.Providers[validatedName]

	promptCfg := providerPromptConfig{
		ProviderName: validatedName,
		Provider:     provider,
		Definition:   definition,
		Secrets:      params.Secrets,
		IsEdit:       exists,
		Exists:       exists,
	}

	displayProviderHeader(promptCfg)

	var envKey string
	if definition.APIKeyEnvVar == "" {
		envKey = promptForEnvKey(promptCfg)
	}

	apiKey := promptForAPIKey(promptCfg)
	if err := definition.ValidateAPIKey(apiKey); err != nil {
		return "", err
	}

	baseURL := promptForBaseURL(promptCfg)
	if err := validate.ValidateURL(baseURL, definition.Name); err != nil {
		return "", err
	}

	model := promptForModel(promptCfg)
	if err := validateConfiguredModel(modelValidationConfig{
		Model:        model,
		ProviderName: validatedName,
		DisplayName:  definition.Name,
	}); err != nil {
		return "", err
	}

	provider = BuildProviderConfig(ProviderBuildConfig{
		Definition: definition,
		BaseURL:    baseURL,
		Model:      model,
		EnvKey:     envKey,
		Exists:     exists,
		Existing:   &provider,
	})

	setAsDefault := params.Cfg.DefaultProvider == ""
	if err := AddAndSaveProvider(AddProviderParams{
		CLIContext:   params.CLIContext,
		ConfigDir:    params.ConfigDir,
		Cfg:          params.Cfg,
		ProviderName: validatedName,
		Provider:     provider,
		SetAsDefault: setAsDefault,
	}); err != nil {
		return "", err
	}

	params.Secrets[APIKeyEnvVarName(validatedName)] = apiKey
	if err := SaveSecrets(params.CLIContext, params.SecretsPath, params.KeyPath, params.Secrets); err != nil {
		return "", err
	}

	tap.Outro(fmt.Sprintf("%s configured successfully", provider.Name), tap.MessageOptions{
		Hint: fmt.Sprintf("Run 'kairo %s' to use this provider", validatedName),
	})

	return validatedName, nil
}

func runResetSecrets(cliCtx *CLIContext, configDir string, secretsResult SecretsResult) error {
	ui.PrintWarn("This will delete your current encryption key and encrypted secrets.")
	ui.PrintInfo("You will need to re-enter all API keys.")
	ui.PrintInfo("")

	confirmed, err := ui.Confirm("Continue")
	if err != nil || !confirmed {
		return kairoerrors.ErrUserCancelled
	}

	if err := ResetSecretsFiles(
		cliCtx.RootCtx(), cliCtx, configDir, secretsResult.SecretsPath, secretsResult.KeyPath,
	); err != nil {
		return err
	}

	ui.PrintSuccess("Encryption key regenerated successfully")

	return nil
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup and edit wizard",
	Long: "Run the interactive wizard to configure new providers or edit existing ones. " +
		"Select a provider to edit or choose 'new provider' to add a new provider.",
	Run: func(cmd *cobra.Command, args []string) {
		cliCtx := CLIContextFromCmd(cmd)
		configDir := cliCtx.ConfigDir()
		if configDir == "" {
			ui.PrintError("Could not determine config directory. Set KAIRO_CONFIG_DIR or provide --config flag.")

			return
		}

		if err := EnsureConfigDir(cliCtx, configDir); err != nil {
			ui.PrintError(err.Error())

			return
		}

		cfg, err := LoadConfig(cliCtx, configDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))

			return
		}

		secretsResult, err := LoadSecrets(cliCtx, configDir)
		if err != nil {
			if setupResetSecrets {
				if err := runResetSecrets(cliCtx, configDir, secretsResult); err != nil {
					ui.PrintError(fmt.Sprintf("Failed to reset secrets: %v", err))
					ui.PrintInfo("Use --verbose for more details.")

					return
				}
				secretsResult.Secrets = make(map[string]string)
			} else {
				ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
				printSecretsRecoveryHelp()

				return
			}
		}

		for _, w := range secretsResult.Warnings {
			ui.PrintWarn(w)
		}

		providerName := promptForProvider(cfg)
		if providerName == "" {
			tap.Cancel("Setup canceled")

			return
		}

		if _, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: providerName,
			Secrets:      secretsResult.Secrets,
			SecretsPath:  secretsResult.SecretsPath,
			KeyPath:      secretsResult.KeyPath,
		}); err != nil {
			tap.Cancel(err.Error())

			return
		}
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupResetSecrets, "reset-secrets", false,
		"Reset encrypted secrets by regenerating encryption key (requires re-entering API keys)")
	rootCmd.AddCommand(setupCmd)
}
