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

func TestExecuteWithAuth_TokenFileWriteFails(t *testing.T) {
	tmpDir := t.TempDir()
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mw.CreateTempAuthDirFn = func() (string, error) {
			return tmpDir, nil
		}
		mw.WriteTempTokenFileFn = func(authDir, token string) (string, error) {
			return "", fmt.Errorf("token file write failed")
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
		APIKey:      "test-api-key",
		Deps:        d,
	}

	executeWithAuth(cfg)

	result := outputOf(cmd)
	if !strings.Contains(result, "Error creating secure token file") {
		t.Errorf("Output should contain 'Error creating secure token file', got: %s", result)
	}
}

func TestExecuteWithAuth_QwenHarness(t *testing.T) {
	tmpDir := t.TempDir()
	var execCalled atomic.Bool
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mw.CreateTempAuthDirFn = func() (string, error) {
			return tmpDir, nil
		}
		tokenPath := filepath.Join(tmpDir, "token")
		mw.WriteTempTokenFileFn = func(authDir, token string) (string, error) {
			return tokenPath, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			execCalled.Store(true)
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
		HarnessToUse:  harnessQwen,
		HarnessBinary: "qwen",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		APIKey:      "test-api-key",
		Deps:        d,
	}

	executeWithAuth(cfg)

	if !execCalled.Load() {
		t.Error("executeWithAuth() should call ExecCommandContext for Qwen harness")
	}
}

func TestExecuteWithAuth_ClaudeHarness(t *testing.T) {
	tmpDir := t.TempDir()
	var execCalled atomic.Bool
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mw.CreateTempAuthDirFn = func() (string, error) {
			return tmpDir, nil
		}
		tokenPath := filepath.Join(tmpDir, "token")
		mw.WriteTempTokenFileFn = func(authDir, token string) (string, error) {
			return tokenPath, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			execCalled.Store(true)
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
		HarnessToUse:  "claude",
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		APIKey:      "test-api-key",
		Deps:        d,
	}

	executeWithAuth(cfg)

	if !execCalled.Load() {
		t.Error("executeWithAuth() should call ExecCommandContext for Claude harness")
	}
}

func TestExecuteWithAuth_YoloModeClaude(t *testing.T) {
	tmpDir := t.TempDir()
	var capturedCfg wrapper.ScriptConfig
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mw.CreateTempAuthDirFn = func() (string, error) {
			return tmpDir, nil
		}
		tokenPath := filepath.Join(tmpDir, "token")
		mw.WriteTempTokenFileFn = func(authDir, token string) (string, error) {
			return tokenPath, nil
		}
		mw.GenerateWrapperScriptFn = func(cfg wrapper.ScriptConfig) (string, bool, error) {
			capturedCfg = cfg
			scriptPath := filepath.Join(tmpDir, "test-wrapper.ps1")
			return scriptPath, true, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
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
		HarnessToUse:  "claude",
		HarnessBinary: "claude",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		APIKey:      "test-api-key",
		Yolo:        true,
		Deps:        d,
	}

	executeWithAuth(cfg)

	found := false
	for _, arg := range capturedCfg.CliArgs {
		if arg == "--dangerously-skip-permissions" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("executeWithAuth(Yolo=true) should pass --dangerously-skip-permissions for claude, got CliArgs: %v", capturedCfg.CliArgs)
	}
}

func TestExecuteWithAuth_YoloModeQwen(t *testing.T) {
	tmpDir := t.TempDir()
	var capturedCfg wrapper.ScriptConfig
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mw.CreateTempAuthDirFn = func() (string, error) {
			return tmpDir, nil
		}
		tokenPath := filepath.Join(tmpDir, "token")
		mw.WriteTempTokenFileFn = func(authDir, token string) (string, error) {
			return tokenPath, nil
		}
		mw.GenerateWrapperScriptFn = func(cfg wrapper.ScriptConfig) (string, bool, error) {
			capturedCfg = cfg
			scriptPath := filepath.Join(tmpDir, "test-wrapper.ps1")
			return scriptPath, true, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
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
		HarnessToUse:  harnessQwen,
		HarnessBinary: "qwen",
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
		HarnessArgs: []string{"--test"},
		APIKey:      "test-api-key",
		Yolo:        true,
		Deps:        d,
	}

	executeWithAuth(cfg)

	found := false
	for _, arg := range capturedCfg.CliArgs {
		if arg == "--yolo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("executeWithAuth(Yolo=true) should pass --yolo for qwen, got CliArgs: %v", capturedCfg.CliArgs)
	}
}

func TestExecuteWithAuth_PiHarness(t *testing.T) {
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
		ProviderEnv:   []string{"ZAI_API_KEY=test-key"},
		HarnessToUse:  harnessPi,
		HarnessBinary: "pi",
		Provider: config.Provider{
			Name:    "Z.AI",
			BaseURL: "https://api.z.ai/api/anthropic",
			Model:   "glm-5.1",
		},
		ProviderName: "zai",
		HarnessArgs:  []string{"--session", "test"},
		APIKey:       "test-api-key",
		Deps:         d,
	}

	executeWithAuth(cfg)

	if len(capturedArgs) == 0 {
		t.Fatal("expected args to be captured")
	}
	if capturedArgs[0] != "--provider" || capturedArgs[1] != "zai" {
		t.Errorf("expected --provider zai, got %v", capturedArgs[:2])
	}
	if capturedArgs[2] != "--model" || capturedArgs[3] != "glm-5.1" {
		t.Errorf("expected --model glm-5.1, got %v", capturedArgs[2:4])
	}
}

func TestExecuteWithAuth_PiYoloMode(t *testing.T) {
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
			Name:    "Z.AI",
			BaseURL: "https://api.z.ai/api/anthropic",
			Model:   "glm-5.1",
		},
		HarnessArgs: []string{},
		APIKey:      "test-key",
		Yolo:        true,
		Deps:        d,
	}

	executeWithAuth(cfg)

	for _, arg := range capturedArgs {
		if arg == "--dangerously-skip-permissions" || arg == "--yolo" {
			t.Errorf("pi harness should not pass yolo flags, got %q", arg)
		}
	}
}
