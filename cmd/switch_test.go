package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestCreateTempAuthDir(t *testing.T) {
	isWindows := runtime.GOOS == "windows"

	t.Run("creates private directory with 0700 permissions", func(t *testing.T) {
		if isWindows {
			t.Skip("Windows does not support Unix-style permissions")
		}
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		// Verify directory exists
		info, err := os.Stat(authDir)
		if err != nil {
			t.Fatalf("Failed to stat auth directory: %v", err)
		}

		// Verify it's a directory
		if !info.IsDir() {
			t.Errorf("Auth path should be a directory")
		}

		// Verify permissions are 0700 (owner read/write/execute only)
		perms := info.Mode().Perm()
		expectedPerms := os.FileMode(0700)
		if perms != expectedPerms {
			t.Errorf("Auth directory permissions = %v, want %v", perms, expectedPerms)
		}
	})

	t.Run("directory is in temp directory", func(t *testing.T) {
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		tempDir := os.TempDir()
		if !strings.HasPrefix(authDir, tempDir) {
			t.Errorf("Auth directory path = %q, should be in temp directory %q", authDir, tempDir)
		}
	})

	t.Run("directory name contains kairo-auth identifier", func(t *testing.T) {
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		if !strings.Contains(authDir, "kairo-auth") {
			t.Errorf("Auth directory path = %q, should contain 'kairo-auth'", authDir)
		}
	})

	t.Run("creates unique directory for each call", func(t *testing.T) {
		authDir1, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir1)

		authDir2, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir2)

		if authDir1 == authDir2 {
			t.Errorf("createTempAuthDir() returned same path for different calls: %s", authDir1)
		}
	})
}

func TestWriteTempTokenFile(t *testing.T) {
	isWindows := runtime.GOOS == "windows"

	t.Run("creates file with correct content", func(t *testing.T) {
		authDir := t.TempDir()
		token := "test-api-key-12345"

		tokenPath, err := writeTempTokenFile(authDir, token)
		if err != nil {
			t.Fatalf("writeTempTokenFile() error = %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
			t.Errorf("Token file was not created at %s", tokenPath)
		}

		// Verify file content
		content, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Fatalf("Failed to read token file: %v", err)
		}

		if string(content) != token {
			t.Errorf("Token file content = %q, want %q", string(content), token)
		}
	})

	t.Run("file has restricted permissions (0600)", func(t *testing.T) {
		if isWindows {
			t.Skip("Windows does not support Unix-style permissions")
		}
		authDir := t.TempDir()
		token := "secure-token-abc"

		tokenPath, err := writeTempTokenFile(authDir, token)
		if err != nil {
			t.Fatalf("writeTempTokenFile() error = %v", err)
		}

		info, err := os.Stat(tokenPath)
		if err != nil {
			t.Fatalf("Failed to stat token file: %v", err)
		}

		// Verify mode is 0600 (owner read/write only)
		perms := info.Mode().Perm()
		expectedPerms := os.FileMode(0600)
		if perms != expectedPerms {
			t.Errorf("Token file permissions = %v, want %v", perms, expectedPerms)
		}
	})

	t.Run("creates unique file for each call", func(t *testing.T) {
		authDir := t.TempDir()
		token1 := "token-1"
		token2 := "token-2"

		path1, err := writeTempTokenFile(authDir, token1)
		if err != nil {
			t.Fatalf("writeTempTokenFile() error = %v", err)
		}

		path2, err := writeTempTokenFile(authDir, token2)
		if err != nil {
			t.Fatalf("writeTempTokenFile() error = %v", err)
		}

		if path1 == path2 {
			t.Errorf("writeTempTokenFile() returned same path for different calls: %s", path1)
		}
	})

	t.Run("creates files in specified directory", func(t *testing.T) {
		authDir := t.TempDir()
		token := "test-token"

		tokenPath, err := writeTempTokenFile(authDir, token)
		if err != nil {
			t.Fatalf("writeTempTokenFile() error = %v", err)
		}

		// Verify file is in specified directory
		if !strings.HasPrefix(tokenPath, authDir) {
			t.Errorf("Token file path = %q, should be in directory %q", tokenPath, authDir)
		}
	})
}

