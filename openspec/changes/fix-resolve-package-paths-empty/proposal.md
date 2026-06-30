## Why

`go test ./...` fails on `main` because `TestResolvePackagePaths_EmptyPatterns` expects empty input to produce empty output, but `go/packages.Load` with no patterns defaults to loading the current directory package (`"."`). This is GitHub issue #117.

Additionally, there are two identical copies of `resolvePackagePaths` — one in `internal/aireport/runner_steps.go` and one in `internal/crap/contract.go` — with explicit "keep in sync" and "consolidation deferred" comments. Both copies share the same empty-input bug.

## What Changes

1. Add an early return guard in `resolvePackagePaths` for empty pattern slices — return `nil, nil` without calling `packages.Load`.
2. Apply the fix to both copies of the function.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `resolvePackagePaths` (internal/aireport): Returns empty result for empty patterns instead of loading the current directory package.
- `resolvePackagePaths` (internal/crap): Same fix applied to the duplicate copy.

### Removed Capabilities
- None

## Impact

- **internal/aireport/runner_steps.go**: `resolvePackagePaths` gains an early-return guard.
- **internal/crap/contract.go**: Same early-return guard applied to the duplicate function.
- **Tests**: `TestResolvePackagePaths_EmptyPatterns` in `internal/aireport/runner_steps_test.go` will pass. A matching test should be added to `internal/crap/contract_test.go` for the duplicate function.
- **Production callers**: No behavior change — `runProductionPipeline` and `BuildContractCoverageFunc` both validate patterns upstream before calling `resolvePackagePaths`, so the empty-input path is currently latent.

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: N/A

No artifact-based communication or hero interaction is affected. This is an internal bug fix.

### II. Composability First

**Assessment**: PASS

The fix is self-contained within two internal functions. No new dependencies are introduced. Each package remains independently usable.

### III. Observable Quality

**Assessment**: PASS

The fix ensures the function's contract (empty in → empty out) matches the test's expectation. Machine-parseable output and provenance are unaffected.

### IV. Testability

**Assessment**: PASS

The fix makes an existing test pass (`TestResolvePackagePaths_EmptyPatterns`). A new symmetric test is added to the `crap` package to cover the same edge case in the duplicate function. Both functions remain testable in isolation.
