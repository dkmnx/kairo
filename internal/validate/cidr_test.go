package validate

import (
	"net"
	"testing"
)

// TestHardcodedCIDRs verifies all hardcoded CIDR constants parse correctly,
// replacing the previous init-only panic with a deterministic build-time check.
// The cidr* constants are the single source of truth shared with production
// var declarations, so there is no drift risk.
func TestHardcodedCIDRs(t *testing.T) {
	tests := []struct {
		name string
		cidr string
	}{
		{"cidrPrivate10", cidrPrivate10},
		{"cidrPrivate172", cidrPrivate172},
		{"cidrPrivate192", cidrPrivate192},
		{"cidrLinkLocal", cidrLinkLocal},
		{"cidrULAv6", cidrULAv6},
		{"cidrLinkLocalV6", cidrLinkLocalV6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipnet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("hardcoded CIDR %q failed to parse: %v", tt.cidr, err)
			}
			if ipnet == nil {
				t.Fatal("net.ParseCIDR returned nil IPNet with no error")
			}
		})
	}
}
