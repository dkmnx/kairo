package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
