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

func TestRunHarnessWithWrapper_HarnessNotFound(t *testing.T) {
	originalLookPath := lookPath
	defer func() { lookPath = originalLookPath }()

	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("command not found: %s", file)
	}

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

	err := runHarnessWithWrapper(run)
	if err == nil {
		t.Fatal("runHarnessWithWrapper() should return error when harness not found")
	}

	expectedSubstr := "'nonexistent-harness' command not found in PATH"
	if !strings.Contains(err.Error(), expectedSubstr) {
		t.Errorf("Error should contain %q, got: %v", expectedSubstr, err)
	}
}

func TestRunHarnessWithWrapper_WrapperGenerationFails(t *testing.T) {
	originalLookPath := lookPath
	defer func() { lookPath = originalLookPath }()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

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

	err := runHarnessWithWrapper(run)
	if err == nil {
		t.Fatal("runHarnessWithWrapper() should return error when wrapper generation fails")
	}

	expectedSubstr := "generating wrapper script"
	if !strings.Contains(err.Error(), expectedSubstr) {
		t.Errorf("Error should contain %q, got: %v", expectedSubstr, err)
	}
}

func TestRunHarnessWithWrapper_Success(t *testing.T) {
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	exitProcess = func(int) {}

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

	err := runHarnessWithWrapper(run)
	if err != nil {
		t.Fatalf("runHarnessWithWrapper() should succeed, got error: %v", err)
	}
}

func TestBuildWrapperCommand_Windows(t *testing.T) {
	originalExecCommandContext := execCommandContext
	defer func() { execCommandContext = originalExecCommandContext }()

	var capturedCmd *exec.Cmd
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		capturedCmd = exec.CommandContext(ctx, name, arg...)
		return capturedCmd
	}

	ctx := context.Background()
	params := WrapperCmd{
		Ctx:           ctx,
		WrapperScript: "C:\\temp\\wrapper.ps1",
		IsWindows:     true,
	}

	cmd := buildWrapperCommand(params)

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
	ctx := context.Background()
	params := WrapperCmd{
		Ctx:           ctx,
		WrapperScript: "/tmp/wrapper.sh",
		IsWindows:     false,
	}

	cmd := buildWrapperCommand(params)

	if cmd == nil {
		t.Fatal("buildWrapperCommand() should return non-nil cmd for Unix")
	}

	if len(cmd.Args) < 1 {
		t.Error("cmd.Args should have at least one element")
	}
}

func TestExecuteWithAuth_TokenFileWriteFails(t *testing.T) {
	originalWriteTempTokenFile := writeTempTokenFileFn
	defer func() { writeTempTokenFileFn = originalWriteTempTokenFile }()

	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return "", fmt.Errorf("token file write failed")
	}

	originalCreateTempAuthDir := createTempAuthDirFn
	defer func() { createTempAuthDirFn = originalCreateTempAuthDir }()

	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	}

	executeWithAuth(cfg)

	result := output.String()
	if !strings.Contains(result, "Error creating secure token file") {
		t.Errorf("Output should contain 'Error creating secure token file', got: %s", result)
	}
}

func TestExecuteWithAuth_QwenHarness(t *testing.T) {
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	originalCreateTempAuthDir := createTempAuthDirFn
	originalWriteTempTokenFile := writeTempTokenFileFn
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
		createTempAuthDirFn = originalCreateTempAuthDir
		writeTempTokenFileFn = originalWriteTempTokenFile
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	tokenPath := filepath.Join(tmpDir, "token")
	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return tokenPath, nil
	}

	var execCalled atomic.Bool
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		execCalled.Store(true)
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	exitProcess = func(int) {}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	}

	executeWithAuth(cfg)

	if !execCalled.Load() {
		t.Error("executeWithAuth() should call execCommandContext for Qwen harness")
	}
}

