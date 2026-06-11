## Why

`brew install unbound-force/tap/gaze` fails on darwin_amd64, darwin_arm64, and linux_amd64 with SHA-256 checksum mismatches. The Homebrew cask at `Casks/gaze.rb` in `unbound-force/homebrew-tap` has checksums shifted by one platform position — each platform gets the SHA from a different platform's artifact.

The root cause is in the `sign-macos` job of `.github/workflows/release.yml`. An `awk` script patches darwin checksums in the GoReleaser-generated cask after code signing. The script sets a flag when it encounters `darwin_amd64` or `darwin_arm64` in a `url` line, then replaces the next `sha256` line. But GoReleaser's `homebrew_casks` template places `sha256` **before** `url` in each platform block — so the flag is set after the `sha256` it should have modified has already been printed, causing each replacement to land on the **next** platform's checksum.

A secondary issue: gaze uses `skip_upload: auto` in `.goreleaser.yaml`, meaning GoReleaser pushes the cask to the tap with unsigned checksums during the `release` job, and `sign-macos` patches it in-place afterward. This creates a race window where someone could install with wrong checksums between the initial push and the patch.

Related issues: [gaze#124](https://github.com/unbound-force/gaze/issues/124), [homebrew-tap#4](https://github.com/unbound-force/homebrew-tap/issues/4). The same class of bug exists in dewey ([dewey#67](https://github.com/unbound-force/dewey/issues/67)) and is latently present in `unbound-force/unbound-force` (which avoids the bug only because its pinned GoReleaser version happens to produce the `url`-first ordering).

## What Changes

### New Capabilities
- None

### Modified Capabilities
- `release.yml sign-macos`: Rewrite the cask checksum patching logic to be order-agnostic, handling both `sha256-before-url` and `url-before-sha256` cask layouts
- `release.yml sign-macos`: Add a verification step that confirms patched checksums match before pushing to the tap
- `release.yml release`: Upload GoReleaser-generated cask as a release asset (instead of letting GoReleaser push directly to the tap)
- `.goreleaser.yaml`: Change `skip_upload` from `auto` to `true` so the sign-macos job controls when the cask reaches the tap
- `.goreleaser.yaml`: Pin GoReleaser action version to v7.2.2 and binary version to v2.14.1 to prevent future template drift

### Removed Capabilities
- None

## Impact

- **Files changed**: `.github/workflows/release.yml`, `.goreleaser.yaml`
- **Affected systems**: macOS signing job, Homebrew tap publishing
- **No production code changes**: This is purely a CI workflow fix
- **Cross-repo note**: The same latent bug exists in `unbound-force/unbound-force` if its GoReleaser version ever changes template ordering. The dewey repo has the same active bug and a parallel fix in progress.
- **Remediation**: After the fix is merged, `Casks/gaze.rb` in `homebrew-tap` should be manually corrected for v1.5.0 using the signed checksums from `checksums.txt`

## Constitution Alignment

Assessed against the Gaze project constitution (v1.3.0).

### I. Accuracy

**Assessment**: N/A

This change modifies CI workflow scripts. It does not affect side effect detection, classification, or any analysis output.

### II. Minimal Assumptions

**Assessment**: N/A

This change does not alter how Gaze interacts with host projects or their test frameworks.

### III. Actionable Output

**Assessment**: N/A

This change does not affect report output, JSON schemas, or metric computation.

### IV. Testability

**Assessment**: N/A

This change modifies shell scripts in a GitHub Actions workflow. The added verification step serves as a built-in test gate — it confirms correctness before pushing to the tap, catching both the current bug and any future GoReleaser template changes. No Go code is changed, so no unit test changes are needed.
