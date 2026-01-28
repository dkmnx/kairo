package validate

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

type KeyFormat struct {
	MinLength int
	Prefix    string
	Pattern   string
}

var providerKeyFormats = map[string]KeyFormat{
	"zai":      {MinLength: 32},
	"minimax":  {MinLength: 32},
	"kimi":     {MinLength: 32},
	"deepseek": {MinLength: 32},
	"custom":   {MinLength: 20},
}

func ValidateAPIKey(key string, providerName string) error {
	if strings.TrimSpace(key) == "" {
		return &ValidationError{
			msg: fmt.Sprintf("%s API key cannot be empty or whitespace", providerName),
		}
	}

	format, knownProvider := providerKeyFormats[providerName]
	if !knownProvider {
		format = KeyFormat{MinLength: 20}
	}

	if len(key) < format.MinLength {
		return &ValidationError{
			msg: fmt.Sprintf("%s API key too short (minimum %d characters, got %d)", providerName, format.MinLength, len(key)),
		}
	}

	if format.Prefix != "" && !strings.HasPrefix(key, format.Prefix) {
		return &ValidationError{
			msg: fmt.Sprintf("%s API key must start with '%s'", providerName, format.Prefix),
		}
	}

	if format.Pattern != "" {
		matched, err := regexp.MatchString(format.Pattern, key)
		if err != nil || !matched {
			return &ValidationError{
				msg: fmt.Sprintf("%s API key format is invalid", providerName),
			}
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
	ErrInvalidAPIKey = &ValidationError{msg: "API key validation failed"}
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
