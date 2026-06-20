// Package update provides self-update logic: release checking, download,
// checksum verification, and install script execution.
package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/httpfetch"
)

const (
	checksumsFilename   = "checksums.txt"
	installScriptExt    = ".sh"
	installScriptExtPS1 = ".ps1"
)

var checksumHashPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// Client holds injectable dependencies for update operations.
type Client struct {
	HTTPClient   *http.Client
	EnvFunc      func(key string) (string, bool)
	LookPathFunc func(string) (string, error)
	ExecCommand  func(ctx context.Context, name string, arg ...string) *exec.Cmd
	GOOS         string
}

// NewClient returns a Client with production defaults.
func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: constants.RequestTimeout},
		EnvFunc: func(key string) (string, bool) {
			if v := os.Getenv(key); v != "" {
				return v, true
			}

			return "", false
		},
		LookPathFunc: exec.LookPath,
		ExecCommand:  exec.CommandContext,
		GOOS:         runtime.GOOS,
	}
}

// Release holds the relevant fields from a GitHub release API response.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// doHTTPGet performs an HTTP GET request and returns the response body.
func (c *Client) doHTTPGet(ctx context.Context, url string) ([]byte, error) {
	return httpfetch.DoHTTPGet(ctx, c.HTTPClient, url)
}

// LatestReleaseURL returns the URL to check for the latest release.
func (c *Client) LatestReleaseURL() string {
	if url, ok := c.EnvFunc("KAIRO_UPDATE_URL"); ok && url != "" {
		return url
	}

	return constants.GitHubAPIReleasesLatest
}

// FetchLatestRelease fetches the latest release information from GitHub.
func (c *Client) FetchLatestRelease(ctx context.Context) (*Release, error) {
	body, err := c.doHTTPGet(ctx, c.LatestReleaseURL())
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to fetch release", err)
	}

	var r Release
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to parse response", err)
	}

	return &r, nil
}

// VersionGreaterThan reports whether latest is a higher semver than current.
func VersionGreaterThan(current, latest string) bool {
	c, err := semver.NewVersion(current)
	if err != nil {
		return false
	}
	l, err := semver.NewVersion(latest)
	if err != nil {
		return false
	}

	return c.LessThan(l)
}

// InstallScriptURL returns the download URL for the install script at the given tag.
func InstallScriptURL(goos, tag string) string {
	if goos == constants.WindowsGOOS {
		return constants.RawGitHubFileURL(tag, "scripts/install.ps1")
	}

	return constants.RawGitHubFileURL(tag, "scripts/install.sh")
}

// DownloadToTempFile downloads a URL to a temporary file and returns its path.
func (c *Client) DownloadToTempFile(ctx context.Context, url string) (string, error) {
	resp, err := httpfetch.DoHTTPRequest(ctx, c.HTTPClient, url)
	if err != nil {
		return "", errors.WrapError(errors.NetworkError,
			"failed to download", err)
	}
	defer resp.Body.Close()

	ext := installScriptExt
	if c.GOOS == constants.WindowsGOOS {
		ext = installScriptExtPS1
	}

	return httpfetch.WriteStreamToTemp(resp.Body, "kairo-install-*"+ext)
}

// RunInstallScript executes the install script at the given path.
func (c *Client) RunInstallScript(scriptPath string) error {
	if c.GOOS == constants.WindowsGOOS {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		pwshCmd := c.ExecCommand(ctx, "powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
		pwshCmd.Stdout = os.Stdout
		pwshCmd.Stderr = os.Stderr
		if err := pwshCmd.Run(); err != nil {
			return errors.WrapError(errors.RuntimeError,
				"powershell execution failed", err)
		}

		return nil
	}

	if err := os.Chmod(scriptPath, constants.FilePermExec); err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to make script executable", err)
	}

	shPath, err := c.LookPathFunc("sh")
	if err != nil {
		return errors.WrapError(errors.RuntimeError,
			"failed to find shell", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	shCmd := c.ExecCommand(ctx, shPath, scriptPath)
	shCmd.Stdout = os.Stdout
	shCmd.Stderr = os.Stderr
	if err := shCmd.Run(); err != nil {
		return errors.WrapError(errors.RuntimeError,
			"shell execution failed", err)
	}

	return nil
}

// ChecksumsURL returns the URL for the checksums file at the given tag.
func ChecksumsURL(tag string) string {
	return constants.RawGitHubFileURL(tag, "scripts/"+checksumsFilename)
}

// ParseChecksumLine extracts a SHA256 hash and filename from a checksums line.
func ParseChecksumLine(line string) (hash, filename string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", "", false
	}

	if !checksumHashPattern.MatchString(parts[0]) {
		return "", "", false
	}

	hash = strings.ToLower(parts[0])
	filename = parts[len(parts)-1]

	return hash, filename, true
}

// DownloadAndParseChecksums downloads and parses a checksums file from the given URL.
func (c *Client) DownloadAndParseChecksums(ctx context.Context, url string) (map[string]string, error) {
	body, err := c.doHTTPGet(ctx, url)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to download checksums", err)
	}

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		hash, filename, ok := ParseChecksumLine(line)
		if ok {
			checksums[filename] = hash
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to parse checksums file", err)
	}

	return checksums, nil
}

// VerifyChecksum verifies that the file at scriptPath matches the expected SHA256 hash.
func VerifyChecksum(scriptPath, expectedHash string) error {
	file, err := os.Open(scriptPath)
	if err != nil {
		return errors.FileError("failed to open file for hashing", scriptPath, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to compute file hash", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return errors.VerificationErr(
			fmt.Sprintf("script integrity check failed (expected: %.8s..., got: %.8s...)",
				expectedHash, actualHash),
			nil,
		).WithContext("expected", expectedHash).
			WithContext("actual", actualHash)
	}

	return nil
}

// ChecksumsBundleURL returns the URL for the cosign sigstore bundle of the checksums file.
func ChecksumsBundleURL(tag string) string {
	return constants.RawGitHubFileURL(tag, "scripts/checksums.txt.sigstore.json")
}

// VerifyCosignBundle downloads the sigstore bundle for the checksums file and verifies
// it using cosign. Verification is silently skipped (returns nil) when cosign is not
// found on PATH, making this a best-effort check.
func (c *Client) VerifyCosignBundle(ctx context.Context, tag string) error {
	cosignPath, err := c.LookPathFunc("cosign")
	if err != nil {
		return nil
	}

	bundleData, err := c.doHTTPGet(ctx, ChecksumsBundleURL(tag))
	if err != nil {
		return errors.WrapError(errors.NetworkError,
			"failed to download cosign bundle", err)
	}

	bundlePath, err := httpfetch.DataToTempFile(bundleData, "kairo-bundle-*.sigstore.json")
	if err != nil {
		return err
	}
	defer os.Remove(bundlePath)

	checksumsData, err := c.doHTTPGet(ctx, ChecksumsURL(tag))
	if err != nil {
		return errors.WrapError(errors.NetworkError,
			"failed to download checksums for verification", err)
	}

	checksumsPath, err := httpfetch.DataToTempFile(checksumsData, "kairo-checksums-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(checksumsPath)

	return httpfetch.CosignVerifyBlob(ctx, c.ExecCommand, cosignPath, bundlePath, checksumsPath)
}

// ScriptNameForChecksums returns the script filename used in the checksums file.
func ScriptNameForChecksums(goos string) string {
	ext := installScriptExt
	if goos == constants.WindowsGOOS {
		ext = installScriptExtPS1
	}

	return "scripts/install" + ext
}
