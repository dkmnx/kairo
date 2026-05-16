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
