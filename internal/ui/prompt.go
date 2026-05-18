// Package ui provides terminal output helpers with ANSI color support.
package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// ANSI color and style escape sequences.
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

// ClearScreen clears the terminal screen.
func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, "cmd", "/c", "cls")
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, "clear")
	}
	cmd.Stdout = os.Stdout
	// Best-effort clear; ignore terminal errors
	_ = cmd.Run()
}

// PrintSuccess prints a green success message to stdout.
func PrintSuccess(msg string) {
	fmt.Printf("%s✓%s %s%s\n", Green, Reset, msg, Reset)
}

// PrintWarn prints a yellow warning message to stdout.
func PrintWarn(msg string) {
	fmt.Printf("%s⚠%s %s%s\n", Yellow, Reset, msg, Reset)
}

// PrintError prints a red error message to stderr.
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "%s✗%s %s%s\n", Red, Reset, msg, Reset)
}

// PrintInfo prints a blue informational message to stdout.
func PrintInfo(msg string) {
	fmt.Printf("%s%s\n", Blue, msg)
}

// PrintWhite prints a white message to stdout.
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

// Banner holds the information displayed at startup.
type Banner struct {
	Version      string
	ModelName    string
	ProviderName string
	Harness      string
}

// PrintBanner displays the kairo startup banner with version and provider info.
func PrintBanner(b Banner) {
	info := ""
	if b.Harness == "pi" {
		banner := `
 __           .__
|  | _______  |__|______  ____
|  |/ /\__  \ |  \_  __ \/  _ \
|    <  / __ \|  ||  | \(  <_> )
|__|_ \(____  /__||__|   \____/
     \/     \/`

		info = fmt.Sprintf("\n\n%s\n", b.Version)
		fmt.Printf("%s%s%s", Gray, banner, Reset)
	} else {
		info = fmt.Sprintf("%s · %s · %s\n\n", b.Version, b.ModelName, b.ProviderName)
	}

	fmt.Printf("%s%s%s", Gray, info, Reset)
}

// Confirm prompts the user for a y/N confirmation.
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
