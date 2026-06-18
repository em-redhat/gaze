## 1. Add Scope-Checking Helpers

- [x] 1.1 Add `unwrapToIdent` helper in `internal/analysis/p1effects.go` that unwraps `*ast.SelectorExpr`, `*ast.IndexExpr`, `*ast.StarExpr`, and `*ast.ParenExpr` chains to find the base `*ast.Ident`. Returns nil if the expression cannot be unwrapped to an identifier.
- [x] 1.2 Add `isExternallyObservable` helper in `internal/analysis/p1effects.go` that resolves an `ast.Expr` via `unwrapToIdent` → `info.Uses` → `types.Object` and checks the scope hierarchy. Returns `true` for parameters (parent scope's parent is package scope), named returns (same scope as params), receivers (same scope as params), package-level variables (parent's parent is Universe), and unresolvable expressions (conservative). Returns `false` for body-local variables. Scope depth: package-level = 2 levels to Universe, param/receiver/named-return = 3 levels, body-local = 4+ levels.
- [x] 1.3 Add code comments on `isExternallyObservable` documenting known limitations: (a) slice aliasing — local slice sub-sliced and returned may produce false negatives; (b) closure capture — a local variable captured by a returned closure is not detected as externally observable.

## 2. Gate Effect Detection Behind Scope Check

- [x] 2.1 In `detectAssignEffects`, gate MapMutation emission (line ~100) with `isExternallyObservable(info, idx.X)`. Only emit MapMutation when the map variable is externally observable.
- [x] 2.2 In `detectAssignEffects`, gate SliceMutation emission (line ~116) with `isExternallyObservable(info, idx.X)`. Only emit SliceMutation when the slice variable is externally observable.
- [x] 2.3 Add `info *types.Info` parameter to `detectSendEffects` signature. Update the call site in `AnalyzeP1Effects` to pass `info`.
- [x] 2.4 In `detectSendEffects`, gate ChannelSend emission with `isExternallyObservable(info, node.Chan)`. Only emit ChannelSend when the channel is externally observable.
- [x] 2.5 In `detectP1CallEffects`, gate ChannelClose emission (line ~214) with `isExternallyObservable(info, node.Args[0])`. Only emit ChannelClose when the channel is externally observable.

## 3. Add Test Fixtures

- [x] 3.1 Add `LocalMapWrite` fixture to `internal/analysis/testdata/src/p1effects/p1effects.go`: creates a local map with `make`, writes to it, returns it. Should NOT produce any P1 effects.
- [x] 3.2 Add `LocalSliceWrite` fixture: creates a local slice with `make`, writes to it, returns it. Should NOT produce any P1 effects.
- [x] 3.3 Add `LocalChannelSend` fixture: creates a local buffered channel with `make(chan int, 1)`, sends on it, receives from it. Should NOT produce any P1 effects.
- [x] 3.4 Add `LocalChannelClose` fixture: creates a local channel with `make(chan int)`, closes it. Should NOT produce any P1 effects.
- [x] 3.5 Add `NamedReturnMapWrite` fixture: function with named return `result map[string]int`, creates with `make`, writes to it, returns. SHOULD produce MapMutation (named return is externally observable).
- [x] 3.6 Add `WriteToStructMap` fixture: method on a struct with a map field, writes to `self.M["key"]` where `self` is the receiver. SHOULD produce MapMutation (exercises `unwrapToIdent` SelectorExpr path).

## 4. Add Tests

- [x] 4.1 Add `TestAnalyzeP1Effects_Direct_LocalMapWrite`: verify `LocalMapWrite` produces exactly 0 P1 effects (not just zero MapMutation — assert `len(effects) == 0`). Verify existing `WriteToMap` (parameter) still produces MapMutation.
- [x] 4.2 Add `TestAnalyzeP1Effects_Direct_LocalSliceWrite`: verify `LocalSliceWrite` produces exactly 0 P1 effects. Verify existing `WriteToSlice` (parameter) still produces SliceMutation.
- [x] 4.3 Add `TestAnalyzeP1Effects_Direct_LocalChannelSend`: verify `LocalChannelSend` produces exactly 0 P1 effects. Verify existing `SendOnChannel` (parameter) still produces ChannelSend.
- [x] 4.4 Add `TestAnalyzeP1Effects_Direct_LocalChannelClose`: verify `LocalChannelClose` produces exactly 0 P1 effects. Verify existing `CloseChannel` (parameter) still produces ChannelClose.
- [x] 4.5 Add `TestAnalyzeP1Effects_Direct_NamedReturnMapWrite`: verify `NamedReturnMapWrite` produces MapMutation (named return is externally observable). Validates the FuncType scope detection path.
- [x] 4.6 Add `TestAnalyzeP1Effects_Direct_WriteToStructMap`: verify `WriteToStructMap` produces MapMutation (receiver field access via SelectorExpr). Validates the `unwrapToIdent` SelectorExpr unwrapping path.
- [x] 4.7 Verify all 9 existing P1 effects tests still pass: `TestAnalyzeP1Effects_Direct_GlobalMutation`, `_TwoGlobals`, `_ChannelSend`, `_ChannelClose`, `_WriterOutput`, `_HTTPResponseWrite`, `_MapMutation`, `_SliceMutation`, `_PureFunction`, `_NilBody`.

## 5. Cross-Cutting Verification

- [x] 5.1 Run `go build ./...` — clean build.
- [x] 5.2 Run `go test -race -count=1 -short ./internal/analysis/...` — all tests pass.
- [x] 5.3 Run `go test -race -count=1 -short ./...` — full suite passes.
- [x] 5.4 Run `golangci-lint run` — no new lint violations.
- [x] 5.5 Verify GazeCRAPload improvement: run `go run ./cmd/gaze crap --format=json ./...` and check that `summary.gaze_crapload` decreased (CI threshold at `--max-gaze-crapload=6` will validate automatically).
- [x] 5.6 Verify constitution alignment: Principle I (Accuracy) — confirm false positives are eliminated for local variables while true positives (parameter, named return, receiver, package-level variables) are preserved.
- [x] 5.7 Update AGENTS.md "Recent Changes" section with a `p1-local-var-false-positives` entry.

<!-- spec-review: passed -->
<!-- code-review: passed -->
