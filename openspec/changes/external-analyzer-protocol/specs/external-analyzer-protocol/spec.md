## ADDED Requirements

### Requirement: JSON-RPC Protocol Client

The `protocol` package MUST implement a JSON-RPC 2.0 client that spawns an external analyzer binary as a subprocess, communicates via stdin/stdout, and supports the 8 protocol methods defined in Issue #95 (5 required: `initialize`, `analyze`, `complexity`, `coverage`, `shutdown`; 3 optional: `discover`, `test_mapping`, `classify_signals`). The client MUST handle request ID sequencing, response matching, error responses, subprocess lifecycle (spawn, communicate, shutdown), and per-method timeouts via `context.Context`.

#### Scenario: Successful protocol session

- **GIVEN** a mock analyzer binary that responds to all required protocol methods and both optional methods (`test_mapping`, `classify_signals`)
- **WHEN** the protocol client spawns the binary and executes a full session (initialize → analyze → complexity → coverage → test_mapping → classify_signals → shutdown)
- **THEN** all responses MUST be correctly parsed and the subprocess MUST exit cleanly

#### Scenario: Analyzer binary not found

- **GIVEN** a non-existent analyzer binary name
- **WHEN** the protocol client attempts to spawn it
- **THEN** the client MUST return an error indicating the binary was not found in PATH

#### Scenario: Analyzer crashes mid-session

- **GIVEN** an analyzer binary that exits unexpectedly after responding to `initialize`
- **WHEN** the protocol client sends the next request
- **THEN** the client MUST detect the closed stdin/stdout and return an error with context

---

### Requirement: External Provider Adapters

The `adapter` package MUST implement four types that satisfy the Phase 1 provider interfaces (`ComplexityProvider`, `LineCoverageProvider`, `SideEffectAnalyzer`, `ContractCoverageProvider`) by translating JSON-RPC protocol responses into the expected Go types. Each adapter MUST hold a reference to the protocol client and call the appropriate protocol method in its interface method.

#### Scenario: Complexity adapter translates protocol response

- **GIVEN** an `ExternalComplexityProvider` connected to a mock analyzer
- **WHEN** `Analyze` is called
- **THEN** the adapter MUST call the `complexity` protocol method and return `[]crap.FunctionComplexity` with correctly mapped fields

#### Scenario: Coverage adapter translates protocol response

- **GIVEN** an `ExternalLineCoverageProvider` connected to a mock analyzer
- **WHEN** `Coverage` is called
- **THEN** the adapter MUST call the `coverage` protocol method and return `[]crap.FuncCoverage` with correctly mapped fields

#### Scenario: Side effect adapter translates protocol response

- **GIVEN** an `ExternalSideEffectAnalyzer` connected to a mock analyzer
- **WHEN** `Analyze` is called
- **THEN** the adapter MUST call the `analyze` protocol method and return `[]taxonomy.AnalysisResult` with `Classification` attached to each `SideEffect`

#### Scenario: Contract coverage adapter with test_mapping

- **GIVEN** an `ExternalContractCoverageProvider` connected to an analyzer that supports `test_mapping`
- **WHEN** `Build` is called
- **THEN** the adapter MUST call `test_mapping`, compute contract coverage using `quality.ComputeContractCoverage`, and return a lookup function

#### Scenario: Contract coverage adapter without test_mapping

- **GIVEN** an `ExternalContractCoverageProvider` connected to an analyzer that does NOT support `test_mapping` (capability is false)
- **WHEN** `Build` is called
- **THEN** the adapter MUST return a lookup function that always returns `(zero, false)` and an empty degraded packages list — GazeCRAP is unavailable

---

### Requirement: Capability Negotiation

The `initialize` method response MUST include a `capabilities` object declaring which optional methods the analyzer supports. The protocol client MUST store these capabilities and the adapter layer MUST check them before calling optional methods.

#### Scenario: Analyzer declares test_mapping capability

- **GIVEN** an analyzer that responds to `initialize` with `capabilities.test_mapping: true`
- **WHEN** gaze runs the analysis pipeline
- **THEN** the `test_mapping` method MUST be called and GazeCRAP scores MUST be computed

#### Scenario: Analyzer declares no optional capabilities

- **GIVEN** an analyzer that responds to `initialize` with `capabilities.test_mapping: false` and `capabilities.classify_signals: false`
- **WHEN** gaze runs the analysis pipeline
- **THEN** gaze MUST skip `test_mapping` and `classify_signals`, produce CRAP scores (from complexity + coverage), and omit GazeCRAP/quadrant data

---

### Requirement: Analyzer Discovery

Gaze MUST discover analyzer binaries through a three-tier mechanism: (1) `--analyzer <name>` CLI flag, (2) `.gaze.yaml` `analyzers.<language>.command` config, (3) `gaze-analyzer-<language>` PATH convention. The first match wins.

#### Scenario: CLI flag overrides config

