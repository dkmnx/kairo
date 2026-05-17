package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/wrapper"
	"github.com/spf13/cobra"
)

func testCmd() *cobra.Command {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	return cmd
}

func outputOf(cmd *cobra.Command) string {
	if cmd.OutOrStdout() == nil {
		return ""
	}
	buf, ok := cmd.OutOrStdout().(*bytes.Buffer)
	if !ok {
		return ""
	}
	return buf.String()
}

func TestRunHarnessWithWrapper_HarnessNotFound(t *testing.T) {
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "", fmt.Errorf("command not found: %s", file)
		}
	})

	run := HarnessRun{
		AuthDir:       "/tmp/test-auth",
		TokenPath:     "/tmp/test-auth/token",
		HarnessBinary: "nonexistent-harness",
		CliArgs:       []string{"--test"},
		ProviderEnv:   []string{"TEST=value"},
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
	}

	err := runHarnessWithWrapper(context.Background(), d, run)
	if err == nil {
		t.Fatal("runHarnessWithWrapper() should return error when harness not found")
	}

	expectedSubstr := "'nonexistent-harness' command not found in PATH"
	if !strings.Contains(err.Error(), expectedSubstr) {
		t.Errorf("Error should contain %q, got: %v", expectedSubstr, err)
	}
}

func TestRunHarnessWithWrapper_WrapperGenerationFails(t *testing.T) {
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mw.GenerateWrapperScriptFn = func(cfg wrapper.ScriptConfig) (string, bool, error) {
			return "", false, fmt.Errorf("wrapper: token path cannot be empty")
		}
	})

	run := HarnessRun{
		AuthDir:       "/tmp/test-auth",
		TokenPath:     "", // Empty token path will cause wrapper generation to fail
		HarnessBinary: "claude",
		CliArgs:       []string{"--test"},
		ProviderEnv:   []string{"TEST=value"},
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
	}

	err := runHarnessWithWrapper(context.Background(), d, run)
	if err == nil {
		t.Fatal("runHarnessWithWrapper() should return error when wrapper generation fails")
	}

	expectedSubstr := "generating wrapper script"
	if !strings.Contains(err.Error(), expectedSubstr) {
		t.Errorf("Error should contain %q, got: %v", expectedSubstr, err)
	}
}

func TestRunHarnessWithWrapper_Success(t *testing.T) {
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.LookPathFn = func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			cmd := exec.Command("echo", "mocked")
			cmd.Env = []string{"TEST=value"}
			return cmd
		}
		mp.ExitProcessFn = func(int) {}
	})

	tmpDir := t.TempDir()

	tokenPath := filepath.Join(tmpDir, "token")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0600); err != nil {
		t.Fatalf("Failed to create token file: %v", err)
	}

	run := HarnessRun{
		AuthDir:       tmpDir,
		TokenPath:     tokenPath,
		HarnessBinary: "claude",
		CliArgs:       []string{"--test"},
		ProviderEnv:   []string{"TEST=value"},
		Provider: config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://test.com",
			Model:   "test-model",
		},
	}

	err := runHarnessWithWrapper(context.Background(), d, run)
	if err != nil {
		t.Fatalf("runHarnessWithWrapper() should succeed, got error: %v", err)
	}
}

func TestBuildWrapperCommand_Windows(t *testing.T) {
	var capturedCmd *exec.Cmd
	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mp.ExecCommandContextFn = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			capturedCmd = exec.CommandContext(ctx, name, arg...)
			return capturedCmd
		}
	})

	ctx := context.Background()
	params := WrapperCmd{
		Ctx:           ctx,
		WrapperScript: "C:\\temp\\wrapper.ps1",
		IsWindows:     true,
	}

	cmd := buildWrapperCommand(d, params)

	if cmd == nil {
		t.Fatal("buildWrapperCommand() should return non-nil cmd for Windows")
	}

	cmdName := ""
	if len(cmd.Args) > 0 {
		cmdName = cmd.Args[0]
	}
	if cmdName != "powershell" {
		t.Errorf("cmd.Args[0] = %q, want 'powershell'", cmdName)
	}

	expectedArgs := []string{"powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "C:\\temp\\wrapper.ps1"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("cmd.Args length = %d, want %d", len(cmd.Args), len(expectedArgs))
	}
}

func TestBuildWrapperCommand_Unix(t *testing.T) {
	d := NewDeps()
	ctx := context.Background()
	params := WrapperCmd{
		Ctx:           ctx,
		WrapperScript: "/tmp/wrapper.sh",
		IsWindows:     false,
	}

	cmd := buildWrapperCommand(d, params)

	if cmd == nil {
		t.Fatal("buildWrapperCommand() should return non-nil cmd for Unix")
	}

	if len(cmd.Args) < 1 {
		t.Error("cmd.Args should have at least one element")
	}
}

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

