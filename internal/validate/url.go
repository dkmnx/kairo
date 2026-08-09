package validate

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/dkmnx/kairo/internal/errors"
)

// hardcodedCIDRs are the private and link-local CIDR ranges blocked by
// ValidateURL. Parsed once at package init into blockedCIDRs; a typo here
// fails TestHardcodedCIDRs rather than panicking at startup.
var hardcodedCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"fc00::/7",
	"fe80::/10",
	"127.0.0.0/8",
	"0.0.0.0/8",
	"::1/128",
}

var blockedCIDRs = mustParseCIDRs(hardcodedCIDRs)

// blockedHosts holds literal hosts that are not already covered by
// blockedCIDRs. "::" (IPv6 unspecified) is not inside ::1/128, and
// "localhost" is a name, so both need explicit entries; all literal
// loopback/private IPs are covered by the CIDR list.
var blockedHosts = []string{
	"localhost",
	"::",
}

// mustParseCIDRs parses each CIDR string and panics if any are malformed.
// Inputs are package constants covered by TestHardcodedCIDRs.
func mustParseCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, len(cidrs))
	for i, c := range cidrs {
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			panic(fmt.Sprintf("validate: malformed hardcoded CIDR %q: %v", c, err))
		}
		out[i] = ipnet
	}

	return out
}

// lookupIP resolves a hostname to its IP addresses. It is a package-level
// variable so tests can stub DNS resolution without network access.
var lookupIP = func(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, net.IP(a.AsSlice()))
	}

	return ips, nil
}

// ValidateURL checks that the given URL is a valid HTTPS URL without blocked hosts.
func ValidateURL(rawURL, providerName string) error {
	if rawURL == "" {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: base URL cannot be empty", providerName))
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: base URL is not a valid URL: %v", providerName, err))
	}

	if parsed.Scheme != "https" {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: base URL must use HTTPS protocol", providerName))
	}

	host := parsed.Hostname()
	if host == "" {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: base URL missing host component", providerName))
	}

	if isBlockedHost(host) {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: base URL cannot use blocked host: %s (localhost/private IPs not allowed)", providerName, host))
	}

	return nil
}

// isBlockedHost reports whether host is blocked: a literal blocked name, a
// private/link-local IP in canonical or legacy (inet_aton) encoding, or a
// hostname that resolves to a private/link-local address.
func isBlockedHost(host string) bool {
	// A trailing dot marks an FQDN; strip it so "localhost." and "127.0.0.1."
	// are checked in their canonical forms.
	host = strings.TrimSuffix(host, ".")

	if slices.Contains(blockedHosts, host) {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return isPrivateIP(ip)
	}

	// net.ParseIP rejects non-canonical numeric hosts (e.g. "127.1",
	// "2130706433", "0x7f000001", "0177.0.0.1"), but many resolvers still
	// interpret them via inet_aton rules, so normalize and check them.
	if ip, ok := parseLegacyNumericHost(host); ok {
		return isPrivateIP(ip)
	}

	// Hostnames: block if any resolved address is private/link-local.
	// A name that fails to resolve now is accepted (it cannot be reached
	// until it resolves); this leaves a DNS-rebinding TOCTOU window, which
	// is documented in the security notes.
	addrs, err := lookupIP(context.Background(), host)
	if err != nil {
		return false
	}
	for _, ip := range addrs {
		if isPrivateIP(ip) {
			return true
		}
	}

	return false
}

// parseLegacyNumericHost interprets host using classic inet_aton grammar:
// one to four dot-separated parts, each decimal, octal (leading 0), or
// hexadecimal (0x/0X prefix). It returns the resulting IPv4 address, or
// ok=false if host is not a legacy numeric form.
func parseLegacyNumericHost(host string) (net.IP, bool) {
	parts := strings.Split(host, ".")
	if len(parts) < 1 || len(parts) > 4 {
		return nil, false
	}

	vals := make([]uint32, len(parts))
	for i, p := range parts {
		v, ok := parseNumericPart(p)
		if !ok {
			return nil, false
		}
		vals[i] = v
	}

	var addr uint32
	switch len(vals) {
	case 1:
		addr = vals[0]
	case 2:
		if vals[0] > 0xff || vals[1] > 0xffffff {
			return nil, false
		}
		addr = vals[0]<<24 | vals[1]
	case 3:
		if vals[0] > 0xff || vals[1] > 0xff || vals[2] > 0xffff {
			return nil, false
		}
		addr = vals[0]<<24 | vals[1]<<16 | vals[2]
	case 4:
		if vals[0] > 0xff || vals[1] > 0xff || vals[2] > 0xff || vals[3] > 0xff {
			return nil, false
		}
		addr = vals[0]<<24 | vals[1]<<16 | vals[2]<<8 | vals[3]
	}

	ip := make(net.IP, net.IPv4len)
	ip[0] = byte(addr >> 24)
	ip[1] = byte(addr >> 16)
	ip[2] = byte(addr >> 8)
	ip[3] = byte(addr)

	return ip, true
}

// parseNumericPart parses a single inet_aton address component in decimal,
// octal (leading 0), or hexadecimal (0x/0X prefix) notation.
func parseNumericPart(p string) (uint32, bool) {
	if p == "" {
		return 0, false
	}

	base := 10
	digits := p
	if len(p) > 2 && (p[0] == '0' && (p[1] == 'x' || p[1] == 'X')) {
		base = 16
		digits = p[2:]
	} else if len(p) > 1 && p[0] == '0' {
		base = 8
		digits = p[1:]
	}

	var v uint64
	for _, r := range digits {
		var d uint64
		switch {
		case r >= '0' && r <= '9':
			d = uint64(r - '0')
		case base == 16 && r >= 'a' && r <= 'f':
			d = uint64(r-'a') + 10
		case base == 16 && r >= 'A' && r <= 'F':
			d = uint64(r-'A') + 10
		default:
			return 0, false
		}
		if d >= uint64(base) {
			return 0, false
		}
		v = v*uint64(base) + d
		if v > 0xffffffff {
			return 0, false
		}
	}

	return uint32(v), true
}

func isPrivateIP(ip net.IP) bool {
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}
