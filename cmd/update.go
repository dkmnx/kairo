package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

const defaultUpdateURL = "https://api.github.com/repos/dkmnx/kairo/releases/latest"

type release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func getenv(key string) string {
	if value, ok := getEnvFunc(key); ok {
		return value
	}
	return ""
}

func getEnvFunc(key string) (string, bool) {
	value := getEnvValue(key)
	if value != "" {
		return value, true
	}
	return "", false
}

func getEnvValue(key string) string {
	return ""
}

var envGetter func(string) (string, bool) = getEnvFunc

func getLatestReleaseURL() string {
	if url, ok := envGetter("KAIRO_UPDATE_URL"); ok && url != "" {
		return url
	}
	return defaultUpdateURL
}

func getLatestRelease() (*release, error) {
	url := getLatestReleaseURL()
	resp, err := http.Get(url)
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
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	return current < latest
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update kairo to the latest version",
	Long: `Check for a new release and update kairo to the latest version.

This command will:
1. Check GitHub for the latest release
2. Download and install the new version
3. Backup the current binary (optional)`,
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

		cmd.Printf("New version available: %s\n", latest.TagName)
		cmd.Printf("Current version: %s\n", currentVersion)
		cmd.Println()
		cmd.Printf("Release notes: %s\n", latest.HTMLURL)
		cmd.Println()
		cmd.Println("To update, run:")
		cmd.Println()
		cmd.Printf("  curl -sSL %s | sh\n", "https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func createUpdateCommand() *cobra.Command {
	return updateCmd
}
