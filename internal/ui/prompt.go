package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
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

func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

func PrintSuccess(msg string) {
	fmt.Printf("\n%s✓%s %s%s\n", Green, Reset, msg, Reset)
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

func PrintWhite(msg string) {
	fmt.Printf("%s%s%s\n", White, msg, Reset)
}

func PrintGray(msg string) {
	fmt.Printf("%s%s%s\n", Gray, msg, Reset)
}

func PrintDefault(msg string) {
	fmt.Printf("%s%s %s(default)%s\n", White, msg, Gray, Reset)
}

func PromptSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	fmt.Print(": ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		if isInterrupted(err) {
			return "", kairoerrors.ErrUserCancelled
		}
		return "", err
	}
	return string(password), nil
}

func isInterrupted(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrClosed) || errors.Is(err, io.EOF) || strings.Contains(err.Error(), "interrupted")
}

func isEmptyInput(err error) bool {
	if err == nil {
		return false
	}
	return !errors.Is(err, io.EOF) && !isInterrupted(err)
}

func Prompt(prompt string) (string, error) {
	fmt.Print(prompt)
	fmt.Print(": ")
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		if isEmptyInput(err) {
			return "", nil
		}
		if errors.Is(err, io.EOF) || isInterrupted(err) {
			return "", kairoerrors.ErrUserCancelled
		}
		return "", err
	}
	return input, nil
}

func PromptWithDefault(prompt, defaultVal string) (string, error) {
	if defaultVal != "" {
		prompt = fmt.Sprintf("%s [%s]", prompt, defaultVal)
	}
	fmt.Print(prompt)
	fmt.Print(": ")
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		if isEmptyInput(err) {
			return defaultVal, nil
		}
		if errors.Is(err, io.EOF) || isInterrupted(err) {
			return defaultVal, kairoerrors.ErrUserCancelled
		}
		return defaultVal, err
	}
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

type ProviderOption struct {
	Number   int
	Name     string
	Config   *config.Config
	Secrets  map[string]string
	Provider string
}

func PrintProviderOption(opts ProviderOption) {
	configured := isProviderConfigured(opts.Config, opts.Secrets, opts.Provider)
	if configured {
		fmt.Printf("  %d. %s✓%s %s\n", opts.Number, Green, Reset, opts.Name)
	} else {
		fmt.Printf("  %d.   %s\n", opts.Number, opts.Name)
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

func PrintBanner(version string, provider config.Provider) {
	banner := fmt.Sprintf("kairo %s · %s · %s", version, provider.Model, provider.Name)
	fmt.Printf("%s%s%s\n\n", Gray, banner, Reset)
}

func Confirm(prompt string) (bool, error) {
	fmt.Printf("%s [y/N]: ", prompt)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		if isEmptyInput(err) {
			return false, nil
		}
		if errors.Is(err, io.EOF) || isInterrupted(err) {
			return false, kairoerrors.ErrUserCancelled
		}
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}