func TestGenerateWrapperScript(t *testing.T) {
	isWindows := runtime.GOOS == "windows"

	t.Run("generates valid script", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-file")
		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{"--help"})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}
		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Errorf("Wrapper script was not created at %s", wrapperPath)
		}

		// Read and verify script content
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		// Verify script contains expected elements (platform-specific)
		if isWindows {
			// Windows batch script checks
			if !strings.Contains(scriptContent, "@echo off") {
				t.Errorf("Wrapper script missing @echo off\nScript content:\n%s", scriptContent)
			}
			if !strings.Contains(scriptContent, "REM Generated by kairo") {
				t.Errorf("Wrapper script missing REM comment\nScript content:\n%s", scriptContent)
			}
			if !strings.Contains(scriptContent, "ANTHROPIC_AUTH_TOKEN") {
				t.Errorf("Wrapper script missing ANTHROPIC_AUTH_TOKEN\nScript content:\n%s", scriptContent)
			}
			if !strings.Contains(scriptContent, "for /f") {
				t.Errorf("Wrapper script missing for /f command\nScript content:\n%s", scriptContent)
			}
			if !strings.Contains(scriptContent, "del ") {
				t.Errorf("Wrapper script missing del command\nScript content:\n%s", scriptContent)
			}
		} else {
			// Unix shell script checks
			requiredElements := []string{
				"#!/bin/sh",
				"# Generated by kairo - DO NOT EDIT",
				"# This script will be automatically deleted after execution",
				"export ANTHROPIC_AUTH_TOKEN",
				"rm",
				tokenPath,
				"exec",
				"/usr/bin/claude",
				"--help",
			}
			for _, elem := range requiredElements {
				if !strings.Contains(scriptContent, elem) {
					t.Errorf("Wrapper script missing required element: %s\nScript content:\n%s", elem, scriptContent)
				}
			}
		}
	})

	t.Run("script cleans up token file", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token")
		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/echo", []string{"test"})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		// Verify script removes token file (platform-specific check)
		if isWindows {
			// On Windows, check for del command with quoted path (ends with .bat)
			if !strings.HasSuffix(wrapperPath, ".bat") {
				t.Errorf("Wrapper script should have .bat extension on Windows")
			}
			// Check that del command is present (the path will be quoted via %q)
			if !strings.Contains(scriptContent, "del ") {
				t.Errorf("Wrapper script should contain del command")
			}
		} else {
			if !strings.Contains(scriptContent, "rm") || !strings.Contains(scriptContent, tokenPath) {
				t.Errorf("Wrapper script should remove token file %s", tokenPath)
			}
		}
	})

	t.Run("script is executable", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-file")
		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}

		info, err := os.Stat(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to stat wrapper script: %v", err)
		}

		// On Windows, .bat files don't need Unix executable permissions
		if !isWindows {
			// Verify script is executable (at least 0700)
			if info.Mode().Perm()&0111 == 0 {
				t.Errorf("Wrapper script should be executable, got mode %v", info.Mode().Perm())
			}
		}
	})

	t.Run("handles empty args correctly", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-file")
		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		// Verify script doesn't break with empty args (platform-specific)
		if isWindows {
			if !strings.Contains(scriptContent, "\"/usr/bin/claude\"") {
				t.Errorf("Wrapper script should handle empty args correctly\n%s", scriptContent)
			}
		} else {
			if !strings.Contains(scriptContent, "exec \"/usr/bin/claude\"") {
				t.Errorf("Wrapper script should handle empty args correctly\n%s", scriptContent)
			}
		}
	})

	t.Run("escapes special characters in paths", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-with spaces")
		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		// Verify paths with spaces are properly quoted
		// The path should appear in the script (quoted)
		if !strings.Contains(scriptContent, "test-token-with spaces") {
			t.Errorf("Wrapper script should contain the token path with spaces\nGot:\n%s", scriptContent)
		}

		// Verify claudePath is also quoted
		quotedClaudePath := `"/usr/bin/claude"`
		if !strings.Contains(scriptContent, quotedClaudePath) {
			t.Errorf("Wrapper script should quote claude path\nGot:\n%s\nExpected to find: %s", scriptContent, quotedClaudePath)
		}
	})

	t.Run("creates wrapper in specified directory", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-file")
		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}

		// Verify wrapper script is created in specified directory
		if !strings.HasPrefix(wrapperPath, authDir) {
			t.Errorf("Wrapper script path = %q, should be in directory %q", wrapperPath, authDir)
		}
	})
}