func TestExecuteWithAuth_ClaudeHarness(t *testing.T) {
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	originalCreateTempAuthDir := createTempAuthDirFn
	originalWriteTempTokenFile := writeTempTokenFileFn
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
		createTempAuthDirFn = originalCreateTempAuthDir
		writeTempTokenFileFn = originalWriteTempTokenFile
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	tokenPath := filepath.Join(tmpDir, "token")
	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return tokenPath, nil
	}

	var execCalled atomic.Bool
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		execCalled.Store(true)
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	exitProcess = func(int) {}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	}

	executeWithAuth(cfg)

	if !execCalled.Load() {
		t.Error("executeWithAuth() should call execCommandContext for Claude harness")
	}
}

func TestExecuteWithAuth_YoloModeClaude(t *testing.T) {
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	originalCreateTempAuthDir := createTempAuthDirFn
	originalWriteTempTokenFile := writeTempTokenFileFn
	originalGenerateWrapperScript := generateWrapperScriptFn
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
		createTempAuthDirFn = originalCreateTempAuthDir
		writeTempTokenFileFn = originalWriteTempTokenFile
		generateWrapperScriptFn = originalGenerateWrapperScript
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	tokenPath := filepath.Join(tmpDir, "token")
	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return tokenPath, nil
	}

	var capturedCfg wrapper.ScriptConfig
	generateWrapperScriptFn = func(cfg wrapper.ScriptConfig) (string, bool, error) {
		capturedCfg = cfg
		scriptPath := filepath.Join(tmpDir, "test-wrapper.ps1")
		return scriptPath, true, nil
	}

	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	exitProcess = func(int) {}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	originalCreateTempAuthDir := createTempAuthDirFn
	originalWriteTempTokenFile := writeTempTokenFileFn
	originalGenerateWrapperScript := generateWrapperScriptFn
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
		createTempAuthDirFn = originalCreateTempAuthDir
		writeTempTokenFileFn = originalWriteTempTokenFile
		generateWrapperScriptFn = originalGenerateWrapperScript
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	tokenPath := filepath.Join(tmpDir, "token")
	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return tokenPath, nil
	}

	var capturedCfg wrapper.ScriptConfig
	generateWrapperScriptFn = func(cfg wrapper.ScriptConfig) (string, bool, error) {
		capturedCfg = cfg
		scriptPath := filepath.Join(tmpDir, "test-wrapper.ps1")
		return scriptPath, true, nil
	}

	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	exitProcess = func(int) {}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	result, err := BuildProviderEnv(cliCtx, tmpDir, EnvProvider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "test-provider")
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
	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	}

	executeWithoutAuth(cfg)
}

func TestExecuteWithoutAuth_HarnessNotFound(t *testing.T) {
	originalLookPath := lookPath
	defer func() { lookPath = originalLookPath }()

	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("command not found: %s", file)
	}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	}

	executeWithoutAuth(cfg)

	result := output.String()
	if !strings.Contains(result, "command not found in PATH") {
		t.Errorf("Output should contain 'command not found in PATH', got: %s", result)
	}
}

func TestExecuteWithoutAuth_ExecutionFails(t *testing.T) {
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := execCommand("false") // 'false' always returns non-zero exit
		return cmd
	}

	exitProcessCalled := false
	exitProcess = func(int) {
		exitProcessCalled = true
	}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	}

	executeWithoutAuth(cfg)

	if !exitProcessCalled {
		t.Error("executeWithoutAuth() should call exitProcess on execution failure")
	}
}

func TestExecuteWithoutAuth_YoloModeClaude(t *testing.T) {
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	var capturedArgs []string
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		capturedArgs = arg
		cmd := execCommand("echo", "mocked")
		return cmd
	}

	exitProcess = func(int) {}

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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
	result, err := BuildProviderEnv(cliCtx, tmpDir, EnvProvider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "ollama")
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
	result, err := BuildProviderEnv(cliCtx, tmpDir, EnvProvider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars}, "ollama")
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
	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

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

		envVars := BuildBuiltInEnvVars(EnvProvider{BaseURL: provider.BaseURL, Model: provider.Model, EnvVars: provider.EnvVars})
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
