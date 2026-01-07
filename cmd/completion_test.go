package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompletionCommandBash generates bash completion script.
func TestCompletionCommandBash(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "bash"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Verify bash completion markers - Cobra uses different formats
	if !strings.Contains(output, "_kairo") && !strings.Contains(output, "kairo") {
		t.Errorf("Output should contain bash completion, got: %s", output[:min(200, len(output))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestCompletionCommandZsh generates zsh completion script.
func TestCompletionCommandZsh(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "zsh"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Verify zsh completion markers
	if !strings.Contains(output, "#compdef kairo") {
		t.Errorf("Output should contain zsh completion directive, got: %s", output[:min(200, len(output))])
	}
}

// TestCompletionCommandFish generates fish completion script.
func TestCompletionCommandFish(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "fish"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Verify fish completion markers
	if !strings.Contains(output, "complete -c kairo") {
		t.Errorf("Output should contain fish completion command, got: %s", output[:min(200, len(output))])
	}
}

// TestCompletionCommandPowerShell generates PowerShell completion script.
func TestCompletionCommandPowerShell(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "powershell"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Verify PowerShell completion markers
	if !strings.Contains(output, "kairo") {
		t.Error("Output should contain kairo reference")
	}
}

// TestCompletionCommandUnknownShell returns error for unknown shell.
func TestCompletionCommandUnknownShell(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "unknown-shell"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected error for unknown shell, got nil")
	}
}

// TestCompletionCommandNoArgs shows error when run without shell argument.
func TestCompletionCommandNoArgs(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() { setConfigDir(originalConfigDir) })

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected error when running completion without args, got nil")
	}
}

// TestCompletionCommandWithOutputFlag saves completion to file.
func TestCompletionCommandWithOutputFlag(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() {
		setConfigDir(originalConfigDir)
		completionOutput = ""
		completionSave = false
	})

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	outputPath := filepath.Join(tmpDir, "kairo-completion.sh")
	rootCmd.SetArgs([]string{"completion", "bash", "--output", outputPath})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file was not created at %s", outputPath)
	}

	// Verify file contains bash completion
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "_kairo") {
		t.Errorf("Output file should contain bash completion, got: %s", string(content)[:min(200, len(content))])
	}
}

// TestCompletionCommandWithShortOutputFlag saves to file using -o flag.
func TestCompletionCommandWithShortOutputFlag(t *testing.T) {
	originalConfigDir := getConfigDir()
	t.Cleanup(func() {
		setConfigDir(originalConfigDir)
		completionOutput = ""
		completionSave = false
	})

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	outputPath := filepath.Join(tmpDir, "kairo-completion.sh")
	rootCmd.SetArgs([]string{"completion", "bash", "-o", outputPath})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file was not created at %s", outputPath)
	}
}

// TestCompletionCommandAutoSaveToDefaultLocation saves completion to default location with --save flag.
func TestCompletionCommandAutoSaveToDefaultLocation(t *testing.T) {
	originalConfigDir := getConfigDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		setConfigDir(originalConfigDir)
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		} else {
			os.Unsetenv("USERPROFILE")
		}
		completionOutput = ""
		completionSave = false
	})

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	// Set both HOME and USERPROFILE for cross-platform compatibility
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)

	// Verify environment is set correctly
	if home := os.Getenv("HOME"); home != tmpDir {
		t.Fatalf("HOME not set correctly: got %s, want %s", home, tmpDir)
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != tmpDir {
		t.Fatalf("USERPROFILE not set correctly: got %s, want %s", userProfile, tmpDir)
	}

	// Verify UserHomeDir returns the test directory
	userHome, _ := os.UserHomeDir()
	if userHome != tmpDir {
		t.Fatalf("UserHomeDir not returning test directory: got %s, want %s", userHome, tmpDir)
	}

	// Test bash auto-save
	rootCmd.SetArgs([]string{"completion", "bash", "--save"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify file was created in default location (under home)
	defaultPath := filepath.Join(tmpDir, ".bash_completion.d", "kairo")
	t.Logf("Expected path: %s", defaultPath)
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Errorf("Auto-save file was not created at %s", defaultPath)
	}

	// Clean up for next test
	os.Remove(defaultPath)
}

func TestGetDefaultCompletionPathBash(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)

	path := getDefaultCompletionPath("bash")

	expected := filepath.Join(home, ".bash_completion.d", "kairo")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(bash) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathZsh(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)

	path := getDefaultCompletionPath("zsh")

	expected := filepath.Join(home, ".zsh", "completion", "_kairo")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(zsh) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathFish(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)

	path := getDefaultCompletionPath("fish")

	expected := filepath.Join(home, ".config", "fish", "completions", "kairo.fish")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(fish) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathPowerShell(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)

	path := getDefaultCompletionPath("powershell")

	expected := filepath.Join(home, "Documents", "PowerShell", "Modules", "kairo-completion", "kairo-completion.psm1")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(powershell) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathUnknown(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)

	path := getDefaultCompletionPath("unknown")

	if path != "kairo-completion.sh" {
		t.Errorf("getDefaultCompletionPath(unknown) = %q, want %q", path, "kairo-completion.sh")
	}
}

func TestGetDefaultCompletionPathNoHomeDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)
	os.Unsetenv("HOME")
	os.Unsetenv("USERPROFILE")

	path := getDefaultCompletionPath("bash")

	if path != "kairo-completion.sh" {
		t.Errorf("getDefaultCompletionPath without HOME = %q, want %q", path, "kairo-completion.sh")
	}
}