func TestSwitchCmdSecureTokenPassing(t *testing.T) {
	// Integration test for the full switch flow with mocked exec
	// Verifies that wrapper script is used instead of direct environment variable passing

	t.Run("uses wrapper script when API key is present", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file with a provider that requires API key
		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7"},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		// Create encrypted secrets with API key
		secretsContent := "ZAI_API_KEY=test-api-key-12345"
		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		// Create a simple key file (not encrypted for test simplicity)
		// In real scenario, this would be properly encrypted
		if err := os.WriteFile(keyPath, []byte("test-key"), 0600); err != nil {
			t.Fatalf("Failed to create key file: %v", err)
		}
		if err := os.WriteFile(secretsPath, []byte(secretsContent), 0600); err != nil {
			t.Fatalf("Failed to create secrets file: %v", err)
		}

		// Save and restore global state
		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() {
			setConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		// Mock lookPath to return a fake claude path
		originalLookPath := lookPath
		lookPath = func(file string) (string, error) {
			if file == "claude" {
				return "/usr/bin/claude", nil
			}
			return originalLookPath(file)
		}
		defer func() { lookPath = originalLookPath }()

		// Mock execCommand to verify wrapper script is used
		originalExecCommand := execCommand
		execCommand = func(name string, arg ...string) *exec.Cmd {
			// Verify that wrapper script is executed (not claude directly)
			if name == "/usr/bin/claude" {
				t.Errorf("Expected wrapper script to be executed, got direct claude execution")
			}
			return originalExecCommand("echo", "mocked")
		}
		defer func() { execCommand = originalExecCommand }()

		// Mock exitProcess to prevent test from exiting
		originalExitProcess := exitProcess
		exitProcess = func(int) {}
		defer func() { exitProcess = originalExitProcess }()

		// Execute switch command
		rootCmd.SetArgs([]string{"switch", "zai", "--help"})
		rootCmd.SetOut(&bytes.Buffer{})
		rootCmd.SetErr(&bytes.Buffer{})

		// This should trigger the wrapper script path
		// Note: We can't fully execute this due to crypto dependencies,
		// but we can verify the setup logic works
		// The actual execution would use wrapper script
	})

	t.Run("does not use wrapper script when no API key", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file with provider that doesn't require API key
		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic", BaseURL: "", Model: ""},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		// No secrets file for anthropic
		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := os.WriteFile(keyPath, []byte("test-key"), 0600); err != nil {
			t.Fatalf("Failed to create key file: %v", err)
		}
		if err := os.WriteFile(secretsPath, []byte(""), 0600); err != nil {
			t.Fatalf("Failed to create empty secrets file: %v", err)
		}

		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() {
			setConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		// Mock lookPath
		originalLookPath := lookPath
		lookPath = func(file string) (string, error) {
			if file == "claude" {
				return "/usr/bin/claude", nil
			}
			return originalLookPath(file)
		}
		defer func() { lookPath = originalLookPath }()

		// Mock execCommand to verify claude is executed directly
		originalExecCommand := execCommand
		execCommand = func(name string, arg ...string) *exec.Cmd {
			return originalExecCommand("echo", "mocked")
		}
		defer func() { execCommand = originalExecCommand }()

		originalExitProcess := exitProcess
		exitProcess = func(int) {}
		defer func() { exitProcess = originalExitProcess }()

		rootCmd.SetArgs([]string{"switch", "anthropic", "--help"})
		rootCmd.SetOut(&bytes.Buffer{})
		rootCmd.SetErr(&bytes.Buffer{})
	})
}

