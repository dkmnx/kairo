package validate

import (
	"net"
	"testing"
)

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
