## ADDED Requirements

### Requirement: CompactForAI method

`ReportPayload` MUST provide a `CompactForAI() ([]byte, error)` method that produces a reduced JSON representation suitable for AI adapter consumption. The compact output MUST be valid JSON and MUST contain the following top-level fields: `summary`, `crap`, `quality`, `classify`, `docscan`, `errors`.

#### Scenario: Compact output is valid JSON
- **GIVEN** a fully populated `ReportPayload` with all four steps succeeded
- **WHEN** `CompactForAI()` is called
- **THEN** the returned bytes MUST be valid JSON that unmarshals without error

#### Scenario: Compact output includes summary
- **GIVEN** a `ReportPayload` with `Summary.CRAPload=12`, `Summary.GazeCRAPload=3`, `Summary.AvgContractCoverage=45`
- **WHEN** `CompactForAI()` is called
- **THEN** the output MUST contain a top-level `"summary"` object with `"crapload": 12`, `"gaze_crapload": 3`, `"avg_contract_coverage": 45`

#### Scenario: Compact output respects step failures
- **GIVEN** a `ReportPayload` where the Quality step failed (Quality is nil, Errors.Quality is non-nil)
- **WHEN** `CompactForAI()` is called
- **THEN** the `"quality"` field MUST be `null` and `"errors"` MUST contain the quality error message

#### Scenario: All steps failed
- **GIVEN** a `ReportPayload` with all four step fields nil and all four `Errors` fields populated
- **WHEN** `CompactForAI()` is called
- **THEN** the output MUST be valid JSON with all step fields as `null` and the `"errors"` object containing all four error messages

#### Scenario: Step succeeds with empty data
- **GIVEN** a `ReportPayload` where Docscan succeeded but returned an empty array (`[]`)
- **WHEN** `CompactForAI()` is called
- **THEN** the `"docscan"` field MUST be `[]` (empty array), not `null`

### Requirement: Docscan content exclusion

The compact payload MUST strip the `content` field from docscan entries. Each docscan entry MUST retain only `path` and `priority` fields.

#### Scenario: Docscan paths without content
- **GIVEN** a `ReportPayload` with docscan containing 3 documents each with `path`, `content`, and `priority`
- **WHEN** `CompactForAI()` is called
- **THEN** the `"docscan"` array MUST contain 3 entries, each with `"path"` and `"priority"` only, and no `"content"` field

#### Scenario: Docscan step failure preserved
- **GIVEN** a `ReportPayload` where Docscan is nil
- **WHEN** `CompactForAI()` is called
- **THEN** the `"docscan"` field MUST be `null`

### Requirement: Quality data compaction

The compact payload MUST replace full `SideEffect` objects in Quality report fields with arrays of side effect ID strings. Specifically:

- `contract_coverage.gaps` (nested under `ContractCoverage`) MUST be replaced with `contract_coverage.gap_ids`
- `contract_coverage.discarded_returns` (nested under `ContractCoverage`) MUST be replaced with `contract_coverage.discarded_return_ids`
- `ambiguous_effects` (top-level on `QualityReport`) MUST be replaced with `ambiguous_effect_ids`

The `contract_coverage.gap_hints` and `contract_coverage.discarded_return_hints` fields MUST be preserved as-is. All scalar fields on `ContractCoverage` and `QualityReport` MUST be preserved.

#### Scenario: Quality gaps reduced to IDs
- **GIVEN** a Quality report with a test-target pair whose `contract_coverage.gaps` contains 3 full `SideEffect` objects with `id` fields `"se-a1b2c3d4"`, `"se-b2c3d4e5"`, `"se-c3d4e5f6"`
- **WHEN** `CompactForAI()` is called
- **THEN** the corresponding quality entry's `contract_coverage` MUST contain `"gap_ids": ["se-a1b2c3d4", "se-b2c3d4e5", "se-c3d4e5f6"]` and MUST NOT contain a `"gaps"` field

#### Scenario: Quality discarded returns reduced to IDs
- **GIVEN** a Quality report with a test-target pair whose `contract_coverage.discarded_returns` contains 2 full `SideEffect` objects with `id` fields `"se-d4e5f6a7"`, `"se-e5f6a7b8"`
- **WHEN** `CompactForAI()` is called
- **THEN** the corresponding quality entry's `contract_coverage` MUST contain `"discarded_return_ids": ["se-d4e5f6a7", "se-e5f6a7b8"]` and MUST NOT contain a `"discarded_returns"` field

#### Scenario: Quality ambiguous effects reduced to IDs
- **GIVEN** a Quality report with a test-target pair whose `ambiguous_effects` contains 2 full `SideEffect` objects with `id` fields `"se-f6a7b8c9"`, `"se-a7b8c9d0"`
- **WHEN** `CompactForAI()` is called
- **THEN** the corresponding quality entry MUST contain `"ambiguous_effect_ids": ["se-f6a7b8c9", "se-a7b8c9d0"]` and MUST NOT contain an `"ambiguous_effects"` field