- **GIVEN** a `.gaze.yaml` with `analyzers.python.command: snake-eyes` and the user runs `gaze crap --analyzer my-analyzer ./...`
- **WHEN** gaze discovers the analyzer
- **THEN** `my-analyzer` MUST be used, not `snake-eyes`

#### Scenario: PATH convention fallback

- **GIVEN** no `--analyzer` flag, no `.gaze.yaml` `analyzers` section, and `gaze-analyzer-python` exists in PATH
- **WHEN** gaze discovers the analyzer for Python
- **THEN** `gaze-analyzer-python` MUST be used

#### Scenario: No analyzer found

- **GIVEN** no `--analyzer` flag, no config, and no `gaze-analyzer-*` in PATH
- **WHEN** gaze runs without `--analyzer` on a Go project
- **THEN** gaze MUST use the default Go providers (existing behavior, no error)

---

### Requirement: CLI Flags

The `crap`, `quality`, and `report` commands MUST accept `--analyzer <name>` and `--language <lang>` flags. When `--analyzer` is specified, external provider adapters MUST be constructed instead of Go providers. When neither flag is specified, behavior MUST be identical to today. (`gaze analyze --analyzer` is deferred to a future phase — see design.md D12.)

#### Scenario: Analyze with external analyzer

- **GIVEN** `--analyzer snake-eyes` specified on `gaze crap`
- **WHEN** gaze runs
- **THEN** gaze MUST spawn `snake-eyes --stdio`, run the protocol session, and produce CRAP scores from the external data

#### Scenario: No analyzer flag preserves Go behavior

- **GIVEN** no `--analyzer` or `--language` flag specified
- **WHEN** gaze runs `gaze crap ./...`
- **THEN** behavior MUST be identical to the current Go-only implementation

---

### Requirement: Fake Analyzer Canned Data

The fake analyzer binary used for testing MUST return deterministic canned data:

- `complexity`: 3 functions — `add` (complexity 2), `multiply` (complexity 3), `divide` (complexity 5)
- `coverage`: same 3 functions — `add` (90%), `multiply` (60%), `divide` (0%)
- `analyze`: `divide` has ReturnValue and ErrorReturn effects; `multiply` has ReturnValue only

This data enables deterministic CRAP score verification in integration tests.

#### Scenario: Fake analyzer produces deterministic CRAP scores

- **GIVEN** the fake analyzer binary with the canned data above
- **WHEN** `gaze crap --analyzer fake_analyzer` is run against a test project
- **THEN** CRAP scores MUST be deterministic and reproducible across runs (e.g., `divide` with complexity 5 and 0% coverage produces CRAP = 30)

---

### Requirement: Analyzers Configuration

The `.gaze.yaml` file MUST support an `analyzers` section for mapping languages to analyzer binaries:

```yaml
analyzers:
  python:
    command: snake-eyes
    args: ["--stdio"]
```

#### Scenario: Config-based analyzer discovery

- **GIVEN** `.gaze.yaml` with `analyzers.python.command: snake-eyes` and `analyzers.python.args: ["--stdio"]`
- **WHEN** gaze discovers the Python analyzer
- **THEN** it MUST spawn `snake-eyes --stdio`

---

### Requirement: Protocol Timeouts

The protocol client MUST enforce per-method timeouts via `context.Context` to prevent hung analyzers from blocking gaze indefinitely.

#### Scenario: Analyzer hangs past timeout

- **GIVEN** an analyzer that hangs (does not respond) after receiving a request
- **WHEN** the timeout expires
- **THEN** the client MUST kill the subprocess and return a timeout error

---

### Requirement: Session Lifecycle

The `Session` struct MUST manage the full protocol lifecycle: spawn binary, initialize, run analysis pipeline (calling all required + optional methods based on capabilities), shutdown.

#### Scenario: Complete session with all capabilities

- **GIVEN** an analyzer that supports all capabilities (`test_mapping: true`, `classify_signals: true`)
- **WHEN** the session runs
- **THEN** all required methods (`initialize`, `analyze`, `complexity`, `coverage`, `shutdown`) and both optional methods (`test_mapping`, `classify_signals`) MUST be called in order and providers MUST be returned

#### Scenario: Session with minimal capabilities

- **GIVEN** an analyzer with no optional capabilities (`test_mapping: false`, `classify_signals: false`)
- **WHEN** the session runs
- **THEN** only required methods MUST be called, optional methods MUST be skipped, and GazeCRAP MUST be unavailable

---

### Requirement: Unknown Side Effect Types

The adapter layer MUST handle unknown `SideEffectType` values gracefully.

#### Scenario: Analyzer returns unknown SideEffectType

- **GIVEN** an analyzer that returns a side effect with an unknown `SideEffectType` (e.g., `"PythonYield"`)
- **WHEN** the adapter processes the response
- **THEN** the unknown effect MUST be included in the results with a warning logged to stderr

---

## MODIFIED Requirements

None — the scoring core (`crap.Analyze`, `Formula`, `ClassifyQuadrant`, `computeScores`) is unchanged. External adapters provide data through the same provider interfaces that Go adapters use.

## REMOVED Requirements

None