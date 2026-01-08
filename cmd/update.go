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
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var r release
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
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

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update kairo to the latest version",
	Long: `Check for a new release and update kairo to the latest version.

This command will:
1. Check GitHub for the latest release
2. Download and install the new version`,
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

		if isWindows(runtime.GOOS) {
			// On Windows, use PowerShell to download and execute the install script
			pwshCmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", "irm "+installScriptURL+" | iex")
			pwshCmd.Stdout = os.Stdout
			pwshCmd.Stderr = os.Stderr

			if err := pwshCmd.Run(); err != nil {
				cmd.Printf("Error during installation: %v\n", err)
				return
			}
		} else {
			// On Unix-like systems, use curl | sh
			curlCmd := exec.Command("curl", "-fsSL", installScriptURL)
			curlCmd.Stderr = os.Stderr

			shCmd := exec.Command("sh")
			shCmd.Stdin, _ = curlCmd.StdoutPipe()
			shCmd.Stdout = os.Stdout
			shCmd.Stderr = os.Stderr

			if err := shCmd.Start(); err != nil {
				cmd.Printf("Error starting update: %v\n", err)
				return
			}
			if err := curlCmd.Run(); err != nil {
				cmd.Printf("Error downloading update: %v\n", err)
				return
			}
			if err := shCmd.Wait(); err != nil {
				cmd.Printf("Error during installation: %v\n", err)
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
