# Quickstart: Baseline GazeCRAP New-Function Threshold

**Feature**: 039-baseline-gazecrap-threshold
**Date**: 2026-06-23

## What This Changes

The `gaze crap --baseline` comparison now evaluates GazeCRAP scores for new functions, not just classic CRAP. A new function is flagged as a violation if either:
- Its CRAP score exceeds `new_function_threshold` (existing behavior), OR
- Its GazeCRAP score exceeds `new_function_gaze_crap_threshold` (new behavior)

## Before (Bug)

A new function with CRAP 25 (below threshold 30) but GazeCRAP 40 would pass the baseline gate — even though it carries high change risk due to poor contract coverage.

## After (Fix)

The same function is now flagged as `new_violation` because GazeCRAP 40 exceeds the GazeCRAP threshold of 30.

## Configuration

The new threshold is configured in `.gaze.yaml`:

```yaml
baseline:
  new_function_threshold: 30              # CRAP threshold (existing)
  new_function_gaze_crap_threshold: 40    # GazeCRAP threshold (new)
```

If not configured, the default is 30 (same as the CRAP threshold default).

## Backward Compatibility

- **No `.gaze.yaml`**: Default threshold of 30 applies. Existing behavior for CRAP is unchanged. GazeCRAP evaluation is new but uses the same default.
- **GazeCRAP unavailable (nil)**: Only CRAP is evaluated. Identical to current behavior.
- **Existing `.gaze.yaml` without new field**: Default of 30 is used. No user action required.

## Key Files

| File | What Changes |
|------|-------------|
| `internal/config/config.go` | New `NewFunctionGazeCRAPThreshold` field in `BaselineConfig` |
| `internal/crap/compare.go` | `CompareOptions` gains field; `buildComparisonSummary` evaluates GazeCRAP |
| `internal/crap/crap.go` | `ComparisonSummary` gains `new_function_gaze_crap_threshold` JSON field |
| `internal/crap/compare_report.go` | Text/JSON output shows GazeCRAP for violations |
| `cmd/gaze/main.go` | Wires config field into `CompareOptions` |

## Testing

Run the comparison tests:
```bash
go test -race -count=1 ./internal/crap/ -run 'TestSC'
go test -race -count=1 ./internal/config/ -run 'TestBaseline'
```

Run all tests:
```bash
go test -race -count=1 -short ./...
```
