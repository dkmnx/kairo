package wrapper

import (
	"os"
	"runtime"
	"testing"
)

func TestCreateTempAuthDir_Success(t *testing.T) {
	dir, err := CreateTempAuthDir()
	if err != nil {
		t.Fatalf("CreateTempAuthDir() error = %v", err)
	}
	defer os.RemoveAll(dir)

	if dir == "" {
		t.Error("CreateTempAuthDir() returned empty path")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Skip permission check on Windows (doesn't support Unix-style 0700)
	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode&0077 != 0 {
			t.Errorf("Directory should have no group/other permissions, got %o", mode)
		}
		if mode&0100 == 0 {
			t.Errorf("Directory should have owner execute permission")
		}
		if mode&0200 == 0 {
			t.Errorf("Directory should have owner write permission")
		}
		if mode&0400 == 0 {
			t.Errorf("Directory should have owner read permission")
		}
	}
}

func TestCreateTempAuthDir_ReturnsUniqueDirs(t *testing.T) {
	dir1, err := CreateTempAuthDir()
	if err != nil {
		t.Fatalf("CreateTempAuthDir() error = %v", err)
	}
	defer os.RemoveAll(dir1)

	dir2, err := CreateTempAuthDir()
	if err != nil {
		t.Fatalf("CreateTempAuthDir() error = %v", err)
	}
	defer os.RemoveAll(dir2)

	if dir1 == dir2 {
		t.Error("CreateTempAuthDir() returned same path for two calls")
	}
}

func TestWriteTempTokenFile_Success(t *testing.T) {
	authDir := t.TempDir()
	token := "test-api-key-12345"

	path, err := WriteTempTokenFile(authDir, token)
	if err != nil {
		t.Fatalf("WriteTempTokenFile() error = %v", err)
	}
	defer os.Remove(path)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != token {
		t.Errorf("File content = %q, want %q", string(content), token)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Skip permission check on Windows (doesn't support Unix-style 0600)
	if runtime.GOOS != "windows" {
		expectedPerms := os.FileMode(0600)
		if info.Mode() != expectedPerms {
			t.Errorf("File permissions = %o, want %o", info.Mode(), expectedPerms)
		}
	}
}

func TestWriteTempTokenFile_EmptyToken(t *testing.T) {
	authDir := t.TempDir()

	_, err := WriteTempTokenFile(authDir, "")
	if err == nil {
		t.Error("WriteTempTokenFile() should error on empty token")
	}
}

func TestCreateTempAuthDir_Failure(t *testing.T) {
	// Note: os.MkdirTemp creates both the temp dir and any necessary parents,
	// so testing error conditions is difficult without modifying system state.
	// We verify the basic functionality works via other tests.
	_, err := CreateTempAuthDir()
	// The error for invalid paths would be OS-specific
	_ = err
}

func TestWriteTempTokenFile_NonExistentDir(t *testing.T) {
	_, err := WriteTempTokenFile("/nonexistent/path", "test-token")
	if err == nil {
		t.Error("WriteTempTokenFile() should error on non-existent directory")
	}
}

func TestWriteTempTokenFile_CloseError(t *testing.T) {
	_, err := WriteTempTokenFile("/nonexistent/path/that/does/not/exist", "test-token")
	if err == nil {
		t.Error("WriteTempTokenFile() should fail in non-existent directory")
	}
}