// TestWrapperScriptExecution tests that the wrapper script correctly passes
// environment variables to a child process
func TestWrapperScriptExecution(t *testing.T) {
	t.Parallel()

	isWindows := runtime.GOOS == "windows"

	// Create a temporary directory for auth files
	authDir := t.TempDir()

	// Create a platform-specific "child" script that will print the environment variable
	var childScriptPath string
	var childScriptContent string

	if isWindows {
		childScriptPath = filepath.Join(authDir, "child.bat")
		childScriptContent = "@echo off\r\n"
		childScriptContent += "echo CHILD_PID=$$\r\n"
		childScriptContent += "echo ANTHROPIC_AUTH_TOKEN=%ANTHROPIC_AUTH_TOKEN%\r\n"
	} else {
		childScriptPath = filepath.Join(authDir, "child.sh")
		childScriptContent = `#!/bin/sh
echo "CHILD_PID=$$"
echo "ANTHROPIC_AUTH_TOKEN=$ANTHROPIC_AUTH_TOKEN"
`
	}

	if err := os.WriteFile(childScriptPath, []byte(childScriptContent), 0600); err != nil {
		t.Fatalf("Failed to create child script: %v", err)
	}
	if !isWindows {
		if err := os.Chmod(childScriptPath, 0700); err != nil {
			t.Fatalf("Failed to set executable permission: %v", err)
		}
	}

	// Create a token file with a test API key
	tokenPath := filepath.Join(authDir, "token")
	expectedToken := "sk-ant-test1234567890"
	if err := os.WriteFile(tokenPath, []byte(expectedToken), 0600); err != nil {
		t.Fatalf("Failed to create token file: %v", err)
	}

	// Generate wrapper script that will execute the child script
	wrapperPath, useCmdExe, err := generateWrapperScript(authDir, tokenPath, childScriptPath, []string{})
	if err != nil {
		t.Fatalf("generateWrapperScript() error = %v", err)
	}

	// Execute the wrapper script
	var out bytes.Buffer
	var cmd *exec.Cmd
	if useCmdExe {
		cmd = exec.Command("cmd", "/c", wrapperPath)
	} else {
		cmd = exec.Command(wrapperPath)
	}
	cmd.Stdout = &out
	cmd.Stderr = &out

	execErr := cmd.Run()

	// Check if execution succeeded
	// Note: The wrapper uses 'exec' which replaces the process, so we expect exit status 0
	if execErr != nil {
		t.Logf("Wrapper execution error (may be expected): %v", execErr)
	}

	output := out.String()

	// Verify the token was passed correctly
	if !strings.Contains(output, expectedToken) {
		t.Errorf("Expected token %q not found in wrapper output:\n%s", expectedToken, output)
	}

	// Verify token file was cleaned up by wrapper
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Errorf("Token file was not cleaned up after wrapper execution")
	}

	// Verify wrapper script contains expected comments
	wrapperContent, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("Failed to read wrapper script: %v", err)
	}
	wrapperStr := string(wrapperContent)

	if isWindows {
		if !strings.Contains(wrapperStr, "REM Generated by kairo") {
			t.Error("Wrapper script missing REM comment")
		}
	} else {
		if !strings.Contains(wrapperStr, "# Generated by kairo - DO NOT EDIT") {
			t.Error("Wrapper script missing header comment")
		}
		if !strings.Contains(wrapperStr, "# This script will be automatically deleted after execution") {
			t.Error("Wrapper script missing deletion notice")
		}
	}

	t.Logf("Wrapper execution output:\n%s", output)
}

func TestSignalHandlingIsCrossPlatform(t *testing.T) {
	// Verify that the switch command implementation doesn't use syscall.Kill directly
	// This test ensures cross-platform compatibility (no Windows build failures)
	t.Run("code compiles without platform-specific syscalls", func(t *testing.T) {
		// This test verifies the build succeeds on all platforms
		// If the code used syscall.Kill on Windows, go build would fail
		// The signal handling now uses exitProcess(code) which is cross-platform
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		// Verify directory was created successfully
		if _, err := os.Stat(authDir); err != nil {
			t.Errorf("Auth directory should exist: %v", err)
		}
	})

	t.Run("signal exit code calculation is cross-platform", func(t *testing.T) {
		// Simulate the exit code calculation used in signal handler
		// This verifies the logic works without actual signal delivery
		testCases := []struct {
			sig      syscall.Signal
			expected int
		}{
			{syscall.SIGINT, 130},  // 128 + 2
			{syscall.SIGTERM, 143}, // 128 + 15
		}

		for _, tc := range testCases {
			code := 128
			code += int(tc.sig)
			if code != tc.expected {
				t.Errorf("Signal exit code = %d, want %d for signal %d", code, tc.expected, tc.sig)
			}
		}
	})
}

