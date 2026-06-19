## Context

`config.Load` (`internal/config/config.go:112-149`) unmarshals `.gaze.yaml` and validates baseline config (epsilon, new_function_threshold) and doc_scan timeout, but performs zero validation on classification thresholds. The `[1, 99]` range check and `contractual > incidental` coherence check live exclusively in the CLI layer's `loadConfig` wrapper (`cmd/gaze/main.go:158-196`).

This creates two gaps:
1. Config-file values bypass range checking (only flag-sourced values are range-checked).
2. Non-CLI callers (`loadGazeConfigBestEffort` in `runner_steps.go` and `contract.go`) bypass all threshold validation entirely.

The fix follows the "validate at the boundary" principle: move threshold validation into `config.Load`, the single function through which all config enters the system.

## Goals / Non-Goals

### Goals
- Validate `contractual` and `incidental` thresholds in `config.Load` so all callers get validated config
- Enforce `1 <= incidental < contractual <= 99` (the same invariant the CLI path enforces)
- Return descriptive errors identifying the invalid field, received value, and valid range
- Add comprehensive test coverage for all validation branches

### Non-Goals
- Changing the valid range boundaries (these remain `[1, 99]` as established)
- Modifying CLI flag validation in `loadConfig` (it stays for flag-specific error messages)
- Adding validation for other config fields beyond classification thresholds (out of scope)
- Changing the `loadGazeConfigBestEffort` fallback behavior (it already returns `DefaultConfig()` on error)

## Decisions

### D1: Validate in `config.Load`, not in a separate `Validate` method

**Decision**: Add validation directly inside `Load()` after YAML unmarshaling, alongside existing baseline and timeout validation.

**Alternatives considered**:
- Separate `Validate(*GazeConfig) error` function: More composable, but adds an extra call that callers can forget. The existing pattern (validate-in-Load) is already established for baseline and timeout fields.
- Validate only in `loadConfig` CLI wrapper: Status quo. Leaves non-CLI paths unprotected.

**Rationale**: Consistency with existing validation pattern in `Load()`. Every caller of `Load` gets validated output without remembering to call a separate function. Aligns with Constitution Principle II (Minimal Assumptions) -- `config.Load` enforces the same constraints already documented in the CLI path, making them explicit at the config boundary.

### D2: Keep CLI `loadConfig` validation intact

**Decision**: Do not remove the range checks or coherence check from `cmd/gaze/main.go:loadConfig`.

**Rationale**: The CLI-layer validation provides better error messages that reference flag names (`--contractual-threshold=500 is invalid`). The `config.Load` errors reference YAML field paths (`classification.thresholds.contractual must be in [1, 99]`). Both are valuable in their respective contexts. The CLI checks also catch invalid states created by merging valid config with invalid flag overrides, which `config.Load` cannot catch (it only sees the file).

### D3: Error message format

**Decision**: Use the pattern `classification.thresholds.<field> must be in [1, 99], got <value>` for range errors and `classification.thresholds.contractual (<value>) must be greater than incidental (<value>)` for coherence errors.

**Rationale**: Matches the existing error format used for baseline validation (`baseline.epsilon must be >= 0, got %g`). Uses YAML field paths so users can locate the problem in their config file.

## Coverage Strategy

Unit tests only. All new validation branches in `config.Load` must achieve 100% branch coverage. The test cases cover: contractual range (above, zero, negative), incidental range (above, zero, negative), coherence (inverted, equal), and valid boundaries (extreme, adjacent) -- covering all 3 validation checks in both pass and fail paths. No integration or e2e tests needed -- `config.Load` is a pure function with file I/O as its only external dependency, tested via isolated YAML fixture files.

## Error Path Change

After this change, for YAML-sourced invalid thresholds, `config.Load` returns the error before `loadConfig`'s own coherence check runs (line 154-156 of `cmd/gaze/main.go`). The error message format changes:

- **Before**: `contractual threshold (50) must be greater than incidental threshold (60); check config file /path/.gaze.yaml` (from `loadConfig`)
- **After**: `classification.thresholds.contractual (50) must be greater than incidental (60)` (from `config.Load`)

The new format uses YAML field paths and is consistent with other `config.Load` errors. The existing CLI test `TestLoadConfig_YAMLInvertedThresholdsRejected` asserts on `"config file"` in the error and must be updated to match the new format.

The CLI-layer coherence check in `loadConfig` (lines 176-196) still fires when CLI flag overrides create an invalid combination with a valid config file -- that path is unchanged.

## Partial Threshold Specification

When only one threshold is set in `.gaze.yaml` (e.g., `incidental: 85` with no `contractual` key), the YAML unmarshaler writes only the specified field, and the other field retains its default from `DefaultConfig()` (contractual=80, incidental=50). Validation runs on the merged result. This means a partial config like `incidental: 85` will fail coherence (80 <= 85) even though the user only set one field.

This is correct behavior -- the merged config would produce degenerate classification regardless of whether the user explicitly set both values. The error message names the fields and values, making it clear what needs fixing. No special handling for "defaulted" vs. "explicit" values is needed because Go's `int` zero-value (0) and YAML's explicit `0` are indistinguishable after parsing, and 0 is outside `[1, 99]` regardless of intent.

## Risks / Trade-offs

### R1: Breaking change for users with invalid configs (Low risk)

Users who have `.gaze.yaml` files with out-of-range thresholds will get errors where they previously got silent degradation. This is intentional -- the previous behavior was a bug that produced incorrect results. The error messages tell users exactly what to fix.

### R2: Redundant validation in CLI path (Accepted trade-off)

After this change, thresholds from `.gaze.yaml` are validated twice when accessed via CLI commands: once in `config.Load` and again in `loadConfig`. This is acceptable because:
- The validations are cheap (integer comparisons)
- They serve different purposes (generic config error vs. flag-specific error)
- Removing CLI-layer validation would regress error message quality for flag users

### R3: `loadGazeConfigBestEffort` silently swallows validation errors (Existing behavior, unchanged)

When `config.Load` returns an error, `loadGazeConfigBestEffort` falls back to `DefaultConfig()` without surfacing the error to the user. This means an invalid `.gaze.yaml` in the `gaze report` or `gaze crap` path produces default thresholds instead of an error. This is the existing fallback pattern and is acceptable for "best effort" callers, but could be improved in a future change by logging a warning.
