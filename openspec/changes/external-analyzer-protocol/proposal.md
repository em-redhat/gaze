## Why

Gaze's core value proposition — side effect detection, contract classification, CRAP scoring, test quality assessment — is language-agnostic. Phase 1 (`provider-interfaces`, PR #165) extracted four provider interfaces that decouple the universal scoring engine from Go-specific analysis:

| Interface | Purpose |
|-----------|---------|
| `ComplexityProvider` | Per-function cyclomatic complexity |
| `LineCoverageProvider` | Per-function line coverage |
| `SideEffectAnalyzer` | Side effect detection + classification |
| `ContractCoverageProvider` | Contract coverage lookup (quality pipeline) |

With these interfaces in place, gaze can accept analysis data from any source — not just Go tooling. The [snake-eyes](https://github.com/zero-dot-force/snake-eyes) project needs Python analysis using the same scoring engine. Other languages (TypeScript, Rust) follow the same pattern.

This change adds the **external analyzer protocol** — a JSON-RPC 2.0 over stdin/stdout transport that lets gaze spawn an external language analyzer process, exchange analysis data through the provider interfaces, and produce CRAP scores, GazeCRAP scores, quadrant classifications, fix strategies, and reports for any language.

This is Phase 2 of Issue #95.

## What Changes

### New: JSON-RPC Protocol (`internal/protocol/`)

A new package implementing JSON-RPC 2.0 client over stdin/stdout. Gaze spawns an external analyzer binary, sends requests, and receives responses through the protocol.

**Protocol methods** (from Issue #95):

| Method | Required | Direction | Purpose |
|--------|----------|-----------|---------|
| `initialize` | Yes | gaze → analyzer | Handshake: root path, config, capabilities |
| `discover` | No | gaze → analyzer | Find source files, test files, framework |
| `analyze` | Yes | gaze → analyzer | Detect side effects per function |
| `complexity` | Yes | gaze → analyzer | Cyclomatic complexity per function |
| `coverage` | Yes | gaze → analyzer | Parse coverage data |
| `test_mapping` | No | gaze → analyzer | Map test assertions to side effects |
| `classify_signals` | No | gaze → analyzer | Language-specific classification signals |
| `shutdown` | Yes | gaze → analyzer | Clean process exit |

**Capability negotiation**: The `initialize` response declares which optional methods the analyzer supports. Gaze adapts its pipeline accordingly (e.g., skips contract coverage if `test_mapping` is not supported).

### New: External Analyzer Adapter (`internal/adapter/`)

A new package that implements the four Phase 1 provider interfaces by translating protocol responses into the expected types:

- `ExternalComplexityProvider` — calls `complexity` method, returns `[]FunctionComplexity`
- `ExternalLineCoverageProvider` — calls `coverage` method, returns `[]FuncCoverage`
- `ExternalSideEffectAnalyzer` — calls `analyze` + optionally `classify_signals`, returns `[]taxonomy.AnalysisResult`
- `ExternalContractCoverageProvider` — calls `test_mapping` + computes coverage, returns lookup function

### New: Analyzer Discovery

Analyzers are discovered via:

1. **CLI flag**: `--analyzer <name>` explicitly names the binary
2. **Config**: `.gaze.yaml` `analyzers` section maps languages to analyzer binaries
3. **PATH convention**: `gaze-analyzer-<language>` naming convention (fallback)

### New: CLI Flags

- `--analyzer <name>` on `crap`, `quality`, `report` commands
- `--language <lang>` for explicit language selection (auto-detected otherwise)

Note: `gaze analyze --analyzer` is deferred to a future phase (see design.md D12).

### Modified: CLI Commands

When `--analyzer` is specified, commands construct external provider adapters instead of Go providers. The scoring pipeline (`crap.Analyze`) receives the external adapters through the same `Options.ComplexityProvider`, `Options.LineCoverageProvider`, `Options.ContractCoverageProvider` fields — the scoring core is unchanged.

## Capabilities

### New Capabilities
- `external-analyzer-protocol`: JSON-RPC 2.0 over stdin/stdout protocol for spawning external language analyzers and exchanging analysis data
- `analyzer-discovery`: Three-tier discovery mechanism (CLI flag, .gaze.yaml config, PATH convention) for finding analyzer binaries
- `external-provider-adapters`: Provider interface implementations that bridge the JSON-RPC protocol to the Phase 1 interfaces
- `capability-negotiation`: Protocol handshake where analyzers declare which optional methods they support
- `multi-language-crap`: CRAP scoring, GazeCRAP, quadrants, and fix strategies for any language with an analyzer

### Modified Capabilities
- `gaze crap`: Extended with `--analyzer` and `--language` flags
- `gaze quality`: Extended with `--analyzer` and `--language` flags
- `gaze report`: Extended with `--analyzer` and `--language` flags
- `.gaze.yaml`: Extended with `analyzers` section for language-to-binary mapping

### Removed Capabilities
- None

## Impact

- **New packages**: `internal/protocol/` (~5 files), `internal/adapter/` (~6 files)
- **Modified packages**: `cmd/gaze/` (CLI flags, analyzer dispatch), `internal/config/` (analyzers config section)
- **New config options**: `analyzers` section in `.gaze.yaml`
- **CLI changes**: New `--analyzer` and `--language` flags on 3 commands (`crap`, `quality`, `report`)
- **Backward compatible**: When no external analyzer is configured, gaze behaves exactly as it does today (built-in Go analysis via `goprovider`)
- **New external dependency**: None — JSON-RPC 2.0 is simple enough to implement without a library

## Phase 1 Foundation (from `provider-interfaces`)

This change builds directly on Phase 1's provider interfaces:

- **Interface definitions**: `internal/crap/provider.go` — `ComplexityProvider`, `LineCoverageProvider`, `SideEffectAnalyzer`, `ContractCoverageProvider`, `FunctionComplexity`
- **Go adapters**: `internal/provider/goprovider/` — default implementations used when no external analyzer is configured
- **Scoring core decoupled**: `internal/crap/analyze.go` has zero Go-specific imports. `computeScores`, `Formula`, `ClassifyQuadrant`, `buildSummary`, `assignFixStrategy` operate on language-neutral types.
- **Key design decisions carried forward**:
  - D1: Interfaces in `internal/crap/provider.go` (no import cycles)
  - D3: Callers construct providers at call sites
  - D5: `SideEffectAnalyzer` consumed by `ContractCoverageProvider`, not `crap.Options`
  - D7: Deprecated `ContractCoverageFunc`/`SSADegradedPackages` preserved for backward compat

## Deferred Decisions (from Issue #95)

1. **Taxonomy evolution**: Deferred to Phase 3. Phase 2 uses option (b): map language-specific concepts to existing gaze taxonomy types (e.g., Python `yield` → `ReturnValue`). The current 38 types include 9 Go-specific ones that are valid but unlikely from non-Go analyzers.
2. **Protocol versioning**: Resolved in D3/D8 — semver in `initialize` response (`protocol_version` field). `minProtocolVersion`/`maxProtocolVersion` negotiation deferred to when a breaking protocol change is needed.
3. **Error handling**: Resolved in D7 — graceful degradation for optional methods, hard error for required methods. Consistent with existing SSA failure handling.
4. **Streaming**: Deferred. Batch only in Phase 2. Streaming (JSONL per function) can be added as a future protocol extension via a new `analyze/stream` method without breaking changes.

## Constitution Alignment

Assessed against the Gaze project constitution (`.specify/memory/constitution.md` v1.3.0).

### I. Accuracy

**Assessment**: PASS

The protocol transmits raw analysis data (effects, coverage, complexity) to gaze's existing scoring engine. Accuracy of the universal scoring is preserved — `Formula`, `ClassifyQuadrant`, `computeScores` are unchanged. Language-specific accuracy depends on the quality of the external analyzer, which is tested independently. Mock analyzers validate the protocol-to-provider bridge.

### II. Minimal Assumptions

**Assessment**: PASS

The protocol makes minimal assumptions about the analyzer's language or implementation. Analyzers report standardized effect types, coverage data, and complexity metrics. No language-specific knowledge is required in gaze's core. When no analyzer is configured, behavior is identical to today.

### III. Actionable Output

**Assessment**: PASS

Reports produced from external analyzer data use the same formatting, fix strategies, and quadrant classification as Go analysis. The output is equally actionable regardless of source language.

### IV. Testability

**Assessment**: PASS

The JSON-RPC protocol is inherently testable: mock analyzers can be written as simple scripts that emit canned responses. Each protocol method can be tested in isolation. The adapter layer can be tested without spawning real analyzer processes — it receives parsed JSON and produces provider interface types.