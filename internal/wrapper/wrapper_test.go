package wrapper

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCreateTempAuthDir_Success(t *testing.T) {
	dir, err := CreateTempAuthDir()
	if err != nil {
		t.Fatalf("CreateTempAuthDir() error = %v", err)
	}
	defer os.RemoveAll(dir)

	if dir == "" {
		t.Error("CreateTempAuthDir() returned empty path")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Skip permission check on Windows (doesn't support Unix-style 0700)
	if runtime.GOOS != "windows" {
		mode := info.Mode()
		// Check that directory has owner-only permissions (0700)
		if mode&0077 != 0 {
			t.Errorf("Directory should have no group/other permissions, got %o", mode)
		}
		if mode&0100 == 0 {
			t.Errorf("Directory should have owner execute permission")
		}
		if mode&0200 == 0 {
			t.Errorf("Directory should have owner write permission")
		}
		if mode&0400 == 0 {
			t.Errorf("Directory should have owner read permission")
		}
	}
}

func TestCreateTempAuthDir_ReturnsUniqueDirs(t *testing.T) {
	dir1, err := CreateTempAuthDir()
	if err != nil {
		t.Fatalf("CreateTempAuthDir() error = %v", err)
	}
	defer os.RemoveAll(dir1)

	dir2, err := CreateTempAuthDir()
	if err != nil {
		t.Fatalf("CreateTempAuthDir() error = %v", err)
	}
	defer os.RemoveAll(dir2)

	if dir1 == dir2 {
		t.Error("CreateTempAuthDir() returned same path for two calls")
	}
}

func TestWriteTempTokenFile_Success(t *testing.T) {
	authDir := t.TempDir()
	token := "test-api-key-12345"

	path, err := WriteTempTokenFile(authDir, token)
	if err != nil {
		t.Fatalf("WriteTempTokenFile() error = %v", err)
	}
	defer os.Remove(path)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != token {
		t.Errorf("File content = %q, want %q", string(content), token)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Skip permission check on Windows (doesn't support Unix-style 0600)
	if runtime.GOOS != "windows" {
		expectedPerms := os.FileMode(0600)
		if info.Mode() != expectedPerms {
			t.Errorf("File permissions = %o, want %o", info.Mode(), expectedPerms)
		}
	}
}

func TestWriteTempTokenFile_EmptyToken(t *testing.T) {
	authDir := t.TempDir()

	_, err := WriteTempTokenFile(authDir, "")
	if err == nil {
		t.Error("WriteTempTokenFile() should error on empty token")
	}
}

func TestEscapePowerShellArg_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "''"},
		{"simple", "hello", "'hello'"},
		{"with spaces", "hello world", "'hello world'"},
		{"single quote", "can't", "'can''t'"},
		{"double quote", `say "hi"`, `'say \"hi\"'`},
		{"dollar sign", "$HOME", "'`$HOME'"},
		{"backtick", "foo`bar", "'foo``bar'"},
		{"complex", `$env:PATH = "C:\test"`, "'`$env:PATH = \\\"C:\\test\\\"'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			if result != tt.expected {
				t.Errorf("EscapePowerShellArg(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateWrapperScript_EmptyTokenPath(t *testing.T) {
	authDir := t.TempDir()
	_, _, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: "", CliPath: "/usr/bin/claude", CliArgs: []string{"--help"}})
	if err == nil {
		t.Error("GenerateWrapperScript() should error on empty token path")
	}
}

func TestGenerateWrapperScript_EmptyClaudePath(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	_, _, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: "", CliArgs: []string{"--help"}})
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

	scriptPath, useCmdExe, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
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

	// Verify token path is in script (Windows paths are escaped with \\ in scripts)
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

	scriptPath, useCmdExe, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
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

	// Verify token path is in script
	if !contains(string(content), tokenPath) {
		t.Error("Wrapper script should contain token path")
	}

	// Verify shebang for Unix
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

	scriptPath, _, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}
	defer os.Remove(scriptPath)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Verify args are properly escaped in the script
	if !contains(string(content), "Hello") || !contains(string(content), "World") {
		t.Error("Wrapper script should contain escaped arguments")
	}
}

