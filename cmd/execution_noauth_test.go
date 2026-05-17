package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestExecuteWithoutAuth_QwenNoAPIKey(t *testing.T) {
	cmd := testCmd()

	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   []string{"TEST=value"},
		HarnessToUse:  harnessQwen,
		HarnessBinary: "qwen",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		Deps:        NewDeps(),
	}

	executeWithoutAuth(cfg)
}

func TestExecuteWithoutAuth_HarnessNotFound(t *testing.T) {
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "", fmt.Errorf("command not found: %s", file)
		}
	})

	cmd := testCmd()

	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   []string{"TEST=value"},
		HarnessToUse:  "claude",
		HarnessBinary: "nonexistent",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		Deps:        d,
	}

	executeWithoutAuth(cfg)

	result := outputOf(cmd)
	if !strings.Contains(result, "command not found in PATH") {
		t.Errorf("Output should contain 'command not found in PATH', got: %s", result)
	}
}

func TestExecuteWithoutAuth_ExecutionFails(t *testing.T) {
	exitProcessCalled := false
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			cmd := exec.Command("false") // 'false' always returns non-zero exit
			return cmd
		}
		mp.ExitProcessFn = func(int) {
			exitProcessCalled = true
		}
	})

	cmd := testCmd()

	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   []string{"TEST=value"},
		HarnessToUse:  "claude",
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		Deps:        d,
	}

	executeWithoutAuth(cfg)

	if !exitProcessCalled {
		t.Error("executeWithoutAuth() should call ExitProcess on execution failure")
	}
}

func TestExecuteWithoutAuth_YoloModeClaude(t *testing.T) {
	var capturedArgs []string
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			capturedArgs = arg
			cmd := exec.Command("echo", "mocked")
			return cmd
		}
		mp.ExitProcessFn = func(int) {}
	})

	cmd := testCmd()

	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   []string{"TEST=value"},
		HarnessToUse:  "claude",
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		Yolo:        true,
		Deps:        d,
	}

	executeWithoutAuth(cfg)

	found := false
	for _, arg := range capturedArgs {
		if arg == "--dangerously-skip-permissions" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("executeWithoutAuth(Yolo=true) should pass --dangerously-skip-permissions for claude, got: %v", capturedArgs)
	}
}

func TestExecuteWithoutAuth_QwenNoAuth(t *testing.T) {
	cmd := testCmd()

	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   []string{"TEST=value"},
		HarnessToUse:  harnessQwen,
		HarnessBinary: "qwen",
		Provider: config.Provider{
			Name:    "Qwen Provider",
			BaseURL: "https://test.com",
			Model:   "qwen-model",
		},
		HarnessArgs: []string{"--test"},
		Deps:        NewDeps(),
	}

	executeWithoutAuth(cfg)
}

func TestExecuteWithoutAuth_PiHarness(t *testing.T) {
	var capturedArgs []string
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			capturedArgs = arg
			cmd := exec.Command("echo", "mocked")
			cmd.Env = []string{"TEST=value"}
			return cmd
		}
		mp.ExitProcessFn = func(int) {}
	})

	cmd := testCmd()

	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   []string{"TEST=value"},
		HarnessToUse:  harnessPi,
		HarnessBinary: "pi",
		Provider: config.Provider{
			Name:    "DeepSeek AI",
			BaseURL: "https://api.deepseek.com/anthropic",
			Model:   "deepseek-v4-pro[1m]",
		},
		ProviderName: "deepseek",
		HarnessArgs:  []string{"--continue"},
		Deps:         d,
	}

	executeWithoutAuth(cfg)

	if len(capturedArgs) == 0 {
		t.Fatal("expected args to be captured")
	}
	if capturedArgs[0] != "--provider" || capturedArgs[1] != "deepseek" {
		t.Errorf("expected --provider deepseek, got %v", capturedArgs[:2])
	}
}

func TestExecuteWithoutAuth_PiHarnessNotFound(t *testing.T) {
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "", fmt.Errorf("not found")
		}
	})

	cmd := testCmd()

	cfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   []string{},
		HarnessToUse:  harnessPi,
		HarnessBinary: "pi",
		Provider: config.Provider{
			Name:  "Test",
			Model: "test-model",
		},
		Deps: d,
	}

	executeWithoutAuth(cfg)

	if !strings.Contains(outputOf(cmd), "'pi' command not found in PATH") {
		t.Errorf("expected 'not found in PATH' error, got %q", outputOf(cmd))
	}
}
