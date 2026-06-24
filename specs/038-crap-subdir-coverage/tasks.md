# Tasks: Fix Zero Coverage When Running from Subdirectory

**Input**: Design documents from `/specs/038-crap-subdir-coverage/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Tests are included — the spec requires verifiable regression coverage and the plan specifies a coverage strategy (R7).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Extract the shared `FindModuleRoot` function that ALL user stories depend on. No user story work can begin until this phase is complete.

- [x] T001 Add `FindModuleRoot(startDir string) (string, error)` to `internal/loader/loader.go` — walk up from `startDir` checking for `go.mod` at each level, stop at filesystem root (`filepath.Dir(dir) == dir`), return error `no go.mod found in %q or any parent directory` when not found. Add `os` and `path/filepath` imports. (FR-001, FR-006, FR-010)
- [x] T002 Add `TestFindModuleRoot_AtRoot` in `internal/loader/loader_test.go` — create temp dir with `go.mod`, call `FindModuleRoot(dir)`, assert returns that dir. (FR-005)
- [x] T003 [P] Add `TestFindModuleRoot_FromSubdirectory` in `internal/loader/loader_test.go` — create temp dir with `go.mod` at root, create nested subdirectory 2-3 levels deep, call `FindModuleRoot(deepDir)`, assert returns root dir. (FR-001)
- [x] T004 [P] Add `TestFindModuleRoot_NoGoMod` in `internal/loader/loader_test.go` — create temp dir with NO `go.mod`, call `FindModuleRoot(dir)`, assert returns error containing "go.mod". (FR-004)
- [x] T005 [P] Add `TestFindModuleRoot_NestedModules` in `internal/loader/loader_test.go` — create temp dir with `go.mod` at root AND in a subdirectory, call `FindModuleRoot(subdir)`, assert returns the subdirectory (nearest ancestor). (FR-006)

**Checkpoint**: `go test -race -count=1 ./internal/loader/...` passes with all `FindModuleRoot` tests green.

---

## Phase 2: User Story 1 - Accurate Coverage from Any Directory (Priority: P1) MVP

**Goal**: `gaze crap ./...` from any subdirectory produces identical coverage/CRAP scores as from the module root.

**Independent Test**: Run `gaze crap ./...` from the module root and a subdirectory; coverage percentages and CRAP scores match for the same functions.

### Implementation for User Story 1

- [x] T006 [US1] In `cmd/gaze/main.go`, `newCrapCmd` RunE closure (line ~703): replace `moduleDir, err := os.Getwd()` with `cwd, err := os.Getwd()` followed by `moduleDir, err := loader.FindModuleRoot(cwd)`, returning the error with context `finding module root: %w`. Add `loader` import if not present. (FR-002, FR-004)
- [x] T007 [US1] In `cmd/gaze/main.go`, `runClassify` function (line ~291): replace `cwd, err := os.Getwd()` with `cwd, err := os.Getwd()` followed by `moduleRoot, err := loader.FindModuleRoot(cwd)`, pass `moduleRoot` to `loader.LoadModule` instead of `cwd`. On `FindModuleRoot` error, log warning and set `moduleRoot = ""` (non-fatal for classification). (FR-011)
- [x] T008 [US1] In `cmd/gaze/main.go`, `loadConfig` function (line ~148): when `path == ""`, replace `cwd, err := os.Getwd()` with `cwd, err := os.Getwd()` followed by `moduleRoot, findErr := loader.FindModuleRoot(cwd)`, use `moduleRoot` for config path. On `FindModuleRoot` error, fall back to `config.DefaultConfig()` (existing behavior for config-not-found). (FR-007)
- [x] T009 [US1] In `cmd/gaze/main.go`, delete `findModuleRoot()` function (line ~1168-1183) — replaced by `loader.FindModuleRoot`. Update `runSelfCheck`: change `selfCheckParams.moduleRootFunc` type from `func() (string, error)` to accept a wrapper that calls `loader.FindModuleRoot(cwd)`. Ensure `runSelfCheck` still works by getting `cwd` via `os.Getwd()` and passing to `loader.FindModuleRoot`. (FR-010)
- [x] T009a [US1] In `cmd/gaze/main.go`, `runDocscan` function (line ~767): replace `repoRoot, err := os.Getwd()` with `cwd, err := os.Getwd()` followed by `repoRoot, findErr := loader.FindModuleRoot(cwd)`, falling back to `cwd` on error (docscan is non-critical). This ensures doc scanning also resolves from the module root when invoked from a subdirectory. (FR-011)
- [x] T010 [US1] In `internal/crap/contract.go`, `classifyResults` function (line ~242): add `moduleDir string` parameter after `pkgPath string`, replace `cwd, err := os.Getwd()` and `loader.LoadModule(cwd)` with `loader.LoadModule(moduleDir)`. Remove `os` import if no longer needed. Update signature in GoDoc comment. (FR-011)
- [x] T011 [US1] In `internal/crap/contract.go`, `loadGazeConfigBestEffort` function (line ~329): add `moduleDir string` parameter, replace body with `cfgPath := filepath.Join(moduleDir, ".gaze.yaml")` (matching the `cmd/gaze/main.go:628` pattern). Remove `os` import if no longer needed. Preserve `NOTE: keep in sync` comment. (FR-007, FR-011)
- [x] T012 [US1] In `internal/crap/contract.go`, update all callers of the modified functions: `BuildContractCoverageFunc` (line ~53) calls `loadGazeConfigBestEffort()` — change to `loadGazeConfigBestEffort(moduleDir)` using the `moduleDir` parameter it already receives. `analyzePackageCoverage` (line ~210) calls `classifyResults(results, pkgPath, gazeConfig)` — change to `classifyResults(results, pkgPath, moduleDir, gazeConfig)`, threading `moduleDir` from `BuildContractCoverageFunc` through `analyzePackageCoverage`. Add `moduleDir string` parameter to `analyzePackageCoverage` if not already present. (FR-011)
- [x] T013 [US1] In `internal/crap/coverage.go`, `ParseCoverProfile` function (line ~56-57): replace `os.Getwd()` fallback with `loader.FindModuleRoot`. When `moduleDir == ""`, call `moduleDir, _ = os.Getwd()` then `moduleDir, findErr := loader.FindModuleRoot(moduleDir)` — on error, set `moduleDir = ""` (will trigger 0/N diagnostics in US3). Add `loader` import. (FR-001)

**Checkpoint**: `go build ./cmd/gaze && go test -race -count=1 -short ./...` passes. `gaze crap ./...` from module root works as before (SC-004).

---

## Phase 3: User Story 2 - Accurate Report Output from Any Directory (Priority: P2)

**Goal**: `gaze report ./... --format=json` from any subdirectory produces identical results as from the module root.

**Independent Test**: Run `gaze report ./... --format=json` from the module root and a subdirectory; CRAP scores and coverage values match.

### Implementation for User Story 2

- [x] T014 [US2] In `cmd/gaze/main.go`, `runReport` function (line ~1329): replace `cwd, err := os.Getwd()` with `cwd, err := os.Getwd()` followed by `moduleDir, findErr := loader.FindModuleRoot(cwd)`, return error with context if `FindModuleRoot` fails. Use `moduleDir` for `aireport.LoadPrompt(moduleDir)` and `RunnerOptions.ModuleDir: moduleDir`. (FR-003, FR-004)
- [x] T015 [US2] In `internal/aireport/runner_steps.go`, `loadGazeConfigBestEffort` function (line ~312): add `moduleDir string` parameter, replace body with `cfgPath := filepath.Join(moduleDir, ".gaze.yaml")` pattern. Remove `os` import if no longer needed. Preserve `NOTE: keep in sync` comment or add one referencing the `contract.go` copy. (FR-007, FR-011)
- [x] T016 [US2] In `internal/aireport/runner_steps.go`, `resolveModulePackages` function (line ~295): replace `os.Getwd()` fallback (lines 297-301) with `loader.FindModuleRoot`. When `moduleDir == ""`, call `cwd, err := os.Getwd()` then `moduleDir, _ = loader.FindModuleRoot(cwd)` — on error, return nil (existing graceful degradation pattern). Add `loader` import if not present. (FR-011)
- [x] T017 [US2] In `internal/aireport/runner_steps.go`, update all callers of `loadGazeConfigBestEffort()` to pass `moduleDir`. Search for calls within `runner_steps.go` and pass the `moduleDir` parameter that flows through from `RunnerOptions.ModuleDir`. (FR-011)

**Checkpoint**: `go build ./cmd/gaze && go test -race -count=1 -short ./...` passes. Both `gaze crap` and `gaze report` build and run correctly from module root (SC-004).

---

## Phase 4: User Story 3 - Silent Failure Prevention (Priority: P2)

**Goal**: Unresolved coverage profile entries produce stderr warnings; 0/N resolution produces an error instead of silent zero coverage.

**Independent Test**: Provide a coverage profile with unresolvable entries; verify warnings on stderr and error when all entries fail.

### Tests for User Story 3

- [x] T018 [P] [US3] Add `TestParseCoverProfile_AllUnresolved` in `internal/crap/crap_test.go` — create a coverage profile with entries that cannot be resolved (e.g., `nonexistent.com/pkg/file.go`), create a temp dir with a valid `go.mod`, call `ParseCoverProfile(profilePath, dir, &stderr)`, assert returns error containing "failed to resolve". (FR-009, SC-006)
- [x] T019 [P] [US3] Add `TestParseCoverProfile_PartialUnresolved` in `internal/crap/crap_test.go` — create a coverage profile with one resolvable and one unresolvable entry, call `ParseCoverProfile`, assert returns nil error (success) AND stderr contains a warning for the unresolvable entry. (FR-008, SC-005)
- [x] T020 [P] [US3] Add `TestParseCoverProfile_WarningContent` in `internal/crap/crap_test.go` — verify the stderr warning message contains the unresolvable profile entry name for diagnostic clarity. (FR-008)

### Implementation for User Story 3

- [x] T021 [US3] In `internal/crap/coverage.go`, `ParseCoverProfile` function: add `totalEntries` and `resolvedEntries` counters. Increment `totalEntries` for each profile entry. Increment `resolvedEntries` when `resolveFilePath` returns a non-empty string. (FR-008, FR-009)
- [x] T022 [US3] In `internal/crap/coverage.go`, `ParseCoverProfile` function: when `resolveFilePath` returns `""`, write warning to stderr: `fmt.Fprintf(stderr, "warning: skipping profile entry %q: could not resolve to file on disk\n", profile.FileName)` — only if stderr is non-nil (matching existing nil-check pattern at line ~72). (FR-008)
- [x] T023 [US3] In `internal/crap/coverage.go`, `ParseCoverProfile` function: after the profile loop, add check: if `totalEntries > 0 && resolvedEntries == 0`, return `nil, fmt.Errorf("all %d coverage profile entries failed to resolve — check that the profile matches the current module", totalEntries)`. (FR-009)
- [x] T024 [US3] Verify `TestResolveFilePath_NoGoMod` in `internal/crap/crap_test.go` (line ~752) still passes — the test creates a temp dir with no `go.mod` and asserts `resolveFilePath` returns `""`, which remains correct since `resolveFilePath` itself does not walk up (callers provide the module root). Add a GoDoc comment clarifying this tests the "outside any module" contract boundary. (FR-012)

**Checkpoint**: `go test -race -count=1 -short ./internal/crap/...` passes with all diagnostic tests green (SC-005, SC-006).

---

## Phase 5: User Story 4 - Clear Error When No Module Root Found (Priority: P3)

**Goal**: Running `gaze crap` from outside any Go module produces a clear error with non-zero exit code.

**Independent Test**: Run from a directory with no ancestor `go.mod`; verify non-zero exit and descriptive error message.

### Implementation for User Story 4

- [x] T025 [US4] Verify that `FindModuleRoot` error propagation is correct in T006 (`newCrapCmd` RunE) and T014 (`runReport`): when `FindModuleRoot` returns error, the error should propagate to Cobra which prints it and exits non-zero. No additional code needed if T006/T014 return the error correctly — this task validates the error path end-to-end. (FR-004, SC-003)
- [x] T026 [US4] Add `TestNewCrapCmd_NoModuleRoot` in `cmd/gaze/main_test.go` (or verify via existing test infrastructure): confirm that when `FindModuleRoot` returns an error, `newCrapCmd` RunE returns a non-nil error containing "go.mod" or "module root". This may require a test helper that temporarily changes the working directory to a temp dir without `go.mod`, or uses the existing `moduleRootFunc` injection pattern from `selfCheckParams`. (FR-004, SC-003)

**Checkpoint**: Error path verified — non-zero exit code and descriptive message (SC-003).

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Regression verification, lint checks, and documentation validation

- [x] T027 Run `go build ./cmd/gaze` — verify build succeeds with all changes
- [x] T028 Run `go test -race -count=1 -short ./...` — verify ALL existing tests pass (SC-004)
- [x] T029 Run `golangci-lint run` — verify no lint issues introduced
- [x] T030 Verify no unused imports remain after removing `os.Getwd()` calls from internal packages (`internal/crap/contract.go`, `internal/aireport/runner_steps.go`)
- [x] T031 Review GoDoc comments on all modified functions — ensure parameter documentation matches new signatures
- [x] T032 Run quickstart.md validation: execute `gaze crap ./...` from module root (baseline), then from `internal/crap/` subdirectory — verify identical coverage values for overlapping functions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — can start immediately. BLOCKS all user stories.
- **User Story 1 (Phase 2)**: Depends on Phase 1 (`FindModuleRoot` must exist before CLI can use it)
- **User Story 2 (Phase 3)**: Depends on Phase 1. Can run in parallel with US1 (different files: `runner_steps.go` vs `contract.go`/`main.go`)
- **User Story 3 (Phase 4)**: Depends on Phase 1 (`ParseCoverProfile` uses `FindModuleRoot` fallback). Tests can start immediately (T018-T020 are independent)
- **User Story 4 (Phase 5)**: Depends on Phase 2 (T006 must be complete for error path to exist)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 1 only. Core bug fix — MVP scope.
- **US2 (P2)**: Depends on Phase 1 only. Can run in parallel with US1 (different files).
- **US3 (P2)**: Depends on Phase 1 only. Can run in parallel with US1 and US2 (different code path in `coverage.go`).
- **US4 (P3)**: Depends on US1 (T006 establishes the error propagation path).

### Within Each User Story

- Signature changes before caller updates (within US1: T010-T011 before T012)
- CLI entry points after internal package changes (T006-T009 depend on T010-T013 being compilable)

### Parallel Opportunities

- **Phase 1**: T003, T004, T005 can run in parallel (independent test cases)
- **Phase 2 + Phase 3**: US1 and US2 can run in parallel after Phase 1 (different files)
- **Phase 4**: T018, T019, T020 can run in parallel (independent test files)
- **Cross-story**: US1, US2, US3 can all start after Phase 1 if three developers are available

---

## Parallel Example: After Phase 1 Completes

```text
# Developer A: User Story 1 (gaze crap fix)
Task: T006 — newCrapCmd RunE in cmd/gaze/main.go
Task: T010 — classifyResults in internal/crap/contract.go
Task: T011 — loadGazeConfigBestEffort in internal/crap/contract.go

