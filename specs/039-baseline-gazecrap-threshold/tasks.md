# Tasks: Baseline GazeCRAP New-Function Threshold

**Input**: Design documents from `specs/039-baseline-gazecrap-threshold/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, quickstart.md
**Issue**: #164

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Foundational Type Changes (Blocking Prerequisites)

**Purpose**: Add the new field to all types that carry the GazeCRAP threshold through the system. These structural changes MUST be complete before any user story logic can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T001 [P] Add `NewFunctionGazeCRAPThreshold float64` field to `CompareOptions` struct in `internal/crap/compare.go` with GoDoc comment per data-model.md
- [x] T002 [P] Add `NewFunctionGazeCRAPThreshold float64` field with `json:"new_function_gaze_crap_threshold"` tag to `ComparisonSummary` struct in `internal/crap/crap.go` with GoDoc comment per data-model.md
- [x] T003 [P] Add `NewFunctionGazeCRAPThreshold float64` field with `yaml:"new_function_gaze_crap_threshold"` tag to `BaselineConfig` struct in `internal/config/config.go` with GoDoc comment per data-model.md
- [x] T004 Set default value `NewFunctionGazeCRAPThreshold: 30` in `DefaultConfig()` in `internal/config/config.go`
- [x] T005 Wire `cfg.Baseline.NewFunctionGazeCRAPThreshold` into `CompareOptions` in `loadAndCompare` function in `cmd/gaze/main.go` (line ~642)
- [x] T006 Propagate `opts.NewFunctionGazeCRAPThreshold` into `summary.NewFunctionGazeCRAPThreshold` in `buildComparisonSummary` in `internal/crap/compare.go` (line ~162)

**Checkpoint**: All types carry the GazeCRAP threshold field. Compilation must succeed. No behavioral changes yet — the new field exists but is not evaluated.

---

## Phase 2: US1 — GazeCRAP Violation Detection for New Functions (Priority: P1) 🎯 MVP

**Goal**: New functions are classified as `new_violation` if either CRAP exceeds its threshold OR GazeCRAP exceeds its threshold (when GazeCRAP is available).

**Independent Test**: Run `Compare()` with a new function that has CRAP below threshold but GazeCRAP above threshold. Verify `new_violation` classification and `Summary.Passed == false`.

### Tests for US1

- [x] T007 [US1] Add test `TestSC001_NewFunction_GazeCRAPAboveThreshold` in `internal/crap/compare_test.go` — new function with CRAP 25 (below 30) and GazeCRAP 40 (above 30) is classified as `new_violation`, `Summary.Passed` is false
- [x] T008 [US1] Add test `TestSC001_NewFunction_BothAboveThreshold` in `internal/crap/compare_test.go` — new function with CRAP 42 and GazeCRAP 55 (both above 30) is classified as `new_violation`
- [x] T009 [US1] Add test `TestSC001_NewFunction_BothBelowThreshold` in `internal/crap/compare_test.go` — new function with CRAP 5 and GazeCRAP 8 (both below 30) is classified as `new` (not violation), `Summary.Passed` is true
- [x] T010 [US1] Add test `TestSC001_NewFunction_GazeCRAPEqualToThreshold` in `internal/crap/compare_test.go` — new function with GazeCRAP exactly 30 (equal to threshold 30) is NOT a violation (strict greater-than per D3)

### Implementation for US1

- [x] T011 [US1] Update `buildComparisonSummary` in `internal/crap/compare.go` — add GazeCRAP threshold check with nil guard and OR semantics per data-model.md algorithm (lines 177-183)

**Checkpoint**: Core violation detection works. `buildComparisonSummary` evaluates both CRAP and GazeCRAP for new functions. All T007-T010 tests pass.

---

## Phase 3: US2 — Independent Threshold Configuration (Priority: P2)

**Goal**: Teams can configure `new_function_gaze_crap_threshold` independently from `new_function_threshold` in `.gaze.yaml`.

**Independent Test**: Create a `.gaze.yaml` with `baseline.new_function_gaze_crap_threshold: 40`, run comparison with a new function at GazeCRAP 38 (passes) and 42 (fails).

### Tests for US2

- [x] T012 [US2] Add test `TestSC004_IndependentThresholds` in `internal/crap/compare_test.go` — CRAP threshold 30, GazeCRAP threshold 40: new function with CRAP 25 and GazeCRAP 38 passes (both below respective thresholds)
- [x] T013 [US2] Add test `TestSC004_IndependentThresholds_GazeCRAPExceeds` in `internal/crap/compare_test.go` — CRAP threshold 30, GazeCRAP threshold 40: new function with CRAP 25 and GazeCRAP 42 is `new_violation`
- [x] T014 [US2] Add test `TestSC007_ConfigValidation_ZeroGazeCRAPThreshold` in `internal/config/config_test.go` — `.gaze.yaml` with `new_function_gaze_crap_threshold: 0` returns validation error
- [x] T015 [US2] Add test `TestSC007_ConfigValidation_NegativeGazeCRAPThreshold` in `internal/config/config_test.go` — `.gaze.yaml` with `new_function_gaze_crap_threshold: -5` returns validation error
- [x] T016 [US2] Add test `TestSC004_ConfigDefault_GazeCRAPThreshold` in `internal/config/config_test.go` — `DefaultConfig()` returns `NewFunctionGazeCRAPThreshold == 30`

### Implementation for US2

- [x] T017 [US2] Add validation for `NewFunctionGazeCRAPThreshold` in `Load()` in `internal/config/config.go` — must be > 0, same error format as `NewFunctionThreshold` validation (after line ~136)

**Checkpoint**: Configuration is independently configurable and validated. All T012-T016 tests pass.

---

## Phase 4: US3 — Backward Compatibility When GazeCRAP Unavailable (Priority: P3)

**Goal**: When `Score.GazeCRAP` is nil, only CRAP threshold is evaluated. Behavior is identical to pre-fix implementation.

**Independent Test**: Run comparison with new functions that have nil GazeCRAP. Verify only CRAP threshold determines violation status.

### Tests for US3

- [x] T018 [US3] Add test `TestSC003_NilGazeCRAP_CRAPBelowThreshold` in `internal/crap/compare_test.go` — new function with CRAP 25 and GazeCRAP nil is classified as `new` (not violation)
- [x] T019 [US3] Add test `TestSC003_NilGazeCRAP_CRAPAboveThreshold` in `internal/crap/compare_test.go` — new function with CRAP 35 and GazeCRAP nil is classified as `new_violation` (CRAP-only evaluation)

### Implementation for US3

No additional implementation needed — the nil guard added in T011 (`s.GazeCRAP != nil &&`) handles this case. These tests verify the backward-compatible behavior.

**Checkpoint**: Nil GazeCRAP is handled correctly. All T018-T019 tests pass. Behavior is identical to pre-fix for nil GazeCRAP.

---

## Phase 5: Report Output Updates

**Purpose**: Update text and JSON report output to display GazeCRAP for new-function violations (FR-006, FR-007).

### Tests

- [x] T020 [P] [US1] Add test `TestSC006_TextReport_ViolationShowsGazeCRAP` in `internal/crap/compare_report_test.go` — text output for a new-function violation includes both `CRAP: X.X` and `GazeCRAP: Y.Y` when GazeCRAP is available
- [x] T021 [P] [US1] Add test `TestSC006_TextReport_ViolationWithoutGazeCRAP` in `internal/crap/compare_report_test.go` — text output for a violation with nil GazeCRAP shows only `CRAP: X.X` (no GazeCRAP label)
- [x] T022 [P] [US1] Add test `TestSC005_JSONOutput_SummaryIncludesGazeCRAPThreshold` in `internal/crap/compare_report_test.go` — JSON `comparison` object includes `new_function_gaze_crap_threshold` field
- [x] T023 [P] [US1] Add test `TestSC005_JSONOutput_NewFunctionStatusUsesGazeCRAP` in `internal/crap/compare_report_test.go` — JSON new-function status is `new_violation` when GazeCRAP exceeds threshold (even if CRAP is below)

### Implementation

- [x] T024 [US1] Update `writeComparisonNewFunctions` in `internal/crap/compare_report.go` — add `gazeCRAPThreshold float64` parameter, use OR logic for violation classification, display GazeCRAP score in violation output when non-nil
- [x] T025 [US1] Update `WriteComparisonText` call site in `internal/crap/compare_report.go` (line ~186) to pass `result.Summary.NewFunctionGazeCRAPThreshold` to `writeComparisonNewFunctions`
- [x] T026 [US1] Update `WriteComparisonJSON` new-function status logic in `internal/crap/compare_report.go` (lines 117-121) — add GazeCRAP threshold check with nil guard, using `result.Summary.NewFunctionGazeCRAPThreshold`

**Checkpoint**: Report output shows GazeCRAP for violations. All T020-T023 tests pass. JSON summary includes the GazeCRAP threshold.

---

## Phase 6: Polish & Validation

**Purpose**: Final verification, documentation, and CI readiness.

- [x] T027 Run `go build ./cmd/gaze` — verify clean compilation
- [x] T028 Run `go test -race -count=1 -short ./internal/crap/...` — verify all comparison tests pass
- [x] T029 Run `go test -race -count=1 -short ./internal/config/...` — verify all config tests pass
- [x] T030 Run `go test -race -count=1 -short ./...` — verify no regressions across the full module
- [x] T031 Run `golangci-lint run` — verify no lint violations
- [x] T032 Verify GoDoc comments on all new/modified exported fields (`CompareOptions.NewFunctionGazeCRAPThreshold`, `ComparisonSummary.NewFunctionGazeCRAPThreshold`, `BaselineConfig.NewFunctionGazeCRAPThreshold`)
- [x] T033 Run quickstart.md validation — verify the testing commands from quickstart.md produce expected results

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Foundational)**: No dependencies — can start immediately. BLOCKS all subsequent phases.
- **Phase 2 (US1 — P1)**: Depends on Phase 1 completion. Core fix.
- **Phase 3 (US2 — P2)**: Depends on Phase 1 completion. Can run in parallel with Phase 2 (different files: config vs. crap).
- **Phase 4 (US3 — P3)**: Depends on Phase 2 completion (T011 provides the nil guard). Tests only — no new implementation.
- **Phase 5 (Report Output)**: Depends on Phase 2 completion (needs the OR logic pattern established).
- **Phase 6 (Polish)**: Depends on all previous phases.

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 1. No dependencies on other stories.
- **US2 (P2)**: Can start after Phase 1. Independent of US1 (config validation is separate from comparison logic).
- **US3 (P3)**: Depends on US1 (the nil guard in T011 is what makes backward compatibility work). Tests only.

### Within Each Phase

- T001, T002, T003 can run in parallel (different files)
- T004 depends on T003 (same file, same function)
- T005 depends on T001 (needs `CompareOptions` field to exist)
- T006 depends on T001, T002 (needs both `CompareOptions` and `ComparisonSummary` fields)
- T007-T010 should be written before T011 (test-first)
- T020-T023 can run in parallel (different test functions, same file — no conflicts)
- T024-T026 are sequential (same file, related functions)

### Parallel Opportunities

```
Phase 1 parallel group:
  T001 (compare.go)  |  T002 (crap.go)  |  T003 (config.go)

Phase 2+3 parallel (after Phase 1):
  US1: T007-T011 (compare.go, compare_test.go)
  US2: T014-T017 (config.go, config_test.go)

Phase 5 test parallel group:
  T020 | T021 | T022 | T023 (all in compare_report_test.go, different test functions)
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Foundational type changes
2. Complete Phase 2: US1 — GazeCRAP violation detection
3. **STOP and VALIDATE**: Run `go test -race -count=1 -short ./internal/crap/...`
4. The core bug (#164) is fixed at this point

### Full Delivery

1. Phase 1 → Foundation ready
2. Phase 2 (US1) + Phase 3 (US2) → in parallel if desired
3. Phase 4 (US3) → backward compatibility verified
4. Phase 5 → report output updated
5. Phase 6 → full validation, CI readiness

---

## Notes

- All tests use synthetic `Score` values and in-memory buffers — no external processes or filesystem access
- The `*float64` nil pattern for GazeCRAP is established in the codebase; follow the same `float64Ptr()` test helper
- Strict greater-than (`>`) comparison per D3 — a score exactly at threshold is NOT a violation
- Total production code: ~50 lines across 5 files
- Total test code: ~150 lines across 2 test files

<!-- spec-review: passed -->
<!-- code-review: passed -->
