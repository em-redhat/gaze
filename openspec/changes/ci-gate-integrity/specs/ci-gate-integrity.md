## ADDED Requirements

### Requirement: Partial Coverage Profile Tolerance

`generateCoverProfile` MUST preserve the coverage profile when `go test` exits
non-zero, provided the profile file exists and has non-zero size. A warning
MUST be emitted to stderr indicating that partial coverage is being used.
`generateCoverProfile` MUST return a hard error only when no usable profile
was produced (file missing or empty).

#### Scenario: go test exits non-zero with usable profile

- **GIVEN** a module with 10 packages, 1 of which fails to build
- **WHEN** `generateCoverProfile` runs `go test -coverprofile`
- **THEN** the coverage profile for the 9 passing packages is preserved
- **AND** a warning is emitted to stderr containing the `go test` exit error
- **AND** the function returns the profile path (not an error)

#### Scenario: go test exits non-zero with no profile

- **GIVEN** a module where `go test` fails before writing any coverage data
- **WHEN** `generateCoverProfile` runs `go test -coverprofile`
- **THEN** the function returns an error indicating no coverage was produced
- **AND** any empty profile file is cleaned up

#### Scenario: go test exits non-zero with empty profile

- **GIVEN** a module where `go test` creates the profile file but writes no data
- **WHEN** `generateCoverProfile` runs `go test -coverprofile`
- **THEN** the function returns an error indicating no coverage was produced
- **AND** the empty profile file is cleaned up

---

### Requirement: GazeCRAPload Unavailability Signal

`ReportSummary.GazeCRAPload` and `crapStepResult.GazeCRAPload` MUST be modeled
as `*int` to distinguish "zero violations" (`*0`) from "not computed" (`nil`).
The nil signal from `crap.Summary.GazeCRAPload` MUST be preserved through the
entire pipeline without lossy conversion. This includes
`compactSummary.GazeCRAPload` in `compact.go` which is in the JSON
serialization path for the AI adapter.

`ThresholdResult.Actual` MUST be modeled as `*int` so that threshold results
can represent "metric unavailable" (`nil`) distinctly from "zero violations"
(`*0`). Formatting logic MUST print "N/A" when `Actual` is nil.

#### Scenario: GazeCRAP data available with zero violations

- **GIVEN** a module where all functions have GazeCRAP scores below threshold
- **WHEN** `gaze report --format=json` is run
- **THEN** `summary.gaze_crapload` in the JSON output is `0` (integer)
- **AND** threshold evaluation treats this as "zero violations" (PASS if limit >= 0)

#### Scenario: GazeCRAP data unavailable

- **GIVEN** a module without contract coverage data
- **WHEN** `gaze report --format=json` is run
- **THEN** `summary.gaze_crapload` in the JSON output is `null`
- **AND** threshold evaluation treats this as "data unavailable" (not PASS)

---

### Requirement: GazeCRAPload Threshold Failure on Unavailable Data

When `--max-gaze-crapload` is set but GazeCRAP data is unavailable,
`EvaluateThresholds` MUST report a threshold failure. The threshold result
MUST indicate that the metric was unavailable, not that it was zero.

#### Scenario: Threshold set, GazeCRAP unavailable

- **GIVEN** `--max-gaze-crapload=5` is set
- **WHEN** GazeCRAP data is unavailable (GazeCRAPload is nil)
- **THEN** the threshold check reports FAIL
- **AND** the result name or message indicates "GazeCRAP unavailable"
- **AND** the command exits non-zero

#### Scenario: Threshold set, GazeCRAP available and within limit

- **GIVEN** `--max-gaze-crapload=5` is set
- **WHEN** GazeCRAPload is 3
- **THEN** the threshold check reports PASS (3 <= 5)

#### Scenario: Threshold set, GazeCRAP available and exceeds limit

- **GIVEN** `--max-gaze-crapload=5` is set
- **WHEN** GazeCRAPload is 7
- **THEN** the threshold check reports FAIL (7 > 5)
- **AND** the command exits non-zero

#### Scenario: Threshold not set, GazeCRAP available

- **GIVEN** no `--max-gaze-crapload` flag is provided
- **WHEN** GazeCRAPload is 3
- **THEN** no GazeCRAPload threshold result is emitted (threshold skipped)

#### Scenario: Threshold not set, GazeCRAP unavailable

- **GIVEN** no `--max-gaze-crapload` flag is provided
- **WHEN** GazeCRAP data is unavailable
- **THEN** no GazeCRAPload threshold result is emitted (threshold skipped)

---

### Requirement: Zero-Result Gate Failure

When a gate command produces zero analyzable functions and threshold flags
were explicitly provided, it MUST return a non-zero exit code. The error
message MUST suggest checking the package patterns. The applicable threshold
flags per command are:

- `gaze crap`: `--max-crapload`, `--max-gaze-crapload`
- `gaze report`: `--max-crapload`, `--max-gaze-crapload`, `--min-contract-coverage`

Threshold flag detection MUST use `cmd.Flags().Changed()` semantics (was the
flag explicitly provided?) rather than `value > 0` (which treats `--max-crapload=0`
as "disabled").

#### Scenario: Gate command with zero functions

- **GIVEN** `gaze crap --max-crapload=5 ./empty/pkg` is run (package exists but
  has no analyzable functions)
- **WHEN** zero functions are found to analyze
- **THEN** the command exits non-zero
- **AND** the error message contains "no functions analyzed"
- **AND** the error message suggests checking package patterns

#### Scenario: Non-gate command with zero functions

- **GIVEN** `gaze crap ./empty/pkg` is run (no threshold flags)
- **WHEN** zero functions are found to analyze
- **THEN** the command logs a warning
- **AND** the command exits 0 (existing behavior preserved)

#### Scenario: Report pipeline with zero results and thresholds

- **GIVEN** `gaze report --max-crapload=5 ./empty/pkg` is run
- **WHEN** the CRAP step produces zero scores
- **THEN** the threshold evaluation fails
- **AND** the command exits non-zero

---

## MODIFIED Requirements

### Requirement: gaze crap and gaze report Consistency

Previously: `gaze crap` and `gaze report` produced different results for the
same input when GazeCRAP was unavailable — `gaze crap` printed "GazeCRAP
unavailable" while `gaze report` silently passed with value 0.

`gaze crap` and `gaze report` MUST produce consistent pass/fail semantics
for the same input. When GazeCRAP data is unavailable:
- Both commands MUST indicate unavailability (not silently use zero).
- Both commands MUST fail a `--max-gaze-crapload` threshold check.

---

## REMOVED Requirements

None.
