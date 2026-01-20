package cmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
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
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "kairo-cli")

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

// getExecutablePath returns the path to the current executable
func getExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}
	return execPath, nil
}

// getArch returns the architecture suffix for the current platform
func getArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm7"
	default:
		return goarch
	}
}

// downloadBinary downloads and extracts the kairo binary for the given version
func downloadBinary(version, repo string) (string, error) {
	arch := getArch(runtime.GOARCH)
	filename := fmt.Sprintf("kairo_windows_%s.zip", arch)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, version, filename)

	tmpDir := os.TempDir()
	archivePath := filepath.Join(tmpDir, filename)

	// Download the archive
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write archive: %w", err)
	}
	out.Close()

	// Extract the archive
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipReader.Close()

	var binaryPath string
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "kairo.exe") {
			rc, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open file in zip: %w", err)
			}
			defer rc.Close()

			binaryPath = filepath.Join(tmpDir, "kairo.exe")
			f, err := os.Create(binaryPath)
			if err != nil {
				return "", fmt.Errorf("failed to create binary: %w", err)
			}
			defer f.Close()

			if _, err := io.Copy(f, rc); err != nil {
				return "", fmt.Errorf("failed to extract binary: %w", err)
			}
			f.Close()
			break
		}
	}

	// Clean up archive
	os.Remove(archivePath)

	if binaryPath == "" {
		return "", fmt.Errorf("binary not found in archive")
	}

	return binaryPath, nil
}

// createSwapScript creates a PowerShell script that replaces the binary after the parent process exits
func createSwapScript(oldPath, newPath, version string) (string, error) {
	// Use escaped quotes for PowerShell - avoid here-string syntax which conflicts with Go raw strings
	scriptContent := fmt.Sprintf(`# Kairo Binary Swap Script
# This script waits for the parent kairo process to exit, then replaces the binary

$ErrorActionPreference = "Stop"
$OldPath = "%s"
$NewPath = "%s"
$Version = "%s"

Write-Host "[kairo] Waiting for kairo process to exit..." -ForegroundColor Green

# Get current process ID (this script's parent)
$ParentPid = $PID
$KairoPid = (Get-Process -Name "kairo" -ErrorAction SilentlyContinue | Where-Object { $_.Id -ne $ParentPid } | Select-Object -First-Object).Id

if ($KairoPid) {
    # Wait for the kairo process to exit
    $Process = Get-Process -Id $KairoPid -ErrorAction SilentlyContinue
    if ($Process) {
        $Process.WaitForExit()
        Start-Sleep -Milliseconds 500
    }
}

# Attempt to replace the binary with retry logic
$MaxAttempts = 5
$Attempt = 0

while ($Attempt -lt $MaxAttempts) {
    $Attempt++
    try {
        # Move old binary to backup
        if (Test-Path $OldPath) {
            $BackupPath = $OldPath + ".old"
            Remove-Item $BackupPath -Force -ErrorAction SilentlyContinue
            Move-Item -Path $OldPath -Destination $BackupPath -Force
        }

        # Move new binary to final location
        Move-Item -Path $NewPath -Destination $OldPath -Force

        Write-Host "[kairo] Successfully updated to $Version" -ForegroundColor Green
        Write-Host "[kairo] Please run 'kairo --version' to verify" -ForegroundColor Green
        Write-Host "[kairo] Backup saved to: $OldPath.old" -ForegroundColor Gray
        Write-Host "[kairo] You can delete the backup manually if needed." -ForegroundColor Gray

        exit 0
    }
    catch {
        Write-Host "[kairo] Attempt $Attempt/$MaxAttempts failed: $_" -ForegroundColor Yellow
        if ($Attempt -lt $MaxAttempts) {
            Start-Sleep -Seconds 2
        } else {
            Write-Host "[kairo] ERROR: Failed to replace binary after $MaxAttempts attempts" -ForegroundColor Red
            Write-Host "[kairo] New binary is at: $NewPath" -ForegroundColor Yellow
            Write-Host "[kairo] You can manually replace $OldPath with $NewPath" -ForegroundColor Yellow
            exit 1
        }
    }
}
`, oldPath, newPath, version)

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, fmt.Sprintf("kairo-swap-%d.ps1", time.Now().Unix()))

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write swap script: %w", err)
	}

	return scriptPath, nil
}

// performWindowsUpdate handles the self-update process on Windows
func performWindowsUpdate(version string, cmd *cobra.Command) error {
	// Get current executable path
	execPath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Download the new binary
	cmd.Println("Downloading new binary...")
	newBinaryPath, err := downloadBinary(version, "dkmnx/kairo")
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	// Create the swap script
	swapScriptPath, err := createSwapScript(execPath, newBinaryPath, version)
	if err != nil {
		return fmt.Errorf("failed to create swap script: %w", err)
	}

	// Spawn the swap script in a hidden window
	cmd.Println("Spawning background update process...")
	cmd.Println("This process will exit now. The update will complete in the background.")
	cmd.Println("")
	cmd.Println("Once the update is complete, you can run 'kairo --version' to verify.")

	pwshCmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", swapScriptPath)
	if err := pwshCmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn swap script: %w", err)
	}

	// Exit the current process to release the file lock
	os.Exit(0)
	return nil
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
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "kairo-install-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tempFile.Name(), nil
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
			// On Windows, use direct binary download with swap-after-exit pattern
			// This avoids file lock issues when updating a running process
			confirmed, err := ui.Confirm("Do you want to proceed with installation?")
			if err != nil {
				cmd.Printf("Error reading input: %v\n", err)
				return
			}
			if !confirmed {
				cmd.Println("Installation cancelled.")
				return
			}

			if err := performWindowsUpdate(latest.TagName, cmd); err != nil {
				cmd.Printf("Error during installation: %v\n", err)
				return
			}
		} else {
			// On Unix-like systems, download to temp file first for security
			tempFile, err := downloadToTempFile(installScriptURL)
			if err != nil {
				cmd.Printf("Error downloading install script: %v\n", err)
				return
			}
			defer os.Remove(tempFile)

			// Show the script source and ask for confirmation
			cmd.Printf("\nInstall script downloaded from: %s\n", installScriptURL)
			cmd.Printf("Script will be executed from: %s\n\n", tempFile)

			confirmed, err := ui.Confirm("Do you want to proceed with installation?")
			if err != nil {
				cmd.Printf("Error reading input: %v\n", err)
				return
			}
			if !confirmed {
				cmd.Println("Installation cancelled.")
				return
			}

			// Make script executable and execute
			if err := os.Chmod(tempFile, 0755); err != nil {
				cmd.Printf("Error making script executable: %v\n", err)
				return
			}

			shCmd := exec.Command(tempFile)
			shCmd.Stdout = os.Stdout
			shCmd.Stderr = os.Stderr

			if err := shCmd.Run(); err != nil {
				cmd.Printf("Error during installation: %v\n", err)
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
