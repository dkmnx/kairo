package cmd

import (
	"context"
	"os/exec"

	"github.com/dkmnx/kairo/internal/update"
	"github.com/dkmnx/kairo/internal/wrapper"
)

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
	GetLatestReleaseFn          func() (*update.Release, error)
	ConfirmUpdateFn             func(message string) (bool, error)
	DownloadToTempFileFn        func(url string) (string, error)
	DownloadAndParseChecksumsFn func(url string) (map[string]string, error)
	VerifyChecksumFn            func(scriptPath, expectedHash string) error
	VerifyCosignBundleFn        func(tag string) error
	RunInstallScriptFn          func(scriptPath string) error
}

func (m *mockUpdate) GetLatestRelease() (*update.Release, error) {
	return m.GetLatestReleaseFn()
}
func (m *mockUpdate) ConfirmUpdate(message string) (bool, error) { return m.ConfirmUpdateFn(message) }
func (m *mockUpdate) DownloadToTempFile(url string) (string, error) {
	return m.DownloadToTempFileFn(url)
}
func (m *mockUpdate) DownloadAndParseChecksums(url string) (map[string]string, error) {
	return m.DownloadAndParseChecksumsFn(url)
}
func (m *mockUpdate) VerifyChecksum(scriptPath, expectedHash string) error {
	return m.VerifyChecksumFn(scriptPath, expectedHash)
}
func (m *mockUpdate) VerifyCosignBundle(tag string) error { return m.VerifyCosignBundleFn(tag) }
func (m *mockUpdate) RunInstallScript(scriptPath string) error {
	return m.RunInstallScriptFn(scriptPath)
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
		GetLatestReleaseFn:          func() (*update.Release, error) { return nil, nil },
		ConfirmUpdateFn:             func(string) (bool, error) { return false, nil },
		DownloadToTempFileFn:        func(string) (string, error) { return "", nil },
		DownloadAndParseChecksumsFn: func(string) (map[string]string, error) { return nil, nil },
		VerifyChecksumFn:            func(string, string) error { return nil },
		VerifyCosignBundleFn:        func(string) error { return nil },
		RunInstallScriptFn:          func(string) error { return nil },
	}
	for _, fn := range overrides {
		fn(mp, mw, mu)
	}

	return &Deps{Process: mp, Wrapper: mw, Update: mu}
}
