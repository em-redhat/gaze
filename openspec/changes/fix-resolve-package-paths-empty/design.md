## Context

`resolvePackagePaths` exists as two identical copies — one in `internal/aireport/runner_steps.go:246` and one in `internal/crap/contract.go:153`. Both call `packages.Load(cfg, patterns...)` without guarding against empty input. When `patterns` is an empty slice, `packages.Load` defaults to loading the current directory package (`"."`), producing a non-empty result that contradicts the function's implicit contract of "empty in, empty out."

The existing test `TestResolvePackagePaths_EmptyPatterns` in `internal/aireport/runner_steps_test.go:35` asserts this contract but fails because of the missing guard. The `crap` package has no equivalent empty-patterns test.

Production callers guard against empty patterns upstream, so the bug is latent in production but breaks `go test ./...` on a clean `main` checkout. Per the proposal's constitution alignment: the fix is Composability-safe (no new dependencies), Observable Quality-safe (no output format changes), and Testability-positive (restores a broken test, adds symmetric coverage).

## Goals / Non-Goals

### Goals
- Fix the empty-input edge case in both copies of `resolvePackagePaths`
- Restore `TestResolvePackagePaths_EmptyPatterns` to passing
- Add symmetric empty-patterns test to `internal/crap/contract_test.go`

### Non-Goals
- Consolidating the two copies into a shared implementation (deferred per existing comments referencing `specs/022-report-gazecrap-pipeline/tasks.md`)
- Adding `pkg.Errors` checking to `resolvePackagePaths` (separate concern from BUG_004)
- Modifying upstream callers or production behavior

## Decisions

**D1: Early-return guard over modifying test expectations.**

The function's contract should be "empty patterns → empty result." The test correctly captures this contract. Changing the test to accept `["."]` for empty input would make the function's behavior surprising and inconsistent with the upstream caller's assumption (which already rejects empty patterns before calling `resolvePackagePaths`). The fix is a 3-line guard at the top of the function:

```go
if len(patterns) == 0 {
    return nil, nil
}
```

**D2: Fix both copies independently rather than consolidating.**

The existing "keep in sync" comments and deferred consolidation note in `specs/022-report-gazecrap-pipeline/tasks.md` indicate consolidation is planned but out of scope. Applying the same 3-line fix to both copies is minimal, safe, and keeps the change focused on the bug.

**D3: Add symmetric test to `crap` package.**

The `crap` package has tests for valid patterns, test-suffix filtering, and invalid patterns — but not for empty patterns. Adding a `TestResolvePackagePaths_EmptyPatterns` to `internal/crap/contract_test.go` ensures both copies have equivalent edge-case coverage.

## Risks / Trade-offs

**Low risk.** The fix is a pure guard clause with no behavioral change for non-empty inputs. Production callers never pass empty patterns, so the fix addresses only the test contract gap. The only trade-off is continuing to maintain two copies of the function, but that is an existing accepted cost deferred to a future spec.
