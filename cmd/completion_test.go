package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompletionCommandBash generates bash completion script
func TestCompletionCommandBash(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

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

// TestCompletionCommandZsh generates zsh completion script
func TestCompletionCommandZsh(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

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

// TestCompletionCommandFish generates fish completion script
func TestCompletionCommandFish(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

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

// TestCompletionCommandPowerShell generates powershell completion script
func TestCompletionCommandPowerShell(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "powershell"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Verify powershell completion markers
	if !strings.Contains(output, "kairo") {
		t.Error("Output should contain kairo reference")
	}
}

// TestCompletionCommandUnknownShell returns error
func TestCompletionCommandUnknownShell(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "unknown-shell"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected error for unknown shell, got nil")
	}
}

// TestCompletionCommandNoArgs shows error (requires exactly 1 arg)
func TestCompletionCommandNoArgs(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected error when running completion without args, got nil")
	}
}

// TestCompletionCommandWithOutputFlag saves to file
func TestCompletionCommandWithOutputFlag(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() {
		configDir = originalConfigDir
		completionOutput = ""
		completionSave = false
	})

	tmpDir := t.TempDir()
	configDir = tmpDir

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

// TestCompletionCommandWithShortOutputFlag
func TestCompletionCommandWithShortOutputFlag(t *testing.T) {
	originalConfigDir := configDir
	t.Cleanup(func() {
		configDir = originalConfigDir
		completionOutput = ""
		completionSave = false
	})

	tmpDir := t.TempDir()
	configDir = tmpDir

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

// TestCompletionCommandAutoSaveToDefaultLocation
func TestCompletionCommandAutoSaveToDefaultLocation(t *testing.T) {
	originalConfigDir := configDir
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		configDir = originalConfigDir
		os.Setenv("HOME", originalHome)
		completionOutput = ""
		completionSave = false
	})

	tmpDir := t.TempDir()
	configDir = tmpDir
	os.Setenv("HOME", tmpDir)

	// Verify HOME is set correctly
	if home := os.Getenv("HOME"); home != tmpDir {
		t.Fatalf("HOME not set correctly: got %s, want %s", home, tmpDir)
	}

	// Verify UserHomeDir reads from HOME
	userHome, _ := os.UserHomeDir()
	if userHome != tmpDir {
		t.Fatalf("UserHomeDir not reading HOME: got %s, want %s", userHome, tmpDir)
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
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", home)

	path := getDefaultCompletionPath("bash")

	expected := filepath.Join(home, ".bash_completion.d", "kairo")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(bash) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathZsh(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", home)

	path := getDefaultCompletionPath("zsh")

	expected := filepath.Join(home, ".zsh", "completion", "_kairo")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(zsh) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathFish(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", home)

	path := getDefaultCompletionPath("fish")

	expected := filepath.Join(home, ".config", "fish", "completions", "kairo.fish")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(fish) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathPowerShell(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", home)

	path := getDefaultCompletionPath("powershell")

	expected := filepath.Join(home, "kairo.ps1")
	if path != expected {
		t.Errorf("getDefaultCompletionPath(powershell) = %q, want %q", path, expected)
	}
}

func TestGetDefaultCompletionPathUnknown(t *testing.T) {
	home := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", home)

	path := getDefaultCompletionPath("unknown")

	if path != "kairo-completion.sh" {
		t.Errorf("getDefaultCompletionPath(unknown) = %q, want %q", path, "kairo-completion.sh")
	}
}

func TestGetDefaultCompletionPathNoHomeDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Unsetenv("HOME")

	path := getDefaultCompletionPath("bash")

	if path != "kairo-completion.sh" {
		t.Errorf("getDefaultCompletionPath without HOME = %q, want %q", path, "kairo-completion.sh")
	}
}