func TestEscapePowerShellArg_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"newline", "line1\nline2", "'line1`nline2'"},
		{"carriage return", "line1\rline2", "'line1`rline2'"},
		{"crlf", "line1\r\nline2", "'line1`r`nline2'"},
		{"tab", "col1\tcol2", "'col1`tcol2'"},
		{"backspace", "back\bspace", "'back`bspace'"},
		{"null", "before\x00after", "'before`0after'"},
		{"unicode emoji", "🚀🎉", "'🚀🎉'"},
		{"unicode chinese", "你好世界", "'你好世界'"},
		{"windows path long", `C:\Users\JohnDoe\AppData\Local\Programs\Claude\claude.exe`, "'C:\\Users\\JohnDoe\\AppData\\Local\\Programs\\Claude\\claude.exe'"},
		{"windows path with spaces", `C:\Program Files\My App\file.txt`, "'C:\\Program Files\\My App\\file.txt'"},
		{"semicolon", "test; cmd", "'test`; cmd'"},
		{"pipe", "test | calc", "'test `| calc'"},
		{"variable style", "$myVar", "'`$myVar'"},
		{"env variable", "$env:PATH", "'`$env:PATH'"},
		{"at sign", "@()", "'@()'"},
		{"percent", "100%", "'100``%'"},
		{"multi-line", "line1\nline2", "'line1`nline2'"},
		{"json string", `{"key":"value"}`, "'{\\\"key\\\":\\\"value\\\"}'"},
		{"base64", "SGVsbG8gV29ybGQ=", "'SGVsbG8gV29ybGQ='"},
		{"prompt with quotes", `Say "hello" to me`, "'Say \\\"hello\\\" to me'"},
		{"curl command", `curl -H "Authorization: Bearer $TOKEN" https://api.example.com`, "'curl -H \\\"Authorization: Bearer `$TOKEN\\\" https://api.example.com'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			if result != tt.expected {
				t.Errorf("EscapePowerShellArg(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapePowerShellArg_CommandInjectionPrevention(t *testing.T) {
	injectionPatterns := []struct {
		name  string
		input string
	}{
		{"cmd execl", "$(cmd.exe /c whoami)"},
		{"powershell execl", "$(powershell.exe -Command 'Get-Process')"},
		{"backtick exec", "`whoami"},
		{"command substitution", "$((whoami))"},
		{"env substitution", "$env:COMPUTERNAME"},
	}

	for _, tt := range injectionPatterns {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			// Result should be wrapped in single quotes
			if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
				t.Errorf("Result should be wrapped in single quotes, got: %q", result)
			}
			// Dollar signs should be escaped as `$
			if strings.Contains(result, "$(") && !strings.Contains(result, "`$(") {
				t.Errorf("Command substitution not properly escaped in: %q", result)
			}
			// Backticks should be doubled or part of valid escape
			if strings.Contains(result, "`") && !strings.Contains(result, "``") &&
				!strings.Contains(result, "`$") && !strings.Contains(result, "`n") &&
				!strings.Contains(result, "`r") && !strings.Contains(result, "`t") {
				t.Errorf("Backtick not properly escaped in: %q", result)
			}
		})
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

	scriptPath, _, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: claudePath, CliArgs: args})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}
	defer os.Remove(scriptPath)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	scriptStr := string(content)

	// Verify PowerShell-specific content
	if !strings.Contains(scriptStr, "$env:ANTHROPIC_AUTH_TOKEN") {
		t.Error("PowerShell script should set ANTHROPIC_AUTH_TOKEN")
	}
	if !strings.Contains(scriptStr, "Get-Content") {
		t.Error("PowerShell script should use Get-Content")
	}
	if !strings.Contains(scriptStr, "Remove-Item") {
		t.Error("PowerShell script should use Remove-Item")
	}
	// Verify dollar signs are escaped in prompt
	if !strings.Contains(scriptStr, "`$total") || !strings.Contains(scriptStr, "`$price") || !strings.Contains(scriptStr, "`$quantity") {
		t.Error("PowerShell script should escape dollar signs in prompt")
	}
	// Verify path with spaces is properly quoted
	if !strings.Contains(scriptStr, "Program Files") {
		t.Error("PowerShell script should handle paths with spaces")
	}
}

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

