# Update Verification

`kairo update` downloads and runs the install script for the latest release
from GitHub. The download pipeline verifies integrity in three layers:

1. **SHA256 checksum** of the install script against the `checksums.txt`
   file in the release.
2. **Cosign bundle** verification of the `checksums.txt` against the
   GitHub Actions OIDC issuer (`token.actions.githubusercontent.com`) and the
   release-workflow identity.
3. **User confirmation** prompt before invoking the install script.

## Why cosign failures are downgraded to warnings

Cosign is not installed on most users' systems, and the release pipeline
attaches a cosign bundle only for tags produced by the release workflow.
For tags without a bundle, `VerifyCosignBundle` returns nil (no error).
For tags with a bundle where verification fails, the failure is logged
to stderr as a warning and the update proceeds.

This is a deliberate UX choice: a missing cosign installation should
not block self-updates. The trade-off is that a successful cosign
verification is **best-effort**, not mandatory.

If you require strict cosign verification, set `KAIRO_REQUIRE_COSIGN=1`
in the environment. When this is set, a cosign failure (including
"cosign not installed") aborts the update.

## What is still enforced

- The SHA256 checksum is **always** required. A mismatch aborts the update
  and deletes the downloaded script.
- The install script runs with the user's normal privileges. It is not
  elevated.
- The update flow never executes the install script without a successful
  SHA256 match.
