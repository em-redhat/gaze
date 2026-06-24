# Research: Fix Zero Coverage When Running from Subdirectory

**Feature**: 038-crap-subdir-coverage
**Date**: 2026-06-22

## R1: Module Root Discovery Approach

**Decision**: Walk up from the current working directory to find the nearest `go.mod` file.

**Rationale**: This approach is already proven in the codebase (`findModuleRoot` in `cmd/gaze/main.go:1168`, test helper in `internal/loader/loader_test.go:36`). It is deterministic, runs in 5-50us, and requires only `os.Stat` calls — no subprocess execution.

**Alternatives considered**:
- `go/packages` with `packages.NeedModule`: Spawns a `go list` subprocess, adds 100-500ms latency per call. Overkill for finding a file that can be located with a simple directory walk. Rejected unanimously by review council.
- `go list -m -json`: Same subprocess overhead, plus requires parsing JSON output. No benefit over walking.
- `go env GOMOD`: Single subprocess call, lighter than `go/packages`, but still orders of magnitude slower than a filesystem walk. Also cannot distinguish "no go.mod" from "go.mod exists but env is not set."

## R2: Where to Place the Shared Function

**Decision**: Add `FindModuleRoot(startDir string) (string, error)` to `internal/loader/loader.go`.

**Rationale**:
- `internal/loader` already wraps `go/packages` and owns `LoadModule(dir string)` — the primary consumer of the module root directory.
- The function has zero external dependencies (only `os` and `path/filepath`).
- A test-only version already exists in `internal/loader/loader_test.go:36-53` with identical logic.
- The production version in `cmd/gaze/main.go:1168` (`findModuleRoot`) can be replaced with a call to `loader.FindModuleRoot(cwd)`.

**Alternatives considered**:
- New package `internal/modroot`: Too much ceremony for a single function. Creates an unnecessary package boundary.
- Keep in `cmd/gaze/main.go`: Already there, but forces `internal/crap/contract.go` and `internal/aireport/runner_steps.go` to either duplicate the logic or import the `cmd` package (which is architecturally wrong — internal packages cannot import cmd packages).
- `internal/crap/coverage.go`: Too narrow — the function is needed by packages beyond `crap`.

## R3: Complete Inventory of Affected Call Sites

**Decision**: 10 call sites use `os.Getwd()` as a proxy for the module root. 9 must be fixed; 1 (`scaffold.go`) is correct as-is.

| # | File:Line | Function | Impact | Fix Strategy |
|---|-----------|----------|--------|-------------|
| 1 | `cmd/gaze/main.go:703` | `newCrapCmd` RunE | PRIMARY BUG — moduleDir = cwd | Replace with `loader.FindModuleRoot(cwd)` |
| 2 | `cmd/gaze/main.go:1329` | `runReport` | Same bug — ModuleDir = cwd | Replace with `loader.FindModuleRoot(cwd)` |
| 3 | `cmd/gaze/main.go:291` | `runClassify` | Classification degrades silently | Replace with `loader.FindModuleRoot(cwd)` |
| 4 | `cmd/gaze/main.go:148` | `loadConfig` | Config not found, silent fallback | Replace with `loader.FindModuleRoot(cwd)` |
| 5 | `internal/crap/contract.go:254` | `classifyResults` | Module loading from cwd | Add `moduleDir` parameter |
| 6 | `internal/crap/contract.go:330` | `loadGazeConfigBestEffort` | Config not found from subdirectory | Add `moduleDir` parameter (like `cmd/gaze/main.go:628`) |
| 7 | `internal/aireport/runner_steps.go:298` | `resolveModulePackages` | Fallback to cwd when empty | Replace fallback with `loader.FindModuleRoot` |
| 8 | `internal/aireport/runner_steps.go:313` | `loadGazeConfigBestEffort` | Duplicate of #6 | Add `moduleDir` parameter (consolidate with #6) |
| 9 | `internal/crap/coverage.go:57` | `ParseCoverProfile` | Fallback to cwd when empty | Replace fallback with `loader.FindModuleRoot` |
| 10 | `internal/scaffold/scaffold.go:159` | `applyDefaults` | NOT A BUG — cwd is correct target | No change needed |

## R4: `loadGazeConfigBestEffort` Duplication

