package validate

import (
	"net"
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

func TestValidationError(t *testing.T) {
	t.Run("NewValidationError creates correct error", func(t *testing.T) {
		msg := "test error message"
		err := NewValidationError(msg)
		if err == nil {
			t.Fatal("NewValidationError returned nil")
		}
		if err.Error() != msg {
			t.Errorf("Error() = %q, want %q", err.Error(), msg)
		}
	})

	t.Run("ValidationError type assertion", func(t *testing.T) {
		err := NewValidationError("test")
		_, ok := err.(*ValidationError)
		if !ok {
			t.Error("Error should be of type *ValidationError")
		}
	})

	t.Run("ErrInvalidAPIKey is ValidationError", func(t *testing.T) {
		var _ error = ErrInvalidAPIKey
		_, ok := any(ErrInvalidAPIKey).(*ValidationError)
		if !ok {
			t.Error("ErrInvalidAPIKey should be of type *ValidationError")
		}
	})

	t.Run("ErrInvalidURL is ValidationError", func(t *testing.T) {
		var _ error = ErrInvalidURL
		_, ok := any(ErrInvalidURL).(*ValidationError)
		if !ok {
			t.Error("ErrInvalidURL should be of type *ValidationError")
		}
	})
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"public IP 8.8.8.8", "8.8.8.8", false},
		{"public IP 1.1.1.1", "1.1.1.1", false},
		{"private 10.x", "10.0.0.1", true},
		{"private 10.255.255.255", "10.255.255.255", true},
		{"private 172.16.0.0", "172.16.0.0", true},
		{"private 172.31.255.255", "172.31.255.255", true},
		{"private 192.168.0.0", "192.168.0.0", true},
		{"private 192.168.255.255", "192.168.255.255", true},
		{"link-local 169.254.0.0", "169.254.0.0", true},
		{"link-local 169.254.255.255", "169.254.255.255", true},
		{"public IP 9.9.9.9", "9.9.9.9", false},
		{"public IP 203.0.113.1", "203.0.113.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsBlockedHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{"localhost", "localhost", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"::1 IPv6 localhost", "::1", true},
		{"public host", "api.example.com", false},
		{"public IP", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBlockedHost(tt.host)
			if got != tt.want {
				t.Errorf("isBlockedHost(%s) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
