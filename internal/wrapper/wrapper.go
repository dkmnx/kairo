// Package wrapper generates platform-specific shell scripts that securely
// pass authentication tokens to CLI harnesses.
package wrapper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/errors"
)

// CreateTempAuthDir creates a temporary directory with restricted permissions
// for storing authentication tokens.
func CreateTempAuthDir() (string, error) {
	authDir, err := os.MkdirTemp("", "kairo-auth-")
	if err != nil {
		return "", errors.WrapError(errors.FileSystemError,
			"failed to create temp auth directory", err)
	}

	if err := os.Chmod(authDir, constants.DirPermSecure); err != nil {
		_ = os.RemoveAll(authDir)

		return "", errors.WrapError(errors.FileSystemError,
			"failed to set auth directory permissions", err)
	}

	return authDir, nil
}

// WriteTempTokenFile writes the given token to a temporary file in authDir
// with restricted permissions and returns its path.
func WriteTempTokenFile(authDir, token string) (string, error) {
	if token == "" {
		return "", errors.NewError(errors.ValidationError,
			"wrapper: token cannot be empty")
	}

	f, err := os.CreateTemp(authDir, "token-")
	if err != nil {
		return "", errors.WrapError(errors.FileSystemError,
			"failed to create temp token file", err)
	}

	if _, err := f.WriteString(token); err != nil {
		_ = f.Close()

		return "", errors.WrapError(errors.FileSystemError,
			"failed to write token to temp file", err)
	}

	if err := f.Close(); err != nil {
		return "", errors.WrapError(errors.FileSystemError,
			"failed to close temp token file", err)
	}

	if err := os.Chmod(f.Name(), constants.FilePermSecure); err != nil {
		return "", errors.WrapError(errors.FileSystemError,
			"failed to set temp file permissions", err)
	}

	return f.Name(), nil
}

// EscapePowerShellArg escapes a string for safe use as a PowerShell argument.
//
// THREAT MODEL: This function protects auth-token delivery into a PowerShell
// subprocess. The invariant is single-quote wrapping plus ' doubling (the only
// escape PowerShell supports inside single quotes). Inside single quotes,
// PowerShell performs zero interpolation: $, `, $( ), and @( ) are all
// literal. The ' doubling is the sole escape hatch — a lone ' would close the
// quoting and enable arbitrary command injection. The escaper also backtick-
// escapes $ outside single quotes (used in the wrapper temp-path) and handles
// %, &, ;, |, and control characters for defense-in-depth. NUL bytes are
// replaced with `0 to match PowerShell's escape convention.
func EscapePowerShellArg(arg string) string {
	arg = strings.ReplaceAll(arg, "`", "``")
	arg = strings.ReplaceAll(arg, "$", "`$")
	arg = strings.ReplaceAll(arg, "\"", "\\\"")
	arg = strings.ReplaceAll(arg, "'", "''")
	arg = strings.ReplaceAll(arg, "&", "`&")
	arg = strings.ReplaceAll(arg, ";", "`;")
	arg = strings.ReplaceAll(arg, "|", "`|")
	arg = strings.ReplaceAll(arg, "%", "``%")
	arg = strings.ReplaceAll(arg, "\n", "`n")
	arg = strings.ReplaceAll(arg, "\r", "`r")
	arg = strings.ReplaceAll(arg, "\t", "`t")
	arg = strings.ReplaceAll(arg, "\b", "`b")
	arg = strings.ReplaceAll(arg, "\x00", "`0")

	return "'" + arg + "'"
}

// ScriptConfig holds the parameters for generating a wrapper script.
type ScriptConfig struct {
	AuthDir    string
	TokenPath  string
	CliPath    string
	CliArgs    []string
	EnvVarName string
}

