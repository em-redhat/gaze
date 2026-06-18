## ADDED Requirements

### Requirement: Scope-Aware MapMutation Detection

`detectAssignEffects` MUST only emit a MapMutation effect when the map
variable is externally observable (parameter, receiver, named return, or
package-level variable). Writes to locally-created maps (`make`, `var`, `:=`)
MUST NOT produce MapMutation effects.

#### Scenario: Map parameter mutation detected

- **GIVEN** a function with a `map[string]int` parameter `m`
- **WHEN** the function contains `m["key"] = 42`
- **THEN** a MapMutation P1 effect is emitted for `m`

#### Scenario: Local map mutation suppressed

- **GIVEN** a function that creates a local map with `m := make(map[string]int)`
- **WHEN** the function contains `m["key"] = 42`
- **THEN** no MapMutation effect is emitted

#### Scenario: Local map returned does not trigger MapMutation

- **GIVEN** a function that creates a local map, writes to it, and returns it
- **WHEN** `AnalyzeP1Effects` is called
- **THEN** no MapMutation effect is emitted (the return value is covered by
  ReturnValue, not MapMutation)

---

### Requirement: Scope-Aware SliceMutation Detection

`detectAssignEffects` MUST only emit a SliceMutation effect when the slice
variable is externally observable. Writes to locally-created slices MUST NOT
produce SliceMutation effects.

#### Scenario: Slice parameter mutation detected

- **GIVEN** a function with a `[]int` parameter `s`
- **WHEN** the function contains `s[0] = 99`
- **THEN** a SliceMutation P1 effect is emitted for `s`

#### Scenario: Local slice mutation suppressed

- **GIVEN** a function that creates a local slice with `s := make([]int, 3)`
- **WHEN** the function contains `s[0] = 42`
- **THEN** no SliceMutation effect is emitted

---

### Requirement: Scope-Aware ChannelSend Detection

`detectSendEffects` MUST only emit a ChannelSend effect when the channel
variable is externally observable. Sends on locally-created channels MUST NOT
produce ChannelSend effects.

#### Scenario: Channel parameter send detected

- **GIVEN** a function with a `chan<- int` parameter `ch`
- **WHEN** the function contains `ch <- 42`
- **THEN** a ChannelSend P1 effect is emitted for `ch`

#### Scenario: Local channel send suppressed

- **GIVEN** a function that creates a local channel with `ch := make(chan int, 1)`
- **WHEN** the function contains `ch <- 42`
- **THEN** no ChannelSend effect is emitted

---

### Requirement: Scope-Aware ChannelClose Detection

`detectP1CallEffects` MUST only emit a ChannelClose effect when the channel
variable is externally observable. Closing locally-created channels MUST NOT
produce ChannelClose effects.

#### Scenario: Channel parameter close detected

- **GIVEN** a function with a `chan int` parameter `ch`
- **WHEN** the function contains `close(ch)`
- **THEN** a ChannelClose P1 effect is emitted for `ch`

#### Scenario: Local channel close suppressed

- **GIVEN** a function that creates a local channel with `ch := make(chan int)`
- **WHEN** the function contains `close(ch)`
- **THEN** no ChannelClose effect is emitted

---

### Requirement: Conservative Default for Unresolvable Expressions

When `isExternallyObservable` cannot resolve an expression to a `types.Object`
(nil type info, unresolved identifier, non-variable object), it MUST return
`true` (assume externally observable). This ensures false negatives are
avoided when type resolution fails.

#### Scenario: Nil type info

- **GIVEN** `info` is nil
- **WHEN** `isExternallyObservable` is called
- **THEN** it returns `true`

#### Scenario: Unresolved identifier

- **GIVEN** an identifier that is not present in `info.Uses`
- **WHEN** `isExternallyObservable` is called
- **THEN** it returns `true`

---

### Requirement: Known Limitation — Slice Aliasing

Slice aliasing (a local slice sub-sliced and returned, sharing the backing
array) is a known limitation. The scope check MAY produce a false negative
for this pattern. This SHOULD be documented in code comments.

---

## MODIFIED Requirements

### Requirement: `detectSendEffects` Signature

Previously: `detectSendEffects` did not receive `info *types.Info`.

`detectSendEffects` MUST accept `info *types.Info` as a parameter so that
`isExternallyObservable` can resolve channel variable scope.

---

## REMOVED Requirements

None.
