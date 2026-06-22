## Context

Gaze's CI gate commands have three classes of silent-pass bugs where "could not
analyze" or "data unavailable" is reported as a passing result. The proposal
(proposal.md) identifies issues #101, #108, and #116 as a cohesive cluster
sharing the theme: CI gates must distinguish "pass" from "no data."

Current state:
- `generateCoverProfile` deletes the coverage file on any `go test` non-zero
  exit, even though `go test` writes usable partial profiles.
- `ReportSummary.GazeCRAPload` is `int` while `crap.Summary.GazeCRAPload` is
  `*int` — the nil signal is lost at the pipeline boundary.
- Zero-result runs exit 0 regardless of whether threshold flags were provided.

## Goals / Non-Goals

### Goals
- Preserve partial coverage profiles when `go test` exits non-zero but wrote
  usable data (#101).
- Propagate the `*int` nil/unavailable signal for GazeCRAPload through the
  entire report pipeline so `EvaluateThresholds` can distinguish "zero
  violations" from "not computed" (#108).
- Exit non-zero when a threshold-bearing command produces zero analyzable
  functions (#116).
- Align `gaze report` and `gaze crap` behavior so the same input produces
  consistent pass/fail semantics.

### Non-Goals
- Changing the CRAP formula or scoring algorithm.
- Adding new threshold flags or configuration options.
- Fixing other silent-failure bugs (#107 analyze single-package, #113
  subdirectory coverage) — those are separate issues with different root causes.
- Changing exit codes for non-gate invocations (no threshold flags).
- Aligning `gaze crap`'s `int`-based threshold model with `gaze report`'s
  `*int` + `cmd.Flags().Changed()` model — that's a separate refactoring task.
  This change uses `cmd.Flags().Changed()` for the zero-result check only.

## Decisions

### D1: Partial coverage profile — warn and continue

**Decision**: When `go test` exits non-zero, check whether the coverage profile
exists and has non-zero size. If so, proceed with the partial profile and emit
a warning to stderr listing the `go test` exit code. If the profile is missing
or empty, return a hard error as today.

**Rationale**: `go test -coverprofile` writes coverage data for each package as
it completes, regardless of whether later packages fail. The profile is usable.
Discarding it violates Principle III (Actionable Output) — users get zero
information instead of partial information.

**Design**: Modify `generateCoverProfile` in `internal/crap/analyze.go`:

```go
output, err := cmd.CombinedOutput()
if err != nil {
    // Check if profile was written despite non-zero exit
    info, statErr := os.Stat(profilePath)
    if statErr != nil || info.Size() == 0 {
        _ = os.Remove(profilePath)
        return "", fmt.Errorf("go test failed and produced no coverage: %s\n%s", err, string(output))
    }
    // Profile exists with data — warn and continue using injected writer
    fmt.Fprintf(stderr, "warning: go test exited with error (partial coverage used): %s\n", err)
}
```

Note: The code snippet above uses an injected `stderr io.Writer` parameter (not
`os.Stderr` directly) for testability. The `crap.Options` struct already carries
a `Stderr io.Writer` field — `generateCoverProfile` should accept it as a
parameter. See task 1.2.

The `--coverprofile` flag bypasses `generateCoverProfile` entirely, so
user-supplied profiles are trusted as-is. The partial-coverage tolerance only
applies to internally-generated profiles.

The caller (`Analyze`) already handles the profile path correctly — it defers
removal for auto-generated profiles and validates user-supplied ones. No
changes needed there.

### D2: GazeCRAPload as `*int` through the pipeline

**Decision**: Change `crapStepResult.GazeCRAPload` and
`ReportSummary.GazeCRAPload` from `int` to `*int`.

**Rationale**: The `crap.Summary` already models this correctly as `*int`. The
lossy `*int` → `int` conversion at `runner_steps.go:68` is the root cause of
#108. Fixing the type propagation aligns with the existing pattern and makes
`EvaluateThresholds` consistent with `checkCIThresholds` in `cmd/gaze/main.go`,
which already checks `!= nil` before comparing.

**Design**:

1. `internal/aireport/runner_steps.go`: Change `crapStepResult.GazeCRAPload`
   to `*int`. Assign directly: `res.GazeCRAPload = rpt.Summary.GazeCRAPload`.

2. `internal/aireport/payload.go`: Change `ReportSummary.GazeCRAPload` to
   `*int`. Note: `ReportSummary` has `json:"-"` on the struct tag so JSON
   serialization tags are irrelevant — this change is purely for threshold
   evaluation. The actual JSON output path flows through `crap.Summary`
   (already `*int`) → `crap.WriteJSON` → `ReportPayload.CRAP` (raw JSON).

3. `internal/aireport/payload.go`: Change `ThresholdResult.Actual` from `int`
   to `*int` so the struct can represent "metric unavailable" (`nil`) vs
   "zero violations" (`*0`). This is needed because task 2.4 appends a FAIL
   result when GazeCRAP is unavailable — setting `Actual = 0` would be
   misleading (the same ambiguity being fixed).

4. `internal/aireport/compact.go`: Change `compactSummary.GazeCRAPload` from
   `int` to `*int`. Update the `CompactForAI()` method to assign the pointer
   directly. This file is in the JSON serialization path for the AI adapter —
   if it stays `int`, the nil/unavailable signal is lost in the compact output
   too.

5. `internal/aireport/threshold.go`: In `EvaluateThresholds`, when
   `cfg.MaxGazeCrapload != nil` but `summary.GazeCRAPload == nil`:
   - Set `Passed = false` with `Actual = nil` and a result name indicating
     "GazeCRAPload (unavailable)."
   - The threshold is explicitly *not met* when the data doesn't exist — this
     is the safe default for CI gates.

6. `internal/aireport/runner.go`: Update the assignment from `crapStepResult`
   to `ReportSummary` to pass through `*int` directly. Update
   `evaluateAndPrintThresholds` formatting to handle `*int` `Actual` (print
   "N/A" when nil).

**JSON impact**: The `gaze_crapload` field in `gaze report --format=json`
output changes from `0` to `null` when GazeCRAP is unavailable. This is a
semantically correct breaking change that aligns with `gaze crap --format=json`
which already emits `null`.

### D3: Zero-result gate failure

**Decision**: When a command with threshold flags produces zero analyzable
functions, return a non-zero exit code with a descriptive error.

**Rationale**: A CI gate that passes when nothing was measured provides false
assurance. The user likely misconfigured the package pattern. Non-gate
invocations (no threshold flags) retain warn-and-exit-0 behavior since the
user may be exploring interactively.

**Design**: Add a zero-result check after analysis completes in each gate path.
The check should fire when *any* threshold flag was explicitly provided, using
`cmd.Flags().Changed()` rather than `value > 0`, to correctly handle
`--max-crapload=0` as a live threshold. Note: `gaze crap` currently uses plain
`int` flags with `0 = no limit` semantics — the check should use
`cmd.Flags().Changed("max-crapload") || cmd.Flags().Changed("max-gaze-crapload")`
to detect intent.

- `cmd/gaze/main.go` `runCrap`: After `crap.Analyze`, if `len(rpt.Scores) == 0`
  and any threshold flag was explicitly set (via `cmd.Flags().Changed`), return
  an error:
  `"no functions analyzed — cannot evaluate thresholds (check package patterns)"`.
  The `runCrap` function will need access to the `*cobra.Command` to check flag
  state — thread it through `crapParams` or pass a `thresholdSet bool` computed
  by the caller.

- `cmd/gaze/main.go` `runAnalyze`: Already warns on zero results. No threshold
  flags exist for `analyze`, so no change needed.

- `internal/aireport/runner.go`: The zero-result check for `gaze report` should
  live in `Run()` between pipeline completion and threshold evaluation (not
  inside `runProductionPipeline`, which doesn't have access to threshold config).
  When the CRAP step succeeds with zero scores and any threshold is configured
  in `ThresholdConfig`, the check reports a threshold failure.

### D4: Stderr for warnings, not a new logger

**Decision**: Use `fmt.Fprintf(stderr, "warning: ...")` for partial-coverage
and zero-result warnings, consistent with existing warning patterns in the
codebase.

**Rationale**: The codebase already uses `charmbracelet/log` for structured
logging and `fmt.Fprintf(stderr, ...)` for user-facing warnings. These
warnings are user-facing ("your go test failed, here's what we did"), so
stderr fprintf is the correct channel.

## Coverage Strategy

Per Constitution Principle IV, coverage strategy for all new code:

- **Unit tests** (tasks 1.3-1.5, 2.6-2.9, 3.3-3.5): Each modified function
  gets direct unit tests covering all branches. `generateCoverProfile` tests
  use dependency injection (a `cmdRunner` function parameter) to avoid spawning
  real `go test` processes. `EvaluateThresholds` tests construct `ReportSummary`
  with `nil` vs `*int` GazeCRAPload. `runCrap` zero-result tests use the
  existing `crapParams` pattern with `io.Writer` injection.
- **Integration test** (task 4.3): Extend `TestSC002_GazeCRAPloadMatchBetweenCrapAndReport`
  to verify the `null`/unavailable case is consistent between commands.
- **Coverage target**: All new code paths (partial coverage branching,
  nil GazeCRAPload threshold logic, zero-result gate) must have >80% branch
  coverage. Existing coverage ratchets continue to apply.

## Risks / Trade-offs

### R1: Breaking JSON change for GazeCRAPload

The `gaze_crapload` field changes from `0` to `null` in `gaze report` JSON
output. Consumers that parse this field as a plain integer will break.

**Mitigation**: This aligns with `gaze crap` JSON output which already emits
`null`. Document the change in release notes. The field was already semantically
wrong (reporting `0` when the value was unknown), so consumers were getting
incorrect data.

### R2: Partial coverage may produce misleading scores

When 5 of 30 packages fail to build, coverage for functions in those 5 packages
reads as 0%. This could inflate CRAP scores for those specific functions.

**Mitigation**: The alternative (no output at all) is strictly worse. The
warning message tells the user which packages failed, so they can interpret
the scores in context. Future work could annotate individual scores with
"coverage unavailable" markers.

### R3: Zero-result exit code may break existing CI

Pipelines that run `gaze crap --max-crapload=5 ./some/empty/pkg` and expect
exit 0 will start failing.

**Mitigation**: This is the correct behavior — a gate that measures nothing
should not pass. The error message is clear and actionable. Users can remove
the threshold flag if they want the exploratory warn-and-exit-0 behavior.
