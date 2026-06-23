## 1. Tests (write before implementation)

- [x] 1.1 Add test `TestDeltaTable_TwoRowFormat_WithGazeCRAP` in `internal/crap/compare_report_test.go` — build a `ComparisonResult` with a regression that has both CRAP and GazeCRAP deltas, call `WriteComparisonText`, verify output contains function name on its own line, indented `GazeCRAP` row with baseline/current/delta, and indented `CRAP` row with baseline/current/delta
- [x] 1.2 Add test `TestDeltaTable_TwoRowFormat_MixedGazeCRAP` in `internal/crap/compare_report_test.go` — one function has GazeCRAP data, another does not; verify the function with GazeCRAP gets both rows, the function without GazeCRAP gets only the `CRAP` row (no empty GazeCRAP row)
- [x] 1.3 Add test `TestDeltaTable_SingleRowFormat_NoGazeCRAP` in `internal/crap/compare_report_test.go` — all functions have nil GazeCRAP deltas; verify output uses the existing single-row format unchanged (backward compatibility)
- [x] 1.4 Add test `TestDeltaTable_GazeCRAPBeforeCRAP` in `internal/crap/compare_report_test.go` — verify `GazeCRAP` row appears before `CRAP` row in two-row output (order constraint from D1)
- [x] 1.5 Add test `TestDeltaTable_TwoRowFormat_WidthCompliance` in `internal/crap/compare_report_test.go` — verify no output line exceeds 80 characters when rendering the two-row format with a 40-char function name

## 2. Implementation

- [x] 2.1 Update `writeComparisonDeltaTable` in `internal/crap/compare_report.go` — add detection of whether any matching delta has non-nil `GazeCRAPDelta`; if so, switch to two-row rendering per D1/D2/D3/D4
- [x] 2.2 In two-row mode: print function name (with file path) on its own line, then indented `GazeCRAP` row (if non-nil) with `Baseline.GazeCRAP`, `Current.GazeCRAP`, `GazeCRAPDelta`, then indented `CRAP` row with `Baseline.CRAP`, `Current.CRAP`, `CRAPDelta`
- [x] 2.3 In single-row mode (no GazeCRAP data in any matching delta): preserve the existing format unchanged

## 3. Validation

- [x] 3.1 Run `go build ./cmd/gaze` — verify clean compilation
- [x] 3.2 Run `go test -race -count=1 -short ./internal/crap/...` — verify all comparison and report tests pass
- [x] 3.3 Run `go test -race -count=1 -short ./...` — verify no regressions across the full module
- [x] 3.4 Run `golangci-lint run` — verify no lint violations in changed packages

## 4. Constitution Alignment Verification

- [x] 4.1 Verify Observable Quality (III): text output now surfaces GazeCRAP delta data that was previously hidden; JSON output unchanged
- [x] 4.2 Verify Testability (IV): all new behavior covered by unit tests using in-memory buffers and synthetic `FunctionDelta` inputs

<!-- spec-review: passed -->
<!-- code-review: passed -->
