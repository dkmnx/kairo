package cmd

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
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

const defaultUpdateURL = "https://api.github.com/repos/dkmnx/kairo/releases/latest"
const requestTimeout = 10 * time.Second
const checksumsFilename = "checksums.txt"
const installScriptExt = ".sh"
const installScriptExtPS1 = ".ps1"

var httpClient = &http.Client{
	Timeout: requestTimeout,
	Transport: &http.Transport{
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	},
}

var (
	getLatestReleaseFn          = getLatestRelease
	confirmUpdateFn             = ui.Confirm
	downloadToTempFileFn        = downloadToTempFile
	downloadAndParseChecksumsFn = downloadAndParseChecksums
	verifyChecksumFn            = verifyChecksum
	runInstallScriptFn          = runInstallScript
)

type release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func getEnvFunc(key string) (string, bool) {
	value := getEnvValue(key)
	if value != "" {
		return value, true
	}

	return "", false
}

func getEnvValue(key string) string {
	return os.Getenv(key)
}

var envGetter func(string) (string, bool) = getEnvFunc

func getLatestReleaseURL() string {
	if url, ok := envGetter("KAIRO_UPDATE_URL"); ok && url != "" {
		return url
	}

	return defaultUpdateURL
}

func getLatestRelease() (*release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	url := getLatestReleaseURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to create request", err)
	}

	req.Header.Set("User-Agent", "kairo-cli")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to fetch release", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, kairoerrors.NewError(kairoerrors.NetworkError,
			fmt.Sprintf("API returned status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to read response", err)
	}

	var r release
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to parse response", err)
	}

	return &r, nil
}

func versionGreaterThan(current, latest string) bool {
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

func isWindows(goos string) bool {
	return goos == constants.WindowsGOOS
}

func getInstallScriptURL(goos, tag string) string {
	if isWindows(goos) {
		return fmt.Sprintf("https://raw.githubusercontent.com/dkmnx/kairo/%s/scripts/install.ps1", tag)
	}

	return fmt.Sprintf("https://raw.githubusercontent.com/dkmnx/kairo/%s/scripts/install.sh", tag)
}

func downloadToTempFile(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to create download request", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to download", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", kairoerrors.NewError(kairoerrors.NetworkError,
			fmt.Sprintf("download failed with status %d", resp.StatusCode))
	}

	ext := installScriptExt
	if runtime.GOOS == constants.WindowsGOOS {
		ext = installScriptExtPS1
	}
	tempFile, err := os.CreateTemp("", "kairo-install-*"+ext)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create temp file", err)
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())

		return "", kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to write to temp file", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())

		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close temp file", err)
	}

	return tempFile.Name(), nil
}

func runInstallScript(scriptPath string) error {
	if runtime.GOOS == constants.WindowsGOOS {
		pwshCmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
		pwshCmd.Stdout = os.Stdout
		pwshCmd.Stderr = os.Stderr

		if err := pwshCmd.Run(); err != nil {
			return kairoerrors.WrapError(kairoerrors.RuntimeError,
				"powershell execution failed", err)
		}

		return nil
	}

	if err := os.Chmod(scriptPath, 0755); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to make script executable", err)
	}

	shPath, err := exec.LookPath("sh")
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.RuntimeError,
			"failed to find shell", err)
	}

	shCmd := exec.Command(shPath, scriptPath)
	shCmd.Stdout = os.Stdout
	shCmd.Stderr = os.Stderr
	if err := shCmd.Run(); err != nil {
		return kairoerrors.WrapError(kairoerrors.RuntimeError,
			"shell execution failed", err)
	}

	return nil
}

func getChecksumsURL(tag string) string {
	baseURL := fmt.Sprintf("https://raw.githubusercontent.com/dkmnx/kairo/%s/scripts", tag)

	return fmt.Sprintf("%s/%s", baseURL, checksumsFilename)
}

