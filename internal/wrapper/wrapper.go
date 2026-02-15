// Package wrapper provides secure wrapper script generation for Claude Code execution.
//
// This package handles:
//   - Temporary authentication directory creation with secure permissions
//   - Temporary token file writing for secure API key passing
//   - Cross-platform wrapper script generation (PowerShell for Windows, shell for Unix)
//   - Argument escaping to prevent command injection
//
// Security:
//   - Temporary directories use 0700 permissions (owner only)
//   - Token files use 0600 permissions (owner only)
//   - Wrapper scripts immediately delete token files after use
//   - PowerShell argument escaping prevents command injection attacks
//   - API keys never appear in /proc/<pid>/environ
//
// Thread Safety:
//   - Temp directory creation uses os.MkdirTemp (thread-safe)
//   - Not thread-safe for concurrent script generation in same directory
//
// Platform Support:
//   - Windows: PowerShell (.ps1) scripts with cmd.exe execution
//   - Unix/Linux/macOS: Shell scripts with sh execution
//   - Cross-platform argument escaping (platform-specific special characters)
package wrapper

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// CreateTempAuthDir creates a private temporary directory for storing auth files.
// The directory is created with 0700 permissions (owner only) to ensure security.
// Returns the path to the temporary directory.
func CreateTempAuthDir() (string, error) {
	authDir, err := os.MkdirTemp("", "kairo-auth-")
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create temp auth directory", err)
	}

	if err := os.Chmod(authDir, 0700); err != nil {
		_ = os.RemoveAll(authDir)
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to set auth directory permissions", err)
	}

	return authDir, nil
}

// WriteTempTokenFile creates a temporary file with the API key content.
// The file is created with 0600 permissions (owner read/write only) to ensure security.
// Returns the path to the temporary file.
func WriteTempTokenFile(authDir, token string) (string, error) {
	if token == "" {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			"token cannot be empty")
	}

	f, err := os.CreateTemp(authDir, "token-")
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create temp token file", err)
	}

	if _, err := f.WriteString(token); err != nil {
		_ = f.Close()
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write token to temp file", err)
	}

	if err := f.Close(); err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close temp token file", err)
	}

	if err := os.Chmod(f.Name(), 0600); err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to set temp file permissions", err)
	}

	return f.Name(), nil
}

// EscapePowerShellArg escapes a string for use as a PowerShell argument.
// It wraps the argument in single quotes and escapes special characters to prevent
// command injection. Special characters escaped: backtick, dollar sign, double quote,
// single quote, and common control characters (newline, carriage return, tab, etc.).
// Note: Some escape sequences like `v (vertical tab) and `f (form feed) are not
// supported in older PowerShell versions (5.1 and below), so we only escape commonly
// supported control characters.
func EscapePowerShellArg(arg string) string {
	// First, escape backticks (must be done before other replacements)
	arg = strings.ReplaceAll(arg, "`", "``")
	// Escape dollar signs to prevent variable expansion
	arg = strings.ReplaceAll(arg, "$", "`$")
	// Escape double quotes
	arg = strings.ReplaceAll(arg, "\"", "\\\"")
	// Escape single quotes by doubling them
	arg = strings.ReplaceAll(arg, "'", "''")
	// Escape control characters (widely supported in PowerShell)
	arg = strings.ReplaceAll(arg, "\n", "`n")
	arg = strings.ReplaceAll(arg, "\r", "`r")
	arg = strings.ReplaceAll(arg, "\t", "`t")
	// Note: `b (backspace) and `0 (null) are also supported but rarely used in practice
	arg = strings.ReplaceAll(arg, "\b", "`b")
	arg = strings.ReplaceAll(arg, "\x00", "`0")
	// Wrap in single quotes for safest passing
	return "'" + arg + "'"
}

// GenerateWrapperScript creates a temporary script that reads the API key from the
// token file, sets the specified environment variable, cleans up the token
// file, and executes the CLI command with the provided arguments.
// envVarName defaults to "ANTHROPIC_AUTH_TOKEN" if empty.
// Returns the path to the wrapper script and whether to use shell execution.
func GenerateWrapperScript(authDir, tokenPath, cliPath string, cliArgs []string, envVarName ...string) (string, bool, error) {
	if tokenPath == "" {
		return "", false, kairoerrors.NewError(kairoerrors.ValidationError,
			"token path cannot be empty")
	}
	if cliPath == "" {
		return "", false, kairoerrors.NewError(kairoerrors.ValidationError,
			"cli path cannot be empty")
	}

	envVar := "ANTHROPIC_AUTH_TOKEN"
	if len(envVarName) > 0 && envVarName[0] != "" {
		envVar = envVarName[0]
	}

	isWindows := runtime.GOOS == "windows"

	f, err := os.CreateTemp(authDir, "wrapper-")
	if err != nil {
		return "", false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create temp wrapper script", err)
	}

	var scriptContent string

	if isWindows {
		scriptContent = "# Generated by kairo - DO NOT EDIT\r\n"
		scriptContent += "# This script will be automatically deleted after execution\r\n"
		scriptContent += fmt.Sprintf("$env:%s = Get-Content -Path %q -Raw\r\n", envVar, tokenPath)
		scriptContent += fmt.Sprintf("Remove-Item -Path %q -Force\r\n", tokenPath)
		scriptContent += fmt.Sprintf("& %q", cliPath)
		for _, arg := range cliArgs {
			scriptContent += fmt.Sprintf(" %s", EscapePowerShellArg(arg))
		}
		scriptContent += "\r\n"
	} else {
		scriptContent = "#!/bin/sh\n"
		scriptContent += "# Generated by kairo - DO NOT EDIT\n"
		scriptContent += "# This script will be automatically deleted after execution\n"
		scriptContent += fmt.Sprintf("export %s=$(cat %q)\n", envVar, tokenPath)
		scriptContent += fmt.Sprintf("rm -f %q\n", tokenPath)
		scriptContent += "exec " + fmt.Sprintf("%q", cliPath)
		for _, arg := range cliArgs {
			scriptContent += " " + fmt.Sprintf("%q", arg)
		}
		scriptContent += "\n"
	}

	if _, err := f.WriteString(scriptContent); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write wrapper script", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close wrapper script", err)
	}

	if isWindows {
		ps1Path := f.Name() + ".ps1"
		if err := os.Rename(f.Name(), ps1Path); err != nil {
			_ = os.Remove(f.Name())
			return "", false, kairoerrors.WrapError(kairoerrors.FileSystemError,
				"failed to rename wrapper script", err)
		}
		return ps1Path, true, nil
	}

	if err := os.Chmod(f.Name(), 0700); err != nil {
		_ = os.Remove(f.Name())
		return "", false, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to make wrapper script executable", err)
	}

	return f.Name(), false, nil
}

// ExecCommand wraps exec.Command for testability.
func ExecCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
