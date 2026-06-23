## Why

The baseline comparison text report (`writeComparisonDeltaTable` in `internal/crap/compare_report.go`) only displays CRAP values in its delta table. When a function regresses due to GazeCRAP (but not CRAP), the user sees it marked as "regression" without understanding why — the GazeCRAP numbers that triggered the classification are invisible.

The JSON format already includes all fields (`baseline_gaze_crap`, `gaze_crap_delta`), so this is a display-only gap in the human-readable text report.

Fixes #163.

## What Changes

Update `writeComparisonDeltaTable` to display GazeCRAP data using a two-row-per-function format. When GazeCRAP deltas are available for any function in the table, each function gets two rows: a GazeCRAP row first, then the CRAP row second. This keeps the output within the 80-column terminal constraint while making both metrics visible.

Before:
```
Regressions:
  Function                                  Baseline    Current     Delta
  doSomething (internal/crap/c...)          12.0        12.0        +0.0
```

After (when GazeCRAP data is available):
```
Regressions:
  Function                                  Baseline    Current     Delta
  doSomething (internal/crap/c...)
    GazeCRAP                                8.5         12.1        +3.6
    CRAP                                    12.0        12.0        +0.0
```

When no function in the table has GazeCRAP data, the current single-row format is preserved unchanged.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `writeComparisonDeltaTable`: Displays GazeCRAP deltas alongside CRAP deltas in a two-row-per-function format when GazeCRAP data is available

### Removed Capabilities
- None

## Impact

- **`internal/crap/compare_report.go`**: `writeComparisonDeltaTable` updated with two-row format and GazeCRAP rendering
- **`internal/crap/compare_report_test.go`**: Tests updated/added for GazeCRAP delta display
- Text output width stays within 80-column constraint
- JSON output is unchanged — this is a text-only fix
- No new types, config fields, or CLI flags

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: N/A

This is a display-only change to a text formatter. No artifact interfaces, inter-hero communication, or self-describing outputs are affected.

### II. Composability First

**Assessment**: N/A

No new dependencies or mandatory couplings are introduced. The change is confined to one internal function in one file.

### III. Observable Quality

**Assessment**: PASS

The change improves observable quality by surfacing GazeCRAP delta data that was already computed and available in the JSON format but hidden from the human-readable text report. Users can now see which metric triggered a regression or improvement classification.

### IV. Testability

**Assessment**: PASS

`writeComparisonDeltaTable` is a pure function that writes to `io.Writer` with synthetic `FunctionDelta` inputs — fully testable in isolation without external services. All new behavior will be covered by unit tests using in-memory buffers.