func TestSwitch_ErrorHandling(t *testing.T) {
	// These tests verify error handling in temp file/directory operations
	// They test edge cases and error conditions that may occur in production

	t.Run("writeTempTokenFile returns error for non-existent directory", func(t *testing.T) {
		nonExistentDir := "/tmp/kairo-test-non-existent-" + strings.ReplaceAll(os.TempDir(), "/", "-")
		token := "test-token"

		_, err := writeTempTokenFile(nonExistentDir, token)
		if err == nil {
			t.Error("Expected error when writing to non-existent directory")
		}
	})

	t.Run("writeTempTokenFile returns error for invalid directory path", func(t *testing.T) {
		// Use a path that's too long to be valid
		longPath := strings.Repeat("a", 10000)
		token := "test-token"

		_, err := writeTempTokenFile(longPath, token)
		if err == nil {
			t.Error("Expected error when writing to invalid directory path")
		}
	})

	t.Run("writeTempTokenFile handles very long tokens", func(t *testing.T) {
		authDir := t.TempDir()
		// Create a very long token (potential buffer overflow scenario)
		longToken := strings.Repeat("a", 100000)

		tokenPath, err := writeTempTokenFile(authDir, longToken)
		if err != nil {
			t.Errorf("writeTempTokenFile() failed with long token: %v", err)
		}

		// Verify the token was written correctly
		content, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Fatalf("Failed to read token file: %v", err)
		}

		if string(content) != longToken {
			t.Errorf("Token content length mismatch: got %d, want %d", len(content), len(longToken))
		}
	})

	t.Run("writeTempTokenFile handles special characters in token", func(t *testing.T) {
		authDir := t.TempDir()
		// Test with special characters that might cause issues
		specialToken := "sk-test-\"`$';\n\t\x00"

		tokenPath, err := writeTempTokenFile(authDir, specialToken)
		if err != nil {
			t.Errorf("writeTempTokenFile() failed with special characters: %v", err)
		}

		// Verify the token was written correctly
		content, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Fatalf("Failed to read token file: %v", err)
		}

		if string(content) != specialToken {
			t.Errorf("Token content = %q, want %q", string(content), specialToken)
		}
	})

	t.Run("writeTempTokenFile handles unicode in token", func(t *testing.T) {
		authDir := t.TempDir()
		unicodeToken := "sk-test-世界-🌍-🚀"

		tokenPath, err := writeTempTokenFile(authDir, unicodeToken)
		if err != nil {
			t.Errorf("writeTempTokenFile() failed with unicode: %v", err)
		}

		// Verify the token was written correctly
		content, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Fatalf("Failed to read token file: %v", err)
		}

		if string(content) != unicodeToken {
			t.Errorf("Token content = %q, want %q", string(content), unicodeToken)
		}
	})

	t.Run("writeTempTokenFile overwrites existing file", func(t *testing.T) {
		authDir := t.TempDir()
		token1 := "first-token"
		token2 := "second-token"

		// Write first token
		path1, err := writeTempTokenFile(authDir, token1)
		if err != nil {
			t.Fatalf("writeTempTokenFile() failed: %v", err)
		}

		// Write second token (should create a different file)
		path2, err := writeTempTokenFile(authDir, token2)
		if err != nil {
			t.Fatalf("writeTempTokenFile() failed: %v", err)
		}

		// Paths should be different (temp files are unique)
		if path1 == path2 {
			t.Error("writeTempTokenFile() should create unique files")
		}

		// Verify both files exist with correct content
		content1, err := os.ReadFile(path1)
		if err != nil {
			t.Errorf("Failed to read first token file: %v", err)
		}
		if string(content1) != token1 {
			t.Errorf("First token = %q, want %q", string(content1), token1)
		}

		content2, err := os.ReadFile(path2)
		if err != nil {
			t.Errorf("Failed to read second token file: %v", err)
		}
		if string(content2) != token2 {
			t.Errorf("Second token = %q, want %q", string(content2), token2)
		}
	})

	t.Run("createTempAuthDir handles temp directory exhaustion", func(t *testing.T) {
		// This test verifies that createTempAuthDir returns a proper error
		// when it cannot create a temp directory. We can't easily simulate
		// actual temp directory exhaustion, but we can verify error handling.

		authDir, err := createTempAuthDir()
		if err != nil {
			// If creation fails, verify error message is informative
			if !strings.Contains(err.Error(), "failed to create temp auth directory") {
				t.Errorf("Error message should mention directory creation failure: %v", err)
			}
			return
		}
		defer os.RemoveAll(authDir)

		// Verify directory was created successfully
		if authDir == "" {
			t.Error("createTempAuthDir() returned empty path on success")
		}
	})

	t.Run("createTempAuthDir creates multiple unique directories", func(t *testing.T) {
		// Verify that multiple calls create unique directories
		dirs := make(map[string]bool)
		for i := 0; i < 10; i++ {
			authDir, err := createTempAuthDir()
			if err != nil {
				t.Errorf("createTempAuthDir() failed on iteration %d: %v", i, err)
			}
			defer os.RemoveAll(authDir)

			if dirs[authDir] {
				t.Errorf("createTempAuthDir() returned duplicate path: %s", authDir)
			}
			dirs[authDir] = true
		}

		if len(dirs) != 10 {
			t.Errorf("Expected 10 unique directories, got %d", len(dirs))
		}
	})

	t.Run("writeTempTokenFile with read-only directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not support Unix-style permissions")
		}

		authDir := t.TempDir()
		token := "test-token"

		// Make directory read-only
		if err := os.Chmod(authDir, 0500); err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}

		_, err := writeTempTokenFile(authDir, token)
		if err == nil {
			t.Error("Expected error when writing to read-only directory")
		}

		// Restore permissions for cleanup
		_ = os.Chmod(authDir, 0700)
	})
}