#### Scenario: Quality hints preserved
- **GIVEN** a Quality report with `contract_coverage.gap_hints: ["assert err != nil", "check return value"]`
- **WHEN** `CompactForAI()` is called
- **THEN** the `contract_coverage.gap_hints` field MUST be `["assert err != nil", "check return value"]`

#### Scenario: Cross-reference resilience when Classify fails
- **GIVEN** a `ReportPayload` where the Classify step failed (Classify is nil) but Quality has gaps with side effect IDs
- **WHEN** `CompactForAI()` is called
- **THEN** the quality `gap_ids` MUST still be emitted (the reporter can degrade gracefully without cross-referencing the Classify output)

### Requirement: Signal stripping in Classify output

The compact payload MUST omit `Classification.Signals` arrays from all side effects in the Classify output. `Classification.Label`, `Classification.Confidence`, and `Classification.Reasoning` MUST be preserved.

#### Scenario: Signals omitted, label and reasoning preserved
- **GIVEN** a Classify result with a side effect whose Classification has `label: "contractual"`, `confidence: 85`, `reasoning: "interface method with documented contract"`, `signals: [{source: "interface", weight: 15, ...}]`
- **WHEN** `CompactForAI()` is called
- **THEN** the side effect's classification MUST have `"label": "contractual"`, `"confidence": 85`, `"reasoning": "interface method with documented contract"`, and no `"signals"` field

### Requirement: CRAP summary deduplication

The compact payload MUST omit `summary.worst_crap`, `summary.worst_gaze_crap`, and `summary.recommended_actions` from the CRAP output. The `scores` array and all other summary fields (`total_functions`, `avg_complexity`, `avg_line_coverage`, `avg_crap`, `crapload`, `crap_threshold`, `gaze_crapload`, `gaze_crap_threshold`, `avg_gaze_crap`, `avg_contract_coverage`, `quadrant_counts`, `fix_strategy_counts`) MUST be preserved.

#### Scenario: Worst offender lists omitted
- **GIVEN** a CRAP report with `summary.worst_crap` containing 5 entries and `scores` containing 150 entries
- **WHEN** `CompactForAI()` is called
- **THEN** the CRAP output MUST contain `"scores"` with 150 entries and `"summary"` without `"worst_crap"`, `"worst_gaze_crap"`, or `"recommended_actions"` fields

### Requirement: Quality summary deduplication

The compact payload MUST omit `quality_summary.worst_coverage_tests` from the Quality output. All other summary fields (`total_tests`, `average_contract_coverage`, `total_over_specifications`, `assertion_detection_confidence`, `ssa_degraded`, `ssa_degraded_packages`) MUST be preserved.

#### Scenario: Worst coverage tests omitted
- **GIVEN** a Quality output with `quality_summary.worst_coverage_tests` containing 5 entries
- **WHEN** `CompactForAI()` is called
- **THEN** the `"quality_summary"` MUST NOT contain `"worst_coverage_tests"` and MUST contain `"total_tests"`, `"average_contract_coverage"`, `"ssa_degraded"`

### Requirement: Compact payload size budget

The compact payload MUST produce output under 300KB for projects with up to 500 analyzed functions. The 300KB threshold provides ~4x margin below the 1M token context window (~250KB per 1M tokens at ~4 chars/token), ensuring room for the system prompt and model response. A unit test MUST verify that a representative synthetic payload compacts to under this threshold.

Synthetic fixture parameters for the size budget test: each doc file has 15KB content (stripped in compact), each function has 3-5 side effects, each side effect has 2-4 signals with 100-char excerpts, each test-target pair has 2 gaps and 1 discarded return.

#### Scenario: Size budget met for medium project
- **GIVEN** a synthetic `ReportPayload` representing 200 functions, 100 test-target pairs, and 30 documentation files with the fixture parameters above
- **WHEN** `CompactForAI()` is called
- **THEN** the output size MUST be less than 300KB

## MODIFIED Requirements

### Requirement: AI adapter payload delivery

Previously: `runTextPath()` calls `json.Marshal(payload)` and passes the full result to the AI adapter.

`runTextPath()` MUST call `payload.CompactForAI()` instead of `json.Marshal(payload)` when delivering the payload to the AI adapter. The `--format=json` path (`runJSONPath()`) MUST remain unchanged, continuing to use `json.Encoder` with full-fidelity output.

#### Scenario: Text path uses compact payload
- **GIVEN** a report run with `--ai=opencode --format=text`
- **WHEN** the pipeline completes and invokes the AI adapter
- **THEN** the adapter MUST receive the compact payload (with docscan content stripped, signals omitted, etc.) not the full `json.Marshal` output

#### Scenario: JSON path unchanged
- **GIVEN** a report run with `--format=json`
- **WHEN** the pipeline completes
- **THEN** stdout MUST contain the full pretty-printed JSON with all fields including docscan content, signals, worst offender lists, and full side effect objects

## REMOVED Requirements

None.
