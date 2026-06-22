## 1. Partial Coverage Profile Tolerance (#101)

- [x] 1.1 Modify `generateCoverProfile` in `internal/crap/analyze.go` to check whether the coverage profile exists and has non-zero size when `go test` exits non-zero. If so, emit a warning to the injected `stderr` writer and return the profile path. If the profile is missing or empty, clean up and return a hard error. Note: `crap.Options` already has a `Stderr io.Writer` field — pass it to `generateCoverProfile` as a parameter.
- [x] 1.2 Thread the `stderr io.Writer` parameter from `crap.Options.Stderr` into `generateCoverProfile`. The function currently has no access to a writer — add `stderr io.Writer` as a parameter. No `Analyze` signature change should be needed since `Options` already carries the writer.
- [x] 1.3 Add unit test: `go test` exits non-zero but profile exists with data — verify `generateCoverProfile` returns the profile path and emits a warning. Use dependency injection for the subprocess call (inject a `cmdRunner` function or pre-write a profile file and test the stat/size logic in isolation).
- [x] 1.4 Add unit test: `go test` exits non-zero and profile is missing — verify hard error returned.
- [x] 1.5 Add unit test: `go test` exits non-zero and profile is empty (0 bytes) — verify hard error returned and file is cleaned up.

## 2. GazeCRAPload Type Propagation (#108)

- [x] 2.1 Change `crapStepResult.GazeCRAPload` from `int` to `*int` in `internal/aireport/runner_steps.go`. Update `runCRAPStep` to assign `res.GazeCRAPload = rpt.Summary.GazeCRAPload` directly (no dereference).
- [x] 2.2 Change `ReportSummary.GazeCRAPload` from `int` to `*int` in `internal/aireport/payload.go`. Note: `ReportSummary` has `json:"-"` so JSON tags are irrelevant — this change is for threshold evaluation only.
- [x] 2.3 Change `ThresholdResult.Actual` from `int` to `*int` in `internal/aireport/payload.go` so threshold results can represent "metric unavailable" (`nil`) vs "zero violations" (`*0`).
- [x] 2.4 Change `compactSummary.GazeCRAPload` from `int` to `*int` in `internal/aireport/compact.go`. Update the `CompactForAI()` method to assign the `*int` pointer directly from `ReportSummary`.
- [x] 2.5 Update `runProductionPipeline` in `internal/aireport/runner.go` to assign `*int` directly from `crapStepResult` to `ReportSummary`.
- [x] 2.6 Update `EvaluateThresholds` in `internal/aireport/threshold.go`: when `cfg.MaxGazeCrapload != nil` but `summary.GazeCRAPload == nil`, append a FAIL result with `Actual = nil` and name "GazeCRAPload (unavailable)", set `allPassed = false`. When both are non-nil, dereference and compare as before.
- [x] 2.7 Update `ThresholdResult` formatting in `evaluateAndPrintThresholds` (`internal/aireport/runner.go`) to handle `*int` `Actual` — print "N/A" when nil, dereference when non-nil.
- [x] 2.8 Update all existing tests that construct `ReportSummary` or `crapStepResult` with `GazeCRAPload` as a bare `int` to use `*int` (pointer). Files: `threshold_test.go`, `pipeline_internal_test.go`, `payload_test.go`, `compact_test.go`.
- [x] 2.9 Add unit test for `EvaluateThresholds`: GazeCRAPload threshold set, data unavailable (nil) — verify FAIL result with `Actual == nil`.
- [x] 2.10 Add unit test for `EvaluateThresholds`: GazeCRAPload threshold set, data available and within limit — verify PASS.
- [x] 2.11 Add unit test for `EvaluateThresholds`: GazeCRAPload threshold set, data available and exceeds limit — verify FAIL.
- [x] 2.12 Add unit test for `EvaluateThresholds`: GazeCRAPload threshold not set, data unavailable — verify no result emitted (threshold skipped).
- [x] 2.13 Add unit test for `EvaluateThresholds`: GazeCRAPload threshold not set, data available — verify no result emitted (threshold skipped).

## 3. Zero-Result Gate Failure (#116)

- [x] 3.1 In `runCrap` (`cmd/gaze/main.go`), after `crap.Analyze` returns, check if `len(rpt.Scores) == 0` and any threshold flag was explicitly provided (use `cmd.Flags().Changed("max-crapload") || cmd.Flags().Changed("max-gaze-crapload")`). Thread the flag-changed state through `crapParams` as a `thresholdSet bool` field. If so, return an error: `"no functions analyzed — cannot evaluate thresholds (check package patterns)"`.
- [x] 3.2 In `Run()` (`internal/aireport/runner.go`), between pipeline completion and threshold evaluation, check if the CRAP step succeeded with zero scores and any threshold is configured in `ThresholdConfig`. If so, report a threshold failure rather than silently passing.
- [x] 3.3 Add unit test for `runCrap`: zero results with `--max-crapload` flag changed — verify non-zero exit.
- [x] 3.4 Add unit test for `runCrap`: zero results without threshold flags — verify exit 0 with warning (existing behavior preserved).
- [x] 3.5 Add unit test for pipeline: zero CRAP scores with thresholds configured — verify threshold evaluation fails.

## 4. Cross-Cutting Verification

- [x] 4.1 Run `go test -race -count=1 -short ./...` — all existing tests must pass.
- [x] 4.2 Run `golangci-lint run` — no new lint violations.
- [x] 4.3 Extend `TestSC002_GazeCRAPloadMatchBetweenCrapAndReport` to verify that both `gaze crap --format=json` and `gaze report --format=json` produce `null` (not `0`) for `gaze_crapload` when GazeCRAP is unavailable — confirm consistency between the two commands as an automated test.
- [x] 4.4 Check whether any JSON Schema definitions (e.g., in `internal/report/schema.go`) need updating to allow `null` for the `gaze_crapload` field, and update them if so.
- [x] 4.5 Verify constitution alignment: Principle III (Actionable Output) — confirm that all three fixes produce machine-parseable output with clear pass/fail semantics and that partial results include provenance (which packages failed). Principle IV (Testability) — confirm all new code paths are covered by unit tests with >80% branch coverage.
- [x] 4.6 Update AGENTS.md "Recent Changes" section with a `ci-gate-integrity` entry summarizing the three fixes.

<!-- spec-review: passed -->
<!-- code-review: passed -->