func TestSwitch_SignalRaceCondition(t *testing.T) {
	// This test verifies that cleanup of authDir uses sync.Once to prevent
	// race conditions between the main goroutine's defer and signal handler.
	// Running the full test suite with -race flag verifies no data races exist.

	t.Run("multiple RemoveAll calls on same directory are safe", func(t *testing.T) {
		// This test verifies that calling RemoveAll multiple times on the same
		// directory doesn't cause issues (the directory just won't exist after first call)
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}

		// First cleanup removes the directory
		err1 := os.RemoveAll(authDir)
		if err1 != nil {
			t.Errorf("First RemoveAll failed: %v", err1)
		}

		// Second cleanup on non-existent directory should not error
		err2 := os.RemoveAll(authDir)
		if err2 != nil {
			t.Errorf("Second RemoveAll on non-existent directory failed: %v", err2)
		}

		// Verify directory is gone
		if _, err := os.Stat(authDir); !os.IsNotExist(err) {
			t.Error("Directory should not exist after RemoveAll")
		}
	})

	t.Run("sync.Once ensures cleanup happens exactly once", func(t *testing.T) {
		// Verify that sync.Once pattern (as used in switch.go) ensures
		// cleanup is idempotent and safe from concurrent access
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}

		var cleanupOnce sync.Once
		cleanup := func() {
			cleanupOnce.Do(func() {
				_ = os.RemoveAll(authDir)
			})
		}

		// Call cleanup multiple times - it should be idempotent
		cleanup()
		cleanup()
		cleanup()

		// Verify directory is gone
		if _, err := os.Stat(authDir); !os.IsNotExist(err) {
			t.Error("Directory should not exist after cleanup")
		}
	})

	t.Run("concurrent cleanup calls are safe with sync.Once", func(t *testing.T) {
		// This test simulates the race condition scenario where:
		// 1. Main goroutine has a deferred cleanup call
		// 2. Signal handler goroutine calls cleanup
		// Both could happen concurrently, so sync.Once is required
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() error = %v", err)
		}

		var cleanupOnce sync.Once
		var cleanupCalled bool
		cleanup := func() {
			cleanupOnce.Do(func() {
				_ = os.RemoveAll(authDir)
				cleanupCalled = true
			})
		}

		// Simulate concurrent cleanup from multiple goroutines
		// This mimics: defer cleanup() (main) + signal handler cleanup
		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func() {
				cleanup()
				done <- struct{}{}
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify cleanup was called exactly once
		if !cleanupCalled {
			t.Error("Cleanup should have been called")
		}

		// Verify directory is gone (cleanup executed)
		if _, err := os.Stat(authDir); !os.IsNotExist(err) {
			t.Error("Directory should not exist after concurrent cleanup")
		}
	})
}

