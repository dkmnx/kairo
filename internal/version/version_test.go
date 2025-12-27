package version

import (
	"regexp"
	"testing"
)

func TestVersionVariablesExist(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if Date == "" {
		t.Error("Date should not be empty")
	}
}

func TestVersionFormat(t *testing.T) {
	if Version == "dev" {
		return
	}

	semverRegex := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	if !semverRegex.MatchString(Version) {
		t.Errorf("Version %q does not match expected semver format (e.g., v1.2.3)", Version)
	}
}

func TestDateFormat(t *testing.T) {
	if Date == "unknown" {
		return
	}
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateRegex.MatchString(Date) {
		t.Errorf("Date %q does not match expected format (YYYY-MM-DD)", Date)
	}
}

func TestCommitNotNoneInProduction(t *testing.T) {
	if Version != "dev" && Commit == "none" {
		t.Error("Commit should not be 'none' in production builds")
	}
}

func TestVersionIsAccessible(t *testing.T) {
	v := Version
	if v != "dev" && v[0] != 'v' {
		t.Errorf("Version should start with 'v' in production, got %q", v)
	}
}
