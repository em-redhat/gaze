## ADDED Requirements

### Requirement: preflight-ci-verification

The release workflow MUST verify that required CI check runs
completed successfully on the HEAD commit before creating a tag
or building release artifacts. The workflow MUST query the GitHub
Checks API for check run conclusions.

Required checks that MUST pass:
- `Unit + Integration Tests (Go 1.24)`
- `Unit + Integration Tests (Go 1.25)`
- `MegaLinter`

If any required check has not passed (conclusion is not `success`),
the workflow MUST exit with a non-zero status and report which
check failed via a `::error::` annotation.

#### Scenario: all required CI checks passed

- **GIVEN** the HEAD commit has check runs for
  `Unit + Integration Tests (Go 1.24)`,
  `Unit + Integration Tests (Go 1.25)`, and
  `MegaLinter`, all with conclusion `success`
- **WHEN** the preflight job runs
- **THEN** the CI verification step succeeds and the
  workflow proceeds to tag creation

#### Scenario: a required CI check has not passed

- **GIVEN** the HEAD commit has `Unit + Integration Tests (Go 1.24)`
  with conclusion `failure`
- **WHEN** the preflight job runs
- **THEN** the workflow exits with a non-zero status and emits
  `::error::Required check 'Unit + Integration Tests (Go 1.24)'
  has not passed (status: failure)`

#### Scenario: a required CI check has not run

- **GIVEN** the HEAD commit has no check run for `MegaLinter`
- **WHEN** the preflight job runs
- **THEN** the workflow exits with a non-zero status and emits
  `::error::Required check 'MegaLinter' has not passed
  (status: not found)`

### Requirement: tag-format-validation

The release workflow MUST validate that the tag input matches
the strict semver format `vMAJOR.MINOR.PATCH` (e.g., `v1.2.3`).
Pre-release suffixes and build metadata MUST be rejected.

#### Scenario: valid tag format

- **GIVEN** the tag input is `v1.2.3`
- **WHEN** the preflight job runs
- **THEN** the format validation step succeeds

#### Scenario: invalid tag format

- **GIVEN** the tag input is `v1.2.3-beta.1`
- **WHEN** the preflight job runs
- **THEN** the workflow exits with a non-zero status and emits
  `::error::Invalid tag format`

### Requirement: tag-uniqueness

The release workflow MUST verify that the tag does not already
exist on the remote before creating it.

#### Scenario: tag does not exist

- **GIVEN** the tag `v1.2.3` does not exist on the remote
- **WHEN** the preflight job runs
- **THEN** the uniqueness check succeeds

#### Scenario: tag already exists

- **GIVEN** the tag `v1.2.3` already exists on the remote
- **WHEN** the preflight job runs
- **THEN** the workflow exits with a non-zero status and emits
  `::error::Tag 'v1.2.3' already exists`

### Requirement: semver-ordering

The release workflow MUST verify that the new tag is strictly
greater than the latest existing release tag, using semantic
version ordering.

#### Scenario: new tag is greater than latest

- **GIVEN** the latest existing tag is `v1.1.0` and the input
  tag is `v1.2.0`
- **WHEN** the preflight job runs
- **THEN** the ordering check succeeds

#### Scenario: new tag is not greater than latest

- **GIVEN** the latest existing tag is `v1.2.0` and the input
  tag is `v1.1.5`
- **WHEN** the preflight job runs
- **THEN** the workflow exits with a non-zero status and emits
  `::error::Tag 'v1.1.5' is not greater than latest release
  'v1.2.0'`

#### Scenario: first release (no existing tags)

- **GIVEN** no `v*` tags exist in the repository
- **WHEN** the preflight job runs
- **THEN** the ordering check succeeds (first release)

### Requirement: unreleased-commits

The release workflow MUST verify that at least one commit exists
since the last release tag. Empty releases (no new commits) MUST
be rejected.

#### Scenario: commits exist since last tag

- **GIVEN** 5 commits exist since tag `v1.1.0`
- **WHEN** the preflight job runs
- **THEN** the unreleased commits check succeeds

#### Scenario: no commits since last tag

- **GIVEN** HEAD is the same commit as the latest tag `v1.1.0`
- **WHEN** the preflight job runs
- **THEN** the workflow exits with a non-zero status and emits
  `::error::No unreleased commits since v1.1.0`

### Requirement: workflow-dispatch-trigger

The release workflow MUST be triggered via `workflow_dispatch`
with a required `tag` string input. The `on: push: tags` trigger
MUST be removed.

#### Scenario: release triggered via workflow dispatch

- **GIVEN** a maintainer runs the workflow with input `tag: v1.2.3`
- **WHEN** the workflow starts
- **THEN** the `RELEASE_TAG` environment variable is set to `v1.2.3`
  and the preflight job begins

### Requirement: tag-creation-by-workflow

After all preflight checks pass, the workflow MUST create an
annotated tag and push it to the remote. If the tag already exists
(e.g., workflow re-run after partial failure), the creation step
MUST be skipped without error.

#### Scenario: tag created after preflight passes

- **GIVEN** all preflight checks have passed and tag `v1.2.3` does
  not exist on the remote
- **WHEN** the tag creation step runs
- **THEN** an annotated tag `v1.2.3` is created and pushed to origin

#### Scenario: tag already exists on re-run

- **GIVEN** all preflight checks have passed but tag `v1.2.3`
  already exists on the remote (from a prior partial run)
- **WHEN** the tag creation step runs
- **THEN** the step succeeds with a skip message and no error

## MODIFIED Requirements

### Requirement: release-job-dependency

The `release` job MUST depend on the `preflight` job via
`needs: preflight`. The `release` job MUST check out the tag ref
(`ref: ${{ inputs.tag }}`).

Previously: The `release` job had no dependencies and checked
out the default ref.

### Requirement: signing-secrets-detection

The signing secrets check MUST be performed in the `preflight`
job and its output forwarded to the `sign-macos` job via the
`release` job's outputs.

Previously: The signing secrets check was performed inline in
the `release` job.

## REMOVED Requirements

### Requirement: tag-push-trigger

The `on: push: tags: v*` trigger is removed. Releases are now
initiated exclusively via `workflow_dispatch`. This prevents
release artifacts from being built for tags pushed from commits
that have not passed CI.
