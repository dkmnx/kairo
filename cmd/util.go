package cmd

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

func requireConfigDir(cmd *cobra.Command) string {
	if cmd == nil {
		// Fall back to default context for backward compatibility
		return defaultCLIContext.GetConfigDir()
	}
	dir := GetCLIContext(cmd).GetConfigDir()
	if dir == "" {
		ui.PrintError("Config directory not found")
	}
	return dir
}

func requireConfigDirWritable(cmd *cobra.Command) string {
	dir := requireConfigDir(cmd)
	if dir == "" {
		return ""
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		ui.PrintError("Error creating config directory: " + err.Error())
		return ""
	}
	return dir
}

func loadConfigOrExit(cmd *cobra.Command) *config.Config {
	dir := requireConfigDir(cmd)
	if dir == "" {
		return nil
	}

	cliCtx := GetCLIContext(cmd)
	cfg, err := cliCtx.GetConfigCache().Get(cliCtx.GetRootCtx(), dir)
	if err != nil {
		if os.IsNotExist(err) {
			ui.PrintWarn("No providers configured")
			ui.PrintInfo("Run 'kairo setup' to get started")
			return nil
		}
		handleConfigError(cmd, err)
		return nil
	}
	return cfg
}

func printSecretsRecoveryHelp() {
	ui.PrintInfo("Restore 'age.key' and 'secrets.age' from backup,")
	ui.PrintInfo("or remove both files and run 'kairo setup --reset-secrets' to re-enter API keys.")
	ui.PrintInfo("Use --verbose for more details.")
}

// Test hooks
var lookPath = exec.LookPath

var execCommand = exec.Command

var execCommandContext = exec.CommandContext

var exitProcess = os.Exit

func parseIntOrZero(input string) int {
	var result int
	for _, c := range input {
		if c < '0' || c > '9' {
			return 0
		}
		result = result*10 + int(c-'0')
	}
	return result
}

func runningWithRaceDetector() bool {
	return strings.Contains(os.Getenv("GOFLAGS"), "-race")
}

func mergeEnvVars(envs ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, envSlice := range envs {
		for _, env := range envSlice {
			idx := strings.IndexByte(env, '=')
			if idx <= 0 {
				continue
			}
			key := env[:idx]
			seen[key] = true
			result = append(result, env)
		}
	}

	if len(seen) == len(result) {
		return result
	}

	seen = make(map[string]bool)
	var deduped []string
	for i := len(result) - 1; i >= 0; i-- {
		env := result[i]
		idx := strings.IndexByte(env, '=')
		if idx <= 0 {
			continue
		}
		key := env[:idx]
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, env)
		}
	}

	for i, j := 0, len(deduped)-1; i < j; i, j = i+1, j-1 {
		deduped[i], deduped[j] = deduped[j], deduped[i]
	}

	return deduped
}

func setupSignalHandler(cancel func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		signal.Stop(sigChan)
		if cancel != nil {
			cancel()
		}
	}()
}
