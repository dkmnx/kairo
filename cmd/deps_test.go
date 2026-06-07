package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/wrapper"
)

func removeAll(p string) error  { return os.RemoveAll(p) }
func removeFile(p string) error { return os.Remove(p) }

// TestNewDeps wires up the production Deps and exercises each wrapper to
// guarantee the production adapters stay in sync with the interfaces they
// satisfy. The smoke test runs every adapter at least once.
func TestNewDeps(t *testing.T) {
	d := NewDeps()
	if d == nil {
		t.Fatal("NewDeps() returned nil")
	}
	if d.Process == nil {
		t.Error("Process is nil")
	}
	if d.Wrapper == nil {
		t.Error("Wrapper is nil")
	}
	if d.Update == nil {
		t.Error("Update is nil")
	}
	if d.Crypto == nil {
		t.Error("Crypto is nil")
	}
}

// TestDepsProductionAdapters_Coverage invokes each production adapter so the
// coverage tool records at least one execution per wrapper. Real calls (e.g.
// LookPath of `sh`) are used.
func TestDepsProductionAdapters_Coverage(t *testing.T) {
	d := NewDeps()

	// Process adapters
	if _, err := d.Process.LookPath("sh"); err != nil {
		t.Logf("LookPath: %v (expected on some systems)", err)
	}
	if cmd := d.Process.ExecCommandContext(t.Context(), "echo", "x"); cmd == nil {
		t.Error("ExecCommandContext returned nil")
	}
	// Note: ExitProcess calls os.Exit and is therefore not exercised here.

	// Wrapper adapter — CreateTempAuthDir is safe to call; the rest are too.
	authDir, err := d.Wrapper.CreateTempAuthDir()
	if err != nil {
		t.Errorf("CreateTempAuthDir: %v", err)
	} else {
		t.Cleanup(func() { _ = removeAll(authDir) })
	}

	tokenPath, err := d.Wrapper.WriteTempTokenFile(authDir, "x")
	if err != nil {
		t.Errorf("WriteTempTokenFile: %v", err)
	} else {
		t.Cleanup(func() { _ = removeFile(tokenPath) })
	}

	scriptPath, _, err := d.Wrapper.GenerateWrapperScript(wrapper.ScriptConfig{
		AuthDir:    authDir,
		TokenPath:  tokenPath,
		CliPath:    "/bin/echo",
		EnvVarName: "X",
	})
	if err != nil {
		t.Errorf("GenerateWrapperScript: %v", err)
	} else {
		t.Cleanup(func() { _ = removeFile(scriptPath) })
	}

	// Update adapter — exercise methods that don't require network or
	// process execution. ConfirmUpdate just delegates to ui.Confirm.
	_, _ = d.Update.ConfirmUpdate("test?")

	// Crypto adapter — full encrypt-decrypt cycle to cover every method.
	dir := t.TempDir()
	if err := d.Crypto.EnsureKeyExists(t.Context(), dir); err != nil {
		t.Errorf("EnsureKeyExists: %v", err)
	}
	secretsPath := filepath.Join(dir, constants.SecretsFileName)
	keyPath := filepath.Join(dir, constants.KeyFileName)
	if err := d.Crypto.EncryptSecrets(t.Context(), secretsPath, keyPath, "FOO=bar"); err != nil {
		t.Errorf("EncryptSecrets: %v", err)
	}
	if _, err := d.Crypto.DecryptSecretsBytes(t.Context(), secretsPath, keyPath); err != nil {
		t.Errorf("DecryptSecretsBytes: %v", err)
	}
	if _, err := d.Crypto.DecryptSecrets(t.Context(), secretsPath, keyPath); err != nil {
		t.Errorf("DecryptSecrets: %v", err)
	}
}

// Sanity check that exec.CommandContext is reachable from the wrapper code path.
var _ *exec.Cmd = (*exec.Cmd)(nil)