**Decision**: Consolidate the three versions by adopting the `cmd/gaze/main.go:628` signature pattern (takes `moduleDir` as parameter) for all call sites.

**Current state**:
- `cmd/gaze/main.go:628`: `func loadGazeConfigBestEffort(moduleDir string) *config.GazeConfig` — correct pattern, takes dir as param
- `internal/crap/contract.go:329`: `func loadGazeConfigBestEffort() *config.GazeConfig` — calls `os.Getwd()` internally
- `internal/aireport/runner_steps.go:312`: `func loadGazeConfigBestEffort() *config.GazeConfig` — calls `os.Getwd()` internally

Both internal copies have `NOTE: keep in sync` comments acknowledging the duplication (deferred from spec 022).

**Rationale**: The `cmd/gaze/main.go` version is the only correct pattern — it accepts the module directory as a parameter, making it testable and immune to cwd issues. The internal copies must adopt the same parameter. Full consolidation into a single shared function is desirable but may require a shared package location; at minimum, both copies must accept `moduleDir` as a parameter.

## R5: Silent Failure Defense-in-Depth

**Decision**: Add two levels of diagnostics to `ParseCoverProfile`:
1. Per-entry warning when `resolveFilePath` returns `""` (to stderr, respecting the existing nil-check pattern)
2. Error return when 0/N profile entries resolve successfully

**Rationale**: Even with the module root walk-up fix, other scenarios can trigger silent 0% coverage: corrupted profile, stale profile from a different module, wrong working directory outside any module. The 0/N case is almost certainly a configuration error, not "legitimately zero coverage." Existing callers already pass `stderr` to `ParseCoverProfile` for other warnings.

**Design**:
- Track `totalEntries` and `resolvedEntries` counters in the `ParseCoverProfile` loop.
- When `resolveFilePath` returns `""`, write `warning: skipping profile entry %q: could not resolve to file on disk` to stderr (only if stderr is non-nil).
- After the loop, if `totalEntries > 0 && resolvedEntries == 0`, return an error: `all %d coverage profile entries failed to resolve — check that the profile matches the current module`.

## R6: Existing Test Update

**Decision**: Update `TestResolveFilePath_NoGoMod` to test the correct contract: "outside any module" should return `""`, while "subdirectory of a module" should resolve correctly via module root walk-up.

**Current test** (`internal/crap/crap_test.go:752-758`):
```go
func TestResolveFilePath_NoGoMod(t *testing.T) {
    dir := t.TempDir()
    got := resolveFilePath("example.com/test/foo.go", dir)
    if got != "" {
        t.Errorf("expected empty string without go.mod, got %q", got)
    }
}
```

This test creates a temp directory with no `go.mod` and asserts that `resolveFilePath` returns `""`. This correctly tests the "outside any module" case. However, `resolveFilePath` currently does not walk up — it only reads `go.mod` from the given `moduleDir`. After the fix, `resolveFilePath` will receive the module root (found by `FindModuleRoot`), so this test still makes sense: a directory with no `go.mod` and no ancestors with `go.mod` should return `""`.

The new tests needed are:
1. `TestFindModuleRoot_FromSubdirectory` — walks up and finds `go.mod`
2. `TestFindModuleRoot_NoGoMod` — returns error when no `go.mod` in any ancestor
3. `TestFindModuleRoot_AtRoot` — finds `go.mod` in the starting directory (no walk)
4. `TestParseCoverProfile_AllUnresolved` — returns error when 0/N entries resolve
5. `TestParseCoverProfile_PartialUnresolved` — warns but succeeds when some resolve

## R7: Coverage Strategy

**Decision**: Unit tests for the new `FindModuleRoot` function and the `ParseCoverProfile` diagnostics. Integration test for the end-to-end subdirectory invocation path.

| Layer | What | Target |
|-------|------|--------|
| Unit | `loader.FindModuleRoot` — walk up, at root, no module, nested modules | 100% branch coverage |
| Unit | `ParseCoverProfile` — 0/N error, partial warnings, all resolved | All three paths |
| Unit | Updated `TestResolveFilePath_NoGoMod` — contract boundary | Existing test updated |
| Integration | `gaze crap ./...` from subdirectory vs root — same results | SC-001 |
| Integration | `gaze report ./... --format=json` from subdirectory | SC-002 |
| Integration | `gaze crap ./...` from outside any module | SC-003 |
