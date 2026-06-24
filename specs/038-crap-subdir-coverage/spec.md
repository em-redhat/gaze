# Feature Specification: Fix Zero Coverage When Running from Subdirectory

**Feature Branch**: `038-crap-subdir-coverage`
**Created**: 2026-06-22
**Status**: Draft
**Input**: User description: "based on issue 113 of the original repository where the fork was created from"
**Upstream Issue**: unbound-force/gaze#113

## Upstream Analysis

### Bug Report (issue author)

`gaze crap` resolves the import-path-relative filenames in a coverage profile to
on-disk files by reading `go.mod` from `moduleDir`, where `moduleDir` is the
current working directory. When `crap` is invoked from a **subdirectory** of the
module (which has no `go.mod`), `readModulePath` returns `""`, `resolveFilePath`
returns `""` for every profile entry, and all coverage is dropped. Every function
reads 0% and CRAP is maximally inflated — with a normal-looking report and exit 0.

**Reproduction:**

```
$ cd <module-root> && gaze crap ./... | grep -E 'G|COVERAGE'
│CRAP│COMPLEXITY│COVERAGE│FUNCTION│FILE      │
│2.1 │2         │66.7%   │G       │sub/s.go:4│      # correct

$ cd sub && gaze crap ./... | grep -E 'G|COVERAGE'
│CRAP│COMPLEXITY│COVERAGE│FUNCTION│FILE  │
│6.0 │2         │0.0%    │G       │s.go:4│          # coverage silently dropped
```

### Review Council Analysis (jflowers)

**Verdict: Valid bug, CRITICAL severity.** Five review council agents (Adversary, Architect, Guard, SRE, Testing) converged on five problems with the original suggested fix:

#### 1. Scope is underestimated (~4x)

The issue frames this as a `gaze crap` bug. It also affects `gaze report` and several internal call sites that use `os.Getwd()` where they should use the module root:

| Location | Impact |
|----------|--------|
| `cmd/gaze/main.go:703` (crap command) | **PRIMARY BUG** — moduleDir from os.Getwd() |
| `cmd/gaze/main.go:1329` (report command) | **Same bug** — cwd used as ModuleDir |
| `internal/crap/contract.go:254` (classifyResults) | Classification degraded — module loading from cwd |
| `internal/crap/contract.go:330` (loadGazeConfigBestEffort) | Config not found from subdirectory |
| `internal/aireport/runner_steps.go:298,313` | Module loading + config — fallback to os.Getwd() |

#### 2. The silence problem is not addressed

The walk-upward fix corrects the *wrong numbers* but does not fix the *silence*. The entire failure chain is silent:

- `readModulePath` returns `""` — no warning
- `resolveFilePath` returns `""` — no warning
- `ParseCoverProfile` silently skips unresolved entries (`continue` at line 65-66)
- `Analyze` returns a report with 0% coverage and `nil` error
- Exit code 0

Defense-in-depth is needed: warn when `resolveFilePath` fails for individual entries, and error when *all* profile entries fail to resolve (0/N resolved is almost certainly a configuration error, not "no coverage").

#### 3. The `go/packages` alternative should be rejected

Walk-upward is 5-50us vs `go/packages` at 100-500ms (subprocess spawn). The walk already exists as `findModuleRoot` and is proven by `self-check`.

#### 4. `findModuleRoot` should be refactored for testability

The current `findModuleRoot()` calls `os.Getwd()` internally, making it untestable without `os.Chdir` (process-global, race-unsafe). It should be split into `findModuleRootFrom(startDir)` + a thin wrapper.

#### 5. Existing test codifies the bug as correct

`TestResolveFilePath_NoGoMod` (`crap_test.go:752-758`) asserts the buggy behavior is expected — it verifies `resolveFilePath` returns `""` when `go.mod` is absent. This test must be updated to distinguish "subdirectory of a module" from "outside any module."

#### Constitutional Violations

| Principle | Violation |
|-----------|-----------|
| **Accuracy** | Reports 0% coverage when actual coverage may be 80%+ |
| **Minimal Assumptions** | Assumes CWD is module root — undocumented, unenforced, silently ignored when wrong |
| **Actionable Output** | Guides user toward writing tests that already exist; metrics not comparable across runs |

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Accurate Coverage from Any Directory (Priority: P1)

