# Implementation Plan: Fix Zero Coverage When Running from Subdirectory

**Branch**: `038-crap-subdir-coverage` | **Date**: 2026-06-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/038-crap-subdir-coverage/spec.md`

## Summary

`gaze crap` and `gaze report` assume the current working directory is the Go module root when resolving coverage profile paths, loading configuration, and loading module packages. When invoked from a subdirectory, `go.mod` is not found, causing all coverage to be silently dropped (0% everywhere, inflated CRAP scores, exit 0). The fix extracts a shared `loader.FindModuleRoot(startDir)` function that walks up to find the nearest `go.mod`, replaces 9 `os.Getwd()` call sites with the correct module root, and adds stderr diagnostics for unresolved coverage profile entries.

## Technical Context

**Language/Version**: Go 1.25+ (per `go.mod` directive)
**Primary Dependencies**: `golang.org/x/tools/go/packages` (existing), standard library (`os`, `path/filepath`)
**Storage**: N/A — no persistence changes
**Testing**: Standard library `testing` package, `-race -count=1`
**Target Platform**: Linux/macOS (cross-platform, no platform-specific code)
**Project Type**: Single CLI binary with layered internal packages
**Performance Goals**: `FindModuleRoot` must complete in <1ms (filesystem stat walk)
**Constraints**: No new external dependencies; no breaking API changes
**Scale/Scope**: 9 call sites across 4 files, 1 new function, ~50 lines of new code + ~100 lines of tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Research Gate (Phase 0)

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. Accuracy** | PASS | This fix directly restores accuracy — eliminates false 0% coverage reports. Regression tests will verify correct coverage from subdirectories. |
| **II. Minimal Assumptions** | PASS | Removes the undocumented assumption that cwd = module root. The tool will now work from any directory within a module without user action. |
| **III. Actionable Output** | PASS | Adds stderr diagnostics for unresolved profile entries and error for 0/N resolution. Metrics become comparable across runs regardless of cwd. |
| **IV. Testability** | PASS | `FindModuleRoot(startDir)` is pure (no internal `os.Getwd()`), testable with temp directories. Coverage strategy specified below. |

### Post-Design Gate (Phase 1)

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. Accuracy** | PASS | `FindModuleRoot` walks up to the correct `go.mod`, ensuring `resolveFilePath` receives the right module path. Defense-in-depth: 0/N error catches any remaining resolution failure. |
| **II. Minimal Assumptions** | PASS | The only assumption is that `go.mod` exists somewhere in an ancestor directory — this is the Go module system's own requirement. When it doesn't exist, a clear error is returned. |
| **III. Actionable Output** | PASS | Stderr warnings identify specific unresolved profile entries. The 0/N error message suggests checking that the profile matches the current module. |
| **IV. Testability** | PASS | All new code is testable in isolation. `FindModuleRoot` takes `startDir` parameter. `ParseCoverProfile` already accepts `stderr io.Writer` for test capture. Coverage strategy: unit tests for `FindModuleRoot` (all branches), unit tests for `ParseCoverProfile` diagnostics, integration test for subdirectory invocation. |

## Project Structure

### Documentation (this feature)

```text
specs/038-crap-subdir-coverage/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output — research decisions
├── data-model.md        # Phase 1 output — signature changes
├── quickstart.md        # Phase 1 output — verification guide
├── checklists/
│   └── requirements.md  # Specification quality checklist
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
  loader/
    loader.go            # ADD FindModuleRoot(startDir) function
    loader_test.go       # ADD FindModuleRoot unit tests
  crap/
    coverage.go          # MODIFY ParseCoverProfile — add diagnostics
    contract.go          # MODIFY classifyResults, loadGazeConfigBestEffort — add moduleDir param
    crap_test.go         # MODIFY TestResolveFilePath_NoGoMod, ADD diagnostic tests
  aireport/
    runner_steps.go      # MODIFY resolveModulePackages, loadGazeConfigBestEffort — use moduleDir
cmd/
  gaze/
    main.go              # MODIFY newCrapCmd, runReport, runClassify, loadConfig — use FindModuleRoot
                         # DELETE findModuleRoot (replaced by loader.FindModuleRoot)
