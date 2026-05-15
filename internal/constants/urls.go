package constants

import "fmt"

const (
	GitHubRepoOwner = "dkmnx"
	GitHubRepoName  = "kairo"
	GitHubRepo      = GitHubRepoOwner + "/" + GitHubRepoName
)

const (
	RawGitHubBase = "https://raw.githubusercontent.com/" + GitHubRepo
	GitHubBase    = "https://github.com/" + GitHubRepo
	GitHubAPIBase = "https://api.github.com/repos/" + GitHubRepo
)

const (
	GitHubAPIReleasesLatest = GitHubAPIBase + "/releases/latest"
)

func RawGitHubFileURL(branchOrTag, filePath string) string {
	return fmt.Sprintf("%s/%s/%s", RawGitHubBase, branchOrTag, filePath)
}

func GitHubBlobURL(branchOrTag, filePath string) string {
	return fmt.Sprintf("%s/blob/%s/%s", GitHubBase, branchOrTag, filePath)
}
