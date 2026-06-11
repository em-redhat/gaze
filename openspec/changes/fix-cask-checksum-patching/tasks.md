<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file —
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Update GoReleaser Configuration

- [x] 1.1 [P] In `.goreleaser.yaml`, change `skip_upload` from `auto` to `true` under `homebrew_casks`. Remove `commit_msg_template` (no longer used when GoReleaser doesn't push). Confirm the `version: 2` field is present and correct.
- [x] 1.2 [P] In `.github/workflows/release.yml`, update the `goreleaser/goreleaser-action` step: change the pinned SHA from `ec59f474b9834571250b370d4735c50f8e2d1e29` (v7.0.0) to the v7.2.2 SHA `5daf1e915a5f0af01ddbcd89a43b8061ff4f1a89`, and change `version: '~> v2'` to `version: 'v2.14.1'`. Verify the SHA against `https://github.com/goreleaser/goreleaser-action/releases/tag/v7.2.2` before applying. Remove `HOMEBREW_TAP_GITHUB_TOKEN` from the GoReleaser step's `env` block (GoReleaser no longer pushes to the tap with `skip_upload: true`).

## 2. Upload Cask and Handle No-Signing Fallback

- [x] 2.1 In `.github/workflows/release.yml`, add a step after the GoReleaser step in the `release` job to upload the generated cask file to the GitHub Release: `gh release upload "${GITHUB_REF_NAME}" dist/homebrew/Casks/gaze.rb --clobber`. Add `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` env var.
- [x] 2.2 In `.github/workflows/release.yml`, add a conditional step in the `release` job that runs when `has_signing_secrets == 'false'` to push the unsigned cask directly to `unbound-force/homebrew-tap`. This preserves the graceful degradation behavior from spec 015 — releases without signing secrets still publish a Homebrew cask with correct (unsigned) checksums.

## 3. Rewrite Cask Checksum Patching

- [x] 3.1 In `.github/workflows/release.yml`, replace the "Update Homebrew cask checksums" step in the `sign-macos` job with order-agnostic logic: (a) Download the GoReleaser-generated cask from the release assets (instead of cloning the tap). (b) Extract the original unsigned darwin checksums from the cask by parsing `on_macos > on_intel` and `on_macos > on_arm` section context using awk — if extraction fails to find exactly two darwin checksums (one for intel, one for arm), exit non-zero with a descriptive error before attempting any patching. (c) Use `sed` to replace each original hash with the corresponding signed hash (since SHA-256 hashes are unique within the file, direct string replacement is unambiguous). (d) Ensure linux sections are untouched. (e) Clone the tap, copy the patched cask, commit, and push.

## 4. Add Post-Patch Verification

- [x] 4.1 In `.github/workflows/release.yml`, immediately after the patching logic (task 3.1), add a verification block that: (a) Greps the patched cask for `$AMD64_SHA` and confirms it appears exactly once. (b) Greps the patched cask for `$ARM64_SHA` and confirms it appears exactly once. (c) Exits non-zero with a descriptive error if either check fails, preventing the push to `homebrew-tap`. Note: since the `sed` approach replaces specific old hashes with specific new hashes, cross-contamination (swapped checksums) is structurally impossible — flat grep is sufficient for verification.

## 5. Validate Release Workflow

- [x] 5.1 Run `goreleaser check` to validate the updated `.goreleaser.yaml` configuration.
- [x] 5.2 Run `goreleaser release --snapshot --clean` to verify the cask is generated to `dist/homebrew/Casks/gaze.rb` without being pushed to the tap.

## 6. Remediate Existing Release

- [ ] 6.1 [MANUAL] Push a corrected `Casks/gaze.rb` to `unbound-force/homebrew-tap` for v1.5.0 using the signed checksums from the v1.5.0 `checksums.txt` release artifact. After pushing, verify by running `brew audit --cask unbound-force/tap/gaze` to confirm the checksums match.

## 7. Cross-Repo Follow-Up

- [ ] 7.1 [P] File a GitHub issue on `unbound-force/unbound-force` noting that the same awk patching logic is latently vulnerable to GoReleaser template ordering changes, and recommending the same order-agnostic fix.
- [ ] 7.2 [P] Close gaze#124 linking to this fix.

## 8. Verify Constitution Alignment

- [ ] 8.1 Confirm Testability (Constitution IV): the verification step (task 4.1) serves as a built-in test gate that catches checksum mismatches before pushing to the tap.
- [ ] 8.2 Confirm Minimal Assumptions (Constitution II): `brew install unbound-force/tap/gaze` succeeds without requiring users to know about or work around the signing pipeline, after remediation (task 6.1).
<!-- spec-review: passed -->
<!-- code-review: passed -->
