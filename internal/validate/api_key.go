package validate

import (
	"net"
	"net/url"
)

func ValidateAPIKey(key string) error {
	if len(key) < 8 {
		return ErrInvalidAPIKey
	}
	return nil
}

func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	if parsed.Scheme != "https" {
		return ErrInvalidURL
	}

	host := parsed.Host
	if host == "" {
		return ErrInvalidURL
	}

	if isBlockedHost(host) {
		return ErrInvalidURL
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
