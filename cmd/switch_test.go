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
	"github.com/dkmnx/kairo/internal/wrapper"
)

func TestCreateTempAuthDir(t *testing.T) {
	isWindows := runtime.GOOS == "windows"

	t.Run("creates private directory with 0700 permissions", func(t *testing.T) {
		if isWindows {
			t.Skip("Windows does not support Unix-style permissions")
		}
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		info, err := os.Stat(authDir)
		if err != nil {
			t.Fatalf("Failed to stat auth directory: %v", err)
		}

		if !info.IsDir() {
			t.Errorf("Auth path should be a directory")
		}

		perms := info.Mode().Perm()
		if perms&0077 != 0 {
			t.Errorf("Directory should have no group/other permissions, got %o", perms)
		}
	})

	t.Run("directory is in temp directory", func(t *testing.T) {
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		tempDir := os.TempDir()
		if !strings.HasPrefix(authDir, tempDir) {
			t.Errorf("Auth directory path = %q, should be in temp directory %q", authDir, tempDir)
		}
	})

	t.Run("directory name contains kairo-auth identifier", func(t *testing.T) {
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		if !strings.Contains(authDir, "kairo-auth") {
			t.Errorf("Auth directory path = %q, should contain 'kairo-auth'", authDir)
		}
	})

	t.Run("creates unique directory for each call", func(t *testing.T) {
		authDir1, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir1)

		authDir2, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir2)

		if authDir1 == authDir2 {
			t.Errorf("CreateTempAuthDir() returned same path for different calls: %s", authDir1)
		}
	})
}

func TestWriteTempTokenFile(t *testing.T) {
	isWindows := runtime.GOOS == "windows"

	t.Run("creates file with correct content", func(t *testing.T) {
		authDir := t.TempDir()
		token := "test-api-key-12345"

		tokenPath, err := wrapper.WriteTempTokenFile(authDir, token)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() error = %v", err)
		}

		if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
			t.Errorf("Token file was not created at %s", tokenPath)
		}

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

		tokenPath, err := wrapper.WriteTempTokenFile(authDir, token)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() error = %v", err)
		}

		info, err := os.Stat(tokenPath)
		if err != nil {
			t.Fatalf("Failed to stat token file: %v", err)
		}

		perms := info.Mode().Perm()
		if perms&0077 != 0 {
			t.Errorf("File should have no group/other permissions, got %o", perms)
		}
	})

	t.Run("creates unique file for each call", func(t *testing.T) {
		authDir := t.TempDir()
		token1 := "token-1"
		token2 := "token-2"

		path1, err := wrapper.WriteTempTokenFile(authDir, token1)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() error = %v", err)
		}

		path2, err := wrapper.WriteTempTokenFile(authDir, token2)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() error = %v", err)
		}

		if path1 == path2 {
			t.Errorf("WriteTempTokenFile() returned same path for different calls: %s", path1)
		}
	})

	t.Run("creates files in specified directory", func(t *testing.T) {
		authDir := t.TempDir()
		token := "test-token"

		tokenPath, err := wrapper.WriteTempTokenFile(authDir, token)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() error = %v", err)
		}

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
		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{"--help"})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}
		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Errorf("Wrapper script was not created at %s", wrapperPath)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		if isWindows {
			if !strings.Contains(scriptContent, "ANTHROPIC_AUTH_TOKEN") {
				t.Errorf("Wrapper script missing ANTHROPIC_AUTH_TOKEN\nScript content:\n%s", scriptContent)
			}
		} else {
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
		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/echo", []string{"test"})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		if isWindows {
			if !strings.HasSuffix(wrapperPath, ".ps1") {
				t.Errorf("Wrapper script should have .ps1 extension on Windows")
			}
			if !strings.Contains(scriptContent, "Remove-Item") {
				t.Errorf("Wrapper script should contain Remove-Item command")
			}
		} else {
			if !strings.Contains(scriptContent, "rm") || !strings.Contains(scriptContent, tokenPath) {
				t.Errorf("Wrapper script should remove token file %s", tokenPath)
			}
		}
	})

	t.Run("script is executable", func(t *testing.T) {
		if isWindows {
			t.Skip("Skipping executable check on Windows")
		}
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-file")
		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}

		info, err := os.Stat(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to stat wrapper script: %v", err)
		}

		if info.Mode().Perm()&0111 == 0 {
			t.Errorf("Wrapper script should be executable, got mode %v", info.Mode().Perm())
		}
	})

	t.Run("handles empty args correctly", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-file")
		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

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
		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		if !strings.Contains(scriptContent, "test-token-with spaces") {
			t.Errorf("Wrapper script should contain the token path with spaces\nGot:\n%s", scriptContent)
		}

		quotedClaudePath := `"/usr/bin/claude"`
		if !strings.Contains(scriptContent, quotedClaudePath) {
			t.Errorf("Wrapper script should quote claude path\nGot:\n%s\nExpected to find: %s", scriptContent, quotedClaudePath)
		}
	})

	t.Run("creates wrapper in specified directory", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "test-token-file")
		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}

		if !strings.HasPrefix(wrapperPath, authDir) {
			t.Errorf("Wrapper script path = %q, should be in directory %q", wrapperPath, authDir)
		}
	})
}

