package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
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
		Harness: "claude",
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
		Harness: "claude",
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
		Harness: "claude",
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
