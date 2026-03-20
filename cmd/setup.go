package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

// providerNamePattern matches valid custom provider names: starting with a letter
// and containing only alphanumeric characters, underscores, and hyphens.
var providerNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// Provider names must:
// - Be 1-50 characters long
// - Start with a letter
// - Contain only alphanumeric characters, underscores, and hyphens
// - Not be a reserved built-in provider name (case-insensitive)
func validateCustomProviderName(name string) (string, error) {
	if name == "" {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			"provider name is required")
	}
	if len(name) > validate.MaxProviderNameLength {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("provider name must be at most %d characters (got %d)", validate.MaxProviderNameLength, len(name)))
	}
	if !providerNamePattern.MatchString(name) {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			"provider name must start with a letter and contain only alphanumeric characters, underscores, and hyphens")
	}
	// Case-insensitive check to prevent shadowing built-in providers
	lowerName := strings.ToLower(name)
	if providers.IsBuiltInProvider(lowerName) {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("reserved provider name: %s", lowerName))
	}

	return name, nil
}

// buildProviderConfig creates a Provider configuration from a ProviderDefinition.
func buildProviderConfig(definition providers.ProviderDefinition, baseURL, model string) config.Provider {
	provider := config.Provider{
		Name:    definition.Name,
		BaseURL: baseURL,
		Model:   model,
	}
	if len(definition.EnvVars) > 0 {
		provider.EnvVars = definition.EnvVars
	}

	return provider
}

// addAndSaveProvider adds a provider to the config and saves it to disk.
// If setAsDefault is true and no default provider is set, the provider becomes the default.
func addAndSaveProvider(
	cliCtx *CLIContext,
	configDir string,
	cfg *config.Config,
	providerName string,
	provider config.Provider,
	setAsDefault bool,
) error {
	cfg.Providers[providerName] = provider
	if setAsDefault && cfg.DefaultProvider == "" {
		cfg.DefaultProvider = providerName
	}
	if err := config.SaveConfig(cliCtx.GetRootCtx(), configDir, cfg); err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"saving config", err)
	}

	cliCtx.InvalidateCache(configDir)

	return nil
}

// ensureConfigDirectory creates the config directory and encryption key if they don't exist.
func ensureConfigDirectory(cliCtx *CLIContext, configDir string) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"creating config directory", err)
	}
	if err := crypto.EnsureKeyExists(cliCtx.GetRootCtx(), configDir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"creating encryption key", err)
	}

	return nil
}

// loadOrInitializeConfig loads an existing config or creates a new empty one.
func loadOrInitializeConfig(cliCtx *CLIContext, configDir string) (*config.Config, error) {
	cfg, err := cliCtx.GetConfigCache().Get(cliCtx.GetRootCtx(), configDir)
	if err != nil && !errors.Is(err, kairoerrors.ErrConfigNotFound) {
		return nil, err
	}
	if err != nil {
		cfg = &config.Config{
			Providers: make(map[string]config.Provider),
		}
	}

	return cfg, nil
}

// LoadAndDecryptSecrets loads and decrypts secrets from the specified directory.
// Returns secrets map, secrets path, and key path. If secrets file doesn't exist
// or decryption fails, returns empty secrets map with appropriate error handling.
func LoadAndDecryptSecrets(ctx context.Context, configDir string) (map[string]string, string, string, error) {
	secretsPath := filepath.Join(configDir, config.SecretsFileName)
	keyPath := filepath.Join(configDir, config.KeyFileName)

	secrets := make(map[string]string)

	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		return secrets, secretsPath, keyPath, nil
	}

	existingSecrets, err := crypto.DecryptSecrets(ctx, secretsPath, keyPath)
	if err != nil {
		return nil, secretsPath, keyPath, err
	}

	secrets = config.ParseSecrets(existingSecrets)

	return secrets, secretsPath, keyPath, nil
}

// SaveProviderParams holds all parameters for saving provider configuration.
type SaveProviderParams struct {
	CLIContext   *CLIContext
	ConfigDir    string
	Cfg          *config.Config
	ProviderName string
	Provider     config.Provider
	APIKey       string
	Secrets      map[string]string
	SecretsPath  string
	KeyPath      string
	IsEdit       bool
	WasExisting  bool
}

// saveProviderConfiguration saves the provider configuration and secrets.
func saveProviderConfiguration(params SaveProviderParams) error {
	setAsDefault := params.Cfg.DefaultProvider == ""
	if err := addAndSaveProvider(
		params.CLIContext,
		params.ConfigDir,
		params.Cfg,
		params.ProviderName,
		params.Provider,
		setAsDefault,
	); err != nil {
		return err
	}

	// Save secrets
	params.Secrets[apiKeyEnvVarName(params.ProviderName)] = params.APIKey
	secretsContent := config.FormatSecrets(params.Secrets)
	encryptErr := crypto.EncryptSecrets(
		params.CLIContext.GetRootCtx(), params.SecretsPath, params.KeyPath, secretsContent)
	if encryptErr != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"saving API key", encryptErr)
	}

	return nil
}

// resolveProviderName handles custom provider name input and validation.
func resolveProviderName(providerName string) (string, error) {
	if providerName != "custom" {
		return providerName, nil
	}

	customName := tap.Text(context.Background(), tap.TextOptions{
		Message: "Provider name",
	})

	return validateCustomProviderName(customName)
}

// getProviderDefinition retrieves the provider definition, using providerName for custom providers.
func getProviderDefinition(providerName string) providers.ProviderDefinition {
	definition, _ := providers.GetBuiltInProvider(providerName)
	if definition.Name == "" {
		definition.Name = providerName
	}

	return definition
}

// BuildProviderConfigParams holds parameters for building provider config from user input.
type BuildProviderConfigParams struct {
	Definition providers.ProviderDefinition
	BaseURL    string
	Model      string
	Exists     bool
	Existing   config.Provider
}

