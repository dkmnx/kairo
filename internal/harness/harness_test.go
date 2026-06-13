package harness

import "testing"

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
		{"both empty defaults to claude", "", "", Claude},
		{"unknown flag defaults to claude", "unknown", "", Claude},
		{"unknown config defaults to claude", "", "unknown", Claude},
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
