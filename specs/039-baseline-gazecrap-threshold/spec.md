# Feature Specification: Baseline GazeCRAP New-Function Threshold

**Feature Branch**: `039-baseline-gazecrap-threshold`
**Created**: 2026-06-23
**Status**: Draft
**Input**: User description: "Baseline new-function threshold should evaluate GazeCRAP in addition to CRAP, with a separate configurable threshold (new_function_gaze_crap_threshold, default 30)"
**Issue**: #164

## User Scenarios & Testing

### User Story 1 - GazeCRAP Violation Detection for New Functions (Priority: P1)

A developer adds a new function to the codebase. The function has moderate complexity and decent line coverage, resulting in a CRAP score of 25 (below the new-function threshold of 30). However, its tests lack contract assertions, resulting in a GazeCRAP score of 40 (above the threshold). The baseline comparison should flag this as a violation and fail the gate, because the function carries high change risk when contract coverage is factored in.

**Why this priority**: This is the core bug fix described in issue #164. Without this, the baseline comparison gate has a blind spot where new functions with poor contract coverage pass unchallenged. This directly undermines the value of GazeCRAP as a risk metric.

**Independent Test**: Can be fully tested by running `gaze crap --baseline` against a baseline that does not contain the new function, where the new function has CRAP below threshold but GazeCRAP above threshold. The comparison should report a `new_violation` and exit non-zero.

**Acceptance Scenarios**:

1. **Given** a baseline with no entry for function `ProcessOrder`, **When** current results show `ProcessOrder` with CRAP 25 and GazeCRAP 40 (threshold 30 for both), **Then** the comparison classifies `ProcessOrder` as `new_violation` and `Summary.Passed` is false.
2. **Given** a baseline with no entry for function `SimpleHelper`, **When** current results show `SimpleHelper` with CRAP 5 and GazeCRAP 8 (both below threshold 30), **Then** the comparison classifies `SimpleHelper` as `new` and `Summary.Passed` is true (assuming no other violations).
3. **Given** a baseline with no entry for function `ComplexUntested`, **When** current results show `ComplexUntested` with CRAP 42 and GazeCRAP 55 (both above threshold 30), **Then** the comparison classifies `ComplexUntested` as `new_violation` and `Summary.Passed` is false.

---

### User Story 2 - Independent GazeCRAP Threshold Configuration (Priority: P2)

A team uses `.gaze.yaml` to configure baseline comparison thresholds. Because GazeCRAP scores tend to be higher than classic CRAP scores (they incorporate contract coverage gaps), the team wants to set a higher tolerance for GazeCRAP on new functions (e.g., 40) while keeping the classic CRAP threshold at 30. The `new_function_gaze_crap_threshold` configuration field allows this independent calibration.

**Why this priority**: This provides flexibility for teams whose GazeCRAP scores naturally run higher than their CRAP scores. Without a separate threshold, teams would either need to raise the CRAP threshold (weakening the CRAP gate) or accept false violations from GazeCRAP.

**Independent Test**: Can be tested by creating a `.gaze.yaml` with `new_function_gaze_crap_threshold: 40`, adding a new function with CRAP 25 and GazeCRAP 38, and verifying it passes (GazeCRAP 38 < threshold 40). Then changing GazeCRAP to 42 and verifying it fails.

**Acceptance Scenarios**:

1. **Given** `.gaze.yaml` with `baseline.new_function_gaze_crap_threshold: 40`, **When** a new function has CRAP 25 and GazeCRAP 38, **Then** the function is classified as `new` (not a violation) because both scores are below their respective thresholds.
2. **Given** `.gaze.yaml` with `baseline.new_function_gaze_crap_threshold: 40`, **When** a new function has CRAP 25 and GazeCRAP 42, **Then** the function is classified as `new_violation` because GazeCRAP exceeds its threshold.
3. **Given** `.gaze.yaml` with no `new_function_gaze_crap_threshold` field, **When** baseline comparison runs, **Then** the default threshold of 30 is used for GazeCRAP evaluation.

---

### User Story 3 - Backward-Compatible Behavior When GazeCRAP Is Unavailable (Priority: P3)

A developer runs baseline comparison against a codebase where GazeCRAP data is not available (e.g., contract coverage analysis was not run, or the function has nil GazeCRAP). The new-function threshold check should evaluate only classic CRAP, maintaining identical behavior to the current implementation.

**Why this priority**: Backward compatibility is essential. Users who do not use GazeCRAP should see no behavior change. This is also the path taken when comparing against baselines created by older gaze versions that predate GazeCRAP.

**Independent Test**: Can be tested by running baseline comparison where new functions have nil GazeCRAP values. Only the CRAP threshold should be evaluated.

