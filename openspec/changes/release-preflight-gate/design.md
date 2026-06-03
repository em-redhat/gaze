## Context

Gaze's release workflow (`.github/workflows/release.yml`) triggers on
`v*` tag pushes and immediately runs GoReleaser. There is no
verification that CI checks passed on the tagged commit. A tag pushed
from a failing commit produces broken binaries distributed via GitHub
Releases and Homebrew.

The `unbound-force/unbound-force` repo uses a `workflow_dispatch`
trigger with a `preflight` job that validates tag format, uniqueness,
semver ordering, CI status, and unreleased commits before creating
the tag and building artifacts. This design replicates that pattern
for gaze.

## Goals / Non-Goals

### Goals
- Prevent release of binaries from commits that have not passed CI
- Validate tag format, uniqueness, and semver ordering before release
- Verify unreleased commits exist to prevent empty releases
- Align with the org-wide release workflow pattern from
  `unbound-force/unbound-force`
- Maintain existing macOS signing and Homebrew cask update behavior

### Non-Goals
- Adding security scan workflows to gaze (gaze has no security scan
  CI; the preflight only checks what exists)
- Changing GoReleaser configuration or build targets
- Adding SBOM generation or Cosign signing (out of scope for this
  change; can be added later independently)
- Changing the macOS signing flow or Homebrew tap structure

## Decisions

### D1: Switch from tag-push to workflow_dispatch trigger

The trigger changes from `on: push: tags: v*` to
`on: workflow_dispatch` with a `tag` string input. This is the
pattern used by `unbound-force/unbound-force` and provides two
key benefits:

1. The workflow can validate preconditions _before_ the tag exists,
   preventing the "tag pushed from broken commit" problem entirely.
2. The tag is created by the workflow itself (as an annotated tag)
   after all preflight checks pass, ensuring the tag always points
   to a validated commit.

Trade-off: Releases now require using the GitHub Actions UI
"Run workflow" button or `gh workflow run release.yml -f tag=v1.2.3`
instead of `git tag v1.2.3 && git push --tags`. This is a minor
ergonomic change that adds safety.

### D2: Query GitHub Checks API for CI verification

The preflight job queries the GitHub Checks API
(`repos/{owner}/{repo}/commits/{sha}/check-runs`) to verify that
specific CI check runs completed successfully on HEAD.

Required checks (all must pass):
- `Unit + Integration Tests (Go 1.24)` — from test.yml
- `Unit + Integration Tests (Go 1.25)` — from test.yml
- `MegaLinter` — from mega-linter.yml

Advisory checks (logged but not blocking):
- `E2E Tests (Go 1.24)` — from test.yml
- `E2E Tests (Go 1.25)` — from test.yml

Rationale: The E2E tests run `TestRunSelfCheck` which is a 15-30
minute self-analysis. While valuable, these are advisory because
they test gaze's own analysis capabilities rather than build
correctness. The unit + integration + lint checks cover compilation,
correctness, and code quality — the minimum bar for a safe release.

Check names are derived from the `name:` fields in the workflow
files, matching the values that appear in the GitHub Checks API.

### D3: Annotated tag creation by the workflow

After all preflight checks pass, the workflow creates an annotated
tag (`git tag -a`) and pushes it. This ensures:
- Tags are only created after validation passes
- Tags are annotated (not lightweight), which is the GoReleaser
  default expectation
- Re-runs after partial failure are idempotent (tag creation is
  skipped if the tag already exists on the remote)

### D4: Concurrency group with cancel-in-progress: false

A concurrency group `release-${{ github.ref }}` prevents parallel
release runs. `cancel-in-progress: false` ensures an in-progress
release is never cancelled by a new dispatch — the new run queues
instead. This prevents partial releases from being interrupted.

### D5: Tightened permissions

The workflow-level `permissions` block is set to `{}` (no
permissions). Each job declares only the permissions it needs:
- `preflight`: `contents: write` (tag creation) + `checks: read`
  (Checks API queries)
- `release`: `contents: write` (release creation)
- `sign-macos`: `contents: write` (asset replacement)

This follows the principle of least privilege.

## Risks / Trade-offs

### R1: Check name drift

If CI workflow job names change (e.g., `Unit + Integration Tests`
is renamed), the preflight step will fail to find them and block
the release. Mitigation: the error message includes the check name
and status, making it clear which check failed or was not found.
The check names are documented in the workflow file comments.

### R2: CI must have run on HEAD

The preflight verifies CI on HEAD of the branch where
`workflow_dispatch` is triggered. If a maintainer triggers the
workflow from a branch where CI has not run (or has not completed),
the preflight will fail. This is intentional — it forces CI to pass
before release.

### R3: Ergonomic change for maintainers

Maintainers must use `gh workflow run` or the GitHub UI instead of
`git push --tags`. This is a deliberate trade-off: slightly more
friction in exchange for guaranteed CI verification. The
`unbound-force/unbound-force` repo has used this pattern
successfully.
