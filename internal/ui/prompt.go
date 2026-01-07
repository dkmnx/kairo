package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"golang.org/x/term"
)

const (
	Green  = "\033[0;32m"
	Yellow = "\033[0;33m"
	Red    = "\033[0;31m"
	Blue   = "\033[0;34m"
	White  = "\033[0;37m"
	Gray   = "\033[0;90m"
	Bold   = "\033[1m"
	Reset  = "\033[0m"
)

func PrintSuccess(msg string) {
	fmt.Printf("%s✓%s %s%s\n", Green, Reset, msg, Reset)
}

func PrintWarn(msg string) {
	fmt.Printf("%s⚠%s %s%s\n", Yellow, Reset, msg, Reset)
}

func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "%s✗%s %s%s\n", Red, Reset, msg, Reset)
}

func PrintInfo(msg string) {
	fmt.Printf("%s%s\n", Blue, msg)
}

func PrintHeader(msg string) {
	fmt.Printf("%s%s%s\n", Bold, msg, Reset)
}

func PrintSection(msg string) {
	fmt.Printf("\n%s=== %s ===%s\n", Bold, msg, Reset)
}

// PrintWhite prints a message in white (no color)
func PrintWhite(msg string) {
	fmt.Printf("%s%s%s\n", White, msg, Reset)
}

// PrintGray prints a message in gray
func PrintGray(msg string) {
	fmt.Printf("%s%s%s\n", Gray, msg, Reset)
}

// PrintDefault prints provider name with "(default)" indicator in gray
func PrintDefault(msg string) {
	fmt.Printf("%s%s %s(default)%s\n", White, msg, Gray, Reset)
}

func PromptSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	fmt.Print(": ")
	fd := int(os.Stdin.Fd())
	password, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func Prompt(prompt string) string {
	fmt.Print(prompt)
	fmt.Print(": ")
	var input string
	// Ignoring error - user can Ctrl+C/D to exit, or input is used as-is
	_, _ = fmt.Scanln(&input)
	return input
}

func PromptWithDefault(prompt, defaultVal string) string {
	if defaultVal != "" {
		prompt = fmt.Sprintf("%s [%s]", prompt, defaultVal)
	}
	fmt.Print(prompt)
	fmt.Print(": ")
	var input string
	// Ignoring error - user can Ctrl+C/D to exit, or input is used as-is
	_, _ = fmt.Scanln(&input)
	if input == "" {
		return defaultVal
	}
	return input
}

func PrintProviderOption(number int, name string, cfg *config.Config, secrets map[string]string, provider string) {
	configured := isProviderConfigured(cfg, secrets, provider)
	if configured {
		fmt.Printf("  %d. %s✓%s %s\n", number, Green, Reset, name)
	} else {
		fmt.Printf("  %d.   %s\n", number, name)
	}
}

func isProviderConfigured(cfg *config.Config, secrets map[string]string, provider string) bool {
	if !providers.RequiresAPIKey(provider) {
		_, exists := cfg.Providers[provider]
		return exists
	}

	apiKeyKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
	for k := range secrets {
		if strings.EqualFold(k, apiKeyKey) {
			return true
		}
	}
	return false
}

func PrintBanner(version, provider string) {
	banner := ` █████                 ███
░░███                 ░░░
 ░███ █████  ██████   ████  ████████   ██████
 ░███░░███  ░░░░░███ ░░███ ░░███░░███ ███░░███
 ░██████░    ███████  ░███  ░███ ░░░ ░███ ░███
 ░███░░███  ███░░███  ░███  ░███     ░███ ░███
 ████ █████░░████████ █████ █████    ░░██████
░░░░ ░░░░░  ░░░░░░░░ ░░░░░ ░░░░░      ░░░░░░   ` + version + ` - ` + provider
	fmt.Printf("%s%s\n", Bold, banner)
}

// Confirm prompts the user for a yes/no confirmation.
// Returns true if the user answers yes/y (case-insensitive), false otherwise.
func Confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