**Acceptance Scenarios**:

1. **Given** a new function with CRAP 25 and GazeCRAP nil, **When** baseline comparison runs with new-function CRAP threshold 30, **Then** the function is classified as `new` (CRAP 25 < 30, GazeCRAP skipped).
2. **Given** a new function with CRAP 35 and GazeCRAP nil, **When** baseline comparison runs with new-function CRAP threshold 30, **Then** the function is classified as `new_violation` (CRAP 35 > 30, GazeCRAP skipped).

---

### Edge Cases

- What happens when GazeCRAP is exactly equal to the threshold? Following the existing CRAP comparison pattern (`s.CRAP > opts.NewFunctionThreshold`), GazeCRAP at exactly the threshold value should not be a violation (strict greater-than comparison).
- What happens when GazeCRAP is zero? A GazeCRAP of 0 is a valid score (fully covered simple function). It should be evaluated normally against the threshold (0 < 30, no violation).
- What happens when the CRAP threshold and GazeCRAP threshold are set to different values in `.gaze.yaml`? Each metric is evaluated independently against its own threshold. A violation in either triggers `new_violation`.
- What happens when `new_function_gaze_crap_threshold` is set to 0 in `.gaze.yaml`? Configuration validation should reject values <= 0, consistent with the existing `new_function_threshold` validation.

## Requirements

### Functional Requirements

- **FR-001**: The new-function threshold check MUST evaluate both CRAP and GazeCRAP (when GazeCRAP is available). A new function MUST be classified as `new_violation` if either its CRAP score exceeds the CRAP threshold or its GazeCRAP score exceeds the GazeCRAP threshold.
- **FR-002**: A `new_function_gaze_crap_threshold` configuration field MUST be added to the baseline configuration, with a default value of 30.
- **FR-003**: When GazeCRAP is unavailable (nil) for a new function, only the CRAP threshold MUST be evaluated. The absence of GazeCRAP data MUST NOT trigger a violation.
- **FR-004**: The GazeCRAP threshold MUST be independently configurable from the CRAP threshold, allowing teams to set different tolerance levels for each metric.
- **FR-005**: Configuration validation MUST reject `new_function_gaze_crap_threshold` values that are zero or negative, consistent with existing `new_function_threshold` validation.
- **FR-006**: The comparison summary output MUST include the GazeCRAP threshold value alongside the existing CRAP threshold for reproducibility.
- **FR-007**: The text and JSON report output for new-function violations MUST display both CRAP and GazeCRAP scores when GazeCRAP is available, so users can see which metric triggered the violation.

### Key Entities

- **CompareOptions**: Extended with a GazeCRAP threshold field, carrying the threshold from configuration to the comparison algorithm.
- **ComparisonSummary**: Extended with the GazeCRAP threshold value for output reproducibility.
- **BaselineConfig**: Extended with the new configuration field and its default/validation rules.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A new function with CRAP below its threshold but GazeCRAP above its threshold is classified as `new_violation` and the comparison gate fails.
- **SC-002**: A new function with both CRAP and GazeCRAP below their respective thresholds is classified as `new` and does not cause the gate to fail.
- **SC-003**: When GazeCRAP is unavailable (nil) for a new function, the comparison behavior is identical to the current implementation (only CRAP evaluated).
- **SC-004**: The CRAP and GazeCRAP new-function thresholds can be configured independently via `.gaze.yaml`, each with its own numeric value.
- **SC-005**: The comparison summary JSON output includes the GazeCRAP threshold value, enabling consumers to understand which thresholds were applied.
- **SC-006**: The text report displays both CRAP and GazeCRAP scores for new-function violations when GazeCRAP data is available.
- **SC-007**: Invalid configuration values for `new_function_gaze_crap_threshold` (zero or negative) produce a clear validation error at configuration load time.

### Assumptions

- The default value of 30 for `new_function_gaze_crap_threshold` matches the existing `new_function_threshold` default. This is a conservative starting point; teams can adjust independently based on their GazeCRAP score distributions.
- The strict greater-than comparison (`>`) is used for the GazeCRAP threshold, consistent with the existing CRAP threshold behavior.
- The `classifyDelta` function (regression detection for existing functions) already evaluates both CRAP and GazeCRAP symmetrically. This spec brings the new-function threshold check into alignment with that established pattern.

### Dependencies

- **baseline-comparison** (prior work): This spec extends the baseline comparison feature. The `Compare`, `CompareOptions`, `ComparisonSummary`, and `buildComparisonSummary` types/functions from that work are the direct targets of modification.
- **Issue #164**: This spec directly addresses the bug described in the issue.
