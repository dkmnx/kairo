package validate

import (
	"fmt"
	"net"
	"net/url"
)

func ValidateAPIKey(key string, providerName string) error {
	if len(key) < 8 {
		return &ValidationError{
			msg: fmt.Sprintf("%s API key must be at least 8 characters (current: %d)", providerName, len(key)),
		}
	}
	return nil
}

func ValidateURL(rawURL string, providerName string) error {
	if rawURL == "" {
		return &ValidationError{
			msg: fmt.Sprintf("%s BaseURL cannot be empty", providerName),
		}
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return &ValidationError{
			msg: fmt.Sprintf("%s BaseURL is not a valid URL: %v", providerName, err),
		}
	}

	if parsed.Scheme != "https" {
		return &ValidationError{
			msg: fmt.Sprintf("%s BaseURL must use HTTPS protocol", providerName),
		}
	}

	host := parsed.Host
	if host == "" {
		return &ValidationError{
			msg: fmt.Sprintf("%s BaseURL missing host component", providerName),
		}
	}

	if isBlockedHost(host) {
		return &ValidationError{
			msg: fmt.Sprintf("%s BaseURL cannot use blocked host: %s (localhost/private IPs not allowed)", providerName, host),
		}
	}

	return nil
}

func isBlockedHost(host string) bool {
	blockedHosts := []string{
		"localhost",
		"127.0.0.1",
		"::1",
	}

	for _, blocked := range blockedHosts {
		if host == blocked {
			return true
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return isPrivateIP(ip)
	}

	return false
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("169.254.0.0"), Mask: net.CIDRMask(16, 32)},
	}

	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}

	return false
}

var (
	ErrInvalidAPIKey = &ValidationError{msg: "API key must be at least 8 characters"}
	ErrInvalidURL    = &ValidationError{msg: "invalid URL: must be HTTPS and not use blocked hosts"}
)

type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string {
	return e.msg
}

func NewValidationError(msg string) error {
	return &ValidationError{msg: msg}
}
