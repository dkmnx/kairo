package env

import (
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("returns custom config dir when set", func(t *testing.T) {
		SetConfigDir("/custom/path")
		defer SetConfigDir("")

		dir := GetConfigDir()
		if dir != "/custom/path" {
			t.Errorf("GetConfigDir() = %q, want %q", dir, "/custom/path")
		}
	})

	t.Run("returns default config dir from home", func(t *testing.T) {
		defer SetConfigDir("")

		home, err := UserHomeDir()
		if err != nil {
			t.Skip("cannot find home directory")
		}

		expected := filepath.Join(home, ".config", "kairo")
		dir := GetConfigDir()
		if dir != expected {
			t.Errorf("GetConfigDir() = %q, want %q", dir, expected)
		}
	})

	t.Run("setConfigDir affects getConfigDir", func(t *testing.T) {
		SetConfigDir("")
		if configDir != "" {
			t.Errorf("configDir = %q, want empty", configDir)
		}
	})
}

func TestUserHomeDir(t *testing.T) {
	home, err := UserHomeDir()
	if err != nil {
		t.Errorf("UserHomeDir() error = %v", err)
	}
	if home == "" {
		t.Error("UserHomeDir() returned empty string")
	}
	if !filepath.IsAbs(home) {
		t.Errorf("UserHomeDir() = %q, want absolute path", home)
	}
}

func TestIsSubPath(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{"true subpath", "/a/b/c", "/a/b/c/d", true},
		{"same path", "/a/b/c", "/a/b/c", false},
		{"parent is subpath of child", "/a/b/c/d", "/a/b/c", false},
		{"sibling", "/a/b/c", "/a/b/d", false},
		{"unrelated", "/a/b", "/x/y", false},
		{"absolute child", "/a/b", "/etc/passwd", false},
		{"dot path", "/a/b", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSubPath(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("IsSubPath(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}

func TestSetConfigDir(t *testing.T) {
	SetConfigDir("/test/path")
	if configDir != "/test/path" {
		t.Errorf("configDir = %q, want %q", configDir, "/test/path")
	}

	SetConfigDir("")
	if configDir != "" {
		t.Errorf("configDir = %q, want empty", configDir)
	}
}
