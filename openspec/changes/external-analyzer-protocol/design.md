## Context

Phase 1 (`provider-interfaces`) established the interface boundary between Go-specific analysis and universal scoring. The four provider interfaces (`ComplexityProvider`, `LineCoverageProvider`, `SideEffectAnalyzer`, `ContractCoverageProvider`) and `FunctionComplexity` type live in `internal/crap/provider.go`. Go implementations in `internal/provider/goprovider/` wrap existing analysis code. The scoring core in `internal/crap/analyze.go` has zero Go-specific imports — it operates entirely on the provider interfaces.

Relevant existing patterns:
- `internal/aireport/adapter.go` — `runSubprocess` helper for spawning external processes with stdin/stdout/stderr capture
- `internal/aireport/adapter_opencode.go` — OpenCode adapter spawning `opencode` binary with `--dir` flag and stdin payload
- `internal/crap/provider.go` — the four provider interfaces external adapters must implement
- `internal/provider/goprovider/` — reference implementations showing how providers are constructed and used
- `internal/provider/mockprovider/` — mock implementations showing how to test providers with synthetic data
- `cmd/gaze/main.go` — caller sites that construct providers and pass them via `crap.Options`
- `internal/config/config.go` — `.gaze.yaml` configuration loading with nested struct patterns

