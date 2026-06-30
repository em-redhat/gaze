# Research: Baseline GazeCRAP New-Function Threshold

**Feature**: 039-baseline-gazecrap-threshold
**Date**: 2026-06-23
**Status**: Complete

## R1: Current Baseline Comparison Architecture

### How `buildComparisonSummary` works today

The `buildComparisonSummary` function in `internal/crap/compare.go` (lines 159-190) iterates over new functions and classifies each as either `new` or `new_violation` based solely on the classic CRAP score:

```go
for _, s := range result.NewFunctions {
    if s.CRAP > opts.NewFunctionThreshold {
        summary.NewViolations++
    } else {
        summary.NewFunctions++
    }
}
```

GazeCRAP is completely ignored for new functions, even when available. This is the bug described in issue #164.

### Contrast with `classifyDelta` (existing functions)

For existing functions (present in both baseline and current), `classifyDelta` (lines 131-155) already evaluates both CRAP and GazeCRAP symmetrically:

```go
if crapRegression || gazeRegression {
    return StatusRegression
}
```

The new-function threshold check is the only place where GazeCRAP is not evaluated. This fix brings it into alignment.

### Data flow: config → CompareOptions → buildComparisonSummary

1. `.gaze.yaml` → `config.Load()` → `BaselineConfig.NewFunctionThreshold`
2. `cmd/gaze/main.go:loadAndCompare()` → `CompareOptions{NewFunctionThreshold: cfg.Baseline.NewFunctionThreshold}`
3. `Compare()` → `buildComparisonSummary(result, opts)`
4. `buildComparisonSummary` uses `opts.NewFunctionThreshold` for the CRAP check

The new GazeCRAP threshold follows the same path: config → CompareOptions → buildComparisonSummary.

## R2: Score.GazeCRAP Nullability

`Score.GazeCRAP` is `*float64` (pointer, nullable). It is nil when:
- Contract coverage analysis was not run
- The function has no side effects to measure
- The baseline was created by an older gaze version

The new-function GazeCRAP check must handle nil gracefully: when `GazeCRAP` is nil, only the CRAP threshold is evaluated. This is consistent with how `classifyDelta` handles nil GazeCRAP (lines 96-101 of compare.go).

## R3: ComparisonSummary JSON Output

`ComparisonSummary` (lines 203-213 of crap.go) currently includes:
- `new_function_threshold` (float64) — the CRAP threshold used

Adding `new_function_gaze_crap_threshold` (float64) to this struct makes the GazeCRAP threshold visible in JSON output for reproducibility (FR-006).

## R4: Report Output for New Functions

### Text output (`writeComparisonNewFunctions`)

Currently shows CRAP score only for violations:
```
New Functions (violations):
  complexFunc (complex.go)                    CRAP: 42.0  [VIOLATION]
```

The fix adds GazeCRAP when available:
```
New Functions (violations):
  complexFunc (complex.go)                    CRAP: 42.0  GazeCRAP: 55.0  [VIOLATION]
```

### JSON output (`WriteComparisonJSON`)

The `newFunctionJSON` struct determines the status for new functions (lines 117-126 of compare_report.go). Currently:
```go
status := StatusNew
if nf.CRAP > result.Summary.NewFunctionThreshold {
    status = StatusNewViolation
}
```

This must be updated to also check GazeCRAP against the GazeCRAP threshold. The GazeCRAP score is already included in the `Score` struct (embedded in `newFunctionJSON`), so no new fields are needed in the JSON output — only the status classification logic changes.

## R5: Configuration Validation Pattern

The existing `new_function_threshold` validation in `config.Load()` (lines 133-136):
```go
if cfg.Baseline.NewFunctionThreshold <= 0 {
    return nil, fmt.Errorf("baseline.new_function_threshold must be > 0, got %g",
        cfg.Baseline.NewFunctionThreshold)
}
```

The new `new_function_gaze_crap_threshold` follows the same pattern: must be > 0, same error format.

## R6: Backward Compatibility

### Existing `.gaze.yaml` files

YAML unmarshaling into a struct with `DefaultConfig()` as the base means missing fields get the default value. An existing `.gaze.yaml` without `new_function_gaze_crap_threshold` will use the default of 30. No user action required.

### Existing baseline JSON files

Baseline JSON files don't contain threshold configuration — thresholds come from `.gaze.yaml` or defaults. No baseline file changes needed.

### Existing comparison JSON output

The `ComparisonSummary` gains a new field (`new_function_gaze_crap_threshold`). Consumers using `json.Unmarshal` with a struct will ignore unknown fields. Consumers parsing raw JSON will see the new field. This is additive and non-breaking.

## R7: Affected Functions (Complete List)

| Function | File | Change |
|----------|------|--------|
| `BaselineConfig` | `internal/config/config.go` | Add `NewFunctionGazeCRAPThreshold float64` field |
| `DefaultConfig` | `internal/config/config.go` | Set default to 30 |
| `Load` | `internal/config/config.go` | Add validation (> 0) |
| `CompareOptions` | `internal/crap/compare.go` | Add `NewFunctionGazeCRAPThreshold float64` field |
| `buildComparisonSummary` | `internal/crap/compare.go` | Check GazeCRAP against threshold for new functions |
| `ComparisonSummary` | `internal/crap/crap.go` | Add `NewFunctionGazeCRAPThreshold float64` JSON field |
| `writeComparisonNewFunctions` | `internal/crap/compare_report.go` | Show GazeCRAP in violation output |
| `WriteComparisonJSON` | `internal/crap/compare_report.go` | Use GazeCRAP threshold for status classification |
| `loadAndCompare` | `cmd/gaze/main.go` | Wire `cfg.Baseline.NewFunctionGazeCRAPThreshold` into `CompareOptions` |

## Open Questions

None — all technical questions resolved through code analysis.
