// Package integrity provides verified HTTP fetching for the provider catalog.
// FetchVerified uses a two-tier verification strategy: cosign sigstore bundle
// verification (preferred) with SHA256 checksum fallback when cosign is absent.
// The provider catalog is a security boundary — accepting an unverified catalog
// would allow an attacker to feed the CLI arbitrary provider definitions.
package integrity

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/httpfetch"
)

// hexHashPattern matches exactly 64 hexadecimal characters (SHA256 digest).
var hexHashPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// FetchVerified downloads the artifact at artifactURL, verifies its integrity,
// and returns the artifact bytes. Verification uses cosign sigstore bundle
// verification when cosign is available, falling back to SHA256 checksum
// verification against checksumURL when cosign is absent. Both paths hard-error
// on verification failure. Returns the verified data on success.
func FetchVerified(
	ctx context.Context,
	httpClient *http.Client,
	lookPath func(string) (string, error),
	execCommand func(context.Context, string, ...string) *exec.Cmd,
	artifactURL, bundleURL, checksumURL string,
) ([]byte, error) {
	artifactData, err := httpfetch.DoHTTPGet(ctx, httpClient, artifactURL)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to download artifact", err)
	}

	cosignPath, err := lookPath("cosign")
	if err == nil {
		return verifyCosign(ctx, httpClient, execCommand, cosignPath, bundleURL, artifactData)
	}

	return verifyChecksum(ctx, httpClient, checksumURL, artifactData)
}

// verifyCosign downloads the sigstore bundle and verifies the artifact using cosign.
func verifyCosign(
	ctx context.Context,
	httpClient *http.Client,
	execCommand func(context.Context, string, ...string) *exec.Cmd,
	cosignPath, bundleURL string,
	artifactData []byte,
) ([]byte, error) {
	bundleData, err := httpfetch.DoHTTPGet(ctx, httpClient, bundleURL)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to download sigstore bundle", err)
	}

	bundlePath, err := httpfetch.DataToTempFile(bundleData, "kairo-bundle-*.sigstore.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(bundlePath)

	artifactPath, err := httpfetch.DataToTempFile(artifactData, "kairo-artifact-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(artifactPath)

	if err := httpfetch.CosignVerifyBlob(ctx, execCommand, cosignPath, bundlePath, artifactPath); err != nil {
		return nil, err
	}

	return artifactData, nil
}

// verifyChecksum downloads the checksum file and verifies the artifact using SHA256.
func verifyChecksum(
	ctx context.Context,
	httpClient *http.Client,
	checksumURL string,
	artifactData []byte,
) ([]byte, error) {
	checksumData, err := httpfetch.DoHTTPGet(ctx, httpClient, checksumURL)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to download checksum", err)
	}

	expectedHash := string(checksumData)
	if !hexHashPattern.MatchString(expectedHash) {
		return nil, errors.NewError(errors.VerificationError,
			"checksum file does not contain a valid 64-character hex hash")
	}

	if err := httpfetch.VerifySHA256(artifactData, expectedHash); err != nil {
		return nil, err
	}

	return artifactData, nil
}