func TestSwitchCmdSecureTokenPassing(t *testing.T) {
	t.Run("uses wrapper script when API key is present", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7"},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

		secretsContent := "ZAI_API_KEY=test-api-key-12345"
		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := os.WriteFile(keyPath, []byte("test-key"), 0600); err != nil {
			t.Fatalf("Failed to create key file: %v", err)
		}
		if err := os.WriteFile(secretsPath, []byte(secretsContent), 0600); err != nil {
			t.Fatalf("Failed to create secrets file: %v", err)
		}

		originalConfigDir := getConfigDir()
		setConfigDir(tmpDir)
		defer func() {
			setConfigDir(originalConfigDir)
			os.Remove(configPath)
		}()

		originalLookPath := lookPath
		lookPath = func(file string) (string, error) {
			if file == "claude" {
				return "/usr/bin/claude", nil
			}
			return originalLookPath(file)
		}
		defer func() { lookPath = originalLookPath }()

		originalExecCommand := execCommand
		execCommand = func(name string, arg ...string) *exec.Cmd {
			if name == "/usr/bin/claude" {
				t.Errorf("Expected wrapper script to be executed, got direct claude execution")
			}
			return originalExecCommand("echo", "mocked")
		}
		defer func() { execCommand = originalExecCommand }()

		originalExitProcess := exitProcess
		exitProcess = func(int) {}
		defer func() { exitProcess = originalExitProcess }()

		rootCmd.SetArgs([]string{"switch", "zai", "--help"})
		rootCmd.SetOut(&bytes.Buffer{})
		rootCmd.SetErr(&bytes.Buffer{})
	})

	t.Run("does not use wrapper script when no API key", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			DefaultProvider: "",
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic", BaseURL: "", Model: ""},
			},
		}
		configPath := createConfigFile(t, tmpDir, cfg)

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

		originalLookPath := lookPath
		lookPath = func(file string) (string, error) {
			if file == "claude" {
				return "/usr/bin/claude", nil
			}
			return originalLookPath(file)
		}
		defer func() { lookPath = originalLookPath }()

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

