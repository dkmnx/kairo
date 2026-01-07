# Kairo PowerShell Installer for Windows
# Usage: irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex

param(
    [string]$BinDir,
    [string]$Version,
    [string]$Repo = "dkmnx/kairo"
)

$ErrorActionPreference = "Stop"

$BINARY_NAME = "kairo"
$DEFAULT_INSTALL_DIR = "$env:LOCALAPPDATA\Programs\kairo"

function Write-Log {
    param([string]$Message)
    Write-Host "[kairo] $Message" -ForegroundColor Green
}

function Write-Error-Log {
    param([string]$Message)
    Write-Host "[kairo] ERROR: $Message" -ForegroundColor Red
}

function Show-Usage {
    @"
Install kairo CLI

Usage: $MyInvocation.MyCommand.Name [OPTIONS]

OPTIONS
    -BinDir DIRECTORY    Install binary to DIRECTORY (default: $DEFAULT_INSTALL_DIR)
    -Version VERSION     Install specific version (default: latest)
    -Repo REPO          Repository in format owner/repo (default: $Repo)

EXAMPLES
    & $MyInvocation.MyCommand.Name                        # Install latest version
    & $MyInvocation.MyCommand.Name -BinDir "C:\Tools"    # Install to custom directory
    & $MyInvocation.MyCommand.Name -Version "v1.2.3"    # Install specific version

SHORTENED (One-liner):
    irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex
"@
    exit 0
}

function Detect-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($arch -eq "AMD64") {
        return "amd64"
    } elseif ($arch -eq "ARM64") {
        return "arm64"
    } elseif ($arch -match "^ARM") {
        return "arm7"
    } else {
        Write-Error-Log "Unsupported architecture: $arch"
        exit 1
    }
}

function Get-LatestVersion {
    param([string]$Repo)

    Write-Log "Fetching latest version..."
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $response.tag_name
    } catch {
        Write-Error-Log "Failed to fetch latest version: $_"
        exit 1
    }
}

function Get-Checksum {
    param([string]$Repo, [string]$Version, [string]$BinaryName)

    $versionNoPrefix = $Version -replace '^v', ''
    $checksumUrl = "https://github.com/$Repo/releases/download/$Version/${BinaryName}_${versionNoPrefix}_checksums.txt"

    Write-Log "Downloading checksums..."
    try {
        $response = Invoke-RestMethod -Uri $checksumUrl -UseBasicParsing
        return $response
    } catch {
        Write-Log "Warning: Checksum file not found, skipping verification"
        return $null
    }
}

function Verify-Checksum {
    param([string]$FilePath, [string]$ChecksumData, [string]$BinaryName)

    if (-not $ChecksumData) {
        return $true
    }

    Write-Log "Verifying checksum..."
    $hash = Get-FileHash -Path $FilePath -Algorithm SHA256

    # Parse checksums.txt to find the matching hash
    $lines = $ChecksumData -split "`n"
    foreach ($line in $lines) {
        if ($line -match "^([a-f0-9]+)\s+\*$($BinaryName)_windows_") {
            $expectedHash = $matches[1].ToLower()
            $actualHash = $hash.Hash.ToLower()

            if ($expectedHash -eq $actualHash) {
                Write-Log "Checksum verified successfully"
                return $true
            } else {
                Write-Error-Log "Checksum verification failed"
                Write-Error-Log "Expected: $expectedHash"
                Write-Error-Log "Got:      $actualHash"
                return $false
            }
        }
    }

    Write-Log "Warning: Could not find checksum for this binary"
    return $true # Continue anyway
}

function Download-And-Install {
    param(
        [string]$Version,
        [string]$Arch,
        [string]$InstallDir,
        [string]$Repo,
        [string]$BinaryName
    )

    $os = "windows"
    $filename = "${BinaryName}_${os}_${Arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$Version/$filename"

    Write-Log "Downloading $url..."

    $tmpDir = [System.IO.Path]::GetTempPath()
    $archivePath = Join-Path $tmpDir $filename

    try {
        Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing
    } catch {
        Write-Error-Log "Failed to download $url : $_"
        exit 1
    }

    # Get checksums
    $checksumData = Get-Checksum -Repo $Repo -Version $Version -BinaryName $BinaryName

    # Verify checksum
    if (-not (Verify-Checksum -FilePath $archivePath -ChecksumData $checksumData -BinaryName $BinaryName)) {
        Remove-Item -Path $archivePath -Force
        exit 1
    }

    Write-Log "Extracting archive..."
    try {
        Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force
    } catch {
        Write-Error-Log "Failed to extract archive: $_"
        Remove-Item -Path $archivePath -Force
        exit 1
    }

    # Clean up archive
    Remove-Item -Path $archivePath -Force

    Write-Log "Installing to $InstallDir..."

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $binaryPath = Join-Path $tmpDir "$BinaryName.exe"
    $destBinaryPath = Join-Path $InstallDir "$BinaryName.exe"

    if (-not (Test-Path $binaryPath)) {
        Write-Error-Log "Binary not found at $binaryPath"
        exit 1
    }

    # Move binary (remove existing first to avoid "file already exists" error)
    try {
        if (Test-Path $destBinaryPath) {
            try {
                Remove-Item -Path $destBinaryPath -Force
            } catch {
                Write-Error-Log "Failed to remove existing binary: $_"
                Write-Error-Log "Please close any running kairo processes and try again."
                Write-Error-Log "You can also run this as Administrator for write access."
                exit 1
            }
        }
        Move-Item -Path $binaryPath -Destination $destBinaryPath
    } catch {
        Write-Error-Log "Failed to move binary: $_"
        exit 1
    }

    # Remove any extracted LICENSE or README
    $licensePath = Join-Path $tmpDir "LICENSE"
    $readmePath = Join-Path $tmpDir "README.md"
    if (Test-Path $licensePath) { Remove-Item $licensePath -Force }
    if (Test-Path $readmePath) { Remove-Item $readmePath -Force }

    Write-Log "Installed $BinaryName $Version to $InstallDir\$BinaryName.exe"
    Write-Host ""
    Write-Log "Add to PATH by running:"
    Write-Host "  `$env:PATH += `";$InstallDir`""
    Write-Host ""
    Write-Log "Or add permanently:"
    Write-Host "  [Environment]::SetEnvironmentVariable(`"PATH`", `$env:PATH + `";$InstallDir`", [EnvironmentVariableTarget]::User)"
}

# Main execution
if ($Help) {
    Show-Usage
}

$arch = Detect-Architecture
$installDir = if ($BinDir) { $BinDir } else { $DEFAULT_INSTALL_DIR }
$version = if ($Version) { $Version } else { (Get-LatestVersion -Repo $Repo) }

Write-Log "Installing $BINARY_NAME $version for windows/$arch..."

Download-And-Install -Version $version -Arch $arch -InstallDir $installDir -Repo $Repo -BinaryName $BINARY_NAME
