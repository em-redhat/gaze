## Why

Gaze's CI gate commands (`gaze crap`, `gaze report`) silently report success
in three distinct failure scenarios, undermining the reliability of any CI
pipeline that depends on them:

1. **#101 — Coverage profile discarded on test failure**: When `go test` exits
   non-zero (one failing test, one non-building package), `generateCoverProfile`
   deletes the coverage profile and returns a hard error. `go test` writes a
   complete, usable profile for every package that *did* pass — gaze throws it
   away. A test-quality tool that refuses to analyze any repository with a
   single failing test is unusable on real-world codebases.

2. **#108 — GazeCRAPload threshold always passes when unavailable**: 
   `ReportSummary.GazeCRAPload` is a plain `int`. When GazeCRAP data is
   unavailable (the common case without contract coverage), the field stays at
   its zero value, and `--max-gaze-crapload` evaluates `0 <= limit` = PASS.
   The `gaze crap` command correctly models this as `*int` and prints
   "GazeCRAP unavailable" — so the two commands disagree.

3. **#116 — Zero-result runs exit 0**: When a command analyzes nothing (mistyped
   pattern, no exported functions), gaze exits 0 with no output. A
   `--max-crapload` gate "passes" on a package with zero analyzable code,
   masking misconfiguration.

The recurring theme is that "could not analyze" is reported as a passing result.
This violates Constitution Principle III (Actionable Output): metrics must be
comparable across runs, and a pass must mean something was measured.

## What Changes

### Error handling in coverage profile generation (#101)

Change `generateCoverProfile` to preserve the coverage profile when `go test`
exits non-zero but the profile file exists and is non-empty. Emit a warning
listing which packages failed, then proceed with partial data. Only return a
hard error when no profile was produced at all.

### GazeCRAPload type consistency across the report pipeline (#108)

Change `ReportSummary.GazeCRAPload` and `crapStepResult.GazeCRAPload` from
`int` to `*int`, preserving the nil/unavailable signal from `crap.Summary`
through the entire pipeline. Update `EvaluateThresholds` to skip or fail
(configurable) when GazeCRAP data is unavailable and a threshold was requested.

### Zero-result exit code for gate commands (#116)

When a threshold-bearing command (`--max-crapload`, `--max-gaze-crapload`,
`--min-contract-coverage`) produces zero analyzable functions, return a
non-zero exit code with a clear error message. Non-gate invocations (no
threshold flags) retain the current warn-and-exit-0 behavior.

## Capabilities

### New Capabilities
- `partial-coverage-analysis`: `gaze crap` and `gaze report` produce results from partial coverage profiles when some packages fail to build or test, with warnings identifying the failed packages.
- `zero-result-gate-failure`: Gate commands exit non-zero when zero functions are analyzed, preventing misconfigured CI from silently passing.

### Modified Capabilities
- `gaze crap`: Tolerates partial `go test` failures; warns instead of aborting.
- `gaze report --max-gaze-crapload`: Fails (or warns) when GazeCRAP is unavailable, instead of silently passing with value 0.
- `gaze report` threshold evaluation: Distinguishes "metric unavailable" from "zero violations."

### Removed Capabilities
- None.

## Impact

- **Files modified**: `internal/crap/analyze.go`, `internal/aireport/payload.go`, `internal/aireport/runner_steps.go`, `internal/aireport/threshold.go`, `internal/aireport/runner.go`, `internal/aireport/compact.go`, `cmd/gaze/main.go`
- **Exit code changes**: Users with CI pipelines that rely on `gaze crap ./...` exiting 0 when `go test` fails will see a behavior change — partial results instead of hard failure. Users with zero-result gate commands will see exit 1 where they previously saw exit 0.
- **JSON output changes**: `ReportSummary.GazeCRAPload` changes from `int` to `*int` (JSON: `null` when unavailable instead of `0`). This is a breaking change for consumers that assume the field is always numeric.
- **No new dependencies**: All changes use standard library only.

## Constitution Alignment

Assessed against the Gaze project constitution (v1.3.0).

### I. Accuracy

**Assessment**: PASS

This change improves accuracy by preserving partial coverage data instead of
discarding it. A repository with 30 passing packages and 1 failing package
currently produces zero output; after this change it produces accurate scores
for the 30 packages that were measured. No new false positives or negatives
are introduced.

### II. Minimal Assumptions

**Assessment**: PASS

No new assumptions about the host project. The change *reduces* assumptions by
no longer assuming that `go test` must exit 0 for any coverage data to be
usable. The behavior change is backwards-compatible for well-configured
pipelines (where all tests pass, zero-result is a misconfiguration).

### III. Actionable Output

**Assessment**: PASS

This is the primary principle served. The change ensures that:
- "PASS" means something was measured and met the threshold.
- "GazeCRAP unavailable" is explicitly surfaced, not hidden behind a zero.
- Zero-result gate failures tell the user to check their package patterns.
- Partial-coverage warnings list exactly which packages failed and why.

### IV. Testability

**Assessment**: PASS

All three fixes are testable in isolation:
- #101: Unit test with a pre-written partial coverage profile + non-zero exit.
- #108: Unit test `EvaluateThresholds` with `nil` GazeCRAPload.
- #116: Unit test `runCrap`/`runAnalyze` with empty results + threshold flags.

Coverage strategy: unit tests for each modified function, plus integration
test verifying end-to-end exit codes.
