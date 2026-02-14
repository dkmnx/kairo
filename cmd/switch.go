package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/internal/wrapper"
	"github.com/spf13/cobra"
)

// execCommand is the function used to execute external commands.
// It can be replaced in tests to avoid actual process execution.
var execCommand = exec.Command

// exitProcess is the function used to terminate the process.
// It can be replaced in tests to avoid actual exit calls.
var exitProcess = os.Exit

// lookPath is the function used to search for executables in PATH.
// It can be replaced in tests to avoid requiring actual executables.
var lookPath = exec.LookPath

var (
	modelFlag   string
	harnessFlag string
)

// getHarness returns the harness to use, checking flag then config then defaulting to claude.
func getHarness(cfg *config.Config, flagHarness string) string {
	harness := flagHarness
	if harness == "" {
		harness = cfg.DefaultHarness
	}
	if harness == "" {
		return "claude"
	}
	if harness != "claude" && harness != "qwen" {
		ui.PrintWarn(fmt.Sprintf("Unknown harness '%s', using 'claude'", harness))
		return "claude"
	}
	return harness
}

// getHarnessBinary returns the CLI binary name for a given harness.
func getHarnessBinary(harness string) string {
	switch harness {
	case "qwen":
		return "qwen"
	case "claude":
		return "claude"
	default:
		return "claude"
	}
}

