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

		logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogSwitch(providerName)
		})

		providerEnv := os.Environ()
		// Environment variable name constants for model configuration
		const (
			envBaseURL     = "ANTHROPIC_BASE_URL"
			envModel       = "ANTHROPIC_MODEL"
			envHaikuModel  = "ANTHROPIC_DEFAULT_HAIKU_MODEL"
			envSonnetModel = "ANTHROPIC_DEFAULT_SONNET_MODEL"
			envOpusModel   = "ANTHROPIC_DEFAULT_OPUS_MODEL"
			envSmallFast   = "ANTHROPIC_SMALL_FAST_MODEL"
		)

		providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", envBaseURL, provider.BaseURL))
		providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", envModel, provider.Model))

		providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", envHaikuModel, provider.Model))
		providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", envSonnetModel, provider.Model))
		providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", envOpusModel, provider.Model))
		providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", envSmallFast, provider.Model))

		providerEnv = append(providerEnv, provider.EnvVars...)

		secretsPath := filepath.Join(dir, "secrets.age")
		keyPath := filepath.Join(dir, "age.key")
		secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			if getVerbose() {
				ui.PrintInfo(fmt.Sprintf("Warning: Could not decrypt secrets: %v", err))
			}
		} else {
			secrets := config.ParseSecrets(secretsContent)
			for key, value := range secrets {
				providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", key, value))
			}
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

				claudeArgs := args[1:]
				claudePath, err := lookPath("claude")
				if err != nil {
					cmd.Println("Error: 'claude' command not found in PATH")
					return
				}

				wrapperScript, useCmdExe, err := wrapper.GenerateWrapperScript(authDir, tokenPath, claudePath, claudeArgs)
				if err != nil {
					cmd.Printf("Error generating wrapper script: %v\n", err)
					return
				}

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
		}

		// No API key found, run claude directly without auth token
		claudeArgs := args[1:]

		claudePath, err := lookPath("claude")
		if err != nil {
			cmd.Println("Error: 'claude' command not found in PATH")
			return
		}

		ui.PrintBanner(version.Version, provider.Name)

		execCmd := execCommand(claudePath, claudeArgs...)
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
	rootCmd.AddCommand(switchCmd)
}
