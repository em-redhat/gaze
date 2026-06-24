# Data Model: Fix Zero Coverage When Running from Subdirectory

**Feature**: 038-crap-subdir-coverage
**Date**: 2026-06-22

## Overview

This feature introduces no new data types or entities. It refactors the module root discovery pattern from an implicit assumption (cwd = module root) to an explicit function call (`loader.FindModuleRoot`). The change affects function signatures and data flow, not data structures.

## Function Signature Changes

### New Function

```
FindModuleRoot(startDir string) (string, error)
```

- **Package**: `internal/loader`
- **Input**: `startDir` — the directory to start walking up from (typically `os.Getwd()` result)
- **Output**: Absolute path to the nearest ancestor directory containing `go.mod`, or error if none found
- **Error cases**: No `go.mod` found in `startDir` or any ancestor up to filesystem root

### Modified Signatures

#### `classifyResults` (internal/crap/contract.go)

**Before**: `func classifyResults(results []taxonomy.AnalysisResult, pkgPath string, cfg *config.GazeConfig) []taxonomy.AnalysisResult`

**After**: `func classifyResults(results []taxonomy.AnalysisResult, pkgPath string, moduleDir string, cfg *config.GazeConfig) []taxonomy.AnalysisResult`

Added `moduleDir` parameter to replace internal `os.Getwd()` call.

#### `loadGazeConfigBestEffort` (internal/crap/contract.go)

**Before**: `func loadGazeConfigBestEffort() *config.GazeConfig`

**After**: `func loadGazeConfigBestEffort(moduleDir string) *config.GazeConfig`

Adopts the same signature as the `cmd/gaze/main.go` version.

#### `loadGazeConfigBestEffort` (internal/aireport/runner_steps.go)

**Before**: `func loadGazeConfigBestEffort() *config.GazeConfig`

**After**: `func loadGazeConfigBestEffort(moduleDir string) *config.GazeConfig`

Same change as above. The `NOTE: keep in sync` comment should be preserved or the two copies consolidated.

### Data Flow Changes

#### `gaze crap` command flow

```
Before:  os.Getwd() → moduleDir → crap.Analyze / BuildContractCoverageFunc
After:   os.Getwd() → loader.FindModuleRoot(cwd) → moduleDir → crap.Analyze / BuildContractCoverageFunc
```

#### `gaze report` command flow

```
Before:  os.Getwd() → cwd → RunnerOptions.ModuleDir → pipeline steps
After:   os.Getwd() → loader.FindModuleRoot(cwd) → moduleDir → RunnerOptions.ModuleDir → pipeline steps
```

#### `ParseCoverProfile` diagnostics (new)

```
Before:  resolveFilePath returns "" → silently skipped
After:   resolveFilePath returns "" → warning to stderr + counter tracked
         After loop: if resolved == 0 && total > 0 → return error
```

## Unchanged Entities

- `crap.Report`, `crap.Score`, `crap.Summary` — no field changes
- `taxonomy.AnalysisResult`, `taxonomy.PackageSummary` — no field changes
- `aireport.RunnerOptions` — `ModuleDir` field already exists, just receives correct value
- `aireport.ReportSummary` — no field changes
- `config.GazeConfig` — no field changes
