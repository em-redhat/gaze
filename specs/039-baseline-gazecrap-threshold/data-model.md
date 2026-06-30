# Data Model: Baseline GazeCRAP New-Function Threshold

**Feature**: 039-baseline-gazecrap-threshold
**Date**: 2026-06-23

## Type Changes

### `config.BaselineConfig` (internal/config/config.go)

```go
type BaselineConfig struct {
    File                        string  `yaml:"file"`
    Epsilon                     float64 `yaml:"epsilon"`
    NewFunctionThreshold        float64 `yaml:"new_function_threshold"`
    // NEW: GazeCRAP threshold for new functions. Must be > 0. Default: 30.
    NewFunctionGazeCRAPThreshold float64 `yaml:"new_function_gaze_crap_threshold"`
}
```

**Default**: 30 (set in `DefaultConfig()`)
**Validation**: `> 0` (same as `NewFunctionThreshold`)
**YAML key**: `baseline.new_function_gaze_crap_threshold`

### `crap.CompareOptions` (internal/crap/compare.go)

```go
type CompareOptions struct {
    Epsilon              float64
    NewFunctionThreshold float64
    // NEW: GazeCRAP score above which a new function is a violation.
    NewFunctionGazeCRAPThreshold float64
}
```

**Purpose**: Carries the GazeCRAP threshold from configuration to the comparison algorithm. No default — set by caller from config.

### `crap.ComparisonSummary` (internal/crap/crap.go)

```go
type ComparisonSummary struct {
    Regressions                  int     `json:"regressions"`
    Improvements                 int     `json:"improvements"`
    NewFunctions                 int     `json:"new_functions"`
    NewViolations                int     `json:"new_violations"`
    RemovedFunctions             int     `json:"removed_functions"`
    Unchanged                    int     `json:"unchanged"`
    Epsilon                      float64 `json:"epsilon"`
    NewFunctionThreshold         float64 `json:"new_function_threshold"`
    // NEW: GazeCRAP threshold for reproducibility in JSON output.
    NewFunctionGazeCRAPThreshold float64 `json:"new_function_gaze_crap_threshold"`
    Passed                       bool    `json:"passed"`
}
```

**JSON field**: `new_function_gaze_crap_threshold`
**Purpose**: Makes the GazeCRAP threshold visible in comparison output for reproducibility (FR-006).

## Algorithm Changes

### `buildComparisonSummary` (internal/crap/compare.go)

Current logic:
```go
for _, s := range result.NewFunctions {
    if s.CRAP > opts.NewFunctionThreshold {
        summary.NewViolations++
    } else {
        summary.NewFunctions++
    }
}
```

New logic:
```go
for _, s := range result.NewFunctions {
    crapViolation := s.CRAP > opts.NewFunctionThreshold
    gazeViolation := s.GazeCRAP != nil && *s.GazeCRAP > opts.NewFunctionGazeCRAPThreshold
    if crapViolation || gazeViolation {
        summary.NewViolations++
    } else {
        summary.NewFunctions++
    }
}
```

Key behaviors:
- **OR semantics**: Either metric exceeding its threshold triggers a violation (D5)
- **Nil-safe**: `s.GazeCRAP != nil` guard prevents nil dereference (D4)
- **Strict greater-than**: Consistent with existing CRAP comparison (D3)

### `writeComparisonNewFunctions` (internal/crap/compare_report.go)

The function signature gains a `gazeCRAPThreshold float64` parameter. The violation classification uses the same OR logic as `buildComparisonSummary`. The text output includes GazeCRAP when available:

```
  complexFunc (complex.go)                    CRAP: 42.0  GazeCRAP: 55.0  [VIOLATION]
```

When GazeCRAP is nil, the output remains unchanged:
```
  complexFunc (complex.go)                    CRAP: 42.0  [VIOLATION]
```

### `WriteComparisonJSON` (internal/crap/compare_report.go)

The new-function status classification (lines 117-126) uses the same OR logic. The GazeCRAP threshold is read from `result.Summary.NewFunctionGazeCRAPThreshold`.

## Configuration Example

```yaml
# .gaze.yaml
baseline:
  file: ".gaze/baseline.json"
  epsilon: 0.5
  new_function_threshold: 30
  new_function_gaze_crap_threshold: 40  # Higher tolerance for GazeCRAP
```

## JSON Output Example

```json
{
  "comparison": {
    "regressions": 0,
    "improvements": 1,
    "new_functions": 2,
    "new_violations": 1,
    "removed_functions": 0,
    "unchanged": 3,
    "epsilon": 0.5,
    "new_function_threshold": 30,
    "new_function_gaze_crap_threshold": 30,
    "passed": false
  }
}
```

## No Changes Required

- `Score` struct — GazeCRAP field already exists (`*float64`)
- `FunctionDelta` struct — no new-function delta tracking needed
- `ComparisonResult` struct — no structural changes
- `classifyDelta` function — already handles both metrics for existing functions
- Baseline JSON format — thresholds come from config, not baseline files