func TestBuildProviderEnvironment_Success(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		Name:    "test-provider",
		BaseURL: "https://api.test.com",
		Model:   "test-model",
		EnvVars: []string{"CUSTOM_VAR=custom_value"},
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	result, err := BuildProviderEnv(cliCtx, tmpDir, config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "test-provider")
	if err != nil {
		t.Fatalf("BuildProviderEnv() should succeed with no secrets file, got: %v", err)
	}

	if result.Secrets == nil {
		t.Error("BuildProviderEnv() should return empty secrets map, not nil")
	}

	if len(result.ProviderEnv) == 0 {
		t.Error("BuildProviderEnv() should return provider environment variables")
	}
}

func TestApiKeyEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"lowercase provider", "anthropic", "ANTHROPIC_API_KEY"},
		{"uppercase provider", "ANTHROPIC", "ANTHROPIC_API_KEY"},
		{"mixed case provider", "MiniMax", "MINIMAX_API_KEY"},
		{"provider with hyphen", "my-provider", "MY-PROVIDER_API_KEY"},
		{"provider with underscore", "my_provider", "MY_PROVIDER_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := APIKeyEnvVarName(tt.provider)
			if result != tt.expected {
				t.Errorf("APIKeyEnvVarName(%q) = %q, want %q", tt.provider, result, tt.expected)
			}
		})
	}
}

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

func TestBuildProviderEnvironment_NoAPIKeyRequired(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		Name:    "Test Provider",
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"CUSTOM_VAR=value"},
	}

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	result, err := BuildProviderEnv(cliCtx, tmpDir, config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "ollama")
	if err != nil {
		t.Fatalf("BuildProviderEnv() for provider without API key should not error, got: %v", err)
	}
	if result.ProviderEnv == nil {
		t.Error("BuildProviderEnv() returned nil env for provider without API key")
	}
	if result.Secrets == nil {
		t.Error("BuildProviderEnv() returned nil secrets map")
	}
}

func TestBuildProviderEnvironment_WithProviderEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		Name:    "Test Provider",
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"PROVIDER_VAR=provider_value", "ANOTHER_VAR=another_value"},
	}

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	result, err := BuildProviderEnv(cliCtx, tmpDir, config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "ollama")
	if err != nil {
		t.Fatalf("BuildProviderEnv() error = %v", err)
	}
	if len(result.ProviderEnv) == 0 {
		t.Error("BuildProviderEnv() should include provider EnvVars")
	}
	if len(result.ProviderEnv) < len(provider.EnvVars) {
		t.Error("BuildProviderEnv() should include all provider EnvVars")
	}
	envStr := strings.Join(result.ProviderEnv, "|")
	if !strings.Contains(envStr, "PROVIDER_VAR=provider_value") {
		t.Error("buildProviderEnvironment() should include provider EnvVars")
	}
	if !strings.Contains(envStr, "ANOTHER_VAR=another_value") {
		t.Error("buildProviderEnvironment() should include all provider EnvVars")
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

func TestBuildBuiltInEnvVars_Extended(t *testing.T) {
	t.Run("provider with special characters in values", func(t *testing.T) {
		provider := config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://api.test.com/path?query=value",
			Model:   "test-model-v1.0-beta",
		}

		envVars := BuildBuiltInEnvVars(config.Provider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars})
		if len(envVars) == 0 {
			t.Error("buildBuiltInEnvVars() returned empty slice")
		}

		hasBaseURL := false
		hasModel := false
		for _, v := range envVars {
			if strings.HasPrefix(v, "ANTHROPIC_BASE_URL=") {
				hasBaseURL = true
			}
			if strings.HasPrefix(v, "ANTHROPIC_MODEL=") {
				hasModel = true
			}
		}

		if !hasBaseURL {
			t.Error("missing ANTHROPIC_BASE_URL")
		}
		if !hasModel {
			t.Error("missing ANTHROPIC_MODEL")
		}
	})
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

func TestBuildPiEnvVars(t *testing.T) {
	provider := config.Provider{
		BaseURL: "https://api.z.ai/api/anthropic",
		Model:   "glm-5.1",
	}
	envVars := BuildPiEnvVars(provider, "zai")

	hasProvider := false
	hasModel := false
	for _, v := range envVars {
		if v == "PI_PROVIDER=zai" {
			hasProvider = true
		}
		if v == "PI_MODEL=glm-5.1" {
			hasModel = true
		}
	}

	if !hasProvider {
		t.Error("missing PI_PROVIDER")
	}
	if !hasModel {
		t.Error("missing PI_MODEL")
	}
}

func TestPiAPIKeyEnvVarMapping(t *testing.T) {
	tests := []struct {
		provider string
		envVar   string
		ok       bool
	}{
		{"zai", "ZAI_API_KEY", true},
		{"minimax", "MINIMAX_API_KEY", true},
		{"deepseek", "DEEPSEEK_API_KEY", true},
		{"kimi", "KIMI_API_KEY", true},
		{"unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			envVar, ok := PiAPIKeyEnvVar(tt.provider)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if envVar != tt.envVar {
				t.Errorf("envVar = %q, want %q", envVar, tt.envVar)
			}
		})
	}
}