# Developer B: User Story 2 (gaze report fix)
Task: T014 — runReport in cmd/gaze/main.go
Task: T015 — loadGazeConfigBestEffort in internal/aireport/runner_steps.go
Task: T016 — resolveModulePackages in internal/aireport/runner_steps.go

# Developer C: User Story 3 (diagnostics)
Task: T018 — TestParseCoverProfile_AllUnresolved in internal/crap/crap_test.go
Task: T021 — Add counters in internal/crap/coverage.go
Task: T022 — Add warnings in internal/crap/coverage.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Foundational (`FindModuleRoot` + tests)
2. Complete Phase 2: User Story 1 (crap command fix)
3. **STOP and VALIDATE**: `gaze crap ./...` from subdirectory produces correct results
4. This alone fixes the primary reported bug (issue #113)

### Incremental Delivery

1. Phase 1 → Foundation ready
2. Phase 2 (US1) → `gaze crap` fixed → Validate SC-001 (MVP)
3. Phase 3 (US2) → `gaze report` fixed → Validate SC-002
4. Phase 4 (US3) → Silent failure prevention → Validate SC-005, SC-006
5. Phase 5 (US4) → Error path → Validate SC-003
6. Phase 6 → Polish → Full CI validation (SC-004)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US1 and US2 both modify `cmd/gaze/main.go` — if done sequentially, US2 builds on US1's changes; if done in parallel, coordinate on `main.go` changes
- T009 (delete `findModuleRoot`) should be done AFTER T006 verifies the replacement works
- The `NOTE: keep in sync` comments on `loadGazeConfigBestEffort` copies should be preserved until full consolidation (deferred from spec 022)

<!-- spec-review: passed -->
<!-- code-review: passed -->
