// Package validate provides input validation for API keys, URLs, and provider configuration.
package validate

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"

	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
)

// ValidateAPIKey checks that the given key meets the format requirements for the provider.
func ValidateAPIKey(key, providerName string) error {
	def, ok := providers.BuiltInProvider(providerName)
	if !ok {
		def = providers.ProviderDefinition{Name: providerName, KeyFormat: providers.DefaultKeyFormat}
	}

	return def.ValidateAPIKey(key)
}

var (
	msgInvalidCIDR = "kairo: invalid hardcoded CIDR %q: %v\n"

	private10   = mustParseCIDR("10.0.0.0/8")
	private172  = mustParseCIDR("172.16.0.0/12")
	private192  = mustParseCIDR("192.168.0.0/16")
	linkLocal   = mustParseCIDR("169.254.0.0/16")
	ulaIPv6     = mustParseCIDR("fc00::/7")
	linkLocalV6 = mustParseCIDR("fe80::/10")
)

func mustParseCIDR(s string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, msgInvalidCIDR, s, err)
		os.Exit(1)
	}

	return *ipnet
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
	blockedHosts := []string{
		"localhost",
		"127.0.0.1",
		"::1",
	}

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
