# Update Verification

`kairo update` downloads and runs the install script for the latest release
from GitHub. The download pipeline verifies integrity in three layers:

1. **SHA256 checksum** of the install script against the `checksums.txt`
   file in the release.
2. **Cosign bundle** verification of the `checksums.txt` against the
   GitHub Actions OIDC issuer (`token.actions.githubusercontent.com`) and the
   release-workflow identity.
3. **User confirmation** prompt before invoking the install script.

## Why cosign verification is best-effort

Cosign is not installed on most users' systems, and the release pipeline
attaches a cosign bundle only for tags produced by the release workflow.
`VerifyCosignBundle` therefore returns nil (no error) when `cosign` is not
found on PATH, and a bundle verification failure is logged to stderr as a
warning while the update proceeds. The SHA256 checksum below remains the
hard integrity gate in all cases.

If you require strict cosign verification, set `KAIRO_REQUIRE_COSIGN=1`
in the environment. When this is set, a **bundle verification failure**
aborts the update. Note: a missing `cosign` binary is not reported as a
failure (verification is skipped), so strict mode only guards against a
present-but-failing cosign, not an absent one.

## What is still enforced

- The SHA256 checksum is **always** required. A mismatch aborts the update
  and deletes the downloaded script.
- The install script runs with the user's normal privileges. It is not
  elevated.
- The update flow never executes the install script without a successful
  SHA256 match.

## Standalone install scripts (`curl | sh` / `irm | iex`)

The install scripts used by the standalone install paths are also
checksum-gated and **fail closed**:

- `scripts/install.sh` aborts if the checksums file cannot be downloaded,
  if the archive has no checksum entry, or if the SHA256 does not match.
- `scripts/install.ps1` aborts if the checksums file cannot be downloaded
  or if the hash for the downloaded binary+arch does not match.

In both scripts the checksums are fetched from the same GitHub release as
the binary (TOFU): an attacker who can compromise the release or the TLS
channel controls both values. Cosign verification of the checksums is
attempted when `cosign` is installed and the bundle is downloadable.
