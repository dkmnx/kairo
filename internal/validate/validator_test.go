package validate

import (
	"testing"
)

func TestProviderValidation(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"empty key", "", true},
		{"short key (7 chars)", "sk-abc", true},
		{"valid key (8 chars)", "sk-abcde", false},
		{"long valid key", "sk-ant-" + string(make([]byte, 50)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestURLValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty URL", "", true},
		{"http instead of https", "http://api.example.com", true},
		{"localhost", "https://localhost/api", true},
		{"127.0.0.1", "https://127.0.0.1/api", true},
		{"private IP 10.x", "https://10.0.0.1/api", true},
		{"private IP 172.16.x", "https://172.16.0.1/api", true},
		{"private IP 192.168.x", "https://192.168.1.1/api", true},
		{"valid HTTPS", "https://api.example.com/anthropic", false},
		{"valid with path", "https://api.example.com/v1/anthropic", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
