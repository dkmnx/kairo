package validate

import (
	"net/url"
	"strings"
	"testing"
)

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
		{"unspecified IPv4", "https://0.0.0.0/api", "TestProvider", true},
		{"unspecified IPv6", "https://[::]/api", "TestProvider", true},
		{"cloud metadata", "https://169.254.169.254/latest/meta-data/", "TestProvider", true},
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

func TestValidateURL_InvalidParse(t *testing.T) {
	err := ValidateURL("://invalid-url", "TestProvider")
	if err == nil {
		t.Error("ValidateURL() should fail for unparsable URL")
	}
}

func TestValidateURL_SchemeNotHTTPS(t *testing.T) {
	err := ValidateURL("http://api.example.com", "TestProvider")
	if err == nil {
		t.Error("ValidateURL() should fail for non-HTTPS URL")
	}
}

func TestValidateURL_EmptyHost(t *testing.T) {
	err := ValidateURL("https:///path", "TestProvider")
	if err == nil {
		t.Error("ValidateURL() should fail for URL with empty host")
	}
}

// FuzzValidateURL fuzzes the ValidateURL function with random inputs.
func FuzzValidateURL(f *testing.F) {
	// Seed with some initial values
	f.Add("https://api.example.com/anthropic", "TestProvider")
	f.Add("", "TestProvider")
	f.Add("http://api.example.com", "TestProvider")
	f.Add("https://localhost/api", "TestProvider")
	f.Add("https://127.0.0.1/api", "TestProvider")
	f.Add("https://10.0.0.1/api", "TestProvider")
	f.Add("https://172.16.0.1/api", "TestProvider")
	f.Add("https://192.168.1.1/api", "TestProvider")
	f.Add("https://api.z.ai/api/anthropic", "zai")
	f.Add("https://api.minimax.io/v1", "minimax")

	f.Fuzz(func(t *testing.T, rawURL, providerName string) {
		err := ValidateURL(rawURL, providerName)

		if err != nil && providerName != "" {
			errMsg := err.Error()
			if !strings.Contains(errMsg, providerName) {
				t.Errorf("ValidateURL() error message should include provider name %q, got: %v", providerName, errMsg)
			}
		}

		if rawURL == "" && err == nil {
			t.Errorf("ValidateURL() should fail for empty URL")
		}

		if strings.HasPrefix(rawURL, "http://") && err == nil {
			t.Errorf("ValidateURL() should fail for HTTP URL: %s", rawURL)
		}

		// Parse the URL to check for blocked hosts
		parsed, parseErr := url.Parse(rawURL)
		if parseErr == nil && parsed.Host != "" {
			host := parsed.Hostname()
			if host == "localhost" && err == nil {
				t.Errorf("ValidateURL() should fail for localhost URL: %s", rawURL)
			}
			if host == "127.0.0.1" && err == nil {
				t.Errorf("ValidateURL() should fail for 127.0.0.1 URL: %s", rawURL)
			}
		}
	})
}
