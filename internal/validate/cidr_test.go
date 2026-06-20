package validate

import (
	"net"
	"testing"
)

// TestHardcodedCIDRs verifies every entry in hardcodedCIDRs parses successfully
// and matches a non-nil *net.IPNet.
func TestHardcodedCIDRs(t *testing.T) {
	for _, cidr := range hardcodedCIDRs {
		t.Run(cidr, func(t *testing.T) {
			_, ipnet, err := net.ParseCIDR(cidr)
			if err != nil {
				t.Fatalf("hardcoded CIDR %q failed to parse: %v", cidr, err)
			}
			if ipnet == nil {
				t.Fatalf("net.ParseCIDR(%q) returned nil IPNet with no error", cidr)
			}
		})
	}
}
