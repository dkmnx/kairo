package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config <provider>",
	Short: "Configure a provider",
	Long:  "Configure a provider with API key, base URL, and model",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		providerName := args[0]

		if !providers.IsBuiltInProvider(providerName) {
			ui.PrintError(fmt.Sprintf("Unknown provider: '%s'", providerName))
			ui.PrintInfo("Available: anthropic, zai, minimax, kimi, deepseek, custom")
			return
		}

		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		if err := os.MkdirAll(dir, 0700); err != nil {
			ui.PrintError(fmt.Sprintf("Error creating config directory: %v", err))
			return
		}

		if err := crypto.EnsureKeyExists(dir); err != nil {
			ui.PrintError(fmt.Sprintf("Error creating encryption key: %v", err))
			return
		}

		builtinDef, _ := providers.GetBuiltInProvider(providerName)

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

		provider, exists := cfg.Providers[providerName]
		if !exists {
			provider = config.Provider{
				Name: builtinDef.Name,
			}
		}

		ui.PrintHeader(fmt.Sprintf("Configuring %s", provider.Name))

		apiKey, err := ui.PromptSecret("API Key")
		if err != nil {
			if errors.Is(err, ui.ErrUserCancelled) {
				cmd.Println("\nConfiguration cancelled.")
				return
			}
			ui.PrintError(fmt.Sprintf("Error reading API key: %v", err))
			return
		}
		if err := validate.ValidateAPIKey(apiKey, provider.Name); err != nil {
			ui.PrintError(err.Error())
			return
		}

		if builtinDef.BaseURL == "" {
			baseURL, err := ui.PromptWithDefault("Base URL", provider.BaseURL)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to read input: %v", err))
				return
			}
			if err := validate.ValidateURL(baseURL, provider.Name); err != nil {
				ui.PrintError(err.Error())
				return
			}
			provider.BaseURL = baseURL
		} else {
			currentBaseURL := provider.BaseURL
			if currentBaseURL == "" {
				currentBaseURL = builtinDef.BaseURL
			}
			baseURL, err := ui.PromptWithDefault("Base URL", currentBaseURL)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to read input: %v", err))
				return
			}
			if err := validate.ValidateURL(baseURL, provider.Name); err != nil {
				ui.PrintError(err.Error())
				return
			}
			provider.BaseURL = baseURL
		}

		if builtinDef.Model == "" {
			model, err := ui.PromptWithDefault("Model", provider.Model)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to read input: %v", err))
				return
			}
			provider.Model = model
		} else {
			currentModel := provider.Model
			if currentModel == "" {
				currentModel = builtinDef.Model
			}
			model, err := ui.PromptWithDefault("Model", currentModel)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to read input: %v", err))
				return
			}
			provider.Model = model
		}

		// Always refresh EnvVars from provider definition
		// This ensures new env vars from updated definitions are merged
		if len(builtinDef.EnvVars) > 0 {
			provider.EnvVars = builtinDef.EnvVars
		}

		secrets, secretsPath, keyPath, err := LoadSecrets(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
			ui.PrintInfo("Your encryption key may be corrupted. Try 'kairo rotate' to fix.")
			ui.PrintInfo("Use --verbose for more details.")
			return
		}

		oldProvider := cfg.Providers[providerName]
		cfg.Providers[providerName] = provider
		if cfg.DefaultProvider == "" {
			cfg.DefaultProvider = providerName
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		secrets[fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))] = apiKey

		var secretsBuilder strings.Builder
		for key, value := range secrets {
			if key != "" && value != "" {
				secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder.String()); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving API key: %v", err))
			return
		}

		ui.PrintSuccess(fmt.Sprintf("Provider '%s' configured successfully", providerName))

		action := "add"
		if exists {
			action = "update"
		}

		var changes []audit.Change
		if provider.BaseURL != "" && provider.BaseURL != oldProvider.BaseURL {
			old := oldProvider.BaseURL
			if old == "" && builtinDef.BaseURL != "" {
				old = builtinDef.BaseURL
			}
			changes = append(changes, audit.Change{Field: "base_url", Old: old, New: provider.BaseURL})
		}
		if provider.Model != "" && provider.Model != oldProvider.Model {
			old := oldProvider.Model
			if old == "" && builtinDef.Model != "" {
				old = builtinDef.Model
			}
			changes = append(changes, audit.Change{Field: "model", Old: old, New: provider.Model})
		}

		logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogConfig(providerName, action, changes)
		})
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

