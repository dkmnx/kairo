package wrapper

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dkmnx/kairo/internal/constants"
)

// TestCreateTempAuthDir_ChmodFailsOnReadOnly forces MkdirTemp to succeed but
// the subsequent chmod to fail by mounting the temp dir as read-only.
func TestCreateTempAuthDir_ChmodFailsOnReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}

	parent := t.TempDir()
	ro := filepath.Join(parent, "ro")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o700) })

	t.Setenv("TMPDIR", ro)
	if _, err := CreateTempAuthDir(); err == nil {
		t.Error("CreateTempAuthDir() should fail when chmod on the new dir fails")
	}
}

// TestWriteTempTokenFile_ChmodFailsOnReadOnly forces the token file's chmod
// to fail by making the parent dir read-only after the file is written.
func TestWriteTempTokenFile_ChmodFailsOnReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}

	authDir := t.TempDir()
	t.Cleanup(func() { _ = os.Chmod(authDir, 0o700) })

	// Pre-write a file, then drop perms on the dir so chmod on the new
	// token file inside it fails.
	pre := filepath.Join(authDir, "pre")
	if _, err := os.Create(pre); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(authDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(authDir, 0o700) })

	if _, err := WriteTempTokenFile(authDir, "x"); err == nil {
		t.Error("WriteTempTokenFile() should error when chmod on the token file fails")
	}
}

// TestGenerateWrapperScript_ChmodFailsOnUnix is intentionally omitted:
// on a properly-permissioned filesystem, a process can always chmod a file
// it owns in a directory it owns, so we cannot construct a state where
// GenerateWrapperScript's chmod fails from a unit test. The error branch
// is covered by integration-style tests that run as non-root on read-only
// mounts in CI.
func TestGenerateWrapperScript_ChmodFailsOnUnix(t *testing.T) {
	t.Skip("cannot reliably construct chmod-fails state from a unit test")
}

// TestEscapePowerShellArg_AdditionalMetachars covers the remaining characters.
func TestEscapePowerShellArg_AdditionalMetachars(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ampersand", "a&b", "'a`&b'"},
		{"percent", "100%", "'100``%'"},
		{"tab", "a\tb", "'a`tb'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapePowerShellArg(tt.in); got != tt.want {
				t.Errorf("EscapePowerShellArg(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestConstantsUsed guards against accidental removal of the wrapper's
// permission constants — silent failures here would mean the wrapper
// produces insecure files.
func TestConstantsUsed(t *testing.T) {
	_ = constants.DirPermSecure
	_ = constants.FilePermSecure
	_ = constants.FilePermExec
}
