package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

var enableANSI = runtime.GOOS != "windows"

func PrintSuccess(msg string) {
	if enableANSI {
		fmt.Printf("\n%s✓%s %s%s\n", Green, Reset, msg, Reset)
	} else {
		fmt.Printf("\n✓ %s\n", msg)
	}
}

func PrintWarn(msg string) {
	if enableANSI {
		fmt.Printf("%s⚠%s %s%s\n", Yellow, Reset, msg, Reset)
	} else {
		fmt.Printf("⚠ %s\n", msg)
	}
}

func PrintError(msg string) {
	if enableANSI {
		fmt.Fprintf(os.Stderr, "%s✗%s %s%s\n", Red, Reset, msg, Reset)
	} else {
		fmt.Fprintf(os.Stderr, "✗ %s\n", msg)
	}
}

func PrintInfo(msg string) {
	if enableANSI {
		fmt.Printf("%s%s\n", Blue, msg)
	} else {
		fmt.Printf("  %s\n", msg)
	}
}

func PrintWhite(msg string) {
	if enableANSI {
		fmt.Printf("%s%s%s\n", White, msg, Reset)
	} else {
		fmt.Printf("%s\n", msg)
	}
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

func PrintBanner(version, modelName, providerName string) {
	banner := fmt.Sprintf("kairo %s · %s · %s", version, modelName, providerName)
	if enableANSI {
		fmt.Printf("%s%s%s\n\n", Gray, banner, Reset)
	} else {
		fmt.Printf("%s\n\n", banner)
	}
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
