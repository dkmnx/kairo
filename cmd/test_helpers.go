package cmd

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/update"
	"github.com/dkmnx/kairo/internal/wrapper"
)

// testEchoCmd returns a command that prints "mocked" to stdout. It uses
// "cmd /c echo" on Windows because echo is a shell built-in there.
func testEchoCmd() *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(context.Background(), "cmd", "/c", "echo", "mocked")
	}

	return exec.CommandContext(context.Background(), "echo", "mocked")
}

// mockProcess is a test double for ProcessRunner with configurable function fields.
type mockProcess struct {
	LookPathFn           func(file string) (string, error)
	ExecCommandContextFn func(ctx context.Context, name string, arg ...string) *exec.Cmd
	ExitProcessFn        func(code int)
}

func (m *mockProcess) LookPath(file string) (string, error) { return m.LookPathFn(file) }
func (m *mockProcess) ExecCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return m.ExecCommandContextFn(ctx, name, arg...)
}
func (m *mockProcess) ExitProcess(code int) { m.ExitProcessFn(code) }

// mockWrapper is a test double for WrapperService.
type mockWrapper struct {
	CreateTempAuthDirFn     func() (string, error)
	WriteTempTokenFileFn    func(authDir, token string) (string, error)
	GenerateWrapperScriptFn func(cfg wrapper.ScriptConfig) (string, bool, error)
}

func (m *mockWrapper) CreateTempAuthDir() (string, error) { return m.CreateTempAuthDirFn() }
func (m *mockWrapper) WriteTempTokenFile(authDir, token string) (string, error) {
	return m.WriteTempTokenFileFn(authDir, token)
}
func (m *mockWrapper) GenerateWrapperScript(cfg wrapper.ScriptConfig) (string, bool, error) {
	return m.GenerateWrapperScriptFn(cfg)
}

// mockUpdate is a test double for UpdateService.
type mockUpdate struct {
	FetchLatestReleaseFn        func(ctx context.Context) (*update.Release, error)
	ConfirmUpdateFn             func(message string) (bool, error)
	DownloadToTempFileFn        func(ctx context.Context, url string) (string, error)
	DownloadAndParseChecksumsFn func(ctx context.Context, url string) (map[string]string, error)
	VerifyChecksumFn            func(scriptPath, expectedHash string) error
	VerifyCosignBundleFn        func(ctx context.Context, tag string) error
	RunInstallScriptFn          func(scriptPath string) error
}

func (m *mockUpdate) FetchLatestRelease(ctx context.Context) (*update.Release, error) {
	return m.FetchLatestReleaseFn(ctx)
}
func (m *mockUpdate) ConfirmUpdate(message string) (bool, error) { return m.ConfirmUpdateFn(message) }
func (m *mockUpdate) DownloadToTempFile(ctx context.Context, url string) (string, error) {
	return m.DownloadToTempFileFn(ctx, url)
}
func (m *mockUpdate) DownloadAndParseChecksums(ctx context.Context, url string) (map[string]string, error) {
	return m.DownloadAndParseChecksumsFn(ctx, url)
}
func (m *mockUpdate) VerifyChecksum(scriptPath, expectedHash string) error {
	return m.VerifyChecksumFn(scriptPath, expectedHash)
}
func (m *mockUpdate) VerifyCosignBundle(ctx context.Context, tag string) error {
	return m.VerifyCosignBundleFn(ctx, tag)
}
func (m *mockUpdate) RunInstallScript(scriptPath string) error {
	return m.RunInstallScriptFn(scriptPath)
}

// mockCrypto is a test double for crypto.Service with configurable function fields.
type mockCrypto struct {
	GenerateKeyFn         func(ctx context.Context, keyPath string) error
	EncryptSecretsFn      func(ctx context.Context, secretsPath, keyPath, secrets string) error
	DecryptSecretsFn      func(ctx context.Context, secretsPath, keyPath string) (string, error)
	DecryptSecretsBytesFn func(ctx context.Context, secretsPath, keyPath string) ([]byte, error)
	EnsureKeyExistsFn     func(ctx context.Context, configDir string) error
}

func (m *mockCrypto) GenerateKey(ctx context.Context, keyPath string) error {
	if m.GenerateKeyFn != nil {
		return m.GenerateKeyFn(ctx, keyPath)
	}

	return nil
}

func (m *mockCrypto) EncryptSecrets(ctx context.Context, secretsPath, keyPath, secrets string) error {
	if m.EncryptSecretsFn != nil {
		return m.EncryptSecretsFn(ctx, secretsPath, keyPath, secrets)
	}

	return nil
}

func (m *mockCrypto) DecryptSecrets(ctx context.Context, secretsPath, keyPath string) (string, error) {
	if m.DecryptSecretsFn != nil {
		return m.DecryptSecretsFn(ctx, secretsPath, keyPath)
	}

	return "", nil
}

func (m *mockCrypto) DecryptSecretsBytes(ctx context.Context, secretsPath, keyPath string) ([]byte, error) {
	if m.DecryptSecretsBytesFn != nil {
		return m.DecryptSecretsBytesFn(ctx, secretsPath, keyPath)
	}

	return []byte{}, nil
}

func (m *mockCrypto) EnsureKeyExists(ctx context.Context, configDir string) error {
	if m.EnsureKeyExistsFn != nil {
		return m.EnsureKeyExistsFn(ctx, configDir)
	}

	return nil
}

// feedStdin replaces os.Stdin with a pipe pre-filled with input and registered
// for cleanup. The test reads from os.Stdin (e.g. via fmt.Scanln).
func feedStdin(t *testing.T, input string) {
	t.Helper()
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdin = r
}

// testDeps creates a Deps with mock implementations. The optional callback
// receives the three mock structs for field-level configuration.
func testDeps(overrides ...func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate)) *Deps {
	mp := &mockProcess{
		LookPathFn:           func(string) (string, error) { return "", nil },
		ExecCommandContextFn: func(context.Context, string, ...string) *exec.Cmd { return nil },
		ExitProcessFn:        func(int) {},
	}
	mw := &mockWrapper{
		CreateTempAuthDirFn:     func() (string, error) { return "", nil },
		WriteTempTokenFileFn:    func(string, string) (string, error) { return "", nil },
		GenerateWrapperScriptFn: func(wrapper.ScriptConfig) (string, bool, error) { return "", false, nil },
	}
	mu := &mockUpdate{
		FetchLatestReleaseFn:        func(context.Context) (*update.Release, error) { return nil, nil },
		ConfirmUpdateFn:             func(string) (bool, error) { return false, nil },
		DownloadToTempFileFn:        func(context.Context, string) (string, error) { return "", nil },
		DownloadAndParseChecksumsFn: func(context.Context, string) (map[string]string, error) { return nil, nil },
		VerifyChecksumFn:            func(string, string) error { return nil },
		VerifyCosignBundleFn:        func(context.Context, string) error { return nil },
		RunInstallScriptFn:          func(string) error { return nil },
	}
	for _, fn := range overrides {
		fn(mp, mw, mu)
	}

	return &Deps{Process: mp, Wrapper: mw, Update: mu, Crypto: crypto.DefaultService{}}
}
