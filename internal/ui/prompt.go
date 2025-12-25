package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func PrintSuccess(msg string) {
	fmt.Printf("✓ %s\n", msg)
}

func PrintWarn(msg string) {
	fmt.Printf("⚠ %s\n", msg)
}

func PrintError(msg string) {
	fmt.Printf("✗ %s\n", msg)
}

func PrintInfo(msg string) {
	fmt.Printf("ℹ %s\n", msg)
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
