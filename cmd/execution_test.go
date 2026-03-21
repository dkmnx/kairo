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
	"github.com/spf13/cobra"
)

// TestRunHarnessWithWrapper tests the runHarnessWithWrapper function
func TestRunHarnessWithWrapper_HarnessNotFound(t *testing.T) {
	// Save original lookPath
	originalLookPath := lookPath
	defer func() { lookPath = originalLookPath }()

	// Mock lookPath to return error (harness not found)
	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("command not found: %s", file)
	}

	params := HarnessWrapperParams{
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

	err := runHarnessWithWrapper(params)
	if err == nil {
		t.Fatal("runHarnessWithWrapper() should return error when harness not found")
	}

	expectedSubstr := "'nonexistent-harness' command not found in PATH"
	if !strings.Contains(err.Error(), expectedSubstr) {
		t.Errorf("Error should contain %q, got: %v", expectedSubstr, err)
	}
}

func TestRunHarnessWithWrapper_WrapperGenerationFails(t *testing.T) {
	// Save original lookPath
	originalLookPath := lookPath
	defer func() { lookPath = originalLookPath }()

	// Mock lookPath to return success
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	// Test with invalid parameters that will cause wrapper generation to fail
	params := HarnessWrapperParams{
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

	err := runHarnessWithWrapper(params)
	if err == nil {
		t.Fatal("runHarnessWithWrapper() should return error when wrapper generation fails")
	}

	expectedSubstr := "generating wrapper script"
	if !strings.Contains(err.Error(), expectedSubstr) {
		t.Errorf("Error should contain %q, got: %v", expectedSubstr, err)
	}
}

func TestRunHarnessWithWrapper_Success(t *testing.T) {
	// Save original functions
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
	}()

	// Mock lookPath
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	// Mock execCommandContext to return a command that succeeds
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	// Mock exitProcess to prevent test from exiting
	exitProcess = func(int) {}

	tmpDir := t.TempDir()

	// Create token file for the test
	tokenPath := filepath.Join(tmpDir, "token")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0600); err != nil {
		t.Fatalf("Failed to create token file: %v", err)
	}

	params := HarnessWrapperParams{
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

	err := runHarnessWithWrapper(params)
	if err != nil {
		t.Fatalf("runHarnessWithWrapper() should succeed, got error: %v", err)
	}
}

// TestBuildWrapperCommand tests the buildWrapperCommand function
func TestBuildWrapperCommand_Windows(t *testing.T) {
	// Save original execCommandContext
	originalExecCommandContext := execCommandContext
	defer func() { execCommandContext = originalExecCommandContext }()

	var capturedCmd *exec.Cmd
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		capturedCmd = exec.CommandContext(ctx, name, arg...)
		return capturedCmd
	}

	ctx := context.Background()
	params := BuildWrapperCommandParams{
		Ctx:           ctx,
		WrapperScript: "C:\\temp\\wrapper.ps1",
		IsWindows:     true,
	}

	cmd := buildWrapperCommand(params)

	if cmd == nil {
		t.Fatal("buildWrapperCommand() should return non-nil cmd for Windows")
	}

	// Verify it's using powershell - check Args[0] since Path may be resolved to full path on Windows
	cmdName := ""
	if len(cmd.Args) > 0 {
		cmdName = cmd.Args[0]
	}
	if cmdName != "powershell" {
		t.Errorf("cmd.Args[0] = %q, want 'powershell'", cmdName)
	}

	// Verify arguments
	expectedArgs := []string{"powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "C:\\temp\\wrapper.ps1"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("cmd.Args length = %d, want %d", len(cmd.Args), len(expectedArgs))
	}
}

func TestBuildWrapperCommand_Unix(t *testing.T) {
	ctx := context.Background()
	params := BuildWrapperCommandParams{
		Ctx:           ctx,
		WrapperScript: "/tmp/wrapper.sh",
		IsWindows:     false,
	}

	cmd := buildWrapperCommand(params)

	if cmd == nil {
		t.Fatal("buildWrapperCommand() should return non-nil cmd for Unix")
	}

	// For Unix, it should directly execute the script
	if len(cmd.Args) < 1 {
		t.Error("cmd.Args should have at least one element")
	}
}

// TestExecuteWithAuth tests the executeWithAuth function
func TestExecuteWithAuth_TokenFileWriteFails(t *testing.T) {
	// Save original writeTempTokenFileFn
	originalWriteTempTokenFile := writeTempTokenFileFn
	defer func() { writeTempTokenFileFn = originalWriteTempTokenFile }()

	// Mock writeTempTokenFileFn to return error
	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return "", fmt.Errorf("token file write failed")
	}

	// Save original createTempAuthDirFn
	originalCreateTempAuthDir := createTempAuthDirFn
	defer func() { createTempAuthDirFn = originalCreateTempAuthDir }()

	// Mock createTempAuthDirFn to return a temp dir
	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	// Create a real cobra command for testing
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
	// Save original functions
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

	// Mock lookPath
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	// Mock createTempAuthDir
	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	// Mock writeTempTokenFile
	tokenPath := filepath.Join(tmpDir, "token")
	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return tokenPath, nil
	}

	// Track if execCommandContext was called
	var execCalled atomic.Bool
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		execCalled.Store(true)
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	// Mock exitProcess
	exitProcess = func(int) {}

	// Create a real cobra command for testing
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
	// Save original functions
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

	// Mock lookPath
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	// Mock createTempAuthDir
	tmpDir := t.TempDir()
	createTempAuthDirFn = func() (string, error) {
		return tmpDir, nil
	}

	// Mock writeTempTokenFile
	tokenPath := filepath.Join(tmpDir, "token")
	writeTempTokenFileFn = func(authDir, token string) (string, error) {
		return tokenPath, nil
	}

	// Track if execCommandContext was called
	var execCalled atomic.Bool
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		execCalled.Store(true)
		cmd := execCommand("echo", "mocked")
		cmd.Env = []string{"TEST=value"}
		return cmd
	}

	// Mock exitProcess
	exitProcess = func(int) {}

	// Create a real cobra command for testing
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