// GenerateWrapperScript creates a platform-appropriate wrapper script that
// loads the auth token, deletes the token file, and execs the CLI.
// Returns the script path, whether it is a Windows script, and any error.
func GenerateWrapperScript(cfg ScriptConfig) (string, bool, error) {
	if cfg.TokenPath == "" {
		return "", false, errors.NewError(errors.ValidationError,
			"wrapper: token path cannot be empty")
	}
	if cfg.CliPath == "" {
		return "", false, errors.NewError(errors.ValidationError,
			"wrapper: CLI path cannot be empty")
	}

	envVar := constants.EnvAuthToken
	if cfg.EnvVarName != "" {
		envVar = cfg.EnvVarName
	}

	isWindows := runtime.GOOS == constants.WindowsGOOS

	f, err := os.CreateTemp(cfg.AuthDir, "wrapper-")
	if err != nil {
		return "", false, errors.WrapError(errors.FileSystemError,
			"failed to create temp wrapper script", err)
	}

	scriptContent := generateScriptContent(isWindows, envVar, cfg)

	if _, err := f.WriteString(scriptContent); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())

		return "", false, errors.WrapError(errors.FileSystemError,
			"failed to write wrapper script", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())

		return "", false, errors.WrapError(errors.FileSystemError,
			"failed to close wrapper script", err)
	}

	if isWindows {
		ps1Path := f.Name() + ".ps1"
		if err := os.Rename(f.Name(), ps1Path); err != nil {
			_ = os.Remove(f.Name())

			return "", false, errors.WrapError(errors.FileSystemError,
				"failed to rename wrapper script", err)
		}

		return ps1Path, true, nil
	}

	if err := os.Chmod(f.Name(), constants.FilePermExec); err != nil {
		_ = os.Remove(f.Name())

		return "", false, errors.WrapError(errors.FileSystemError,
			"failed to make wrapper script executable", err)
	}

	return f.Name(), false, nil
}

func generateScriptContent(isWindows bool, envVar string, cfg ScriptConfig) string {
	if isWindows {
		return GenerateWindowsScript(envVar, cfg)
	}

	return generateUnixScript(envVar, cfg)
}

// GenerateWindowsScript returns the PowerShell script content for the wrapper.
func GenerateWindowsScript(envVar string, cfg ScriptConfig) string {
	var sb strings.Builder
	sb.WriteString("# Generated by kairo - DO NOT EDIT\r\n")
	sb.WriteString("# This script will be automatically deleted after execution\r\n")
	fmt.Fprintf(&sb, "$env:%s = Get-Content -Path %q -Raw\r\n", envVar, cfg.TokenPath)
	fmt.Fprintf(&sb, "Remove-Item -Path %q -Force\r\n", cfg.TokenPath)
	fmt.Fprintf(&sb, "& %q", cfg.CliPath)
	for _, arg := range cfg.CliArgs {
		fmt.Fprintf(&sb, " %s", EscapePowerShellArg(arg))
	}
	sb.WriteString("\r\n")

	return sb.String()
}

// shellQuotePOSIX wraps s in single quotes safe for /bin/sh.
// A single quote inside the string is escaped as '\” (end quote,
// literal single quote, resume quote). This defeats all shell
// expansions ($, `, \, !) because single quotes are the strongest
// quoting mechanism in POSIX sh.
func shellQuotePOSIX(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func generateUnixScript(envVar string, cfg ScriptConfig) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n")
	sb.WriteString("# Generated by kairo - DO NOT EDIT\n")
	sb.WriteString("# This script will be automatically deleted after execution\n")
	fmt.Fprintf(&sb, "export %s=$(cat %s)\n", envVar, shellQuotePOSIX(cfg.TokenPath))
	fmt.Fprintf(&sb, "rm -f %s\n", shellQuotePOSIX(cfg.TokenPath))
	sb.WriteString("exec ")
	sb.WriteString(shellQuotePOSIX(cfg.CliPath))
	for _, arg := range cfg.CliArgs {
		sb.WriteString(" ")
		sb.WriteString(shellQuotePOSIX(arg))
	}
	sb.WriteString("\n")

	return sb.String()
}

// ExecCommandContext creates an exec.Cmd for the given command and arguments.
func ExecCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}
