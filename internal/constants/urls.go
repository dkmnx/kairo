package constants

import "fmt"

// GitHub repository identifiers used to construct API and web URLs.
const (
	GitHubRepoOwner = "dkmnx"
	GitHubRepoName  = "kairo"
	GitHubRepo      = GitHubRepoOwner + "/" + GitHubRepoName
)

// Base URLs for GitHub raw content, web, and API access.
const (
	RawGitHubBase = "https://raw.githubusercontent.com/" + GitHubRepo
	GitHubBase    = "https://github.com/" + GitHubRepo
	GitHubAPIBase = "https://api.github.com/repos/" + GitHubRepo
)

// GitHubAPIReleasesLatest is the API endpoint for the latest release.
const (
	GitHubAPIReleasesLatest = GitHubAPIBase + "/releases/latest"
)

// RawGitHubFileURL returns the raw content URL for a file at the given branch or tag.
func RawGitHubFileURL(branchOrTag, filePath string) string {
	return fmt.Sprintf("%s/%s/%s", RawGitHubBase, branchOrTag, filePath)
}

// GitHubBlobURL returns the web blob URL for a file at the given branch or tag.
func GitHubBlobURL(branchOrTag, filePath string) string {
	return fmt.Sprintf("%s/blob/%s/%s", GitHubBase, branchOrTag, filePath)
}

// CatalogDownloadURL returns the download URL for the catalog.json artifact at the given tag.
func CatalogDownloadURL(tag string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/catalog.json",
		GitHubRepo, tag)
}

// CatalogBundleDownloadURL returns the download URL for the catalog sigstore bundle.
func CatalogBundleDownloadURL(tag string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/catalog.json.sigstore.json",
		GitHubRepo, tag)
}

// CatalogChecksumURL returns the download URL for the catalog SHA256 checksum.
func CatalogChecksumURL(tag string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/catalog.json.sha256",
		GitHubRepo, tag)
}
