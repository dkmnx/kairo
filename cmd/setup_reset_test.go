package cmd

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// TestRunResetSecrets_UserCancels verifies that runResetSecrets returns
// ErrUserCancelled when the user declines the confirmation prompt.
func TestRunResetSecrets_UserCancels(t *testing.T) {
	feedStdin(t, "n\n")

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)

	err := runResetSecrets(cliCtx, configDir, SecretsResult{})
	if !errors.Is(err, kairoerrors.ErrUserCancelled) {
		t.Errorf("expected ErrUserCancelled, got: %v", err)
	}
}

// TestRunResetSecrets_Confirmed verifies that runResetSecrets succeeds when
// the user confirms and the crypto operations succeed.
func TestRunResetSecrets_Confirmed(t *testing.T) {
	feedStdin(t, "y\n")

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)
	cliCtx.SetDeps(resetDeps(nil))

	err := runResetSecrets(cliCtx, configDir, SecretsResult{
		Secrets:     map[string]string{},
		SecretsPath: filepath.Join(configDir, "secrets.age"),
		KeyPath:     filepath.Join(configDir, "key.age"),
	})
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

// TestRunResetSecrets_ResetFails verifies that an error from key generation
// is propagated to the caller.
func TestRunResetSecrets_ResetFails(t *testing.T) {
	feedStdin(t, "y\n")

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)

	wantErr := errors.New("key generation failed")
	cliCtx.SetDeps(resetDeps(func(ctx context.Context, keyPath string) error {
		return wantErr
	}))

	err := runResetSecrets(cliCtx, configDir, SecretsResult{
		Secrets:     map[string]string{},
		SecretsPath: filepath.Join(configDir, "secrets.age"),
		KeyPath:     filepath.Join(configDir, "key.age"),
	})
	if err == nil || !strings.Contains(err.Error(), wantErr.Error()) {
		t.Errorf("expected error containing %q, got: %v", wantErr.Error(), err)
	}
}

// resetDeps builds a Deps for the reset flow with a mockCrypto whose
// GenerateKey returns genErr (nil → real key generation).
func resetDeps(genErr func(ctx context.Context, keyPath string) error) *Deps {
	mc := &mockCrypto{}
	if genErr != nil {
		mc.GenerateKeyFn = genErr
	}
	return &Deps{
		Process: &mockProcess{
			LookPathFn:    func(string) (string, error) { return "", nil },
			ExitProcessFn: func(int) {},
		},
		Wrapper: &mockWrapper{},
		Update:  &mockUpdate{},
		Crypto:  mc,
	}
}
