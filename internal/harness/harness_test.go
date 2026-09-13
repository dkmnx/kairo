package harness

import (
	"os"
	"testing"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"claude", Claude, true},
		{"qwen", Qwen, true},
		{"pi", Pi, true},
		{"crush", Crush, true},
		{"empty", "", false},
		{"unknown", "unknown", false},
		{"partial", "claud", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValid(tt.input)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name   string
		flag   string
		config string
		want   string
	}{
		{"flag takes precedence", Qwen, Claude, Qwen},
		{"config fallback", "", Qwen, Qwen},
		{"both empty means detect", "", "", ""},
		{"unknown flag falls back to config", "unknown", Claude, Claude},
		{"unknown flag and config empty", "unknown", "", ""},
		{"unknown config with empty flag", "", "unknown", ""},
		{"pi over config", Pi, Claude, Pi},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.flag, tt.config)
			if got != tt.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tt.flag, tt.config, got, tt.want)
			}
		})
	}
}

func TestSupportedOrder(t *testing.T) {
	got := Supported()
	want := []string{Pi, Claude, Qwen, Crush}
	if len(got) != len(want) {
		t.Fatalf("Supported() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Supported()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDetectInstalled(t *testing.T) {
	t.Run("first installed in priority order", func(t *testing.T) {
		installed := map[string]bool{Claude: true, Crush: true}
		got := DetectInstalled(func(file string) (string, error) {
			if installed[file] {
				return "/usr/bin/" + file, nil
			}

			return "", os.ErrNotExist
		})
		if got != Claude {
			t.Errorf("DetectInstalled() = %q, want %q (before crush)", got, Claude)
		}
	})

	t.Run("prefers pi when present", func(t *testing.T) {
		got := DetectInstalled(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		})
		if got != Pi {
			t.Errorf("DetectInstalled() = %q, want %q", got, Pi)
		}
	})

	t.Run("empty when none installed", func(t *testing.T) {
		got := DetectInstalled(func(string) (string, error) {
			return "", os.ErrNotExist
		})
		if got != "" {
			t.Errorf("DetectInstalled() = %q, want empty", got)
		}
	})

	t.Run("nil lookPath", func(t *testing.T) {
		if got := DetectInstalled(nil); got != "" {
			t.Errorf("DetectInstalled(nil) = %q, want empty", got)
		}
	})
}

func TestDispatch(t *testing.T) {
	tests := []struct {
		name         string
		harness      string
		providerName string
		model        string
		wantDisplay  string
		wantEnv      string
		wantExtraLen int
	}{
		{
			name: "claude", harness: Claude, providerName: "test",
			wantDisplay: "Claude", wantEnv: "", wantExtraLen: 0,
		},
		{
			name: "qwen", harness: Qwen, providerName: "test", model: "qwen-plus",
			wantDisplay: "Qwen", wantEnv: "ANTHROPIC_API_KEY", wantExtraLen: 4,
		},
		{
			name: "pi", harness: Pi, providerName: "test",
			wantDisplay: "Pi", wantEnv: "", wantExtraLen: 0,
		},
		{
			name: "crush", harness: Crush, providerName: "test",
			wantDisplay: "Crush", wantEnv: "TEST_API_KEY", wantExtraLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, e, x := Dispatch(tt.harness, tt.providerName, tt.model)
			if d != tt.wantDisplay {
				t.Errorf("display = %q, want %q", d, tt.wantDisplay)
			}
			if e != tt.wantEnv {
				t.Errorf("envVar = %q, want %q", e, tt.wantEnv)
			}
			if len(x) != tt.wantExtraLen {
				t.Errorf("extraArgs len = %d, want %d", len(x), tt.wantExtraLen)
			}
		})
	}
}

func TestYoloFlag(t *testing.T) {
	tests := []struct {
		name    string
		harness string
		want    string
	}{
		{"claude", Claude, "--dangerously-skip-permissions"},
		{"qwen", Qwen, "--yolo"},
		{"pi", Pi, ""},
		{"crush", Crush, "--yolo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := YoloFlag(tt.harness)
			if got != tt.want {
				t.Errorf("YoloFlag(%q) = %q, want %q", tt.harness, got, tt.want)
			}
		})
	}
}

func TestPiEnvVars(t *testing.T) {
	vars := PiEnvVars("zai", "glm-5")
	if len(vars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(vars))
	}
	if vars[0] != "PI_PROVIDER=zai" {
		t.Errorf("PI_PROVIDER = %q", vars[0])
	}
	if vars[1] != "PI_MODEL=glm-5" {
		t.Errorf("PI_MODEL = %q", vars[1])
	}
}

func TestAPIKeyEnvVar(t *testing.T) {
	if got := APIKeyEnvVar("testprovider"); got != "TESTPROVIDER_API_KEY" {
		t.Errorf("APIKeyEnvVar = %q, want TESTPROVIDER_API_KEY", got)
	}
}

func TestAPIKeyEnvVar_Hyphenated(t *testing.T) {
	if got := APIKeyEnvVar("cloudflare-workers-ai"); got != "CLOUDFLARE_WORKERS_AI_API_KEY" {
		t.Errorf("APIKeyEnvVar = %q, want CLOUDFLARE_WORKERS_AI_API_KEY", got)
	}
}
