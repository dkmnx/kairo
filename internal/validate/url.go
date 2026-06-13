package validate

import (
	"fmt"
	"net"
	"net/url"
	"slices"

	"github.com/dkmnx/kairo/internal/errors"
)

const (
	cidrPrivate10   = "10.0.0.0/8"
	cidrPrivate172  = "172.16.0.0/12"
	cidrPrivate192  = "192.168.0.0/16"
	cidrLinkLocal   = "169.254.0.0/16"
	cidrULAv6       = "fc00::/7"
	cidrLinkLocalV6 = "fe80::/10"
)

// CIDR prefixes validated by TestHardcodedCIDRs.
var (
	_, private10, _   = net.ParseCIDR(cidrPrivate10)
	_, private172, _  = net.ParseCIDR(cidrPrivate172)
	_, private192, _  = net.ParseCIDR(cidrPrivate192)
	_, linkLocal, _   = net.ParseCIDR(cidrLinkLocal)
	_, ulaIPv6, _     = net.ParseCIDR(cidrULAv6)
	_, linkLocalV6, _ = net.ParseCIDR(cidrLinkLocalV6)

	blockedHosts = []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"::",
		"0.0.0.0",
		"169.254.169.254",
	}
)

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
	return private10.Contains(ip) ||
		private172.Contains(ip) ||
		private192.Contains(ip) ||
		linkLocal.Contains(ip) ||
		ulaIPv6.Contains(ip) ||
		linkLocalV6.Contains(ip)
}
