## Why

`config.Load` (`internal/config/config.go:112-149`) parses `.gaze.yaml` but performs no validation on classification thresholds (`contractual`, `incidental`). Out-of-range values (e.g., 500, -10, 0), negative values, and inverted pairs (`contractual < incidental`) all load without error.

The `[1, 99]` range check and `contractual > incidental` coherence check exist only in the CLI layer's `loadConfig` wrapper (`cmd/gaze/main.go:158-196`). This creates an asymmetry: the same invalid value supplied via a CLI flag is rejected with a clear error, but supplied via the config file it is accepted silently and degrades classification output.

Additionally, non-CLI callers (`loadGazeConfigBestEffort` in `internal/aireport/runner_steps.go` and `internal/crap/contract.go`) bypass `loadConfig` entirely, calling `config.Load` directly with zero threshold validation.

Reported in [#114](https://github.com/unbound-force/gaze/issues/114). Confirmed and reproduced: `contractual: 500` in `.gaze.yaml` causes all effects to be classified as incidental regardless of confidence score.

## What Changes

Move classification threshold validation into `config.Load()` -- the single boundary where config enters the system. This ensures all callers (CLI `loadConfig`, `loadGazeConfigBestEffort`, and any future consumers) receive validated configuration.

## Capabilities

### New Capabilities
- `config.Load threshold validation`: `config.Load` now validates that `contractual` and `incidental` thresholds are within `[1, 99]` and that `contractual > incidental`. Invalid `.gaze.yaml` files produce descriptive errors instead of silently loading.

### Modified Capabilities
- `config.Load`: Adds three validation checks after YAML unmarshaling (range check for each threshold, coherence check for the pair).

### Removed Capabilities
- None.

## Impact

- **`internal/config/config.go`**: Add threshold validation to `Load()` (~10 lines).
- **`internal/config/config_test.go`**: Add test cases for out-of-range, inverted, equal, zero, and boundary-valid thresholds (~6 new tests).
- **`internal/config/testdata/`**: Add YAML fixture files for new test cases (~6 files).
- **`cmd/gaze/main.go`**: No production code changes needed. The existing CLI-layer validation in `loadConfig` remains for flag-specific error messages (e.g., mentioning `--contractual-threshold`). The post-merge coherence check also remains because CLI overrides can invalidate a config that was valid on disk.
- **`cmd/gaze/main_test.go`**: `TestLoadConfig_YAMLInvertedThresholdsRejected` must update its assertion — the error now comes from `config.Load` (field-path format) instead of `loadConfig` (source-attribution format), so the `"config file"` substring check changes to `"classification.thresholds.contractual"`.
- **`loadGazeConfigBestEffort` callers**: No code changes needed. These already fall back to `DefaultConfig()` on `Load` errors, so invalid configs now trigger safe fallback instead of silently degrading.
- **No API surface changes**: `Load` signature is unchanged; it just returns errors for previously-accepted invalid inputs.

## Constitution Alignment

Assessed against the Gaze project constitution (v1.3.0).

### I. Accuracy

**Assessment**: PASS

This change directly supports accuracy by preventing degenerate classification thresholds from silently corrupting output. An unreachable `contractual: 500` threshold forces every effect to be classified as incidental regardless of actual confidence, producing systematically inaccurate results. Validating at the config boundary eliminates this failure mode.

### II. Minimal Assumptions

**Assessment**: PASS

The fix enforces the same explicit constraints (`[1, 99]` range, `contractual > incidental`) that already exist in the CLI path. No new assumptions are introduced. Invalid inputs that were silently accepted are now explicitly rejected with descriptive errors, making the system's constraints more transparent to users.

### III. Actionable Output

**Assessment**: PASS

The validation errors are specific and actionable: they name the invalid field, show the value received, and state the valid range. Users get immediate feedback at config load time rather than discovering degraded output after a full analysis run.

### IV. Testability

**Assessment**: PASS

All validation logic is in `config.Load`, which is independently testable with YAML fixture files. No external services or shared state required. Each new validation rule has a corresponding test case with a dedicated fixture file.
