package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	Green  = "\033[0;32m"
	Yellow = "\033[0;33m"
	Red    = "\033[0;31m"
	Blue   = "\033[0;34m"
	Bold   = "\033[1m"
	Reset  = "\033[0m"

	green  = Green
	yellow = Yellow
	red    = Red
	blue   = Blue
	bold   = Bold
	reset  = Reset
)

func PrintSuccess(msg string) {
	fmt.Printf("%s✓%s %s%s\n", green, reset, msg, reset)
}

func PrintWarn(msg string) {
	fmt.Printf("%s⚠%s %s%s\n", yellow, reset, msg, reset)
}

func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "%s✗%s %s%s\n", red, reset, msg, reset)
}

func PrintInfo(msg string) {
	fmt.Printf("%sℹ%s %s%s\n", blue, reset, msg, reset)
}

func PrintHeader(msg string) {
	fmt.Printf("%s%s%s\n", bold, msg, reset)
}

func PrintSection(msg string) {
	fmt.Printf("\n%s=== %s ===%s\n", bold, msg, reset)
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
	fmt.Scanln(&input)
	return input
}

func PromptWithDefault(prompt, defaultVal string) string {
	if defaultVal != "" {
		prompt = fmt.Sprintf("%s [%s]", prompt, defaultVal)
	}
	fmt.Print(prompt)
	fmt.Print(": ")
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return defaultVal
	}
	return input
}