The JSON-RPC 2.0 protocol was chosen (per Issue #95) because it uses the same stdin/stdout transport as LSP, is simple to implement without external dependencies, and provides structured request/response with error handling and capability negotiation.

## Goals / Non-Goals

### Goals
- Define a JSON-RPC 2.0 protocol with 8 methods for gaze ↔ analyzer communication
- Implement a JSON-RPC client in `internal/protocol/` for spawning and communicating with analyzer processes
- Implement provider adapters in `internal/adapter/` that translate protocol responses to Phase 1 interface types
- Add `--analyzer` and `--language` CLI flags to `crap`, `quality`, and `report` commands
- Add `analyzers` configuration section to `.gaze.yaml`
- Validate the protocol with a mock analyzer binary (test fixture)
- Document the protocol specification for external analyzer authors

### Non-Goals
- Building a Python analyzer (snake-eyes) — that's a separate project
- Taxonomy generalization (renaming Go-specific effect types) — deferred to a future spec
- Streaming protocol mode — batch responses only in Phase 2
- GUI or web-based analyzer management
- Analyzer auto-installation or version management
- Changes to the scoring core (`computeScores`, `Formula`, `ClassifyQuadrant`) — these are already language-neutral
- Removal of deprecated `ContractCoverageFunc`/`SSADegradedPackages` — Phase 3 cleanup

## Decisions

### D1: JSON-RPC 2.0 over stdin/stdout

The protocol uses JSON-RPC 2.0 (https://www.jsonrpc.org/specification) over stdin/stdout, consistent with the LSP transport model. The analyzer binary is spawned as a subprocess with `--stdio` flag. Gaze writes JSON-RPC requests to the analyzer's stdin and reads responses from stdout. Stderr is captured for diagnostics.

**Why not HTTP?** Subprocess stdin/stdout is simpler (no port allocation, no TLS, no connection management), works in sandboxed environments, and is the established pattern for language tools (LSP, DAP).

**Why not gRPC?** Adds a protobuf dependency, requires code generation, and is overkill for a request-response protocol with 8 methods.

### D2: Protocol lifecycle matches Issue #95

```
gaze analyze --analyzer snake-eyes ./src
│
├─ 1. Find analyzer binary (CLI flag, .gaze.yaml, PATH)
├─ 2. Spawn: snake-eyes --stdio
├─ 3. initialize → capabilities handshake
├─ 4. discover → find source + test files (optional)
├─ 5. analyze → detect side effects per function
├─ 6. complexity → cyclomatic complexity per function
├─ 7. coverage → parse language-specific coverage data
├─ 8. test_mapping → map assertions to effects (optional)
├─ 9. classify_signals → language-specific classification signals (optional)
├─ 10. shutdown → clean exit
└─ 11. Gaze computes CRAP, GazeCRAP, quadrants, fix strategies, renders reports
```

### D3: Capability negotiation via `initialize` response

The `initialize` response declares which optional methods the analyzer supports:

```json
{
  "capabilities": {
    "discover": true,
    "test_mapping": true,
    "classify_signals": false
  },
  "protocol_version": "1.0.0",
  "analyzer_name": "snake-eyes",
  "language": "python"
}
```

When `test_mapping` is false, gaze skips contract coverage computation (GazeCRAP is unavailable). When `classify_signals` is false, gaze uses the `SideEffectAnalyzer`'s built-in classification (effects arrive pre-classified from the `analyze` method).

### D4: External adapters implement Phase 1 interfaces

Each adapter type in `internal/adapter/` implements exactly one Phase 1 interface:

```
ExternalComplexityProvider   implements crap.ComplexityProvider
ExternalLineCoverageProvider implements crap.LineCoverageProvider
ExternalSideEffectAnalyzer   implements crap.SideEffectAnalyzer
ExternalContractCoverageProvider implements crap.ContractCoverageProvider
```

The adapters hold a reference to the protocol client. Their methods translate between protocol JSON messages and Go types (`FunctionComplexity`, `FuncCoverage`, `taxonomy.AnalysisResult`, `ContractCoverageInfo`).

Note: `ExternalSideEffectAnalyzer` is a composition dependency of
`ExternalContractCoverageProvider`, not a standalone adapter passed
to `crap.Options`. This matches Phase 1 D5. The `analyze` protocol
method returns per-function results for the whole project; the
adapter caches results and filters by pkgPath when `Analyze(pkgPath)`
is called.

### D5: Analyzer discovery order

1. `--analyzer <name>` CLI flag — explicit binary name
2. `.gaze.yaml` `analyzers.<language>.command` — config-based lookup
3. `gaze-analyzer-<language>` — PATH convention fallback

Language is determined by: (1) `--language` flag, (2) `.gaze.yaml` `analyzers` keys, (3) `initialize` response `language` field.

### D6: Mock analyzer for testing

A test binary at `internal/adapter/testdata/fake_analyzer/main.go` implements the protocol with canned responses. It reads JSON-RPC requests from stdin, returns hardcoded responses for each method. This validates the protocol client and adapter layer without requiring a real language analyzer.

This follows the established pattern: `internal/aireport/testdata/fake_opencode/main.go` does the same for the OpenCode AI adapter.

### D7: Error handling — graceful degradation

Consistent with the existing SSA degradation pattern and Phase 1's `ContractCoverageProvider` error handling:

- Analyzer binary not found → clear error, exit non-zero
- `initialize` handshake fails → clear error, exit non-zero
- Required method (`analyze`, `complexity`, `coverage`) returns error → propagate error, exit non-zero
- Optional method (`discover`, `test_mapping`, `classify_signals`) returns error → warn on stderr, degrade gracefully (skip that data)
- Analyzer process crashes mid-session → detect via stdin/stdout close, return error
- Malformed JSON response → parse error with context

### D8: Protocol message format

Request:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "analyze",
  "params": {
    "root_path": "/path/to/project",
    "patterns": ["./..."],
    "config": {}
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "functions": [...]
  }
}
```

Error:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid request",
    "data": "..."
  }
}
```

### D9: `analyze` response maps to `taxonomy.AnalysisResult`

The `analyze` method returns side effects using gaze's existing taxonomy types (the `SideEffectType` string constants). External analyzers map their language concepts to gaze's taxonomy:

- Python `yield` → `ReturnValue`
- Python `self.x = ...` → `ReceiverMutation`
- Python `open().write()` → `FileSystemWrite`
- Go-specific types (`ChannelSend`, `DeferredReturnMutation`) are valid but unlikely from non-Go analyzers

The analyzer returns effects with `Classification` already attached (consistent with Phase 1 D5 — `SideEffectAnalyzer` returns pre-classified results). Optionally, `classify_signals` provides raw signals that gaze's scoring engine (`ComputeScore`) can use for classification.

### D10: Protocol timeouts via context.Context

`Client.Call` accepts a `context.Context` parameter for deadline/timeout
control. The `Session` struct sets a per-method timeout (default 5 minutes
for `analyze`/`complexity`/`coverage`, 30 seconds for `initialize`/
`discover`/`shutdown`). When the context deadline expires, the client
kills the subprocess and returns a timeout error. This prevents hung
analyzers from blocking gaze indefinitely.

### D11: `discover` is optional

`discover` is optional in Phase 2. Its output (source files, test files,
framework) is not consumed by any adapter — the provider interfaces
accept patterns and rootDir directly. `discover` is reserved for future
use cases (auto-detecting language, populating IDE file lists). Analyzers
that don't implement `discover` are fully functional.

### D12: `gaze analyze --analyzer` deferred to future phase

The `gaze analyze` command calls `analysis.LoadAndAnalyze` directly, not
through `crap.Options`. Adding `--analyzer` to it requires a separate
integration path. Phase 2 adds `--analyzer` to `crap`, `quality`, and
`report` only — these all flow through `crap.Options` providers. The
`analyze` command is deferred to avoid splitting the implementation
across two integration patterns.

## Risks / Trade-offs

### R1: Protocol stability

**Risk**: Once published, the protocol becomes a public API. Changes break external analyzers.

**Mitigation**: Semantic versioning in `initialize` handshake. Version 1.0.0 for this spec. Breaking changes require major version bump. Optional methods allow non-breaking extensions.

### R2: Subprocess overhead

**Risk**: Spawning an external process adds latency compared to in-process Go analysis.

**Mitigation**: The subprocess is spawned once per analysis run. The JSON-RPC communication is sequential — no concurrent message storms. For large codebases, the analysis time dominates subprocess startup.

### R3: JSON serialization of large results

**Risk**: Large codebases could produce megabytes of JSON for the `analyze` response.

**Mitigation**: Batch response for Phase 2. Streaming (JSONL per function) can be added as a future protocol extension without breaking changes (new `analyze/stream` method). Most projects have <10K functions — JSON serialization is fast.

### R4: External analyzer quality

**Risk**: A buggy external analyzer produces incorrect data, leading to misleading CRAP scores.

**Mitigation**: Gaze validates incoming data structurally (required fields, valid types, score ranges). The `initialize` handshake includes protocol version — gaze can reject incompatible analyzers. Invalid `SideEffectType` values are reported as warnings.

## Coverage Strategy

- `internal/protocol/`: ≥80% line coverage via unit tests with fake analyzer binary. Tests cover: successful session, binary not found, mid-session crash, malformed JSON, timeout, error response.
- `internal/adapter/`: ≥80% line coverage via unit tests with mock protocol client. Tests cover: each adapter translates canned responses correctly, optional method skipped when capability is false, error propagation.
- `internal/config/`: existing coverage preserved; new `AnalyzersConfig` covered by config loading tests.
- `cmd/gaze/`: CLI integration covered by E2E test with fake analyzer binary (task 5.4). Build via `go build` in TestMain.
- Existing test suite regression: zero failures (task 6.1, 6.2).
- No new ratchet — coverage targets are verified manually until the packages stabilize.