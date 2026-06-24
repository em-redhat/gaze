## ADDED Requirements

### Requirement: GazeCRAP delta display in text comparison table

When GazeCRAP delta data is available for any function in a regression or improvement table, `writeComparisonDeltaTable` MUST render each function using a two-row format: the function name on its own line, followed by indented GazeCRAP and CRAP metric rows. GazeCRAP MUST be rendered before CRAP.

#### Scenario: Function with both CRAP and GazeCRAP deltas

- **GIVEN** a baseline comparison result where function `doSomething` has CRAP delta +0.0 and GazeCRAP delta +3.6
- **WHEN** `writeComparisonDeltaTable` renders the regressions table
- **THEN** the output contains the function name on its own line, followed by a `GazeCRAP` row showing baseline 8.5, current 12.1, delta +3.6, followed by a `CRAP` row showing baseline 12.0, current 12.0, delta +0.0

#### Scenario: Function with CRAP delta only (no GazeCRAP)

- **GIVEN** a baseline comparison result where function `helperFunc` has CRAP delta -2.0 and GazeCRAP delta is nil, but another function in the same table has GazeCRAP data
- **WHEN** `writeComparisonDeltaTable` renders the improvements table
- **THEN** the output contains the function name on its own line, followed by only a `CRAP` row (no GazeCRAP row), and the `CRAP` row shows the correct baseline, current, and delta values

### Requirement: Conditional format based on GazeCRAP data availability

`writeComparisonDeltaTable` MUST detect whether any matching delta has non-nil `GazeCRAPDelta`. When no matching delta has GazeCRAP data, the function MUST use the existing single-row format unchanged.

#### Scenario: No GazeCRAP data available in any delta

- **GIVEN** a baseline comparison result where all functions have nil GazeCRAP deltas
- **WHEN** `writeComparisonDeltaTable` renders the regressions table
- **THEN** the output uses the current single-row format: function name, baseline CRAP, current CRAP, and CRAP delta on one line

### Requirement: 80-column terminal width compliance

All output lines from `writeComparisonDeltaTable` MUST fit within 80 columns.

#### Scenario: Two-row format width compliance

- **GIVEN** a function name of maximum typical length (40 chars including file path)
- **WHEN** `writeComparisonDeltaTable` renders the two-row format
- **THEN** no output line exceeds 80 characters

## MODIFIED Requirements

_None._

## REMOVED Requirements

_None._
