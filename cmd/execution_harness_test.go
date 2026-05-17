package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/wrapper"
)

func TestQwenAuthArgs(t *testing.T) {
	args := qwenAuthArgs("qwen-plus")
	if len(args) != 4 {
		t.Fatalf("qwenAuthArgs should return 4 elements, got %d", len(args))
	}
	if args[0] != "--auth-type" || args[1] != "anthropic" {
		t.Errorf("first two args should be --auth-type anthropic, got %v", args[:2])
	}
	if args[2] != "--model" || args[3] != "qwen-plus" {
		t.Errorf("last two args should be --model qwen-plus, got %v", args[2:])
	}
}

func TestExecutePi_HarnessNotFound(t *testing.T) {
	d := testDeps(func(d *Deps) {
		d.LookPath = func(file string) (string, error) {
			return "", fmt.Errorf("not found: %s", file)
		}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessPi,
		HarnessBinary: "pi",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		ProviderName: "test",
		Deps:         d,
	}

	err := executePi(cfg)
	if err != nil {
		t.Errorf("executePi with harness not found should return nil error (prints to cmd), got: %v", err)
	}

	output := outputOf(cmd)
	if !strings.Contains(output, "command not found in PATH") {
		t.Errorf("output should contain 'command not found in PATH', got: %s", output)
	}
}

func TestExecutePi_ExecutionFails(t *testing.T) {
	d := testDeps(func(d *Deps) {
		d.LookPath = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		d.ExecCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.Command("false")
		}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessPi,
		HarnessBinary: "pi",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		ProviderName: "test",
		Deps:         d,
	}

	err := executePi(cfg)
	if err == nil {
		t.Error("executePi with execution failure should return error")
	}
}

func TestExecuteWrapperWithAuth_AuthDirFails(t *testing.T) {
	d := testDeps(func(d *Deps) {
		d.CreateTempAuthDir = func() (string, error) {
			return "", fmt.Errorf("auth dir creation failed")
		}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessClaude,
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		Deps: d,
	}

	executeWrapperWithAuth(cfg)

	output := outputOf(cmd)
	if !strings.Contains(output, "Error creating auth directory") {
		t.Errorf("output should contain 'Error creating auth directory', got: %s", output)
	}
}

func TestExecuteWrapperWithAuth_TokenFileFails(t *testing.T) {
	tmpDir := t.TempDir()
	d := testDeps(func(d *Deps) {
		d.CreateTempAuthDir = func() (string, error) {
			return tmpDir, nil
		}
		d.WriteTempTokenFile = func(authDir, token string) (string, error) {
			return "", fmt.Errorf("token write failed")
		}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessClaude,
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		APIKey: "test-key",
		Deps:   d,
	}

	executeWrapperWithAuth(cfg)

	output := outputOf(cmd)
	if !strings.Contains(output, "Error creating secure token file") {
		t.Errorf("output should contain 'Error creating secure token file', got: %s", output)
	}
}

func TestExecuteWrapperWithAuth_WrapperExecFails_Claude(t *testing.T) {
	tmpDir := t.TempDir()
	exitCalled := false
	d := testDeps(func(d *Deps) {
		d.CreateTempAuthDir = func() (string, error) {
			return tmpDir, nil
		}
		tokenPath := filepath.Join(tmpDir, "token")
		d.WriteTempTokenFile = func(authDir, token string) (string, error) {
			return tokenPath, nil
		}
		d.LookPath = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		d.ExecCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.Command("false")
		}
		d.ExitProcess = func(int) {
			exitCalled = true
		}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessClaude,
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		APIKey: "test-key",
		Deps:   d,
	}

	executeWrapperWithAuth(cfg)

	if !exitCalled {
		t.Error("executeWrapperWithAuth should call ExitProcess on Claude wrapper execution failure")
	}
}

func TestExecuteWrapperWithAuth_QwenPassesAuthArgs(t *testing.T) {
	tmpDir := t.TempDir()
	var capturedArgs []string
	d := testDeps(func(d *Deps) {
		d.CreateTempAuthDir = func() (string, error) {
			return tmpDir, nil
		}
		tokenPath := filepath.Join(tmpDir, "token")
		d.WriteTempTokenFile = func(authDir, token string) (string, error) {
			return tokenPath, nil
		}
		d.LookPath = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		d.GenerateWrapperScript = func(cfg wrapper.ScriptConfig) (string, bool, error) {
			capturedArgs = cfg.CliArgs
			scriptPath := filepath.Join(tmpDir, "wrapper.sh")
			return scriptPath, false, nil
		}
		d.ExecCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.Command("echo", "mocked")
		}
		d.ExitProcess = func(int) {}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessQwen,
		HarnessBinary: "qwen",
		Provider: config.Provider{
			Name:  "Qwen",
			Model: "qwen-plus",
		},
		APIKey:      "test-key",
		HarnessArgs: []string{"--test"},
		Deps:        d,
	}

	executeWrapperWithAuth(cfg)

	hasAuthType := false
	hasModel := false
	for _, arg := range capturedArgs {
		if arg == "--auth-type" {
			hasAuthType = true
		}
		if arg == "qwen-plus" {
			hasModel = true
		}
	}
	if !hasAuthType {
		t.Errorf("Qwen wrapper should include --auth-type in CliArgs, got: %v", capturedArgs)
	}
	if !hasModel {
		t.Errorf("Qwen wrapper should include model name in CliArgs, got: %v", capturedArgs)
	}
}

func TestExecuteWithAuth_PiExecutionError(t *testing.T) {
	d := testDeps(func(d *Deps) {
		d.LookPath = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		d.ExecCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.Command("false")
		}
	})

	exitCalled := false
	d.ExitProcess = func(int) { exitCalled = true }

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessPi,
		HarnessBinary: "pi",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		ProviderName: "test",
		APIKey:       "test-key",
		Deps:         d,
	}

	executeWithAuth(cfg)

	if !exitCalled {
		t.Error("executeWithAuth with Pi execution failure should call ExitProcess")
	}
}

func TestExecuteWithoutAuth_YoloModeQwen(t *testing.T) {
	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessQwen,
		HarnessBinary: "qwen",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		Yolo: true,
		Deps: NewDeps(),
	}

	executeWithoutAuth(cfg)
}

