## ADDED Requirements

### Requirement: Cask checksum post-patch verification

The `sign-macos` job MUST verify that the patched cask file contains the correct signed checksums before pushing to the Homebrew tap. If verification fails, the job MUST exit with a non-zero status and log which checksums are missing.

#### Scenario: Checksums match after patching
- **GIVEN** the `sign-macos` job has signed darwin binaries and patched the cask file
- **WHEN** the verification step runs
- **THEN** the cask file MUST contain the exact `AMD64_SHA` value within the `on_intel` section and the exact `ARM64_SHA` value within the `on_arm` section, and the step MUST exit 0

#### Scenario: Checksums do not match after patching
- **GIVEN** the cask patching logic has a bug or GoReleaser changed its template
- **WHEN** the verification step runs
- **THEN** the step MUST exit non-zero with an error message identifying which platform's checksum is missing, and the job MUST NOT push to the Homebrew tap

### Requirement: Cask upload as release asset

The `release` job MUST upload the GoReleaser-generated cask file as a GitHub Release asset so that the `sign-macos` job can download, patch, and push it to the tap.

#### Scenario: Cask available for sign-macos job
- **GIVEN** GoReleaser has generated the cask to `dist/homebrew/Casks/gaze.rb` with `skip_upload: true`
- **WHEN** the `release` job completes
- **THEN** the cask file MUST be available as a release asset named `gaze.rb`, downloadable by the `sign-macos` job

#### Scenario: Cask download failure in sign-macos
- **GIVEN** the cask file was not uploaded as a release asset (e.g., the upload step failed)
- **WHEN** the `sign-macos` job attempts to download `gaze.rb`
- **THEN** the job MUST fail with a clear error message and MUST NOT push any cask to the tap

### Requirement: GoReleaser version pinning

The release workflow MUST pin the GoReleaser binary to a specific version rather than using a floating range.

#### Scenario: Reproducible cask generation
- **GIVEN** a release is triggered by a `v*` tag push
- **WHEN** GoReleaser runs
- **THEN** it MUST use the exact pinned version `v2.14.1` (not a range like `~> v2`), ensuring consistent cask template output across releases

## MODIFIED Requirements

### Requirement: Cask checksum patching logic

The `sign-macos` job MUST patch darwin SHA-256 checksums in the GoReleaser-generated cask to match the signed binary checksums. The patching logic MUST be order-agnostic — it MUST produce correct results regardless of whether `sha256` appears before or after `url` in each platform section.

Previously: The `awk` script assumed `url` (containing the platform identifier) appeared before `sha256`, setting a flag on the `url` line and replacing the next `sha256` line. This failed when GoReleaser placed `sha256` before `url`.

#### Scenario: sha256-before-url ordering (current GoReleaser output)
- **GIVEN** GoReleaser generates a cask where each platform section has `sha256` before `url`
- **WHEN** the `sign-macos` job patches darwin checksums
- **THEN** the `on_intel` section within `on_macos` MUST contain the signed `darwin_amd64` checksum, the `on_arm` section within `on_macos` MUST contain the signed `darwin_arm64` checksum, and all `on_linux` sections MUST remain unchanged

#### Scenario: url-before-sha256 ordering (alternative GoReleaser output)
- **GIVEN** GoReleaser generates a cask where each platform section has `url` before `sha256`
- **WHEN** the `sign-macos` job patches darwin checksums
- **THEN** the same correct results MUST be produced as in the sha256-before-url scenario

#### Scenario: Linux checksums are not modified
- **GIVEN** a GoReleaser-generated cask with both darwin and linux platform sections
- **WHEN** the `sign-macos` job patches darwin checksums
- **THEN** all checksum values within `on_linux` sections MUST remain identical to the GoReleaser-generated values

### Requirement: Cask skip_upload configuration

The `.goreleaser.yaml` `homebrew_casks[].skip_upload` field MUST be set to `true` so that GoReleaser generates the cask to `dist/` without pushing to the Homebrew tap. The `sign-macos` job MUST control when the cask reaches the tap.

Previously: `skip_upload` was set to `auto`, causing GoReleaser to push the cask to the tap with unsigned checksums during the `release` job.

#### Scenario: No race window for unsigned checksums
- **GIVEN** `skip_upload` is set to `true` in `.goreleaser.yaml`
- **WHEN** the `release` job completes
- **THEN** GoReleaser MUST NOT have pushed the cask to `unbound-force/homebrew-tap`; the cask MUST only reach the tap after the `sign-macos` job patches it with signed checksums

#### Scenario: Cask published without signing secrets
- **GIVEN** `skip_upload` is `true` and the `sign-macos` job is skipped (no signing secrets configured)
- **WHEN** the `release` job completes
- **THEN** the `release` job MUST push the unsigned cask directly to `unbound-force/homebrew-tap`, preserving the graceful degradation behavior established by spec 015

## REMOVED Requirements

None.