```

**Structure Decision**: No new packages or directories. All changes fit within existing package boundaries. `FindModuleRoot` is added to `internal/loader` (its natural home alongside `Load` and `LoadModule`).

## Design Decisions

### D1: Function Placement

`FindModuleRoot(startDir string) (string, error)` lives in `internal/loader/loader.go`. See [research.md](research.md) R2 for rationale and alternatives considered.

### D2: Signature Change Strategy

Internal functions that called `os.Getwd()` as a module root proxy gain a `moduleDir string` parameter. The `os.Getwd() → FindModuleRoot(cwd)` resolution happens once at the CLI entry point (command RunE closures), and the result flows down through the call chain. This is the "push cwd resolution to the edge" pattern — internal packages never call `os.Getwd()` for module root resolution.

### D3: Diagnostic Strategy

Two levels of diagnostics in `ParseCoverProfile`:
1. **Per-entry warning** (stderr): `warning: skipping profile entry %q: could not resolve to file on disk`
2. **Aggregate error** (return value): When 0/N entries resolve, return `fmt.Errorf("all %d coverage profile entries failed to resolve — check that the profile matches the current module", totalEntries)`

### D4: Error vs Warning Threshold

0/N resolved = error (hard stop). 1+/N resolved = warnings only (partial results may still be useful — e.g., some packages were removed since the profile was generated). This matches the existing `findFunctions` pattern in `ParseCoverProfile` where individual file parse errors are warnings, not fatal.

### D5: `loadGazeConfigBestEffort` Consolidation

The three copies are reduced to two with matching signatures. Full consolidation into a single shared function would require moving it to a shared package (e.g., `internal/config`) or making it a method — deferred to avoid scope creep. The immediate fix is ensuring both internal copies accept `moduleDir` as a parameter instead of calling `os.Getwd()`.

## Implementation Phases

### Phase 1: Core Function + Unit Tests

**Files**: `internal/loader/loader.go`, `internal/loader/loader_test.go`

1. Add `FindModuleRoot(startDir string) (string, error)` to `internal/loader/loader.go`
   - Walk up from `startDir`, checking for `go.mod` at each level
   - Stop at filesystem root (`filepath.Dir(dir) == dir`)
   - Return error: `no go.mod found in %q or any parent directory`
2. Add unit tests in `internal/loader/loader_test.go`:
   - `TestFindModuleRoot_AtRoot` — `go.mod` in starting directory
   - `TestFindModuleRoot_FromSubdirectory` — walks up 1-3 levels
   - `TestFindModuleRoot_NoGoMod` — returns error
   - `TestFindModuleRoot_NestedModules` — finds nearest (not deepest)

**FR coverage**: FR-001, FR-005, FR-006, FR-010

### Phase 2: ParseCoverProfile Diagnostics

**Files**: `internal/crap/coverage.go`, `internal/crap/crap_test.go`

1. Add warning when `resolveFilePath` returns `""` in `ParseCoverProfile`
2. Add 0/N resolution error after the profile loop
3. Replace `os.Getwd()` fallback in `ParseCoverProfile` with `FindModuleRoot`
4. Update `TestResolveFilePath_NoGoMod` — verify it still tests the correct contract
5. Add `TestParseCoverProfile_AllUnresolved` — 0/N returns error
6. Add `TestParseCoverProfile_PartialUnresolved` — warns but succeeds
7. Add `TestParseCoverProfile_WarningOutput` — verify stderr content

**FR coverage**: FR-008, FR-009, FR-012

### Phase 3: Internal Package Call Site Updates

**Files**: `internal/crap/contract.go`, `internal/aireport/runner_steps.go`

1. `classifyResults` — add `moduleDir string` parameter, pass to `loader.LoadModule(moduleDir)` instead of `os.Getwd()`
2. `loadGazeConfigBestEffort` in `contract.go` — add `moduleDir string` parameter
3. `loadGazeConfigBestEffort` in `runner_steps.go` — add `moduleDir string` parameter
4. `resolveModulePackages` in `runner_steps.go` — replace `os.Getwd()` fallback with `loader.FindModuleRoot`
5. Update all callers of these modified functions to pass `moduleDir`
6. Verify `BuildContractCoverageFunc` already receives `moduleDir` and threads it correctly

**FR coverage**: FR-007, FR-011

### Phase 4: CLI Entry Point Updates

**Files**: `cmd/gaze/main.go`

1. `newCrapCmd` RunE (line 703) — replace `os.Getwd()` with `loader.FindModuleRoot(cwd)`
2. `runReport` (line 1329) — replace `os.Getwd()` with `loader.FindModuleRoot(cwd)`
3. `runClassify` (line 291) — replace `os.Getwd()` with `loader.FindModuleRoot(cwd)`
4. `loadConfig` (line 148) — replace `os.Getwd()` with `loader.FindModuleRoot(cwd)`
5. Delete `findModuleRoot()` (line 1168) — replaced by `loader.FindModuleRoot`
6. Update `runSelfCheck` to use `loader.FindModuleRoot` via `selfCheckParams.moduleRootFunc`

**FR coverage**: FR-002, FR-003, FR-004

### Phase 5: Integration Tests + Regression Verification

1. Run `go test -race -count=1 -short ./...` — verify all existing tests pass
2. Run `go build ./cmd/gaze` — verify build succeeds
3. Run `golangci-lint run` — verify no lint issues
4. Manual verification: `gaze crap ./...` from module root (regression check)

**FR coverage**: FR-005, SC-004

## Coverage Strategy

| Component | Test Type | Coverage Target |
|-----------|-----------|----------------|
| `loader.FindModuleRoot` | Unit (temp dirs) | 100% branch — all 4 paths (at root, walk up, no module, nested) |
| `ParseCoverProfile` diagnostics | Unit (synthetic profiles) | 100% — 0/N error, partial warn, all resolved |
| `classifyResults` moduleDir param | Existing tests + param update | Existing coverage maintained |
| `loadGazeConfigBestEffort` param | Existing tests + param update | Existing coverage maintained |
| CLI entry points | Integration (existing e2e) | Covered by existing `TestRunSelfCheck` and unit tests |
| Subdirectory invocation | Integration | New: verify SC-001 (subdirectory = same results as root) |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Signature changes break callers | Low | Medium | All callers are internal; compiler catches missing args |
| `FindModuleRoot` walk hits symlink loops | Very Low | Low | `filepath.Dir` handles symlinks; Go's `os.Stat` follows them |
| 0/N error is too aggressive | Low | Medium | Only triggers when ALL entries fail; partial resolution still succeeds with warnings |
| Nested module edge case | Low | Low | `FindModuleRoot` returns nearest, which is correct for the user's cwd context |
| Performance regression from walk | Very Low | Very Low | Walk is <1ms even at 50 levels; happens once per command invocation |

## Complexity Tracking

No constitution violations requiring justification. All principles pass in both pre-research and post-design gates.
