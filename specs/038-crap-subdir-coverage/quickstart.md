# Quickstart: Fix Zero Coverage When Running from Subdirectory

**Feature**: 038-crap-subdir-coverage
**Date**: 2026-06-22

## What This Fixes

Running `gaze crap` or `gaze report` from a subdirectory of a Go module silently drops all coverage data. Every function shows 0% coverage and CRAP scores are maximally inflated — with no warning and exit code 0.

## After This Fix

```bash
# Both commands produce the same correct output:
cd <module-root> && gaze crap ./...    # works (always worked)
cd <module-root>/sub && gaze crap ./...  # now also works correctly

# Clear error when outside any Go module:
cd /tmp && gaze crap ./...
# Error: no go.mod found in "/tmp" or any parent directory
```

## Key Changes

1. **Module root auto-discovery**: `gaze crap` and `gaze report` walk up from the current directory to find the nearest `go.mod`, instead of assuming cwd is the module root.

2. **Diagnostic warnings**: When coverage profile entries cannot be resolved to files on disk, warnings appear on stderr. When zero entries resolve, the tool returns an error.

3. **Config resolution**: `.gaze.yaml` and `.gaze/baseline.json` are resolved from the discovered module root, not from cwd.

## Verification

```bash
# From module root (baseline — should be unchanged):
gaze crap ./...

# From a subdirectory (was broken, now fixed):
cd internal/crap && gaze crap ./...

# Compare: coverage values should match for the same functions
```

## Implementation Scope

- New shared function `loader.FindModuleRoot(startDir)` in `internal/loader/`
- Updated 9 call sites across `cmd/gaze/main.go`, `internal/crap/contract.go`, `internal/aireport/runner_steps.go`, and `internal/crap/coverage.go`
- Added stderr diagnostics for unresolved coverage profile entries
- No new CLI flags, no new output fields, no breaking changes
