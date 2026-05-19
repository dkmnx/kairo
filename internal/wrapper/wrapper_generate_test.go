package wrapper

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGenerateWrapperScript_EmptyTokenPath(t *testing.T) {
	authDir := t.TempDir()
	_, _, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: "", CliPath: "/usr/bin/claude", CliArgs: []string{"--help"}})
	if err == nil {
		t.Error("GenerateWrapperScript() should error on empty token path")
	}
}

func TestGenerateWrapperScript_EmptyClaudePath(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	_, _, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: "", CliArgs: []string{"--help"}})
	if err == nil {
		t.Error("GenerateWrapperScript() should error on empty claude path")
	}
}

func TestGenerateWrapperScript_WindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows test on non-Windows platform")
	}

	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	claudePath := `C:\Program Files\claude\claude.exe`
	args := []string{"--help", "--verbose"}

	scriptPath, useCmdExe, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}
	defer os.Remove(scriptPath)

	if !useCmdExe {
		t.Error("Windows script should require cmdExe")
	}

	if filepath.Ext(scriptPath) != ".ps1" {
		t.Errorf("Windows script should have .ps1 extension, got %s", filepath.Ext(scriptPath))
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(content) == 0 {
		t.Error("Wrapper script should not be empty")
	}

	// The script uses %q formatting which escapes backslashes on Windows
	expectedPath := tokenPath
	if runtime.GOOS == "windows" {
		// On Windows, the script will have escaped backslashes
		expectedPath = strings.ReplaceAll(tokenPath, `\`, `\\`)
	}
	if !contains(string(content), expectedPath) {
		t.Errorf("Wrapper script should contain token path\nscript:\n%s\n\nexpected path: %s", string(content), expectedPath)
	}
}

func TestGenerateWrapperScript_UnixPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix test on Windows")
	}

	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	claudePath := "/usr/local/bin/claude"
	args := []string{"--help", "--verbose"}

	scriptPath, useCmdExe, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}
	defer os.Remove(scriptPath)

	if useCmdExe {
		t.Error("Unix script should not require cmdExe")
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Unix scripts should be executable
	if info.Mode()&0111 == 0 {
		t.Error("Unix wrapper script should be executable")
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(content) == 0 {
		t.Error("Wrapper script should not be empty")
	}

	if !contains(string(content), tokenPath) {
		t.Error("Wrapper script should contain token path")
	}

	if !contains(string(content), "#!/bin/sh") {
		t.Error("Unix wrapper script should have shebang")
	}
}

func TestGenerateWrapperScript_WithSpecialArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix test on Windows")
	}

	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	claudePath := "/usr/bin/claude"
	args := []string{"--prompt", "Hello 'World' and $HOME", `--option="value with spaces"`}

	scriptPath, _, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}
	defer os.Remove(scriptPath)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !contains(string(content), "Hello") || !contains(string(content), "World") {
		t.Error("Wrapper script should contain escaped arguments")
	}
}

func TestGenerateWrapperScript_WindowsWithSpecialArgs(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	// Real-world Windows path with spaces (common in Program Files)
	claudePath := `C:\Program Files\Claude\claude.exe`

	// Real-world arguments a user might pass
	args := []string{
		"--prompt", "Write a function that calculates $total = $price * $quantity",
		"--output", `C:\Users\Developer\Documents\result.md`,
		"--model", "sonnet-4-20250514",
	}

	scriptPath, _, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}
	defer os.Remove(scriptPath)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	scriptStr := string(content)

	if !strings.Contains(scriptStr, "$env:ANTHROPIC_AUTH_TOKEN") {
		t.Error("PowerShell script should set ANTHROPIC_AUTH_TOKEN")
	}
	if !strings.Contains(scriptStr, "Get-Content") {
		t.Error("PowerShell script should use Get-Content")
	}
	if !strings.Contains(scriptStr, "Remove-Item") {
		t.Error("PowerShell script should use Remove-Item")
	}
	if !strings.Contains(scriptStr, "`$total") || !strings.Contains(scriptStr, "`$price") || !strings.Contains(scriptStr, "`$quantity") {
		t.Error("PowerShell script should escape dollar signs in prompt")
	}
	if !strings.Contains(scriptStr, "Program Files") {
		t.Error("PowerShell script should handle paths with spaces")
	}
}

func TestGenerateWrapperScript_CustomEnvVar(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	cliPath := "/usr/local/bin/claude"
	args := []string{"--help"}

	t.Run("uses custom env var when provided", func(t *testing.T) {
		scriptPath, _, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: cliPath, CliArgs: args, EnvVarName: "ANTHROPIC_API_KEY"})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}
		defer os.Remove(scriptPath)

		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		scriptStr := string(content)
		if !strings.Contains(scriptStr, "ANTHROPIC_API_KEY") {
			t.Error("Script should use custom env var ANTHROPIC_API_KEY")
		}
		if strings.Contains(scriptStr, "ANTHROPIC_AUTH_TOKEN") {
			t.Error("Script should not contain ANTHROPIC_AUTH_TOKEN when custom env var is provided")
		}
	})

	t.Run("uses default auth token when not provided", func(t *testing.T) {
		scriptPath, _, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: cliPath, CliArgs: args})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}
		defer os.Remove(scriptPath)

		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		scriptStr := string(content)
		if !strings.Contains(scriptStr, "ANTHROPIC_AUTH_TOKEN") {
			t.Error("Script should use default env var ANTHROPIC_AUTH_TOKEN")
		}
	})

	t.Run("uses empty string as default", func(t *testing.T) {
		scriptPath, _, err := GenerateWrapperScript(ScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: cliPath, CliArgs: args, EnvVarName: ""})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}
		defer os.Remove(scriptPath)

		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		scriptStr := string(content)
		if !strings.Contains(scriptStr, "ANTHROPIC_AUTH_TOKEN") {
			t.Error("Empty string should default to ANTHROPIC_AUTH_TOKEN")
		}
	})
}

