## Context

P1 effect detection in `internal/analysis/p1effects.go` has four detection
sites that check type but not scope: MapMutation (line 100), SliceMutation
(line 116), ChannelSend (line 175), and ChannelClose (line 214). The existing
`GlobalMutation` detection (line 83) already has the correct pattern — it uses
`isGlobalIdent` to verify the variable is package-scoped before emitting.

The `isGlobalIdent` function resolves an `*ast.Ident` via `info.Uses` to its
`types.Object` and checks `v.Parent().Parent() == types.Universe` (package
scope). The fix follows the same approach but inverts the question: instead of
"is this a global?", we ask "is this externally observable?" (parameter,
receiver, or package-level variable).

## Goals / Non-Goals

### Goals
- Eliminate false positive MapMutation, SliceMutation, ChannelSend, and
  ChannelClose effects on body-local variables.
- Use the same `types.Object` scope resolution approach that `isGlobalIdent`
  already uses — no new analysis infrastructure.
- Add comprehensive test fixtures covering local vs. parameter variables for
  each effect type.
- Maintain conservative behavior: if scope cannot be resolved, assume the
  variable is externally observable (emit the effect rather than risk a false
  negative).

### Non-Goals
- Escape analysis (tracking whether a local variable is returned, stored in
  a struct field, or passed to another goroutine). This is a future enhancement.
- Fixing WriterOutput or HTTPResponseWrite scope (these are always on
  parameters by language convention — `io.Writer` and `http.ResponseWriter`
  are interface types received as parameters).
- Slice aliasing detection (a local slice sub-sliced and returned shares the
  backing array). Documented as a known limitation.

## Decisions

### D1: Single `isExternallyObservable` helper

**Decision**: Add one helper function that answers "is this expression
externally observable?" Used at all four detection sites.

**Rationale**: All four bugs share the same root cause (type check without
scope check), so a single helper avoids duplicating resolution logic.

**Design**:

```go
// isExternallyObservable returns true if expr refers to a variable
// that is observable from outside the function: a parameter, receiver,
// named return, or package-level variable. Returns false for body-local
// variables (make, var, :=). Returns true (conservative) when the
// expression cannot be resolved.
func isExternallyObservable(info *types.Info, expr ast.Expr) bool {
    ident := unwrapToIdent(expr)
    if ident == nil {
        return true // can't resolve selector/index — conservative
    }
    if info == nil {
        return true // no type info — conservative
    }
    obj := info.Uses[ident]
    if obj == nil {
        return true // unresolved — conservative
    }
    v, ok := obj.(*types.Var)
    if !ok {
        return true // not a variable — conservative
    }
    // Package-level variable: externally observable.
    if v.Parent() != nil && v.Parent().Parent() == types.Universe {
        return true
    }
    // Function signature-level (parameter, receiver, named return):
    // the parent scope is the function type scope, which is a child
    // of the package scope. Check if the variable is declared in a
    // scope whose parent is the package scope.
    if v.Parent() != nil && v.Parent().Parent() != nil &&
        v.Parent().Parent().Parent() == types.Universe {
        return true
    }
    // Body-local variable: not externally observable.
    return false
}
```

The scope hierarchy for Go functions is:
- Universe → Package → FuncType (params) → FuncBody (locals)

A parameter's `v.Parent()` is the FuncType scope, whose parent is the Package
scope, whose parent is Universe. A body-local's `v.Parent()` is the FuncBody
scope (or a nested block scope), which has more ancestors before reaching
Universe.

### D2: `unwrapToIdent` helper for expression resolution

**Decision**: Add a helper that unwraps selector expressions and index
expressions to find the base `*ast.Ident`.

**Rationale**: Map/slice mutations use `idx.X` which may be a bare identifier
(`m[key] = v`) or a selector (`s.field[key] = v`). Channel sends use
`node.Chan` which may be a selector. The helper handles these cases.

**Design**:

```go
func unwrapToIdent(expr ast.Expr) *ast.Ident {
    for {
        switch e := expr.(type) {
        case *ast.Ident:
            return e
        case *ast.SelectorExpr:
            expr = e.X
        case *ast.IndexExpr:
            expr = e.X
        default:
            return nil
        }
    }
}
```

