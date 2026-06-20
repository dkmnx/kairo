// Package integrity provides sigstore-verified HTTP fetching using cosign.
// Both the update system and the provider catalog refresh depend on it.
//
// NOTE on cosign-not-found policy: unlike update.VerifyCosignBundle, which
// silently skips verification when cosign is absent (the install-script
// download is best-effort), FetchVerified hard-errors. The provider catalog
// is a security boundary — accepting an unverified catalog would allow an
// attacker to feed the CLI arbitrary provider definitions. This is by design.
package integrity

import (
	"context"
	"net/http"
	"os"
	"os/exec"

	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/httpfetch"
)

// FetchVerified downloads the artifact at artifactURL, verifies its sigstore
// bundle at bundleURL using cosign, and returns the artifact bytes. Cosign
// lookup and subprocess use the given functions. Returns the verified data
// on success.
func FetchVerified(
	ctx context.Context,
	httpClient *http.Client,
	lookPath func(string) (string, error),
	execCommand func(context.Context, string, ...string) *exec.Cmd,
	artifactURL, bundleURL string,
) ([]byte, error) {
	bundleData, err := httpfetch.DoHTTPGet(ctx, httpClient, bundleURL)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to download sigstore bundle", err)
	}

	artifactData, err := httpfetch.DoHTTPGet(ctx, httpClient, artifactURL)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to download artifact", err)
	}

	cosignPath, err := lookPath("cosign")
	if err != nil {
		return nil, errors.WrapError(errors.RuntimeError,
			"cosign not found on PATH", err)
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