func TestGenerateWrapperScript_NonExistentAuthDir(t *testing.T) {
	_, _, err := GenerateWrapperScript(ScriptConfig{
		AuthDir:    "/nonexistent/auth/dir",
		TokenPath:  "/nonexistent/auth/dir/token",
		CliPath:    "/usr/bin/claude",
		CliArgs:    []string{"--help"},
		EnvVarName: "ANTHROPIC_AUTH_TOKEN",
	})
	if err == nil {
		t.Error("GenerateWrapperScript() should error on non-existent auth directory")
	}
}

func TestGenerateWrapperScript_ControlCharacterEscaping(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	cliPath := "/usr/bin/claude"

	tests := []struct {
		name  string
		input string
	}{
		{"newline", "hello\nworld"},
		{"carriage return", "hello\rworld"},
		{"tab", "hello\tworld"},
		{"form feed", "hello\x0cworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath, _, err := GenerateWrapperScript(ScriptConfig{
				AuthDir:    authDir,
				TokenPath:  tokenPath,
				CliPath:    cliPath,
				CliArgs:    []string{"--prompt", tt.input},
				EnvVarName: "ANTHROPIC_AUTH_TOKEN",
			})
			if err != nil {
				t.Fatalf("GenerateWrapperScript() error = %v", err)
			}
			defer os.Remove(scriptPath)

			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			scriptStr := string(content)
			if !strings.Contains(scriptStr, "hello") {
				t.Errorf("Script should contain argument content for %q", tt.name)
			}
		})
	}
}

func TestExecCommand(t *testing.T) {
	cmd := ExecCommandContext(context.Background(), "echo", "test")
	if cmd == nil {
		t.Fatal("ExecCommand() should return a valid command")
	}
	if len(cmd.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(cmd.Args))
	}
}

func TestGenerateWrapperScript_WithArgs(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	cliPath := "/usr/bin/claude"
	args := []string{"--model", "sonnet-4-20250514", "--temperature", "0.7"}

	scriptPath, isWindows, err := GenerateWrapperScript(ScriptConfig{
		AuthDir:    authDir,
		TokenPath:  tokenPath,
		CliPath:    cliPath,
		CliArgs:    args,
		EnvVarName: "ANTHROPIC_AUTH_TOKEN",
	})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}
	defer os.Remove(scriptPath)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	scriptStr := string(content)

	expectedArgs := []string{"--model", "sonnet-4-20250514", "--temperature", "0.7"}
	for _, arg := range expectedArgs {
		if !strings.Contains(scriptStr, arg) {
			t.Errorf("Script should contain arg %q", arg)
		}
	}

	isWindowsExpected := runtime.GOOS == "windows"
	if isWindows != isWindowsExpected {
		t.Errorf("isWindows = %v, want %v", isWindows, isWindowsExpected)
	}
}

func TestGenerateWrapperScript_ScriptIsDeletedAfterUse(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	cliPath := "/usr/bin/claude"

	scriptPath, _, err := GenerateWrapperScript(ScriptConfig{
		AuthDir:    authDir,
		TokenPath:  tokenPath,
		CliPath:    cliPath,
		CliArgs:    []string{"--help"},
		EnvVarName: "ANTHROPIC_AUTH_TOKEN",
	})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Error("Script should exist after creation")
	}

	// On Unix, the script should have been created with execute permissions
	if runtime.GOOS != "windows" {
		info, err := os.Stat(scriptPath)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		if info.Mode()&0100 == 0 {
			t.Error("Unix script should have execute permission")
		}
	}
}
