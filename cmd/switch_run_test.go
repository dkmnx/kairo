package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

// Temporarily disabled - Cobra output not captured
func TestSwitchCmd_ProviderNotFound(t *testing.T) {
	t.Skip("Temporarily disabled - Cobra output capture needs refactoring")
}

// Temporarily disabled - Cobra output not captured
func TestSwitchCmd_ClaudeNotFound(t *testing.T) {
	t.Skip("Temporarily disabled - Cobra output capture needs refactoring")
}

func TestSwitchCmd_WithAPIKey_Success(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-4.7"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secretsContent := "ZAI_API_KEY=test-key-12345\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	originalConfigDir := getConfigDir()
	defer setConfigDir(originalConfigDir)
	setConfigDir(tmpDir)

	oldLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/bin/claude", nil
		}
		return oldLookPath(file)
	}
	defer func() { lookPath = oldLookPath }()

	oldExec := execCommand
	executedCmds := []string{}
	execCommand = func(name string, args ...string) *exec.Cmd {
		if strings.Contains(name, "wrapper") || strings.Contains(name, "tmp") || strings.Contains(name, "kairo-auth") {
			executedCmds = append(executedCmds, name)
			cmd := exec.Command("echo", "mock claude execution")
			cmd.Env = []string{}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd
		}
		return oldExec(name, args...)
	}
	defer func() { execCommand = oldExec }()

	oldExit := exitProcess
	var exitCalled bool
	exitProcess = func(code int) {
		exitCalled = true
	}
	defer func() { exitProcess = oldExit }()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	go func() {
		switchCmd.Run(switchCmd, []string{"zai", "--help"})
		w.Close()
	}()

	var bufErr error
	go func() {
		_, bufErr = buf.ReadFrom(r)
	}()

	time.Sleep(100 * time.Millisecond)
	os.Stdout = oldStdout

	if bufErr != nil {
		t.Logf("Warning: io.Copy failed: %v", bufErr)
	}

	output := buf.String()
	if len(executedCmds) == 0 {
		t.Error("Expected wrapper script to be executed")
	}
	if !strings.Contains(output, "Z.AI") {
		t.Errorf("Expected provider name in output, got: %s", output)
	}
	_ = exitCalled
}

func TestSwitchCmd_WithoutAPIKey_Success(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := crypto.EncryptSecrets(secretsPath, keyPath, ""); err != nil {
		t.Fatal(err)
	}

	originalConfigDir := getConfigDir()
	defer setConfigDir(originalConfigDir)
	setConfigDir(tmpDir)

	oldLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/bin/claude", nil
		}
		return oldLookPath(file)
	}
	defer func() { lookPath = oldLookPath }()

	oldExec := execCommand
	executedCmds := []string{}
	execCommand = func(name string, args ...string) *exec.Cmd {
		if strings.Contains(name, "claude") {
			executedCmds = append(executedCmds, name)
			cmd := exec.Command("echo", "mock claude execution")
			cmd.Env = []string{}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd
		}
		return oldExec(name, args...)
	}
	defer func() { execCommand = oldExec }()

	oldExit := exitProcess
	var exitCalled bool
	exitProcess = func(code int) {
		exitCalled = true
	}
	defer func() { exitProcess = oldExit }()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	go func() {
		switchCmd.Run(switchCmd, []string{"anthropic", "--help"})
		w.Close()
	}()

	var bufErr error
	go func() {
		_, bufErr = buf.ReadFrom(r)
	}()

	time.Sleep(100 * time.Millisecond)
	os.Stdout = oldStdout

	if bufErr != nil {
		t.Logf("Warning: io.Copy failed: %v", bufErr)
	}

	output := buf.String()
	if len(executedCmds) == 0 {
		t.Error("Expected claude command to be executed")
	}
	if !strings.Contains(output, "Native Anthropic") {
		t.Errorf("Expected provider name in output, got: %s", output)
	}
	_ = exitCalled
}
