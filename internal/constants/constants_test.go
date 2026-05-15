package constants

import "testing"

func TestURLConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GitHubRepoOwner", GitHubRepoOwner, "dkmnx"},
		{"GitHubRepoName", GitHubRepoName, "kairo"},
		{"GitHubRepo", GitHubRepo, "dkmnx/kairo"},
		{"RawGitHubBase", RawGitHubBase, "https://raw.githubusercontent.com/dkmnx/kairo"},
		{"GitHubBase", GitHubBase, "https://github.com/dkmnx/kairo"},
		{"GitHubAPIBase", GitHubAPIBase, "https://api.github.com/repos/dkmnx/kairo"},
		{"GitHubAPIReleasesLatest", GitHubAPIReleasesLatest, "https://api.github.com/repos/dkmnx/kairo/releases/latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestPathConstants(t *testing.T) {
	if KeyFileName != "age.key" {
		t.Errorf("KeyFileName = %q, want %q", KeyFileName, "age.key")
	}
	if SecretsFileName != "secrets.age" {
		t.Errorf("SecretsFileName = %q, want %q", SecretsFileName, "secrets.age")
	}
}

func TestPlatformConstants(t *testing.T) {
	if WindowsGOOS != "windows" {
		t.Errorf("WindowsGOOS = %q, want %q", WindowsGOOS, "windows")
	}
}

func TestRawGitHubFileURL(t *testing.T) {
	tests := []struct {
		name        string
		branchOrTag string
		filePath    string
		expected    string
	}{
		{
			name:        "install script for tag",
			branchOrTag: "v1.0.0",
			filePath:    "scripts/install.sh",
			expected:    "https://raw.githubusercontent.com/dkmnx/kairo/v1.0.0/scripts/install.sh",
		},
		{
			name:        "install script for main branch",
			branchOrTag: "main",
			filePath:    "scripts/install.sh",
			expected:    "https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh",
		},
		{
			name:        "ps1 script",
			branchOrTag: "v2.0.0",
			filePath:    "scripts/install.ps1",
			expected:    "https://raw.githubusercontent.com/dkmnx/kairo/v2.0.0/scripts/install.ps1",
		},
		{
			name:        "checksums file",
			branchOrTag: "v1.5.0",
			filePath:    "scripts/checksums.txt",
			expected:    "https://raw.githubusercontent.com/dkmnx/kairo/v1.5.0/scripts/checksums.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RawGitHubFileURL(tt.branchOrTag, tt.filePath)
			if got != tt.expected {
				t.Errorf("RawGitHubFileURL(%q, %q) = %q, want %q", tt.branchOrTag, tt.filePath, got, tt.expected)
			}
		})
	}
}

func TestGitHubBlobURL(t *testing.T) {
	tests := []struct {
		name        string
		branchOrTag string
		filePath    string
		expected    string
	}{
		{
			name:        "user guide for main",
			branchOrTag: "main",
			filePath:    "docs/guides/user-guide.md",
			expected:    "https://github.com/dkmnx/kairo/blob/main/docs/guides/user-guide.md",
		},
		{
			name:        "install script for tag",
			branchOrTag: "v1.0.0",
			filePath:    "scripts/install.sh",
			expected:    "https://github.com/dkmnx/kairo/blob/v1.0.0/scripts/install.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GitHubBlobURL(tt.branchOrTag, tt.filePath)
			if got != tt.expected {
				t.Errorf("GitHubBlobURL(%q, %q) = %q, want %q", tt.branchOrTag, tt.filePath, got, tt.expected)
			}
		})
	}
}