func parseChecksumLine(line string) (hash, filename string, ok bool) {
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

func downloadAndParseChecksums(url string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to create checksums request", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to download checksums", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, kairoerrors.NewError(kairoerrors.NetworkError,
			fmt.Sprintf("checksums download failed with status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to read checksums response", err)
	}

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		hash, filename, ok := parseChecksumLine(line)
		if ok {
			checksums[filename] = hash
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to parse checksums file", err)
	}

	return checksums, nil
}

func computeSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", kairoerrors.FileError("failed to open file for hashing", filePath, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to compute file hash", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func verifyChecksum(scriptPath, expectedHash string) error {
	actualHash, err := computeSHA256(scriptPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.VerificationError,
			"failed to verify script integrity", err)
	}

	if !strings.EqualFold(actualHash, expectedHash) {
		return kairoerrors.VerificationErr(
			fmt.Sprintf("script integrity check failed (expected: %.8s..., got: %.8s...)",
				expectedHash, actualHash),
			nil,
		).WithContext("expected", expectedHash).
			WithContext("actual", actualHash)
	}

	return nil
}

func getScriptNameForChecksums(goos string) string {
	ext := installScriptExt
	if goos == constants.WindowsGOOS {
		ext = installScriptExtPS1
	}

	return "scripts/install" + ext
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update kairo to the latest version",
	Long: `Check for a new release and update kairo to the latest version.

This command will:
1. Check GitHub for the latest release
2. Download the install script and its SHA256 checksum
3. Verify script integrity before execution
4. Run the verified install script

Security considerations:
- The install script is downloaded from the specific release tag to ensure
  the script matches the version being installed.
- SHA256 checksums are verified before execution to ensure script integrity.
- You will be prompted for confirmation before installation.
- The script is executed with your current user permissions.

For manual verification, you can download and inspect the install script and checksums from:
https://github.com/dkmnx/kairo/blob/<tag>/scripts/install.sh (Unix)
https://github.com/dkmnx/kairo/blob/<tag>/scripts/install.ps1 (Windows)
https://github.com/dkmnx/kairo/blob/<tag>/scripts/checksums.txt`,
	Run: func(cmd *cobra.Command, args []string) {
		currentVersion := version.Version
		if currentVersion == "dev" {
			cmd.Println("Cannot update development version")

			return
		}

		latest, err := getLatestReleaseFn()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error checking for updates: %v", err))

			return
		}

		if !versionGreaterThan(currentVersion, latest.TagName) {
			cmd.Printf("You are already on the latest version: %s\n", currentVersion)

			return
		}

		cmd.Printf("Updating to %s...\n", latest.TagName)

		installScriptURL := getInstallScriptURL(runtime.GOOS, latest.TagName)

		confirmed, err := confirmUpdateFn("Do you want to proceed with installation?")
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error reading input: %v", err))

			return
		}
		if !confirmed {
			cmd.Println("Installation cancelled.")

			return
		}

		cmd.Printf("\nDownloading install script from: %s\n", installScriptURL)

		tempFile, err := downloadToTempFileFn(installScriptURL)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error downloading install script: %v", err))

			return
		}
		defer os.Remove(tempFile)

		scriptName := getScriptNameForChecksums(runtime.GOOS)
		checksumsURL := getChecksumsURL(latest.TagName)

		cmd.Printf("Downloading checksums from: %s\n", checksumsURL)

		checksums, err := downloadAndParseChecksumsFn(checksumsURL)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error downloading checksums: %v", err))

			return
		}

		expectedHash, ok := checksums[scriptName]
		if !ok {
			ui.PrintError(fmt.Sprintf("Checksum for %s not found in checksums file", scriptName))

			return
		}

		cmd.Printf("Verifying script integrity...\n")

		if err := verifyChecksumFn(tempFile, expectedHash); err != nil {
			ui.PrintError(fmt.Sprintf("Security verification failed: %v", err))
			cmd.Println("Downloaded script has been removed. Please try again later or report this issue.")

			return
		}

		cmd.Printf("Running install script...\n\n")

		if err := runInstallScriptFn(tempFile); err != nil {
			ui.PrintError(fmt.Sprintf("Error during installation: %v", err))

			return
		}

	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
