## Why

`release.yml` triggers on any `v*` tag push and runs GoReleaser
immediately with no CI preflight verification. A tag pushed from a
commit that never passed tests will produce and distribute broken
release binaries to GitHub Releases and Homebrew. There is no
automated check that build, test, or lint CI passed on the tagged
commit before artifacts are published.

The `unbound-force/unbound-force` repo already solves this with a
`preflight` job (lines 30-163 of its `release.yml`) that validates
tag format, uniqueness, semver ordering, CI check status, and
unreleased commits before creating the tag and proceeding to build.
Gaze should replicate this pattern for consistency across the org.

Closes https://github.com/unbound-force/gaze/issues/98

## What Changes

Replace the current tag-push-triggered release workflow with a
`workflow_dispatch`-triggered workflow that includes a `preflight`
job before GoReleaser runs.

The preflight job:
1. Validates tag format (strict `vMAJOR.MINOR.PATCH`)
2. Checks tag uniqueness (not already pushed)
3. Verifies semver ordering (new tag > latest existing tag)
4. Queries the GitHub Checks API to verify CI passed on HEAD
5. Verifies unreleased commits exist since the last tag
6. Creates and pushes the annotated tag
7. Checks signing secrets availability

The `release` job becomes dependent on `preflight` via `needs:`.
The `sign-macos` job remains unchanged in behavior.

## Capabilities

### New Capabilities
- `preflight-ci-verification`: Queries GitHub Checks API to verify
  that required CI checks (unit tests, integration tests, E2E tests,
  MegaLinter) passed on HEAD before building release artifacts.
- `semver-validation`: Validates tag format, uniqueness, and ordering
  before tag creation.
- `unreleased-commit-guard`: Prevents empty releases by verifying at
  least one commit exists since the last tag.
- `workflow-dispatch-trigger`: Release is initiated via manual
  `workflow_dispatch` with a tag input, replacing automatic tag-push
  trigger.

### Modified Capabilities
- `release-job`: Now depends on `preflight` job completing
  successfully. Checks out the tag ref instead of the default ref.
- `sign-macos`: Receives `has_signing_secrets` from `preflight`
  outputs (routed through `release` outputs) instead of computing
  it inline.

### Removed Capabilities
- `tag-push-trigger`: The `on: push: tags: v*` trigger is removed
  in favor of `workflow_dispatch`. Tags are created by the workflow
  itself after preflight validation passes.

## Impact

- **File changed**: `.github/workflows/release.yml` (single file)
- **No Go code changes**: This is CI-only.
- **No test changes**: No production code is modified.
- **Release process change**: Maintainers will use the GitHub Actions
  "Run workflow" button (or `gh workflow run`) instead of pushing a
  tag manually. The workflow creates the tag after validation.
- **Permissions**: Adds `checks: read` permission for the preflight
  job to query the Checks API.
- **Concurrency**: Adds a concurrency group to prevent parallel
  release runs.

## Constitution Alignment

Assessed against the Gaze project constitution (v1.3.0).

### I. Accuracy

**Assessment**: N/A

This change modifies CI workflow configuration only. It does not
affect side effect detection, assertion mapping, or any analysis
output. No accuracy impact.

### II. Minimal Assumptions

**Assessment**: PASS

The preflight job queries the GitHub Checks API for specific check
run names derived from the actual workflow files (`test.yml` and
`mega-linter.yml`). The check names are explicit and match the
`name:` fields in those workflows. No assumptions about CI state
are made beyond what the API reports.

### III. Actionable Output

**Assessment**: N/A

This change does not affect Gaze's analysis output, reports, or
metrics. CI workflow changes have no bearing on user-facing output
formats.

### IV. Testability

**Assessment**: N/A

No production code or test code is modified. The workflow itself is
validated by GitHub Actions execution. The preflight steps produce
clear pass/fail output with `::error::` annotations for each
validation failure.
