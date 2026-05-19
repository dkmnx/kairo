// Package update provides self-update logic: release checking, download,
// checksum verification, and install script execution.
package update

import (
	"bufio"
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
)

const (
	checksumsFilename   = "checksums.txt"
	installScriptExt    = ".sh"
	installScriptExtPS1 = ".ps1"
)

// Client holds injectable dependencies for update operations.
type Client struct {
	HTTPClient   *http.Client
	EnvFunc      func(key string) (string, bool)
	LookPathFunc func(string) (string, error)
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
	}
}

// Release holds the relevant fields from a GitHub release response.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// doHTTPGet performs an HTTP GET request and returns the response body.
func (c *Client) doHTTPGet(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to create request", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to fetch", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewError(errors.NetworkError,
			fmt.Sprintf("request failed with status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to read response", err)
	}

	return body, nil
}

// GetLatestReleaseURL returns the URL to check for the latest release.
func (c *Client) GetLatestReleaseURL() string {
	if url, ok := c.EnvFunc("KAIRO_UPDATE_URL"); ok && url != "" {
		return url
	}

	return constants.GitHubAPIReleasesLatest
}

// GetLatestRelease fetches the latest release information from GitHub.
func (c *Client) GetLatestRelease() (*Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
	defer cancel()

	url := c.GetLatestReleaseURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to create request", err)
	}

	req.Header.Set("User-Agent", "kairo-cli")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to fetch release", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewError(errors.NetworkError,
			fmt.Sprintf("API returned status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to read response", err)
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

// GetInstallScriptURL returns the download URL for the install script at the given tag.
func GetInstallScriptURL(goos, tag string) string {
	if goos == constants.WindowsGOOS {
		return constants.RawGitHubFileURL(tag, "scripts/install.ps1")
	}

	return constants.RawGitHubFileURL(tag, "scripts/install.sh")
}

// DownloadToTempFile downloads a URL to a temporary file and returns its path.
func (c *Client) DownloadToTempFile(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", errors.WrapError(errors.NetworkError,
			"failed to create download request", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", errors.WrapError(errors.NetworkError,
			"failed to download", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewError(errors.NetworkError,
			fmt.Sprintf("download failed with status %d", resp.StatusCode))
	}

	ext := installScriptExt
	if runtime.GOOS == constants.WindowsGOOS {
		ext = installScriptExtPS1
	}
	tempFile, err := os.CreateTemp("", "kairo-install-*"+ext)
	if err != nil {
		return "", errors.WrapError(errors.FileSystemError,
			"failed to create temp file", err)
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())

		return "", errors.WrapError(errors.NetworkError,
			"failed to write to temp file", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())

		return "", errors.WrapError(errors.FileSystemError,
			"failed to close temp file", err)
	}

	return tempFile.Name(), nil
}

// RunInstallScript executes the install script at the given path.
func RunInstallScript(scriptPath string) error {
	if runtime.GOOS == constants.WindowsGOOS {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		pwshCmd := exec.CommandContext(ctx, "powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
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

	shPath, err := exec.LookPath("sh")
	if err != nil {
		return errors.WrapError(errors.RuntimeError,
			"failed to find shell", err)
	}

	shCmd := exec.CommandContext(context.Background(), shPath, scriptPath)
	shCmd.Stdout = os.Stdout
	shCmd.Stderr = os.Stderr
	if err := shCmd.Run(); err != nil {
		return errors.WrapError(errors.RuntimeError,
			"shell execution failed", err)
	}

	return nil
}

// GetChecksumsURL returns the URL for the checksums file at the given tag.
func GetChecksumsURL(tag string) string {
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

	hashPattern := regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
	if !hashPattern.MatchString(parts[0]) {
		return "", "", false
	}

	hash = strings.ToLower(parts[0])
	filename = parts[len(parts)-1]

	return hash, filename, true
}

// DownloadAndParseChecksums downloads and parses a checksums file from the given URL.
func (c *Client) DownloadAndParseChecksums(url string) (map[string]string, error) {
	body, err := c.doHTTPGet(url)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to download checksums", err)
	}

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
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

// GetChecksumsBundleURL returns the URL for the cosign sigstore bundle of the checksums file.
func GetChecksumsBundleURL(tag string) string {
	return constants.RawGitHubFileURL(tag, "scripts/checksums.txt.sigstore.json")
}

// VerifyCosignBundle downloads the sigstore bundle for the checksums file and verifies
// it using cosign. Returns nil if cosign is not installed (best-effort verification).
func (c *Client) VerifyCosignBundle(tag string) error {
	cosignPath, err := c.LookPathFunc("cosign")
	if err != nil {
		return nil
	}

	bundleURL := GetChecksumsBundleURL(tag)
	bundleData, err := c.doHTTPGet(bundleURL)
	if err != nil {
		return errors.WrapError(errors.NetworkError,
			"failed to download cosign bundle", err)
	}

	bundleFile, err := os.CreateTemp("", "kairo-bundle-*.sigstore.json")
	if err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to create temp bundle file", err)
	}
	defer os.Remove(bundleFile.Name())

	if _, err := bundleFile.Write(bundleData); err != nil {
		bundleFile.Close()

		return errors.WrapError(errors.FileSystemError,
			"failed to write bundle file", err)
	}
	bundleFile.Close()

	checksumsURL := GetChecksumsURL(tag)
	checksumsData, err := c.doHTTPGet(checksumsURL)
	if err != nil {
		return errors.WrapError(errors.NetworkError,
			"failed to download checksums for verification", err)
	}

	checksumsFile, err := os.CreateTemp("", "kairo-checksums-*.txt")
	if err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to create temp checksums file", err)
	}
	defer os.Remove(checksumsFile.Name())

	if _, err := checksumsFile.Write(checksumsData); err != nil {
		checksumsFile.Close()

		return errors.WrapError(errors.FileSystemError,
			"failed to write checksums file", err)
	}
	checksumsFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cosignPath,
		"verify-blob",
		"--bundle="+bundleFile.Name(),
		checksumsFile.Name(),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.WrapError(errors.VerificationError,
			"cosign bundle verification failed", err).
			WithContext("output", string(output))
	}

	return nil
}

// GetScriptNameForChecksums returns the script filename used in the checksums file.
func GetScriptNameForChecksums(goos string) string {
	ext := installScriptExt
	if goos == constants.WindowsGOOS {
		ext = installScriptExtPS1
	}

	return "scripts/install" + ext
}
