<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file —
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Add empty-patterns guard to both resolvePackagePaths copies

- [x] 1.1 [P] Add early return guard to `resolvePackagePaths` in `internal/aireport/runner_steps.go` — if `len(patterns) == 0`, return `nil, nil` before calling `packages.Load`. Insert guard after the function signature, before the `packages.Config` construction.
- [x] 1.2 [P] Add the same early return guard to `resolvePackagePaths` in `internal/crap/contract.go` — identical 3-line guard.

## 2. Add symmetric test coverage

- [x] 2.1 Add `TestResolvePackagePaths_EmptyPatterns` to `internal/crap/contract_test.go`, matching the existing test pattern in `internal/aireport/runner_steps_test.go:35`. Guard with `testing.Short()` skip. Assert that empty input produces empty output with nil error.

## 3. Verification

- [x] 3.1 Run `go test -race -count=1 -short ./internal/aireport/` and verify `TestResolvePackagePaths_EmptyPatterns` passes.
- [x] 3.2 Run `go test -race -count=1 -short ./internal/crap/` and verify `TestResolvePackagePaths_EmptyPatterns` passes.
- [x] 3.3 Run `go test -race -count=1 -short ./...` and verify no regressions.
- [x] 3.4 Run `golangci-lint run` and verify no lint violations.
