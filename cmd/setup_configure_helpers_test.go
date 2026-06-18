package cmd

import (
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/yarlson/tap"
)

// startConfigureProvider spins up configureProvider in a goroutine using a
// mock-backed CLIContext and returns the input handle, config, and a channel
// that receives the provider name or "error:<msg>" on completion.
func startConfigureProvider(
	t *testing.T,
	providerName string,
	cfg *config.Config,
) (in *tap.MockReadable, cfgOut *config.Config, resultCh chan string) {
	return startConfigureProviderWithSecrets(t, providerName, cfg, nil)
}

// startConfigureProviderWithSecrets is like startConfigureProvider but lets the
// caller pass a pre-seeded secrets map (needed when the test exercises the
// "edit existing" branch, which reads the existing key).
func startConfigureProviderWithSecrets(
	t *testing.T,
	providerName string,
	cfg *config.Config,
	seedSecrets map[string]string,
) (in *tap.MockReadable, cfgOut *config.Config, resultCh chan string) {
	t.Helper()

	in, _ = setupTapTest(t)
	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)
	cliCtx.SetDeps(&Deps{Crypto: &mockCrypto{}})

	if cfg == nil {
		cfg = &config.Config{
			DefaultProvider: "",
			Providers:       map[string]config.Provider{},
		}
	}

	secrets := seedSecrets
	if secrets == nil {
		secrets = map[string]string{}
	}
	secretsPath := filepath.Join(configDir, "secrets.age")
	keyPath := filepath.Join(configDir, "key.age")

	resultCh = make(chan string)
	go func() {
		result, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: providerName,
			Secrets:      secrets,
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
		})
		if err != nil {
			resultCh <- "error:" + err.Error()
			return
		}
		resultCh <- result
	}()

	return in, cfg, resultCh
}