// mergeEnvVars merges environment variable slices, deduplicating by key.
// If duplicate keys are found, the last value wins (preserves order of precedence).
// Env vars should be in "KEY=VALUE" format.
func mergeEnvVars(envs ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, envSlice := range envs {
		for _, env := range envSlice {
			// Extract the key (everything before the first '=')
			idx := strings.IndexByte(env, '=')
			if idx <= 0 {
				// Invalid format, skip
				continue
			}
			key := env[:idx]

			// Remove any previous occurrence of this key
			if seen[key] {
				// Find and remove previous entry with this key
				for i, e := range result {
					if strings.HasPrefix(e, key+"=") {
						result = append(result[:i], result[i+1:]...)
						break
					}
				}
			}

			// Add the new entry
			result = append(result, env)
			seen[key] = true
		}
	}

	return result
}

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

		cfg, err := configCache.Get(dir)
		if err != nil {
			handleConfigError(cmd, err)
			return
		}

		provider, ok := cfg.Providers[providerName]
		if !ok {
			cmd.Printf("Error: provider '%s' not configured\n", providerName)
			return
		}

		if err := logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogSwitch(providerName)
		}); err != nil {
			ui.PrintWarn(fmt.Sprintf("Audit logging failed: %v", err))
		}

		harnessToUse := getHarness(cfg, harnessFlag)
		harnessBinary := getHarnessBinary(harnessToUse)

		// Environment variable name constants for model configuration
		const (
			envBaseURL     = "ANTHROPIC_BASE_URL"
			envModel       = "ANTHROPIC_MODEL"
			envHaikuModel  = "ANTHROPIC_DEFAULT_HAIKU_MODEL"
			envSonnetModel = "ANTHROPIC_DEFAULT_SONNET_MODEL"
			envOpusModel   = "ANTHROPIC_DEFAULT_OPUS_MODEL"
			envSmallFast   = "ANTHROPIC_SMALL_FAST_MODEL"
		)

		// Build environment variables with proper deduplication
		// Order of precedence (last wins):
		// 1. System environment variables
		// 2. Built-in Kairo environment variables
		// 3. Provider custom EnvVars
		// 4. Secrets (API keys, etc.)
		builtInEnvVars := []string{
			fmt.Sprintf("%s=%s", envBaseURL, provider.BaseURL),
			fmt.Sprintf("%s=%s", envModel, provider.Model),
			fmt.Sprintf("%s=%s", envHaikuModel, provider.Model),
			fmt.Sprintf("%s=%s", envSonnetModel, provider.Model),
			fmt.Sprintf("%s=%s", envOpusModel, provider.Model),
			fmt.Sprintf("%s=%s", envSmallFast, provider.Model),
		}

		secretsPath := filepath.Join(dir, "secrets.age")
		keyPath := filepath.Join(dir, "age.key")

		var secrets map[string]string
		secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			if providers.RequiresAPIKey(providerName) {
				ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
				ui.PrintInfo("Your encryption key may be corrupted. Try 'kairo rotate' to fix.")
				ui.PrintInfo("Use --verbose for more details.")
				return
			}
			secrets = make(map[string]string)
		} else {
			secrets = config.ParseSecrets(secretsContent)
		}

		// Convert secrets to env var slice
		secretsEnvVars := make([]string, 0, len(secrets))
		for key, value := range secrets {
			secretsEnvVars = append(secretsEnvVars, fmt.Sprintf("%s=%s", key, value))
		}

		// Merge all environment variables with deduplication
		providerEnv := mergeEnvVars(os.Environ(), builtInEnvVars, provider.EnvVars, secretsEnvVars)
		apiKeyKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
		if apiKey, ok := secrets[apiKeyKey]; ok {
			// SECURE: Create private auth directory and use wrapper script
			// This prevents API key from being visible in /proc/<pid>/environ
			// and ensures files are only accessible to the current user
			authDir, err := wrapper.CreateTempAuthDir()
			if err != nil {
				cmd.Printf("Error creating auth directory: %v\n", err)
				return
			}

			var cleanupOnce sync.Once
			cleanup := func() {
				cleanupOnce.Do(func() {
					_ = os.RemoveAll(authDir)
				})
			}
			defer cleanup()

			tokenPath, err := wrapper.WriteTempTokenFile(authDir, apiKey)
			if err != nil {
				cmd.Printf("Error creating secure token file: %v\n", err)
				return
			}

			cliArgs := args[1:]

			// Handle Qwen harness - use wrapper for secure API key
			if harnessToUse == "qwen" {
				modelToUse := modelFlag
				if modelToUse == "" {
					modelToUse = provider.Model
				}

				cliArgs = append([]string{"--model", modelToUse}, cliArgs...)

				ui.ClearScreen()
				ui.PrintBanner(version.Version, provider.Name)

				qwenPath, err := lookPath(harnessBinary)
				if err != nil {
					cmd.Printf("Error: '%s' command not found in PATH\n", harnessBinary)
					cmd.Printf("Please install %s CLI or use 'kairo harness set claude'\n", harnessToUse)
					return
				}

				wrapperScript, useCmdExe, err := wrapper.GenerateWrapperScript(authDir, tokenPath, qwenPath, cliArgs, "ANTHROPIC_API_KEY")
				if err != nil {
					cmd.Printf("Error generating wrapper script: %v\n", err)
					return
				}

				// Set up signal handling for cleanup on SIGINT/SIGTERM
				sigChan := make(chan os.Signal, 1)
				defer func() {
					signal.Stop(sigChan)
					close(sigChan)
				}()
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

				go func() {
					sig := <-sigChan
					cleanup()
					code := 128
					if s, ok := sig.(syscall.Signal); ok {
						code += int(s)
					}
					exitProcess(code)
				}()

				var execCmd *exec.Cmd
				if useCmdExe {
					execCmd = execCommand("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", wrapperScript)
				} else {
					execCmd = execCommand(wrapperScript)
				}
				execCmd.Env = providerEnv
				execCmd.Stdin = os.Stdin
				execCmd.Stdout = os.Stdout
				execCmd.Stderr = os.Stderr

				if err := execCmd.Run(); err != nil {
					cmd.Printf("Error running Qwen: %v\n", err)
					exitProcess(1)
				}
				return
			}

			// Claude harness - existing wrapper script logic
			claudePath, err := lookPath(harnessBinary)
			if err != nil {
				cmd.Printf("Error: '%s' command not found in PATH\n", harnessBinary)
				return
			}

			wrapperScript, useCmdExe, err := wrapper.GenerateWrapperScript(authDir, tokenPath, claudePath, cliArgs)
			if err != nil {
				cmd.Printf("Error generating wrapper script: %v\n", err)
				return
			}

			ui.ClearScreen()
			ui.PrintBanner(version.Version, provider.Name)

			// Set up signal handling for cleanup on SIGINT/SIGTERM
			sigChan := make(chan os.Signal, 1)
			defer func() {
				signal.Stop(sigChan)
				close(sigChan)
			}()
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				sig := <-sigChan
				cleanup()
				// Exit with signal code (cross-platform)
				code := 128
				if s, ok := sig.(syscall.Signal); ok {
					code += int(s)
				}
				exitProcess(code)
			}()

			// Execute the wrapper script instead of claude directly
			// The wrapper script will:
			// 1. Read the API key from the temp file
			// 2. Set ANTHROPIC_AUTH_TOKEN environment variable
			// 3. Delete the temp file
			// 4. Execute claude with the proper arguments
			var execCmd *exec.Cmd
			if useCmdExe {
				// On Windows, use cmd /c to execute the batch file
				execCmd = execCommand("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", wrapperScript)
			} else {
				execCmd = execCommand(wrapperScript)
			}
			execCmd.Env = providerEnv
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			if err := execCmd.Run(); err != nil {
				cmd.Printf("Error running Claude: %v\n", err)
				exitProcess(1)
			}
			return
		}

		// No API key found
		cliArgs := args[1:]

		// Handle Qwen harness
		if harnessToUse == "qwen" {
			ui.PrintError(fmt.Sprintf("API key not found for provider '%s'", providerName))
			ui.PrintInfo("Qwen Code requires API keys to be set in environment variables.")
			return
		}

		// Claude harness - run directly without auth token
		claudePath, err := lookPath(harnessBinary)
		if err != nil {
			cmd.Printf("Error: '%s' command not found in PATH\n", harnessBinary)
			return
		}

		ui.ClearScreen()
		ui.PrintBanner(version.Version, provider.Name)

		execCmd := execCommand(claudePath, cliArgs...)
		execCmd.Env = providerEnv
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			cmd.Printf("Error running Claude: %v\n", err)
			exitProcess(1)
		}
	},
}

func init() {
	switchCmd.Flags().StringVar(&modelFlag, "model", "", "Model to use (passed through to CLI harness)")
	switchCmd.Flags().StringVar(&harnessFlag, "harness", "", "CLI harness to use (claude or qwen)")
	rootCmd.AddCommand(switchCmd)
}