// buildProviderConfigFromInput creates a Provider config from user input.
func buildProviderConfigFromInput(params BuildProviderConfigParams) config.Provider {
	if !params.Exists {
		return config.Provider{
			Name:    params.Definition.Name,
			BaseURL: params.BaseURL,
			Model:   params.Model,
		}
	}
	params.Existing.BaseURL = params.BaseURL
	params.Existing.Model = params.Model

	return params.Existing
}

// ConfigureProviderParams holds all parameters for configuring a provider.
type ConfigureProviderParams struct {
	CLIContext   *CLIContext
	ConfigDir    string
	Cfg          *config.Config
	ProviderName string
	Secrets      map[string]string
	SecretsPath  string
	KeyPath      string
	IsEdit       bool
}

// configureProvider configures a provider with interactive prompts.
func configureProvider(params ConfigureProviderParams) (string, error) {
	validatedName, err := resolveProviderName(params.ProviderName)
	if err != nil {
		return "", err
	}

	definition := getProviderDefinition(validatedName)
	provider, exists := params.Cfg.Providers[validatedName]

	displayProviderHeader(provider, params.IsEdit, exists)

	apiKey := promptForAPIKey(validatedName, params.Secrets, params.IsEdit, exists)
	if err := validate.ValidateAPIKey(apiKey, definition.Name); err != nil {
		return "", err
	}

	baseURL := promptForBaseURL(provider, definition, params.IsEdit, exists)
	if err := validate.ValidateURL(baseURL, definition.Name); err != nil {
		return "", err
	}

	model := promptForModel(provider, definition, params.IsEdit, exists)
	if err := validate.ValidateProviderModel(model, definition.Name); err != nil {
		return "", err
	}

	if !providers.IsBuiltInProvider(validatedName) {
		model = strings.TrimSpace(model)
		if model == "" {
			return "", kairoerrors.NewError(kairoerrors.ValidationError,
				"model name is required for custom providers")
		}
	}

	provider = buildProviderConfigFromInput(BuildProviderConfigParams{
		Definition: definition,
		BaseURL:    baseURL,
		Model:      model,
		Exists:     exists,
		Existing:   provider,
	})

	if err := saveProviderConfiguration(SaveProviderParams{
		CLIContext:   params.CLIContext,
		ConfigDir:    params.ConfigDir,
		Cfg:          params.Cfg,
		ProviderName: validatedName,
		Provider:     provider,
		APIKey:       apiKey,
		Secrets:      params.Secrets,
		SecretsPath:  params.SecretsPath,
		KeyPath:      params.KeyPath,
		IsEdit:       params.IsEdit,
		WasExisting:  exists,
	}); err != nil {
		return "", err
	}

	tap.Outro(fmt.Sprintf("%s configured successfully", provider.Name), tap.MessageOptions{
		Hint: fmt.Sprintf("Run 'kairo %s' to use this provider", validatedName),
	})

	return validatedName, nil
}

var (
	setupResetSecrets bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup and edit wizard",
	Long: "Run the interactive wizard to configure new providers or edit existing ones. " +
		"Select a provider to edit or choose 'Setup new provider' to add a new provider.",
	Run: func(cmd *cobra.Command, args []string) {
		cliCtx := GetCLIContext(cmd)
		configDir := cliCtx.GetConfigDir()
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				ui.PrintError("Cannot find home directory")

				return
			}
			configDir = filepath.Join(home, ".config", "kairo")
		}

		if err := ensureConfigDirectory(cliCtx, configDir); err != nil {
			ui.PrintError(err.Error())

			return
		}

		cfg, err := loadOrInitializeConfig(cliCtx, configDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))

			return
		}

		secrets, secretsPath, keyPath, err := LoadAndDecryptSecrets(cliCtx.GetRootCtx(), configDir)
		if err != nil {
			if setupResetSecrets {
				if err := resetSecrets(cliCtx, configDir, secretsPath, keyPath); err != nil {
					ui.PrintError(fmt.Sprintf("Failed to reset secrets: %v", err))
					ui.PrintInfo("Use --verbose for more details.")

					return
				}
				secrets = make(map[string]string)
			} else {
				ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
				printSecretsRecoveryHelp()

				return
			}
		}

		providerName := promptForProvider(cfg)
		if providerName == "" {
			ui.PrintInfo("Setup cancelled")

			return
		}

		_, exists := cfg.Providers[providerName]
		if _, err := configureProvider(ConfigureProviderParams{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: providerName,
			Secrets:      secrets,
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
			IsEdit:       exists,
		}); err != nil {
			ui.PrintError(err.Error())

			return
		}
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupResetSecrets, "reset-secrets", false,
		"Reset encrypted secrets by regenerating encryption key (requires re-entering API keys)")
	rootCmd.AddCommand(setupCmd)
}

// resetSecrets handles the --reset-secrets flag by removing old key and secrets files
// and generating a fresh encryption key.
func resetSecrets(cliCtx *CLIContext, configDir, secretsPath, keyPath string) error {
	ui.PrintWarn("This will delete your current encryption key and encrypted secrets.")
	ui.PrintInfo("You will need to re-enter all API keys.")
	ui.PrintInfo("")

	confirmed, err := ui.Confirm("Continue")
	if err != nil || !confirmed {
		return errors.New("operation cancelled by user")
	}

	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old key file", err)
	}

	if err := os.Remove(secretsPath); err != nil && !os.IsNotExist(err) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old secrets file", err)
	}

	if err := crypto.EnsureKeyExists(cliCtx.GetRootCtx(), configDir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate new encryption key", err)
	}

	ui.PrintSuccess("Encryption key regenerated successfully")

	return nil
}
