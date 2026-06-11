## Context

The `sign-macos` job in `.github/workflows/release.yml` signs darwin binaries, replaces them in the GitHub Release, and then patches the GoReleaser-generated Homebrew cask with the new SHA-256 checksums before pushing to `unbound-force/homebrew-tap`.

The patching uses an `awk` script that sets a flag when it encounters `darwin_amd64` or `darwin_arm64` in a line, then substitutes the next `sha256` line. This works when `url` precedes `sha256` (as in `unbound-force/unbound-force` with GoReleaser v2.14.1) but fails when `sha256` precedes `url` (as gaze's GoReleaser output does with floating `~> v2`). The flag is set on the `url` line but the `sha256` line has already been printed — causing each platform's hash to land on the wrong line.

Additionally, gaze uses `skip_upload: auto` in `.goreleaser.yaml`, which causes GoReleaser to push the cask to the tap with unsigned checksums during the `release` job. The `sign-macos` job then clones the tap and patches the cask in-place. This creates a race window and differs from the `unbound-force/unbound-force` pattern which uses `skip_upload: true` and has the sign-macos job control the entire tap push.

## Goals / Non-Goals

### Goals
- Fix the cask checksum patching to work regardless of `sha256`/`url` ordering in GoReleaser output
- Add a verification gate that catches mismatches before pushing to the tap
- Align the release workflow with the `unbound-force/unbound-force` reference pattern: `skip_upload: true`, upload cask as release asset, download in sign-macos, patch, then push
- Pin GoReleaser version to prevent future template drift
- Remediate the current v1.5.0 cask in `homebrew-tap`

### Non-Goals
- Fixing the same bug in `unbound-force/unbound-force` or other repos (separate issues)
- Changing the GoReleaser cask template or switching from cask to formula
- Adding automated testing of the release workflow (would require a separate spec)
- Adding a preflight job to the release workflow (separate improvement)

## Decisions

### D1: Use `sed` direct-replacement instead of flag-based awk

**Decision**: Replace the `awk` script with a two-step approach: (a) extract the original unsigned darwin checksums from the GoReleaser-generated cask by parsing `on_macos > on_intel` and `on_macos > on_arm` section context, then (b) use `sed` to replace those exact hash strings with the signed values.

**Rationale**: The `awk` flag approach is inherently fragile — it depends on line ordering within sections. Since each SHA-256 hash is unique within the file, direct string replacement is unambiguous and order-agnostic. This matches the approach chosen by the dewey repo's parallel fix.

**Alternatives considered**:
- Rewrite the `awk` to buffer `sha256` lines and emit them after seeing the next line: more complex, still coupled to template structure.
- Use GoReleaser's `cask_template` to control the generated layout: works but adds a template file to maintain and doesn't address the signing flow.

### D2: Add post-patch verification step

**Decision**: After patching, verify that the cask file contains the expected signed checksums in the correct platform sections. Fail the job if they are not found.

**Rationale**: Aligns with Observable Quality — the signing pipeline should fail loudly on mismatch rather than silently publishing wrong checksums. This catches both the original bug and any future GoReleaser template changes.

### D3: Change `skip_upload` from `auto` to `true`

**Decision**: Change `.goreleaser.yaml` `homebrew_casks[].skip_upload` from `auto` to `true`. Add a step in the `release` job to upload the generated cask as a release asset. The `sign-macos` job downloads it, patches checksums, and pushes to the tap.

**Rationale**: With `skip_upload: auto`, GoReleaser pushes the cask to the tap with unsigned checksums, creating a race window where `brew install` could fetch wrong hashes. With `skip_upload: true`, the cask only reaches the tap after signing and patching are complete — matching the proven `unbound-force/unbound-force` pattern.

### D4: Pin GoReleaser to a specific version

**Decision**: Update the goreleaser-action SHA to v7.2.2 and pin the binary version to `v2.14.1` (matching `unbound-force/unbound-force`).

**Rationale**: Floating version ranges (`~> v2`) are a reliability risk for reproducible builds. The cask template ordering that caused this bug could change between minor versions. Pinning ensures consistent behavior.

## Risks / Trade-offs

- **Risk**: The `sed` approach assumes each SHA-256 hash appears exactly once in the cask file. GoReleaser-generated casks do not repeat hashes, so this is safe. The verification step (D2) catches any violation.
- **Risk**: Pinning GoReleaser version means manual updates are needed for new features. This is acceptable — reproducibility is more important than auto-updates for release tooling.
- **Risk**: Changing `skip_upload` from `auto` to `true` changes the release flow. If the `sign-macos` job is skipped (no signing secrets), the cask will not be pushed to the tap at all. This is actually better than the current behavior (pushing with wrong checksums), but the release job should handle the no-signing case by pushing the unsigned cask directly — matching the unbound-force pattern where the cask still gets published even without signing.
- **Trade-off**: The `sign-macos` job now owns the tap push instead of GoReleaser. This adds a step but provides a single point of control for the cask's final state.
