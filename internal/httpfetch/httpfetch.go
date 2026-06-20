// Package httpfetch provides shared HTTP-fetch, temp-file, and cosign
// verify-blob primitives used by the update and provider-catalog paths.
package httpfetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/errors"
)

// MaxBodySize is the maximum number of bytes an HTTP response body will be
// read into memory. 10 MB covers release JSON, checksums, cosign bundles,
// and provider catalogs with generous headroom.
const MaxBodySize = 10 * 1024 * 1024

// DoHTTPRequest performs an HTTP GET and returns the response.
// The caller must close resp.Body.
func DoHTTPRequest(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to create request", err)
	}
	req.Header.Set("User-Agent", "kairo-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to fetch", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()

		return nil, errors.NewError(errors.NetworkError,
			fmt.Sprintf("request failed with status %d", resp.StatusCode))
	}

	return resp, nil
}

// DoHTTPGet performs an HTTP GET and returns the response body,
// enforcing MaxBodySize.
func DoHTTPGet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	resp, err := DoHTTPRequest(ctx, client, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodySize))
	if err != nil {
		return nil, errors.WrapError(errors.NetworkError,
			"failed to read response", err)
	}

	if err := ensureBodyWithinLimit(resp.Body); err != nil {
		return nil, err
	}

	return body, nil
}

// ensureBodyWithinLimit returns an error if r has data beyond MaxBodySize.
func ensureBodyWithinLimit(r io.Reader) error {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err == nil {
		return errors.NewError(errors.NetworkError,
			"response body exceeded maximum size")
	}

	return nil
}

// WriteStreamToTemp reads from r into a temp file, enforcing MaxBodySize.
// The temp file is cleaned up on any write/close error.
func WriteStreamToTemp(r io.Reader, pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", errors.WrapError(errors.FileSystemError,
			"failed to create temp file", err)
	}

	removeOnErr := true
	defer func() {
		if removeOnErr {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	if _, err := io.Copy(f, io.LimitReader(r, MaxBodySize)); err != nil {
		return "", errors.WrapError(errors.NetworkError,
			"failed to write to temp file", err)
	}

	if err := ensureBodyWithinLimit(r); err != nil {
		return "", err
	}

	if err := f.Close(); err != nil {
		return "", errors.WrapError(errors.FileSystemError,
			"failed to close temp file", err)
	}

	removeOnErr = false

	return f.Name(), nil
}

// DataToTempFile writes data to a temp file with the given pattern and
// returns its path.
func DataToTempFile(data []byte, pattern string) (string, error) {
	return WriteStreamToTemp(bytes.NewReader(data), pattern)
}

// CosignVerifyBlob runs cosign verify-blob against the given artifact file
// using the sigstore bundle at bundlePath. The certificate identity regexp
// is built from constants.GitHubRepo.
func CosignVerifyBlob(
	ctx context.Context,
	execCommand func(context.Context, string, ...string) *exec.Cmd,
	cosignPath, bundlePath, artifactPath string,
) error {
	cosignCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	certIdentityRegexp := fmt.Sprintf(
		"^https://github\\.com/%s/\\.github/workflows/release\\.yml$",
		constants.GitHubRepo,
	)

	cmd := execCommand(cosignCtx, cosignPath,
		"verify-blob",
		"--bundle="+bundlePath,
		"--certificate-identity-regexp="+certIdentityRegexp,
		"--certificate-oidc-issuer=https://token.actions.githubusercontent.com",
		artifactPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.WrapError(errors.VerificationError,
			"cosign bundle verification failed", err).
			WithContext("output", string(output))
	}

	return nil
}