func TestExecuteWithAuth_AuthDirCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	var execCalled atomic.Bool
	d := testDeps(func(d *Deps) {
		d.CreateTempAuthDir = func() (string, error) {
			return tmpDir, nil
		}
		tokenPath := filepath.Join(tmpDir, "token")
		d.WriteTempTokenFile = func(authDir, token string) (string, error) {
			return tokenPath, nil
		}
		d.LookPath = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		d.ExecCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			execCalled.Store(true)
			return exec.Command("echo", "mocked")
		}
		d.ExitProcess = func(int) {}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		HarnessToUse:  harnessClaude,
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		APIKey: "test-key",
		Deps:   d,
	}

	executeWrapperWithAuth(cfg)

	if !execCalled.Load() {
		t.Error("executeWrapperWithAuth should execute the wrapper command")
	}
}

func TestExecuteWithoutAuth_ClaudeSuccess(t *testing.T) {
	var capturedEnv []string
	d := testDeps(func(d *Deps) {
		d.LookPath = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		d.ExecCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			cmd := exec.Command("echo", "mocked")
			cmd.Env = []string{"TEST=value"}
			return cmd
		}
		d.ExitProcess = func(int) {}
	})

	cmd := testCmd()
	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   capturedEnv,
		HarnessToUse:  harnessClaude,
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		Deps: d,
	}

	executeWithoutAuth(cfg)

	output := outputOf(cmd)
	if strings.Contains(output, "Error") {
		t.Errorf("executeWithoutAuth successful Claude run should not print errors, got: %s", output)
	}
}