### D3: Thread `info` to `detectSendEffects`

**Decision**: Add `info *types.Info` as a parameter to `detectSendEffects`.

**Rationale**: `detectSendEffects` currently doesn't receive `info`, so it
can't call `isExternallyObservable`. The caller (`AnalyzeP1Effects`) already
has `info` and passes it to `detectAssignEffects` and `detectP1CallEffects`.
Adding it to `detectSendEffects` is a one-line signature change + one-line
call-site change.

### D4: Conservative default

**Decision**: When scope cannot be resolved (nil info, unresolved identifier,
non-variable object), return `true` (externally observable).

**Rationale**: A false negative (missing a real side effect) is worse than a
false positive (reporting a non-existent one) from a safety perspective. The
constitution prioritizes driving false negatives toward zero. In practice,
type info is always available in gaze's analysis pipeline, so the conservative
path is rarely taken.

### D5: Slice aliasing — known limitation

**Decision**: Document slice aliasing as a known limitation, do not attempt
escape analysis.

**Rationale**: A locally-created slice that is sub-sliced and returned shares
the backing array. Mutations to the original slice after sub-slicing are
observable through the returned sub-slice. Detecting this requires escape
analysis (tracking whether the slice is returned, stored, or passed), which
is significantly more complex and belongs in a separate spec. The scope check
still improves accuracy for the common case (local slices used as scratch
buffers and never returned).

## Coverage Strategy

- **Unit tests**: One positive test (parameter variable → effect emitted) and
  one negative test (local variable → no effect) for each of the four effect
  types: MapMutation, SliceMutation, ChannelSend, ChannelClose. Plus additional
  positive tests for named return variables and receiver field access to cover
  the FuncType scope and SelectorExpr unwrapping paths.
- **Coverage target**: 100% branch coverage for `isExternallyObservable` and
  `unwrapToIdent`. All new code paths in `detectAssignEffects`,
  `detectSendEffects`, and `detectP1CallEffects` must be exercised by at least
  one positive and one negative test case.
- **Regression**: All 9 existing P1 effects tests must continue to pass —
  parameter-based fixtures produce the same effects as before.

## Risks / Trade-offs

### R1: Slice aliasing false negatives

Local slices that are sub-sliced and returned share the backing array.
Mutations to the original are observable. The scope check will suppress
these as "local," producing a false negative.

**Mitigation**: Document as known limitation. The common case (local scratch
slices) benefits from the fix. Aliasing detection is a future enhancement.

### R2: Selector expression resolution

`m.field[key] = v` where `m` is a parameter struct with a map field — the
`unwrapToIdent` will resolve to `m`, which is a parameter, so the effect is
correctly emitted. But `m.field[key] = v` where `m` is a local struct — the
effect should be suppressed. This is correct because `unwrapToIdent` returns
the base `m` which is local.

### R3: Channel passed to goroutine

A locally-created channel passed to a goroutine launched within the function
is not "externally observable" by our scope check, but the send/close effects
on it are real side effects (they synchronize with the goroutine). This is
another case where escape analysis would be needed.

**Mitigation**: Same as R1 — document as known limitation. The common case
(local channels used for internal coordination within the function) benefits.

### R4: Closure-captured variables

A locally-created map/slice/channel that is captured by a returned closure is
not "externally observable" by the scope check, but mutations through the
closure are observable from outside the function. Example:

```go
func Outer() func() {
    m := make(map[string]int)
    return func() { m["key"] = 42 }  // m is captured, mutation is observable
}
```

The scope check will classify `m` as body-local and suppress the MapMutation,
producing a false negative. Detecting this requires escape analysis (tracking
whether the variable is captured by a returned closure).

**Mitigation**: Document as known limitation alongside slice aliasing (R1) and
channel-to-goroutine (R3). Closure capture detection is a future enhancement.
Note: when `AnalyzeP1Effects` analyzes a parameter captured by an inner
closure, `info.Uses` correctly resolves to the outer function's parameter
object, so the scope check returns `true` — this common case is handled
correctly.