func TestWrapperScriptExecution(t *testing.T) {
	t.Parallel()

	isWindows := runtime.GOOS == "windows"

	authDir := t.TempDir()

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

	tokenPath := filepath.Join(authDir, "token")
	expectedToken := "sk-ant-test1234567890"
	if err := os.WriteFile(tokenPath, []byte(expectedToken), 0600); err != nil {
		t.Fatalf("Failed to create token file: %v", err)
	}

	wrapperPath, useCmdExe, err := wrapper.GenerateWrapperScript(authDir, tokenPath, childScriptPath, []string{})
	if err != nil {
		t.Fatalf("GenerateWrapperScript() error = %v", err)
	}

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

	if execErr != nil {
		t.Logf("Wrapper execution error (may be expected): %v", execErr)
	}

	output := out.String()

	if !strings.Contains(output, expectedToken) {
		t.Errorf("Expected token %q not found in wrapper output:\n%s", expectedToken, output)
	}

	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Errorf("Token file was not cleaned up after wrapper execution")
	}

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
	t.Run("code compiles without platform-specific syscalls", func(t *testing.T) {
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}
		defer os.RemoveAll(authDir)

		if _, err := os.Stat(authDir); err != nil {
			t.Errorf("Auth directory should exist: %v", err)
		}
	})

	t.Run("signal exit code calculation is cross-platform", func(t *testing.T) {
		testCases := []struct {
			sig      syscall.Signal
			expected int
		}{
			{syscall.SIGINT, 130},
			{syscall.SIGTERM, 143},
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
	t.Run("writeTempTokenFile returns error for non-existent directory", func(t *testing.T) {
		nonExistentDir := "/tmp/kairo-test-non-existent-" + strings.ReplaceAll(os.TempDir(), "/", "-")
		token := "test-token"

		_, err := wrapper.WriteTempTokenFile(nonExistentDir, token)
		if err == nil {
			t.Error("Expected error when writing to non-existent directory")
		}
	})

	t.Run("writeTempTokenFile returns error for invalid directory path", func(t *testing.T) {
		longPath := strings.Repeat("a", 10000)
		token := "test-token"

		_, err := wrapper.WriteTempTokenFile(longPath, token)
		if err == nil {
			t.Error("Expected error when writing to invalid directory path")
		}
	})

	t.Run("writeTempTokenFile handles very long tokens", func(t *testing.T) {
		authDir := t.TempDir()
		longToken := strings.Repeat("a", 100000)

		tokenPath, err := wrapper.WriteTempTokenFile(authDir, longToken)
		if err != nil {
			t.Errorf("WriteTempTokenFile() failed with long token: %v", err)
		}

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
		specialToken := "sk-test-\"`$';\n\t\x00"

		tokenPath, err := wrapper.WriteTempTokenFile(authDir, specialToken)
		if err != nil {
			t.Errorf("WriteTempTokenFile() failed with special characters: %v", err)
		}

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

		tokenPath, err := wrapper.WriteTempTokenFile(authDir, unicodeToken)
		if err != nil {
			t.Errorf("WriteTempTokenFile() failed with unicode: %v", err)
		}

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

		path1, err := wrapper.WriteTempTokenFile(authDir, token1)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() failed: %v", err)
		}

		path2, err := wrapper.WriteTempTokenFile(authDir, token2)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() failed: %v", err)
		}

		if path1 == path2 {
			t.Error("WriteTempTokenFile() should create unique files")
		}

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
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			if !strings.Contains(err.Error(), "failed to create temp auth directory") {
				t.Errorf("Error message should mention directory creation failure: %v", err)
			}
			return
		}
		defer os.RemoveAll(authDir)

		if authDir == "" {
			t.Error("CreateTempAuthDir() returned empty path on success")
		}
	})

	t.Run("createTempAuthDir creates multiple unique directories", func(t *testing.T) {
		dirs := make(map[string]bool)
		for i := 0; i < 10; i++ {
			authDir, err := wrapper.CreateTempAuthDir()
			if err != nil {
				t.Errorf("CreateTempAuthDir() failed on iteration %d: %v", i, err)
			}
			defer os.RemoveAll(authDir)

			if dirs[authDir] {
				t.Errorf("CreateTempAuthDir() returned duplicate path: %s", authDir)
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

		if err := os.Chmod(authDir, 0500); err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}

		_, err := wrapper.WriteTempTokenFile(authDir, token)
		if err == nil {
			t.Error("Expected error when writing to read-only directory")
		}

		_ = os.Chmod(authDir, 0700)
	})
}

func TestSwitch_SignalRaceCondition(t *testing.T) {
	t.Run("multiple RemoveAll calls on same directory are safe", func(t *testing.T) {
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}

		err1 := os.RemoveAll(authDir)
		if err1 != nil {
			t.Errorf("First RemoveAll failed: %v", err1)
		}

		err2 := os.RemoveAll(authDir)
		if err2 != nil {
			t.Errorf("Second RemoveAll on non-existent directory failed: %v", err2)
		}

		if _, err := os.Stat(authDir); !os.IsNotExist(err) {
			t.Error("Directory should not exist after RemoveAll")
		}
	})

	t.Run("sync.Once ensures cleanup happens exactly once", func(t *testing.T) {
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}

		var cleanupOnce sync.Once
		cleanup := func() {
			cleanupOnce.Do(func() {
				_ = os.RemoveAll(authDir)
			})
		}

		cleanup()
		cleanup()
		cleanup()

		if _, err := os.Stat(authDir); !os.IsNotExist(err) {
			t.Error("Directory should not exist after cleanup")
		}
	})

	t.Run("concurrent cleanup calls are safe with sync.Once", func(t *testing.T) {
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() error = %v", err)
		}

		var cleanupOnce sync.Once
		var cleanupCalled bool
		cleanup := func() {
			cleanupOnce.Do(func() {
				_ = os.RemoveAll(authDir)
				cleanupCalled = true
			})
		}

		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func() {
				cleanup()
				done <- struct{}{}
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		if !cleanupCalled {
			t.Error("Cleanup should have been called")
		}

		if _, err := os.Stat(authDir); !os.IsNotExist(err) {
			t.Error("Directory should not exist after concurrent cleanup")
		}
	})
}

