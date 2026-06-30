# Implementation Plan: Baseline GazeCRAP New-Function Threshold

**Branch**: `039-baseline-gazecrap-threshold` | **Date**: 2026-06-23 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/039-baseline-gazecrap-threshold/spec.md`
**Issue**: #164

## Summary

The baseline comparison `buildComparisonSummary` function only evaluates classic CRAP against `new_function_threshold` for new functions. GazeCRAP is not evaluated, creating a blind spot where new functions with poor contract coverage pass unchallenged. This fix adds a separate `new_function_gaze_crap_threshold` (default 30) to `BaselineConfig` and evaluates both metrics independently in `buildComparisonSummary`. A new function is classified as `new_violation` if either its CRAP exceeds the CRAP threshold or its GazeCRAP exceeds the GazeCRAP threshold.

## Technical Context

**Language/Version**: Go 1.25+ (per `go.mod` directive)
**Primary Dependencies**: Standard library only (no new dependencies). Existing: `gopkg.in/yaml.v3` (config), `encoding/json` (report output)
**Storage**: N/A — no persistence changes
**Testing**: Standard library `testing` package, `-race -count=1`
**Target Platform**: CLI binary (darwin/linux, amd64/arm64)
**Project Type**: Single Go module
**Performance Goals**: N/A — pure function changes, no performance-sensitive paths
**Constraints**: Backward compatibility with existing `.gaze.yaml` files and baseline JSON
**Scale/Scope**: 7 files modified, ~50 lines of production code, ~150 lines of test code

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Check

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. Accuracy** | ✅ PASS | This fix closes a false-negative gap: new functions with high GazeCRAP were not flagged as violations. The change makes the gate more accurate by evaluating both risk metrics. Automated regression tests (SC-001 through SC-007) will verify correctness. |
| **II. Minimal Assumptions** | ✅ PASS | No new assumptions about host projects. When GazeCRAP is unavailable (nil), only CRAP is evaluated — identical to current behavior. The new config field has a sensible default (30) and requires no user action. |
| **III. Actionable Output** | ✅ PASS | The text and JSON reports will display both CRAP and GazeCRAP scores for new-function violations (FR-007), telling users exactly which metric triggered the violation. The GazeCRAP threshold is included in the comparison summary for reproducibility (FR-006). |
| **IV. Testability** | ✅ PASS | All modified functions (`buildComparisonSummary`, `writeComparisonNewFunctions`, `WriteComparisonJSON`) are pure functions with injected dependencies. Tests use synthetic `Score` values — no external resources needed. Coverage strategy: unit tests for each acceptance scenario. |

**Gate Result**: PASS — no violations. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/039-baseline-gazecrap-threshold/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/gaze/
  main.go                    # Wire new config field into CompareOptions

internal/
  config/
    config.go                # Add NewFunctionGazeCRAPThreshold to BaselineConfig
  crap/
    compare.go               # Add field to CompareOptions, update buildComparisonSummary
    compare_report.go         # Update violation check in JSON/text output
    compare_test.go           # New tests for GazeCRAP threshold evaluation
    compare_report_test.go    # Update report output tests
    crap.go                   # Add NewFunctionGazeCRAPThreshold to ComparisonSummary
```

**Structure Decision**: All changes are within the existing package structure. No new packages, files, or directories are introduced. The change touches the `config`, `crap`, and `cmd/gaze` packages — the same packages that implement the existing baseline comparison feature.

## Design Decisions

### D1: Separate threshold field (not reuse existing)

GazeCRAP and CRAP are different metrics with different score distributions. GazeCRAP scores tend to be higher because they incorporate contract coverage gaps. A single shared threshold would force teams to choose between weakening the CRAP gate or accepting false violations from GazeCRAP. Independent thresholds allow calibration per metric.

**Alternative rejected**: Reusing `new_function_threshold` for both — rejected because it conflates two metrics with different scales.

### D2: Default value of 30 (matches CRAP default)

The default of 30 matches the existing `new_function_threshold` default. This is conservative — teams whose GazeCRAP scores naturally run higher can raise it. Starting equal avoids surprising users who haven't configured thresholds.

**Alternative rejected**: A higher default (e.g., 50) — rejected because it would weaken the gate for new adopters. Users who need a higher threshold can configure it explicitly.

### D3: Strict greater-than comparison (consistent with CRAP)

The existing CRAP threshold uses `s.CRAP > opts.NewFunctionThreshold` (strict greater-than). The GazeCRAP check uses the same operator for consistency. A score exactly at the threshold is not a violation.

### D4: Skip GazeCRAP check when nil (backward compatibility)

When `Score.GazeCRAP` is nil (contract coverage not available), only the CRAP threshold is evaluated. This preserves identical behavior for users who don't use GazeCRAP and for baselines created by older gaze versions.

### D5: OR semantics for violation (either metric triggers)

A new function is a `new_violation` if CRAP exceeds its threshold OR GazeCRAP exceeds its threshold. This is consistent with how `classifyDelta` handles regressions (any regression signal wins). The conservative approach catches risk from either dimension.

**Alternative rejected**: AND semantics (both must exceed) — rejected because it would allow a function with extreme GazeCRAP to pass if its CRAP is low, defeating the purpose of the GazeCRAP metric.

### D6: Report output shows both scores for violations

When a new function is a violation, the text report shows both CRAP and GazeCRAP scores so users can see which metric triggered it. This follows Constitution Principle III (Actionable Output) — users need to know what to fix.

## Coverage Strategy

| Layer | Scope | Target |
|-------|-------|--------|
| **Unit** | `buildComparisonSummary` — GazeCRAP above threshold, below threshold, nil, equal to threshold, both metrics above | 100% branch coverage of new logic |
| **Unit** | `writeComparisonNewFunctions` — violation with GazeCRAP displayed, violation without GazeCRAP | 100% of new output paths |
| **Unit** | `WriteComparisonJSON` — new function status includes GazeCRAP-triggered violations | Verify JSON status field |
| **Unit** | `config.Load` — validation of `new_function_gaze_crap_threshold` (zero, negative, valid) | 100% of validation branches |
| **Integration** | Full `Compare` → `WriteComparisonJSON`/`WriteComparisonText` flow with GazeCRAP threshold scenarios | End-to-end data flow |

All tests use synthetic `Score` values and in-memory buffers. No external processes, no filesystem access beyond test fixtures.

## Complexity Tracking

No constitution violations to justify. All changes are within existing patterns and structures.

## Post-Design Constitution Re-Check

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. Accuracy** | ✅ PASS | Closes the false-negative gap (issue #164). Both CRAP and GazeCRAP are evaluated for new functions. Nil GazeCRAP is handled correctly (no false positives). |
| **II. Minimal Assumptions** | ✅ PASS | Default config works without `.gaze.yaml`. New field is optional with sensible default. No user action required for existing setups. |
| **III. Actionable Output** | ✅ PASS | Violation output shows both scores. Summary includes both thresholds. Users can see exactly which metric triggered the violation and what the threshold was. |
| **IV. Testability** | ✅ PASS | Coverage strategy specified above. All new logic is in pure functions with injectable inputs. No global state, no I/O in the comparison algorithm. |

**Gate Result**: PASS — design is constitution-compliant.
