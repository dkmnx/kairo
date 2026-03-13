package validate

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

type KeyFormat struct {
	MinLength int
	Prefix    string
	Pattern   string
	compiled  *regexp.Regexp
}

// compilePattern compiles the regex pattern if not already compiled.
func (kf *KeyFormat) compilePattern() error {
	if kf.Pattern == "" {
		return nil
	}
	if kf.compiled != nil {
		return nil
	}
	compiled, err := regexp.Compile(kf.Pattern)
	if err != nil {
		return err
	}
	kf.compiled = compiled

	return nil
}

// matchesPattern checks if the key matches the compiled pattern.
func (kf *KeyFormat) matchesPattern(key string) (bool, error) {
	if kf.Pattern == "" {
		return true, nil
	}
	if err := kf.compilePattern(); err != nil {
		return false, err
	}

	return kf.compiled.MatchString(key), nil
}

var providerKeyFormats = map[string]KeyFormat{
	"zai":      {MinLength: 32},
	"minimax":  {MinLength: 32},
	"kimi":     {MinLength: 32},
	"deepseek": {MinLength: 32},
	"custom":   {MinLength: 20},
}

// Private IP CIDR blocks for URL validation.
// Defined at package level for efficiency (parsed once at startup).
// We manually define these rather than using net.InterfaceAddrs() because:
// 1. This is more explicit and covers all RFC 1918 private ranges
// 2. Includes link-local addresses (169.254.0.0/16) which should also be blocked
// 3. No need to enumerate network interfaces at runtime
var (
	private10  = mustParseCIDR("10.0.0.0/8")
	private172 = mustParseCIDR("172.16.0.0/12")
	private192 = mustParseCIDR("192.168.0.0/16")
	linkLocal  = mustParseCIDR("169.254.0.0/16")
)

func mustParseCIDR(s string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR %s: %v", s, err))
	}

	return *ipnet
}

func ValidateAPIKey(key string, providerName string) error {
	if strings.TrimSpace(key) == "" {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("%s: API key cannot be empty or whitespace", providerName))
	}

	format, knownProvider := providerKeyFormats[providerName]
	if !knownProvider {
		format = KeyFormat{MinLength: 20}
	}

	if len(key) < format.MinLength {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("%s: API key too short (minimum %d characters, got %d)", providerName, format.MinLength, len(key)))
	}

	if format.Prefix != "" && !strings.HasPrefix(key, format.Prefix) {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("%s: API key must start with '%s'", providerName, format.Prefix))
	}

	if format.Pattern != "" {
		matched, err := format.matchesPattern(key)
		if err != nil || !matched {
			return kairoerrors.NewError(kairoerrors.ValidationError,
				fmt.Sprintf("%s: API key format is invalid", providerName))
		}
	}

	return nil
}

func ValidateURL(rawURL string, providerName string) error {
	if rawURL == "" {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("%s: base URL cannot be empty", providerName))
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("%s: base URL is not a valid URL: %v", providerName, err))
	}

	if parsed.Scheme != "https" {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("%s: base URL must use HTTPS protocol", providerName))
	}

	host := parsed.Hostname()
	if host == "" {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("%s: base URL missing host component", providerName))
	}

	if isBlockedHost(host) {
		return kairoerrors.NewError(kairoerrors.ValidationError,
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
		linkLocal.Contains(ip)
}