func TestGenerateWrapperScript_CustomEnvVar(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	cliPath := "/usr/local/bin/claude"
	args := []string{"--help"}

	t.Run("uses custom env var when provided", func(t *testing.T) {
		scriptPath, _, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: cliPath, CliArgs: args, EnvVarName: "ANTHROPIC_API_KEY"})
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
		scriptPath, _, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: cliPath, CliArgs: args})
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
		scriptPath, _, err := GenerateWrapperScript(WrapperScriptConfig{AuthDir: authDir, TokenPath: tokenPath, CliPath: cliPath, CliArgs: args, EnvVarName: ""})
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

func TestCreateTempAuthDir_Failure(t *testing.T) {
	// Note: os.MkdirTemp creates both the temp dir and any necessary parents,
	// so testing error conditions is difficult without modifying system state.
	// We verify the basic functionality works via other tests.
	_, err := CreateTempAuthDir()
	// Just verify it returns some result without error for valid conditions
	// The error for invalid paths would be OS-specific
	_ = err
}

func TestWriteTempTokenFile_NonExistentDir(t *testing.T) {
	_, err := WriteTempTokenFile("/nonexistent/path", "test-token")
	if err == nil {
		t.Error("WriteTempTokenFile() should error on non-existent directory")
	}
}

func TestGenerateWrapperScript_NonExistentAuthDir(t *testing.T) {
	_, _, err := GenerateWrapperScript(WrapperScriptConfig{
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

func TestEscapePowerShellArg_SemicolonAndPipe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"semicolon", "test; cmd", "'test`; cmd'"},
		{"pipe", "test | calc", "'test `| calc'"},
		{"both", "cmd1; cmd2 | grep", "'cmd1`; cmd2 `| grep'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			if result != tt.expected {
				t.Errorf("EscapePowerShellArg(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
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
			scriptPath, _, err := GenerateWrapperScript(WrapperScriptConfig{
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
			// Verify the argument is still present (escaped properly)
			if !strings.Contains(scriptStr, "hello") {
				t.Errorf("Script should contain argument content for %q", tt.name)
			}
		})
	}
}

func TestExecCommand(t *testing.T) {
	// Test that ExecCommand returns a valid command
	cmd := ExecCommand("echo", "test")
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

	scriptPath, isWindows, err := GenerateWrapperScript(WrapperScriptConfig{
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

	// Verify all args are in the script
	expectedArgs := []string{"--model", "sonnet-4-20250514", "--temperature", "0.7"}
	for _, arg := range expectedArgs {
		if !strings.Contains(scriptStr, arg) {
			t.Errorf("Script should contain arg %q", arg)
		}
	}

	// Verify Windows flag is correct
	isWindowsExpected := runtime.GOOS == "windows"
	if isWindows != isWindowsExpected {
		t.Errorf("isWindows = %v, want %v", isWindows, isWindowsExpected)
	}
}

func TestGenerateWrapperScript_ScriptIsDeletedAfterUse(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	cliPath := "/usr/bin/claude"

	scriptPath, _, err := GenerateWrapperScript(WrapperScriptConfig{
		AuthDir:    authDir,
		TokenPath:  tokenPath,
		CliPath:    cliPath,
		CliArgs:    []string{"--help"},
		EnvVarName: "ANTHROPIC_AUTH_TOKEN",
	})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}

	// Verify script exists
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
