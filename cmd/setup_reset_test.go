package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// TestRunResetSecrets_UserCancels verifies that runResetSecrets returns
// ErrUserCancelled when the user declines the confirmation prompt.
func TestRunResetSecrets_UserCancels(t *testing.T) {
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.WriteString("n\n")
	w.Close()
	os.Stdin = r

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)

	err = runResetSecrets(cliCtx, configDir, SecretsResult{})
	if !errors.Is(err, kairoerrors.ErrUserCancelled) {
		t.Errorf("expected ErrUserCancelled, got: %v", err)
	}
}

// TestRunResetSecrets_Confirmed verifies that runResetSecrets succeeds when
// the user confirms and the crypto operations succeed.
func TestRunResetSecrets_Confirmed(t *testing.T) {
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.WriteString("y\n")
	w.Close()
	os.Stdin = r

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)

	// Replace crypto with a mock so EnsureKeyExists succeeds without real age.
	cliCtx.SetDeps(&Deps{
		Process: &mockProcess{
			LookPathFn:           func(string) (string, error) { return "", nil },
			ExecCommandContextFn: nil,
			ExitProcessFn:        func(int) {},
		},
		Wrapper: &mockWrapper{},
		Update:  &mockUpdate{},
		Crypto:  &mockCrypto{},
	})

	err = runResetSecrets(cliCtx, configDir, SecretsResult{
		Secrets:     map[string]string{},
		SecretsPath: filepath.Join(configDir, "secrets.age"),
		KeyPath:     filepath.Join(configDir, "key.age"),
	})
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

// TestRunResetSecrets_ResetFails verifies that an error from EnsureKeyExists
// is propagated to the caller.
func TestRunResetSecrets_ResetFails(t *testing.T) {
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.WriteString("y\n")
	w.Close()
	os.Stdin = r

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)

	wantErr := errors.New("key generation failed")
	cliCtx.SetDeps(&Deps{
		Process: &mockProcess{
			LookPathFn:           func(string) (string, error) { return "", nil },
			ExecCommandContextFn: nil,
			ExitProcessFn:        func(int) {},
		},
		Wrapper: &mockWrapper{},
		Update:  &mockUpdate{},
		Crypto: &mockCrypto{
			EnsureKeyExistsFn: func(ctx context.Context, configDir string) error {
				return wantErr
			},
		},
	})

	err = runResetSecrets(cliCtx, configDir, SecretsResult{
		Secrets:     map[string]string{},
		SecretsPath: filepath.Join(configDir, "secrets.age"),
		KeyPath:     filepath.Join(configDir, "key.age"),
	})
	if err == nil || !strings.Contains(err.Error(), wantErr.Error()) {
		t.Errorf("expected error containing %q, got: %v", wantErr.Error(), err)
	}
}
