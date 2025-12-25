package validate

import (
	"strings"
	"testing"
)

func TestProviderValidation(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		providerName string
		wantErr      bool
	}{
		{"empty key", "", "TestProvider", true},
		{"short key (7 chars)", "sk-abc", "TestProvider", true},
		{"valid key (8 chars)", "sk-abcde", "TestProvider", false},
		{"long valid key", "sk-ant-" + string(make([]byte, 50)), "TestProvider", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.key, tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				errMsg := err.Error()
				if tt.providerName != "" && !strings.Contains(errMsg, tt.providerName) {
					t.Errorf("ValidateAPIKey() error message should include provider name, got: %v", errMsg)
				}
			}
		})
	}
}

func TestURLValidation(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		providerName string
		wantErr      bool
	}{
		{"empty URL", "", "TestProvider", true},
		{"http instead of https", "http://api.example.com", "TestProvider", true},
		{"localhost", "https://localhost/api", "TestProvider", true},
		{"127.0.0.1", "https://127.0.0.1/api", "TestProvider", true},
		{"private IP 10.x", "https://10.0.0.1/api", "TestProvider", true},
		{"private IP 172.16.x", "https://172.16.0.1/api", "TestProvider", true},
		{"private IP 192.168.x", "https://192.168.1.1/api", "TestProvider", true},
		{"valid HTTPS", "https://api.example.com/anthropic", "TestProvider", false},
		{"valid with path", "https://api.example.com/v1/anthropic", "TestProvider", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				errMsg := err.Error()
				if tt.providerName != "" && !strings.Contains(errMsg, tt.providerName) {
					t.Errorf("ValidateURL() error message should include provider name, got: %v", errMsg)
				}
			}
		})
	}
}
