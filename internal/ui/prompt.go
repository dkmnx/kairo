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

func PrintWhite(msg string) {
	fmt.Printf("%s%s%s\n", White, msg, Reset)
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
