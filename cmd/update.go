package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

const defaultUpdateURL = "https://api.github.com/repos/dkmnx/kairo/releases/latest"
const requestTimeout = 10 * time.Second

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

	resp, err := http.DefaultClient.Do(req)
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

// isWindows checks if the given OS is Windows
func isWindows(goos string) bool {
	return goos == "windows"
}

// getInstallScriptURL returns the appropriate install script URL based on OS
func getInstallScriptURL(goos string) string {
	if isWindows(goos) {
		return "https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1"
	}
	return "https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh"
}

// downloadToTempFile downloads a file from URL and saves to a temporary file
func downloadToTempFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.NetworkError,
			"failed to download", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", kairoerrors.NewError(kairoerrors.NetworkError,
			fmt.Sprintf("download failed with status %d", resp.StatusCode))
	}

	ext := ".sh"
	if runtime.GOOS == "windows" {
		ext = ".ps1"
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

// runInstallScript executes the downloaded install script
func runInstallScript(scriptPath string) error {
	if runtime.GOOS == "windows" {
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

	shCmd := exec.Command("/bin/sh", scriptPath)
	shCmd.Stdout = os.Stdout
	shCmd.Stderr = os.Stderr
	if err := shCmd.Run(); err != nil {
		return kairoerrors.WrapError(kairoerrors.RuntimeError,
			"shell execution failed", err)
	}

	return nil
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update kairo to the latest version",
	Long: `Check for a new release and update kairo to the latest version.

This command will:
1. Check GitHub for the latest release
2. Download and run the platform-appropriate install script`,
	Run: func(cmd *cobra.Command, args []string) {
		currentVersion := version.Version
		if currentVersion == "dev" {
			cmd.Println("Cannot update development version")
			return
		}

		latest, err := getLatestRelease()
		if err != nil {
			cmd.Printf("Error checking for updates: %v\n", err)
			return
		}

		if !versionGreaterThan(currentVersion, latest.TagName) {
			cmd.Printf("You are already on the latest version: %s\n", currentVersion)
			return
		}

		cmd.Printf("Updating to %s...\n", latest.TagName)

		installScriptURL := getInstallScriptURL(runtime.GOOS)

		confirmed, err := ui.Confirm("Do you want to proceed with installation?")
		if err != nil {
			cmd.Printf("Error reading input: %v\n", err)
			return
		}
		if !confirmed {
			cmd.Println("Installation cancelled.")
			return
		}

		cmd.Printf("\nDownloading install script from: %s\n", installScriptURL)

		tempFile, err := downloadToTempFile(installScriptURL)
		if err != nil {
			cmd.Printf("Error downloading install script: %v\n", err)
			return
		}
		defer os.Remove(tempFile)

		cmd.Printf("Running install script...\n\n")

		if err := runInstallScript(tempFile); err != nil {
			cmd.Printf("Error during installation: %v\n", err)
			return
		}

		dir := getConfigDir()
		if dir != "" {
			changes, err := config.MigrateConfigOnUpdate(dir)
			if err != nil {
				cmd.Printf("Warning: config migration failed: %v\n", err)
			} else if len(changes) > 0 {
				cmd.Printf("%s\n", config.FormatMigrationChanges(changes))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