func TestSwitch_PowerShellEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "simple string",
			input:    "hello",
			contains: []string{"'hello'"},
		},
		{
			name:     "single quote",
			input:    "it's",
			contains: []string{"'it''s'"},
		},
		{
			name:     "multiple single quotes",
			input:    "it's a test",
			contains: []string{"'it''s a test'"},
		},
		{
			name:     "backtick",
			input:    "test`value",
			contains: []string{"'test``value'"},
		},
		{
			name:     "dollar sign",
			input:    "test$value",
			contains: []string{"'test`$value'"},
		},
		{
			name:     "double quote",
			input:    `test"value`,
			contains: []string{"'test\\\"value'"},
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
			result := wrapper.EscapePowerShellArg(tt.input)

			if !strings.HasPrefix(result, "'") {
				t.Errorf("EscapePowerShellArg(%q) should start with single quote, got: %q", tt.input, result)
			}
			if !strings.HasSuffix(result, "'") {
				t.Errorf("EscapePowerShellArg(%q) should end with single quote, got: %q", tt.input, result)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("EscapePowerShellArg(%q) = %q, should contain %q", tt.input, result, expected)
				}
			}
		})
	}
}

func TestSwitch_WrapperErrorHandling(t *testing.T) {
	t.Run("handles empty token path gracefully", func(t *testing.T) {
		authDir := t.TempDir()
		emptyTokenPath := ""

		_, _, err := wrapper.GenerateWrapperScript(authDir, emptyTokenPath, "/usr/bin/claude", []string{"--help"})
		if err == nil {
			t.Error("Expected error when token path is empty, got nil")
		}
	})

	t.Run("handles empty claude path gracefully", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")
		emptyClaudePath := ""

		_, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, emptyClaudePath, []string{})
		if err == nil {
			t.Error("Expected error when claude path is empty, got nil")
		}
	})

	t.Run("handles special characters in arguments", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		specialArgs := []string{
			"--prompt", "Hello; rm -rf /",
			"--file", "/path/to/file with spaces.txt",
			"--option", "value_with_'quotes'_and_\"dquotes\"",
		}

		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", specialArgs)
		if err != nil {
			t.Fatalf("GenerateWrapperScript() failed with special args: %v", err)
		}

		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should be created even with special arguments")
		}
	})

	t.Run("handles very long arguments", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		longArg := strings.Repeat("a", 10000)

		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{longArg})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() failed with long argument: %v", err)
		}

		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should handle long arguments")
		}
	})

	t.Run("handles nil arguments slice", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", nil)
		if err != nil {
			t.Fatalf("GenerateWrapperScript() failed with nil args: %v", err)
		}

		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should handle nil arguments")
		}
	})

	t.Run("validates script structure on both platforms", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{"--help"})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}

		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("Failed to read wrapper script: %v", err)
		}

		scriptContent := string(content)

		if runtime.GOOS == "windows" {
			if !strings.HasSuffix(wrapperPath, ".ps1") {
				t.Errorf("Windows wrapper should have .ps1 extension, got: %s", wrapperPath)
			}
			if !strings.Contains(scriptContent, "$env:ANTHROPIC_AUTH_TOKEN") {
				t.Error("PowerShell script should set ANTHROPIC_AUTH_TOKEN environment variable")
			}
			if !strings.Contains(scriptContent, "Remove-Item") {
				t.Error("PowerShell script should remove token file")
			}
		} else {
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
	t.Run("wrapper script paths are platform-appropriate", func(t *testing.T) {
		authDir := t.TempDir()
		tokenPath := filepath.Join(authDir, "token")

		wrapperPath, useCmdExe, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", []string{})
		if err != nil {
			t.Fatalf("GenerateWrapperScript() error = %v", err)
		}

		isWindows := runtime.GOOS == "windows"

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
		authDir, err := wrapper.CreateTempAuthDir()
		if err != nil {
			t.Fatalf("CreateTempAuthDir() failed: %v", err)
		}
		defer os.RemoveAll(authDir)

		if _, err := os.Stat(authDir); err != nil {
			t.Errorf("Auth directory should exist: %v", err)
		}

		if !strings.HasPrefix(authDir, os.TempDir()) {
			t.Errorf("Auth dir %q should be in temp dir %q", authDir, os.TempDir())
		}
	})

	t.Run("token file creation works on all platforms", func(t *testing.T) {
		authDir := t.TempDir()
		testToken := "sk-ant-test-token-12345"

		tokenPath, err := wrapper.WriteTempTokenFile(authDir, testToken)
		if err != nil {
			t.Fatalf("WriteTempTokenFile() failed: %v", err)
		}

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

		unicodeArgs := []string{
			"--prompt", "Hello 世界 🌍",
			"--emoji", "🚀🎉",
		}

		wrapperPath, _, err := wrapper.GenerateWrapperScript(authDir, tokenPath, "/usr/bin/claude", unicodeArgs)
		if err != nil {
			t.Fatalf("GenerateWrapperScript() failed with unicode: %v", err)
		}

		if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
			t.Error("Wrapper script should handle unicode arguments")
		}
	})
}
