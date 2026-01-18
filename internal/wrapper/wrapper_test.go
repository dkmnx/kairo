package wrapper

import (
	"os"
	"path/filepath"
	"runtime"
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
	_, _, err := GenerateWrapperScript(authDir, "", "/usr/bin/claude", []string{"--help"})
	if err == nil {
		t.Error("GenerateWrapperScript() should error on empty token path")
	}
}

func TestGenerateWrapperScript_EmptyClaudePath(t *testing.T) {
	authDir := t.TempDir()
	tokenPath := filepath.Join(authDir, "token")
	_, _, err := GenerateWrapperScript(authDir, tokenPath, "", []string{"--help"})
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

	scriptPath, useCmdExe, err := GenerateWrapperScript(authDir, tokenPath, claudePath, args)
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

	// Verify token path is in script
	if !contains(string(content), tokenPath) {
		t.Error("Wrapper script should contain token path")
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

	scriptPath, useCmdExe, err := GenerateWrapperScript(authDir, tokenPath, claudePath, args)
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

	scriptPath, _, err := GenerateWrapperScript(authDir, tokenPath, claudePath, args)
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