// createConfigBackup creates a backup of the current configuration file.
// Returns the path to the backup file or an error if the backup fails.
// The backup file is named with a timestamp to allow for multiple backups.
func createConfigBackup(configDir string) (string, error) {
	configPath := getConfigPath(configDir)

	// Read the current config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Create backup filename with timestamp
	backupPath := getBackupPath(configDir)

	// Write the backup
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// rollbackConfig restores the configuration from a backup file.
// If successful, the current config is replaced with the backup.
// The backup file is preserved after rollback for safety.
func rollbackConfig(configDir, backupPath string) error {
	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Read backup data
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write to config file
	configPath := getConfigPath(configDir)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to restore config from backup: %w", err)
	}

	return nil
}

// withConfigTransaction executes a function within a transaction-like context.
// If the function returns an error, changes are rolled back automatically.
// This provides atomic-like behavior for configuration updates.
func withConfigTransaction(configDir string, fn func(txDir string) error) error {
	// Create backup before transaction
	backupPath, err := createConfigBackup(configDir)
	if err != nil {
		return fmt.Errorf("failed to create transaction backup: %w", err)
	}

	// Execute the transaction function
	err = fn(configDir)

	// If transaction failed, rollback
	if err != nil {
		if rbErr := rollbackConfig(configDir, backupPath); rbErr != nil {
			// Rollback failed - this is a critical situation
			return fmt.Errorf("transaction failed and rollback also failed: tx_err=%w, rollback_err=%w", err, rbErr)
		}
		return fmt.Errorf("transaction failed, changes rolled back: %w", err)
	}

	return nil
}

// getConfigPath returns the full path to the config file.
func getConfigPath(configDir string) string {
	return filepath.Join(configDir, "config.yaml")
}

// getBackupPath returns a backup file path with timestamp.
func getBackupPath(configDir string) string {
	timestamp := time.Now().Format("20060102-150405")
	return filepath.Join(configDir, fmt.Sprintf("config.yaml.backup.%s", timestamp))
}

// validateCrossProviderConfig validates configuration across all providers to detect conflicts.
// Returns an error if environment variable collisions are detected.
func validateCrossProviderConfig(cfg *config.Config) error {
	// Build a map of env var names to their values and which providers set them
	type envVarSource struct {
		provider string
		value    string
	}
	envVarMap := make(map[string][]envVarSource)

	for providerName, provider := range cfg.Providers {
		for _, envVar := range provider.EnvVars {
			// Parse env var to get key and value
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			envVarMap[key] = append(envVarMap[key], envVarSource{
				provider: providerName,
				value:    value,
			})
		}
	}

	// Check for collisions - env vars set by multiple providers with different values
	for key, sources := range envVarMap {
		if len(sources) > 1 {
			// Check if all sources have the same value
			firstValue := sources[0].value
			allSame := true
			for _, s := range sources {
				if s.value != firstValue {
					allSame = false
					break
				}
			}
			if !allSame {
				return fmt.Errorf("environment variable collision: '%s' is set to different values by providers: %v",
					key, sources)
			}
		}
	}

	return nil
}

// validateProviderModel validates a model name against provider capabilities.
// For built-in providers with default models, this ensures the model is reasonable.
// Returns an error if the model name is invalid.
func validateProviderModel(providerName, modelName string) error {
	if modelName == "" {
		return nil // Empty model is allowed (will use provider default)
	}

	// Check if this is a built-in provider
	if def, ok := providers.GetBuiltInProvider(providerName); ok {
		// If provider has a default model, do basic validation
		if def.Model != "" {
			// Check model name length (most LLM model names are reasonable length)
			if len(modelName) > 100 {
				return fmt.Errorf("model name '%s' for provider '%s' is too long (max 100 characters)", modelName, providerName)
			}
			// Check for valid characters (alphanumeric, hyphens, underscores, dots)
			for _, r := range modelName {
				if !isValidModelRune(r) {
					return fmt.Errorf("model name '%s' for provider '%s' contains invalid characters", modelName, providerName)
				}
			}
		}
	}

	return nil
}

// isValidModelRune returns true if the rune is valid in a model name.
func isValidModelRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.'
}
