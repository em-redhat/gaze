## 1. Replace trigger and add workflow structure

- [x] 1.1 Change `on:` from `push: tags: v*` to `workflow_dispatch` with required `tag` string input (description: `Release tag (e.g., v0.15.0)`)
- [x] 1.2 Set workflow-level `permissions: {}` (no default permissions)
- [x] 1.3 Add concurrency group `release-${{ github.ref }}` with `cancel-in-progress: false`
- [x] 1.4 Add `RELEASE_TAG: ${{ inputs.tag }}` env to jobs that reference the tag (replace `${GITHUB_REF_NAME}` references in `sign-macos`)

## 2. Add preflight job

- [x] 2.1 Add `preflight` job with `runs-on: ubuntu-latest`, `timeout-minutes: 10`, permissions `contents: write` + `checks: read`
- [x] 2.2 Add checkout step with `fetch-depth: 0` (needed for tag listing and commit counting)
- [x] 2.3 Add tag format validation step: regex check for `^v[0-9]+\.[0-9]+\.[0-9]+$`, emit `::error::` on mismatch
- [x] 2.4 Add tag uniqueness step: `git ls-remote --tags origin` check, emit `::error::` if tag already exists
- [x] 2.5 Add semver ordering step: compare new tag against latest `v*` tag using `sort -V`, skip if no existing tags (first release)
- [x] 2.6 Add CI verification step: query GitHub Checks API for required check names (`Unit + Integration Tests (Go 1.24)`, `Unit + Integration Tests (Go 1.25)`, `MegaLinter`), fail if any conclusion is not `success`
- [x] 2.7 Add unreleased commits step: `git rev-list --count` between latest tag and HEAD, fail if count is 0
- [x] 2.8 Add tag creation step: `git tag -a` + `git push origin`, with idempotent skip if tag already exists on remote
- [x] 2.9 Move signing secrets check from `release` job to `preflight` job, expose via `outputs.has_signing_secrets`

## 3. Update release job dependency chain

- [x] 3.1 Add `needs: preflight` to `release` job
- [x] 3.2 Add `ref: ${{ inputs.tag }}` to the checkout step in `release` job
- [x] 3.3 Update `release` job `outputs` to forward `has_signing_secrets` from `needs.preflight.outputs.has_signing_secrets`
- [x] 3.4 Set `release` job permissions to only `contents: write`

## 4. Update sign-macos job references

- [x] 4.1 Replace all `${GITHUB_REF_NAME}` references in `sign-macos` steps with `${{ inputs.tag }}` via the `RELEASE_TAG` env variable
- [x] 4.2 Verify `sign-macos` conditional reads from `needs.release.outputs.has_signing_secrets`

## 5. Validation

- [x] 5.1 Verify the complete workflow YAML is valid (no syntax errors) by reviewing the final file structure
- [x] 5.2 Verify all `${{ inputs.tag }}` / `$RELEASE_TAG` references are consistent across all three jobs
- [x] 5.3 Verify check names in the CI verification step match the `name:` fields in `test.yml` and `mega-linter.yml`
- [x] 5.4 Verify the action SHAs are pinned (not floating tags) in all steps

## 6. Documentation

- [x] 6.1 Update AGENTS.md "CI/CD" section to document the new release trigger mechanism (workflow_dispatch instead of tag push)
- [x] 6.2 Add inline comments to the workflow file documenting the preflight pattern and check name source
