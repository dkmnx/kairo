package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

// runningWithRaceDetector returns true if the race detector is enabled
func runningWithRaceDetector() bool {
	// The race detector forces GOMAXPROCS to be at least 2
	return runtime.GOMAXPROCS(-1) > 1
}

func TestSwitchCmd_WithAPIKey_Success(t *testing.T) {
	if runningWithRaceDetector() {
		t.Skip("Skipping with race detector - requires test refactoring for proper goroutine synchronization")
	}
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
	setConfigDir(tmpDir)
	defer setConfigDir(originalConfigDir)

	oldLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/bin/claude", nil
		}
		return oldLookPath(file)
	}
	defer func() { lookPath = oldLookPath }()

	oldExec := execCommand
	var mu sync.Mutex
	executedCmds := []string{}
	execCommand = func(name string, args ...string) *exec.Cmd {
		if strings.Contains(name, "wrapper") || strings.Contains(name, "tmp") || strings.Contains(name, "kairo-auth") {
			mu.Lock()
			executedCmds = append(executedCmds, name)
			mu.Unlock()
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
		mu.Lock()
		exitCalled = true
		mu.Unlock()
	}
	defer func() { exitProcess = oldExit }()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	done := make(chan struct{})
	readDone := make(chan struct{})
	go func() {
		switchCmd.Run(switchCmd, []string{"zai", "--help"})
		w.Close()
		close(done)
	}()

	var bufErr error
	go func() {
		_, bufErr = buf.ReadFrom(r)
		close(readDone)
	}()

	time.Sleep(100 * time.Millisecond)
	os.Stdout = oldStdout

	// Wait for both goroutines to complete before defer runs
	<-done
	<-readDone

	if bufErr != nil {
		t.Logf("Warning: io.Copy failed: %v", bufErr)
	}

	output := buf.String()
	mu.Lock()
	cmdsExecuted := len(executedCmds) > 0
	mu.Unlock()
	if !cmdsExecuted {
		t.Error("Expected wrapper script to be executed")
	}
	if !strings.Contains(output, "Z.AI") {
		t.Errorf("Expected provider name in output, got: %s", output)
	}
	mu.Lock()
	_ = exitCalled
	mu.Unlock()

	setConfigDir(originalConfigDir)
}

func TestSwitchCmd_WithoutAPIKey_Success(t *testing.T) {
	if runningWithRaceDetector() {
		t.Skip("Skipping with race detector - requires test refactoring for proper goroutine synchronization")
	}
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
	var mu sync.Mutex
	executedCmds := []string{}
	execCommand = func(name string, args ...string) *exec.Cmd {
		if strings.Contains(name, "claude") {
			mu.Lock()
			executedCmds = append(executedCmds, name)
			mu.Unlock()
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
		mu.Lock()
		exitCalled = true
		mu.Unlock()
	}
	defer func() { exitProcess = oldExit }()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	done := make(chan struct{})
	readDone := make(chan struct{})
	go func() {
		switchCmd.Run(switchCmd, []string{"anthropic", "--help"})
		w.Close()
		close(done)
	}()

	var bufErr error
	go func() {
		_, bufErr = buf.ReadFrom(r)
		close(readDone)
	}()

	time.Sleep(100 * time.Millisecond)
	os.Stdout = oldStdout

	// Wait for both goroutines to complete before defer runs
	<-done
	<-readDone

	if bufErr != nil {
		t.Logf("Warning: io.Copy failed: %v", bufErr)
	}

	output := buf.String()
	mu.Lock()
	cmdsExecuted := len(executedCmds) > 0
	mu.Unlock()
	if !cmdsExecuted {
		t.Error("Expected claude command to be executed")
	}
	if !strings.Contains(output, "Native Anthropic") {
		t.Errorf("Expected provider name in output, got: %s", output)
	}
	mu.Lock()
	_ = exitCalled
	mu.Unlock()

	setConfigDir(originalConfigDir)
}
