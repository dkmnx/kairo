package validate

import (
	"fmt"
	"net"
	"net/url"
	"slices"

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
}

var blockedCIDRs = mustParseCIDRs(hardcodedCIDRs)

var blockedHosts = []string{
	"localhost",
	"127.0.0.1",
	"::1",
	"::",
	"0.0.0.0",
	"169.254.169.254",
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

func isBlockedHost(host string) bool {
	if slices.Contains(blockedHosts, host) {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return isPrivateIP(ip)
	}

	return false
}

func isPrivateIP(ip net.IP) bool {
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}
