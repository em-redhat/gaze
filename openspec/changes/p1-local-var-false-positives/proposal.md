## Why

`AnalyzeP1Effects` in `internal/analysis/p1effects.go` produces false positive
side effects for four P1 effect types — MapMutation, SliceMutation,
ChannelSend, and ChannelClose — when the target variable is a locally-created
value that never escapes the function scope. A map created with `make()` and
only used as an internal lookup table is not an "externally detectable change,"
yet gaze flags every write to it as a P1 side effect.

This affects real-world analysis: `Compare`, `WriteComparisonJSON`, and
`BuildContractCoverageFunc` in `internal/crap/` all create local maps for
internal bookkeeping and get flagged with bogus MapMutation effects, inflating
their side effect counts, reducing contract coverage, and pushing GazeCRAPload
from 4 to 6 in CI.

The root cause is consistent: `detectAssignEffects` checks type (is this a
map? a slice?) but not scope (is this variable a parameter, receiver, or
package-level variable?). Compare with `GlobalMutation` detection in the same
file, which correctly uses `isGlobalIdent` to verify the variable is
package-scoped before emitting the effect.

Closes #121 (MapMutation), #144 (SliceMutation), #145 (ChannelSend/ChannelClose).

## What Changes

Add a scope-checking helper function `isExternallyObservable` to
`internal/analysis/p1effects.go` that resolves an expression to its
`types.Object` and determines whether the variable is externally observable
(parameter, receiver, or package-level variable) or body-local (`:=`, `var`,
`make`). Gate the emission of MapMutation, SliceMutation, ChannelSend, and
ChannelClose effects behind this check.

Thread the `info *types.Info` parameter to `detectSendEffects` (which
currently doesn't receive it) so the scope check can be performed for
channel send operations.

Add test fixtures and negative test cases for all four effect types with
local variables.

## Capabilities

### New Capabilities
- `scope-aware-p1-detection`: MapMutation, SliceMutation, ChannelSend, and ChannelClose effects are only emitted when the target variable is externally observable (parameter, receiver, or package-level variable). Locally-created variables are excluded.

### Modified Capabilities
- `AnalyzeP1Effects`: Now performs scope-aware detection for map, slice, and channel effects, consistent with the existing scope check for GlobalMutation.
- `detectSendEffects`: Gains `info *types.Info` parameter for scope resolution.

### Removed Capabilities
- None.

## Impact

- **Files modified**: `internal/analysis/p1effects.go`, `internal/analysis/p1effects_test.go`, `internal/analysis/testdata/src/p1effects/p1effects.go`
- **Effect count changes**: Functions with locally-created maps, slices, or channels will have fewer reported P1 effects. This reduces false positive counts and improves contract coverage accuracy.
- **GazeCRAPload impact**: Expected to drop from 6 to 4 or lower, enabling CI threshold reversion from `--max-gaze-crapload=6` to `--max-gaze-crapload=5`.
- **No API surface changes**: `AnalyzeP1Effects` signature is unchanged. Only internal helper signatures change.
- **No new dependencies**: Uses existing `go/types` infrastructure.

## Constitution Alignment

Assessed against the Gaze project constitution (v1.3.0).

### I. Accuracy

**Assessment**: PASS

This is the primary principle served. The constitution states: "false positives
erode trust and MUST be treated as bugs." Local variable mutations are not
externally detectable changes — reporting them as P1 side effects is a false
positive. The fix restores accuracy by applying the same scope-checking
discipline that `GlobalMutation` already uses.

### II. Minimal Assumptions

**Assessment**: PASS

No new assumptions introduced. The fix uses Go's type system (`types.Object`
scope resolution) to determine variable provenance — no heuristics, naming
conventions, or annotations required.

### III. Actionable Output

**Assessment**: PASS

Removing false positive effects makes the remaining effects more actionable.
A function with 1 real P0 ReturnValue effect instead of 1 ReturnValue + 2
bogus MapMutations gives the user a clearer picture of what to test.

### IV. Testability

**Assessment**: PASS

Each effect type gets dedicated positive and negative test fixtures. The
`isExternallyObservable` helper is directly testable via the existing
`AnalyzeP1Effects` test infrastructure. Coverage strategy: unit tests for
each modified detection path.