A developer working in a Go module runs `gaze crap` from a subdirectory (e.g., `cd pkg/server && gaze crap ./...`) and expects the same CRAP scores and coverage percentages as running from the module root. Today, running from a subdirectory silently drops all coverage data, inflating CRAP scores to their maximum values with no warning — producing confidently wrong output.

**Why this priority**: This is the core bug. Users get incorrect, misleading output with no indication that anything is wrong. It violates three of four constitutional principles simultaneously.

**Independent Test**: Can be fully tested by running `gaze crap ./...` from both the module root and a subdirectory, verifying that coverage percentages and CRAP scores match for the same functions.

**Acceptance Scenarios**:

1. **Given** a Go module with tests providing 66.7% coverage on function `G`, **When** the user runs `gaze crap ./...` from a subdirectory within that module, **Then** the reported coverage for `G` is 66.7% (same as from the root).
2. **Given** a Go module with a `go.mod` at the root, **When** the user runs `gaze crap ./...` from any nested subdirectory at any depth, **Then** the tool locates the module root automatically and resolves all coverage profile paths correctly.
3. **Given** a Go module with tests, **When** `gaze crap` is run from the module root, **Then** the behavior is identical to the current behavior (no regression).

---

### User Story 2 - Accurate Report Output from Any Directory (Priority: P2)

A developer or CI pipeline runs `gaze report` from a subdirectory and expects the same quality analysis results as running from the module root. The report pipeline shares the same coverage resolution path as `gaze crap`, so the same bug applies.

**Why this priority**: `gaze report` is the primary CI integration point. Silent data loss in CI produces misleading quality gates that either pass when they should fail or produce inflated CRAPload numbers.

**Independent Test**: Can be tested by running `gaze report ./... --format=json` from both the module root and a subdirectory, verifying the CRAP scores and coverage values in the JSON payload match.

**Acceptance Scenarios**:

1. **Given** a Go module with tests, **When** the user runs `gaze report ./... --format=json` from a subdirectory, **Then** the CRAP scores and coverage percentages in the report match those produced from the module root.
2. **Given** a CI pipeline that changes to a subdirectory before running `gaze report`, **When** the report executes, **Then** threshold checks (`--max-crapload`, `--max-gaze-crapload`, `--min-contract-coverage`) evaluate against correct coverage data.

---

### User Story 3 - Silent Failure Prevention (Priority: P2)

When coverage profile path resolution fails — whether due to missing `go.mod`, corrupted profile, or wrong module — the tool provides diagnostic feedback instead of silently producing zero-coverage results.

**Why this priority**: Even after fixing the walk-up, other scenarios can trigger silent 0% coverage. Defense-in-depth prevents future silent failures from any cause.

**Independent Test**: Can be tested by providing a coverage profile with entries that cannot be resolved and verifying that warnings appear on stderr and/or an error is returned when all entries fail.

**Acceptance Scenarios**:

1. **Given** a coverage profile where some entries cannot be resolved to files on disk, **When** the tool processes the profile, **Then** a warning is written to stderr for each unresolved entry.
2. **Given** a coverage profile where zero out of N entries resolve successfully, **When** the tool processes the profile, **Then** an error is returned (not a silent zero-coverage report).

---

### User Story 4 - Clear Error When No Module Root Found (Priority: P3)

A user runs `gaze crap` from a directory that is not inside any Go module (no `go.mod` exists in any ancestor directory). Instead of silently producing zero-coverage results, the tool should produce a clear error message.

**Why this priority**: Edge case, but the current behavior (silent 0% with exit 0) is the worst possible outcome.

**Independent Test**: Can be tested by running `gaze crap ./...` from a directory with no `go.mod` in any ancestor and verifying that a descriptive error is returned.

**Acceptance Scenarios**:

1. **Given** a directory with no `go.mod` in the current directory or any ancestor, **When** the user runs `gaze crap ./...`, **Then** the tool exits with a non-zero exit code and an error message indicating that no Go module root was found.

---

### Edge Cases

