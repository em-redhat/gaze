## Context

The baseline comparison text report (`writeComparisonDeltaTable` in `internal/crap/compare_report.go`) renders a table of function deltas for regressions and improvements. Currently it only shows CRAP values (baseline, current, delta). The `FunctionDelta` struct already carries `GazeCRAPDelta *float64` and the baseline/current `Score` structs carry `GazeCRAP *float64`, but these are not rendered in the text output.

The 80-column terminal width constraint (AGENTS.md) prevents simply adding three more columns to the existing table — the current header is already 78 chars wide.

## Goals / Non-Goals

### Goals
- Display GazeCRAP baseline, current, and delta values in the text comparison table when GazeCRAP data is available
- Stay within the 80-column terminal width constraint
- Preserve the existing single-row format when no GazeCRAP data is available (backward compatible output)

### Non-Goals
- Modifying JSON output (already includes all GazeCRAP fields)
- Adding new types or config fields
- Changing regression/improvement classification logic (already correct)
- Changing the new-function or removed-function sections (separate concern)

## Decisions

### D1: Two-row-per-function format with GazeCRAP first

When any matching delta in the table has GazeCRAP data, switch to a two-row format where the function name appears on its own line, followed by indented metric rows. GazeCRAP is rendered first, CRAP second.

```
Regressions:
  Function                                  Baseline    Current     Delta
  doSomething (internal/crap/c...)
    GazeCRAP                                8.5         12.1        +3.6
    CRAP                                    12.0        12.0        +0.0
```

**Rationale**: GazeCRAP first because it incorporates contract coverage — the richer, more actionable metric. When a function regresses on GazeCRAP but not CRAP, the GazeCRAP row (first) immediately explains the classification. CRAP second provides the traditional baseline for comparison.

**Alternative considered**: GazeCRAP second. Rejected because the eye naturally reads top-down, and the metric that most often explains the classification should come first.

### D2: Conditional format based on data availability

Detect whether any function in the matching set has GazeCRAP delta data. If none do, use the current single-row format unchanged. This avoids visual noise when GazeCRAP data is absent (e.g., older baselines, runs without contract coverage).

**Detection**: Check if any `d.GazeCRAPDelta != nil` in the matching deltas.

### D3: Function name on its own row in two-row mode

In two-row mode, the function name (with file path) is printed on a line by itself, with the metric rows indented below it. This keeps each metric row's column values cleanly aligned without needing to truncate function names further.

**Rationale**: Putting the function name on the same row as GazeCRAP values would require reducing the function name column width from 40 to ~30 chars to fit all values, causing more name truncation. A separate name row avoids this tradeoff.

### D4: Omit GazeCRAP row for functions without GazeCRAP data

When the table is in two-row mode (because at least one function has GazeCRAP data), functions that lack GazeCRAP data still show the CRAP row but omit the GazeCRAP row. This avoids printing empty/zero GazeCRAP lines.

### D5: No signature change to writeComparisonDeltaTable

The function already receives `[]FunctionDelta` which contains all the needed data (`GazeCRAPDelta`, `Baseline.GazeCRAP`, `Current.GazeCRAP`). No parameter changes required.

## Risks / Trade-offs

- **Vertical space increase**: Two-row mode uses ~2x vertical space per function. For tables with many regressions/improvements, this could produce longer output. Accepted because the additional context (GazeCRAP values) justifies the space, and the regression/improvement tables are typically short (most functions are unchanged).
- **Mixed format within a table**: When some functions have GazeCRAP data and others don't, the table will have a mix of two-row entries (with GazeCRAP) and entries with just the CRAP row (no GazeCRAP). This is slightly uneven visually but preferable to printing empty GazeCRAP rows.
- **Observable Quality** (Constitution III): This change improves text output observability by surfacing computed GazeCRAP data that was previously hidden. The JSON format is unchanged and continues to carry full provenance.