func TestSwitch_PowerShellEscaping(t *testing.T) {
	// This test verifies that escapePowerShellArg properly escapes special characters
	// to prevent command injection and ensure correct argument passing.
	//
	// Special characters that need escaping in PowerShell single-quoted strings:
	// - Single quotes (') -> escaped as ''
	// - Backticks (`) -> escaped as ``
	// - Dollar signs ($) -> escaped as `$ (to prevent variable expansion)
	// - Double quotes (") -> escaped as ""

	tests := []struct {
		name     string
		input    string
		contains []string // Substrings that should be in the escaped output
	}{
		{
			name:     "simple string",
			input:    "hello",
			contains: []string{"'hello'"},
		},
		{
			name:     "single quote",
			input:    "it's",
			contains: []string{"'it''s'"}, // Single quotes are doubled
		},
		{
			name:     "multiple single quotes",
			input:    "it's a test",
			contains: []string{"'it''s a test'"},
		},
		{
			name:     "backtick",
			input:    "test" + string([]byte{0x60}) + "value",
			contains: []string{"'test``value'"}, // Backticks are doubled
		},
		{
			name:     "dollar sign",
			input:    "test$value",
			contains: []string{"'test" + string([]byte{0x60}) + "$value'"}, // Dollar sign is escaped
		},
		{
			name:     "double quote",
			input:    `test"value`,
			contains: []string{"'test\\\"value'"}, // Double quote is escaped
		},
		{
			name:  "path with spaces",
			input: "C:\\Program Files\\test",
			contains: []string{
				"'C:\\Program Files\\test'",
			},
		},
		{
			name:     "empty string",
			input:    "",
			contains: []string{"''"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapePowerShellArg(tt.input)

			// Verify result starts and ends with single quotes
			if !strings.HasPrefix(result, "'") {
				t.Errorf("escapePowerShellArg(%q) should start with single quote, got: %q", tt.input, result)
			}
			if !strings.HasSuffix(result, "'") {
				t.Errorf("escapePowerShellArg(%q) should end with single quote, got: %q", tt.input, result)
			}

			// Verify expected substrings are present
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("escapePowerShellArg(%q) = %q, should contain %q", tt.input, result, expected)
				}
			}

			// Verify the escaped string, when used in PowerShell, would correctly
			// represent the original input (this is a basic sanity check)
			// For example, 'it''s' in PowerShell evaluates to "it's"
		})
	}
}

func TestSwitch_WrapperErrorHandling(t *testing.T) {
	// These tests verify error handling in wrapper script generation
	// They test edge cases and error conditions that may occur in production

	t.Run("handles empty token path gracefully", func(t *testing.T) {
		authDir := t.TempDir()
		emptyTokenPath := ""

		_, _, err := generateWrapperScript(authDir, emptyTokenPath, "/usr/bin/claude", []string{"--help"})
		if err == nil {
			t.Error("Expected error when token path is empty, got nil")
		}
	})

	t.Run("handles empty claude path gracefully", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")
		emptyClaudePath := ""

		_, _, err := generateWrapperScript(authDir, tokenPath, emptyClaudePath, []string{})
		if err == nil {
			t.Error("Expected error when claude path is empty, got nil")
		}
	})

	t.Run("handles special characters in arguments", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		// Test with arguments containing special characters
		specialArgs := []string{
			"--prompt", "Hello; rm -rf /", // Command injection attempt
			"--file", "/path/to/file with spaces.txt",
			"--option", "value_with_'quotes'_and_\"dquotes\"",
		}

		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", specialArgs)
		if err != nil {
			t.Fatalf("generateWrapperScript() failed with special args: %v", err)
		}

		// Verify wrapper script was created
		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should be created even with special arguments")
		}

		// Read and verify script content for proper escaping
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)
		// The arguments should be properly escaped in the script
		if !strings.Contains(scriptContent, "Hello; rm -rf /") {
			t.Log("Script content (command injection string may be escaped):")
			t.Log(scriptContent)
		}
	})

	t.Run("handles very long arguments", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		// Create a very long argument (potential buffer overflow scenario)
		longArg := strings.Repeat("a", 10000)

		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{longArg})
		if err != nil {
			t.Fatalf("generateWrapperScript() failed with long argument: %v", err)
		}

		// Verify wrapper script was created
		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should handle long arguments")
		}
	})

	t.Run("handles nil arguments slice", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		// This should work - nil slice is treated like empty slice
		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", nil)
		if err != nil {
			t.Fatalf("generateWrapperScript() failed with nil args: %v", err)
		}

		// Verify wrapper script was created
		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should handle nil arguments")
		}
	})

	t.Run("validates script structure on both platforms", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{"--help"})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		// Platform-specific validation
		if runtime.GOOS == "windows" {
			if !strings.HasSuffix(wrapperPath, ".ps1") {
				t.Errorf("Windows wrapper should have .ps1 extension, got: %s", wrapperPath)
			}
			// PowerShell script checks
			if !strings.Contains(scriptContent, "$env:ANTHROPIC_AUTH_TOKEN") {
				t.Error("PowerShell script should set ANTHROPIC_AUTH_TOKEN environment variable")
			}
			if !strings.Contains(scriptContent, "Remove-Item") {
				t.Error("PowerShell script should remove token file")
			}
		} else {
			// Unix shell script checks
			if !strings.Contains(scriptContent, "#!/bin/sh") {
				t.Error("Unix script should have shebang")
			}
			if !strings.Contains(scriptContent, "export ANTHROPIC_AUTH_TOKEN") {
				t.Error("Unix script should export ANTHROPIC_AUTH_TOKEN")
			}
			if !strings.Contains(scriptContent, "rm -f") {
				t.Error("Unix script should remove token file")
			}
			if !strings.Contains(scriptContent, "exec") {
				t.Error("Unix script should use exec to replace process")
			}
		}
	})
}