// TestBuildProviderEnvironment tests the buildProviderEnvironment function
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
	providerEnv, secrets, err := buildProviderEnvironment(cliCtx, tmpDir, provider, "test-provider")
	if err != nil {
		t.Fatalf("buildProviderEnvironment() should succeed with no secrets file, got: %v", err)
	}

	if secrets == nil {
		t.Error("buildProviderEnvironment() should return empty secrets map, not nil")
	}

	if len(providerEnv) == 0 {
		t.Error("buildProviderEnvironment() should return provider environment variables")
	}
}

// TestApiKeyEnvVarName tests the apiKeyEnvVarName function
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
			result := apiKeyEnvVarName(tt.provider)
			if result != tt.expected {
				t.Errorf("apiKeyEnvVarName(%q) = %q, want %q", tt.provider, result, tt.expected)
			}
		})
	}
}

// TestExecuteWithoutAuth tests the executeWithoutAuth function
func TestExecuteWithoutAuth_QwenNoAPIKey(t *testing.T) {
	// Create a real cobra command for testing
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

	// executeWithoutAuth for Qwen prints error via ui.PrintError which goes to stderr
	// So we just verify the function runs without crashing
}

func TestExecuteWithoutAuth_HarnessNotFound(t *testing.T) {
	// Save original lookPath
	originalLookPath := lookPath
	defer func() { lookPath = originalLookPath }()

	// Mock lookPath to return error
	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("command not found: %s", file)
	}

	// Create a real cobra command for testing
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
	// Save original functions
	originalLookPath := lookPath
	originalExecCommandContext := execCommandContext
	originalExitProcess := exitProcess
	defer func() {
		lookPath = originalLookPath
		execCommandContext = originalExecCommandContext
		exitProcess = originalExitProcess
	}()

	// Mock lookPath
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	// Mock execCommandContext to return a failing command
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := execCommand("false") // 'false' always returns non-zero exit
		return cmd
	}

	// Mock exitProcess
	exitProcessCalled := false
	exitProcess = func(int) {
		exitProcessCalled = true
	}

	// Create a real cobra command for testing
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

// TestBuildProviderEnvironment_NoAPIKeyRequired tests buildProviderEnvironment for providers that don't require API keys
func TestBuildProviderEnvironment_NoAPIKeyRequired(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a provider that doesn't require API key
	provider := config.Provider{
		Name:    "Test Provider",
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"CUSTOM_VAR=value"},
	}

	// Generate key first
	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	// Test with a provider that doesn't require API key (should not fail on missing secrets)
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	env, secrets, err := buildProviderEnvironment(cliCtx, tmpDir, provider, "ollama")
	if err != nil {
		t.Errorf("buildProviderEnvironment() for provider without API key should not error, got: %v", err)
	}

	if env == nil {
		t.Error("buildProviderEnvironment() returned nil env for provider without API key")
	}

	if secrets == nil {
		t.Error("buildProviderEnvironment() returned nil secrets map")
	}
}

// TestBuildProviderEnvironment_WithProviderEnvVars tests buildProviderEnvironment includes provider EnvVars
func TestBuildProviderEnvironment_WithProviderEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	provider := config.Provider{
		Name:    "Test Provider",
		BaseURL: "https://test.com",
		Model:   "test-model",
		EnvVars: []string{"PROVIDER_VAR=provider_value", "ANOTHER_VAR=another_value"},
	}

	// Generate key
	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	env, _, err := buildProviderEnvironment(cliCtx, tmpDir, provider, "ollama")
	if err != nil {
		t.Fatalf("buildProviderEnvironment() error = %v", err)
	}

	// Check that provider env vars are included
	envStr := strings.Join(env, "|")
	if !strings.Contains(envStr, "PROVIDER_VAR=provider_value") {
		t.Error("buildProviderEnvironment() should include provider EnvVars")
	}
	if !strings.Contains(envStr, "ANOTHER_VAR=another_value") {
		t.Error("buildProviderEnvironment() should include all provider EnvVars")
	}
}

// TestExecuteWithoutAuth_QwenNoAuth tests executeWithoutAuth for Qwen without API key
func TestExecuteWithoutAuth_QwenNoAuth(t *testing.T) {
	// Create a real cobra command for testing
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

	// Should print error and return without crashing
	executeWithoutAuth(cfg)
}

// TestBuildBuiltInEnvVars_Extended tests additional cases for buildBuiltInEnvVars
func TestBuildBuiltInEnvVars_Extended(t *testing.T) {
	t.Run("provider with special characters in values", func(t *testing.T) {
		provider := config.Provider{
			Name:    "Test Provider",
			BaseURL: "https://api.test.com/path?query=value",
			Model:   "test-model-v1.0-beta",
		}

		envVars := buildBuiltInEnvVars(provider)
		if len(envVars) == 0 {
			t.Error("buildBuiltInEnvVars() returned empty slice")
		}

		// Verify all expected vars are present
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
