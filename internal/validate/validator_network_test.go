package validate

import (
	"net"
	"strings"
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
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		if strings.Contains(host, "private") {
			return []net.IP{net.ParseIP("10.0.0.1")}, nil
		}

		return stubPublicResolver(host)
	})

	tests := []struct {
		name string
		host string
		want bool
	}{
		{"localhost", "localhost", true},
		{"localhost trailing dot", "localhost.", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"127.0.0.1 trailing dot", "127.0.0.1.", true},
		{"::1 IPv6 localhost", "::1", true},
		{"shortened loopback", "127.1", true},
		{"shortened loopback 3-part", "127.0.1", true},
		{"decimal loopback", "2130706433", true},
		{"hex loopback", "0x7f000001", true},
		{"octal loopback", "0177.0.0.1", true},
		{"hex dotted loopback", "0x7f.0.0.1", true},
		{"hostname resolving to private", "internal.private", true},
		{"public host", "api.example.com", false},
		{"public IP", "8.8.8.8", false},
		{"numeric-looking hostname with letter", "x1", false},
		{"hostname with x not 0x-prefixed", "examplex.com", false},
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