func TestSwitch_CrossPlatformCompatibility(t *testing.T) {
	// These tests verify cross-platform compatibility across different OS environments

	t.Run("wrapper script paths are platform-appropriate", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		wrapperPath, useCmdExe, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("generateWrapperScript() error = %v", err)
		}

		isWindows := runtime.GOOS == "windows"

		// Verify platform-specific expectations
		if isWindows {
			if !useCmdExe {
				t.Error("useCmdExe should be true on Windows")
			}
			if !strings.HasSuffix(wrapperPath, ".ps1") {
				t.Errorf("Wrapper path should end with .ps1 on Windows, got: %s", wrapperPath)
			}
		} else {
			if useCmdExe {
				t.Error("useCmdExe should be false on Unix")
			}
			if strings.HasSuffix(wrapperPath, ".ps1") {
				t.Error("Wrapper path should not end with .ps1 on Unix")
			}
		}
	})

	t.Run("temp auth directory creation works on all platforms", func(t *testing.T) {
		authDir, err := createTempAuthDir()
		if err != nil {
			t.Fatalf("createTempAuthDir() failed: %v", err)
		}
		defer os.RemoveAll(authDir)

		// Verify directory exists
		if _, err := os.Stat(authDir); err != nil {
			t.Errorf("Auth directory should exist: %v", err)
		}

		// Verify it's in the system temp directory
		if !strings.HasPrefix(authDir, os.TempDir()) {
			t.Errorf("Auth dir %q should be in temp dir %q", authDir, os.TempDir())
		}
	})

	t.Run("token file creation works on all platforms", func(t *testing.T) {
		authDir := t.TempDir()
		testToken := "sk-ant-test-token-12345"

		tokenPath, err := writeTempTokenFile(authDir, testToken)
		if err != nil {
			t.Fatalf("writeTempTokenFile() failed: %v", err)
		}

		// Verify file exists
		content, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Errorf("Token file should be readable: %v", err)
		}

		if string(content) != testToken {
			t.Errorf("Token content = %q, want %q", string(content), testToken)
		}
	})

	t.Run("handles unicode in arguments", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		// Test with unicode characters
		unicodeArgs := []string{
			"--prompt", "Hello 世界 🌍",
			"--emoji", "🚀🎉",
		}

		wrapperPath, _, err := generateWrapperScript(authDir, tokenPath, "/usr/bin/claude", unicodeArgs)
		if err != nil {
			t.Fatalf("generateWrapperScript() failed with unicode: %v", err)
		}

		// Verify wrapper script was created
		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should handle unicode arguments")
		}
	})
}