- What happens when the module root is at the filesystem root (`/`)?
- What happens when `go.mod` exists in the current directory (no walk-up needed)? This must continue to work identically to current behavior.
- What happens when there are nested `go.mod` files (a module within a module)? The nearest ancestor `go.mod` should be used.
- What happens when `go.mod` is unreadable (permission denied)? The tool should return a clear error rather than silently proceeding with zero coverage.
- What happens when multiple coverage profiles reference different modules? Only the enclosing module's profile entries should resolve.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST locate the nearest ancestor directory containing a `go.mod` file when resolving the module root, rather than assuming the current working directory is the module root.
- **FR-002**: When running `gaze crap` from any subdirectory within a Go module, the tool MUST produce the same coverage percentages and CRAP scores as when running from the module root.
- **FR-003**: When running `gaze report` from any subdirectory within a Go module, the tool MUST produce the same coverage data, CRAP scores, and threshold evaluations as when running from the module root.
- **FR-004**: When no `go.mod` file is found in the current directory or any ancestor directory (up to the filesystem root), the tool MUST exit with a non-zero exit code and a descriptive error message.
- **FR-005**: When `go.mod` exists in the current working directory (the common case today), the tool MUST behave identically to its current behavior — no regression.
- **FR-006**: The tool MUST use the nearest ancestor `go.mod` when nested modules exist.
- **FR-007**: Configuration files (`.gaze.yaml`, `.gaze/baseline.json`) MUST be resolved relative to the discovered module root, not relative to the current working directory.
- **FR-008**: When individual coverage profile entries fail to resolve to files on disk, a warning MUST be written to stderr identifying the unresolved entry.
- **FR-009**: When zero out of N coverage profile entries resolve successfully, the tool MUST return an error rather than producing a silent zero-coverage report.
- **FR-010**: The module-root-finding function MUST be extracted to a shared, testable location and accept a starting directory parameter for testability.
- **FR-011**: All internal call sites that use `os.Getwd()` as a proxy for the module root MUST be updated to use the shared module-root-finding function. This includes `internal/crap/contract.go` (classifyResults, loadGazeConfigBestEffort) and `internal/aireport/runner_steps.go` (resolveModulePackages, loadGazeConfigBestEffort).
- **FR-012**: The existing `TestResolveFilePath_NoGoMod` test MUST be updated to reflect the corrected contract boundary (distinguishing "subdirectory of a module" from "outside any module").

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `gaze crap ./...` from any subdirectory within a Go module produces identical coverage percentages (within floating-point rounding) to running from the module root.
- **SC-002**: Running `gaze report ./... --format=json` from a subdirectory produces identical CRAP and coverage values to running from the module root.
- **SC-003**: Running `gaze crap ./...` from a directory with no ancestor `go.mod` produces a non-zero exit code and an error message containing "go.mod" or "module root".
- **SC-004**: All existing tests pass (with `TestResolveFilePath_NoGoMod` updated), confirming no regression in the module-root invocation path.
- **SC-005**: When a coverage profile has entries that cannot be resolved, at least one warning appears on stderr.
- **SC-006**: When zero profile entries resolve, the tool exits with a non-zero exit code.

## Dependencies & Assumptions

### Dependencies

- **Spec 009 (crapload-reduction)**: Decomposed `buildContractCoverageFunc` into `resolvePackagePaths`/`analyzePackageCoverage` — the coverage resolution path affected by this bug.
- **Spec 022 (report-gazecrap-pipeline)**: Wired `BuildContractCoverageFunc` into the report pipeline — `gaze report` shares the same `moduleDir` resolution bug.
- **Spec 020 (report-coverprofile)**: Added `--coverprofile` flag threading through `RunnerOptions.CoverProfile` — the `ModuleDir` in `RunnerOptions` is set from `os.Getwd()`.

### Assumptions

- The `findModuleRoot` function already exists in `cmd/gaze/main.go` (used by `self-check`) and will be extracted to a shared location.
- Go modules are the only supported project structure (no GOPATH-mode projects).
- The nearest ancestor `go.mod` is always the correct module root for the user's intended analysis scope.
- The walk-upward approach is used (not `go/packages`), per review council consensus: 5-50us vs 100-500ms, already proven by `self-check`.
