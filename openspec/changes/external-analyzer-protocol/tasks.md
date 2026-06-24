## 1. Protocol Layer

- [x] 1.1 Create `internal/protocol/types.go` with JSON-RPC 2.0 message types: `Request` (jsonrpc, id, method, params), `Response` (jsonrpc, id, result, error), `Error` (code, message, data). Add protocol-specific types: `InitializeParams` (root_path, config), `InitializeResult` (capabilities, protocol_version, analyzer_name, language), `Capabilities` (discover, test_mapping, classify_signals booleans). Add method constants for all 8 methods.
- [x] 1.2 Create `internal/protocol/client.go` with `Client` struct that manages a subprocess (exec.Cmd, stdin io.Writer, stdout bufio.Reader, stderr bytes.Buffer). Methods: `NewClient(binary string, args []string) (*Client, error)` spawns the subprocess, `Call(ctx context.Context, method string, params any) (*Response, error)` sends a request and reads a response (sequential request/response, no multiplexing) with deadline/timeout via `context.Context` (see design.md D10), `Close()` sends shutdown and waits for process exit. Include request ID sequencing (atomic counter) and response ID matching. When the context deadline expires, the client kills the subprocess and returns a timeout error.
- [x] 1.3 Create `internal/protocol/client_test.go` with tests using a fake analyzer binary at `internal/protocol/testdata/fake_analyzer/main.go`. Tests: (1) successful full session (initialize → analyze → complexity → coverage → shutdown), (2) binary not found error, (3) analyzer crash mid-session (fake exits after initialize), (4) malformed JSON response, (5) JSON-RPC error response propagation, (6) timeout via context.Context cancellation.
- [x] 1.4 Create `internal/protocol/testdata/fake_analyzer/main.go` — a Go binary that reads JSON-RPC requests from stdin and writes canned responses to stdout. Supports all 8 methods with hardcoded response data. Accepts `--stdio` flag. Optionally `--crash-after=initialize` to simulate mid-session crashes.

## 2. Protocol Response Types

- [x] 2.1 Add protocol response types to `internal/protocol/types.go`: `AnalyzeResult` (functions: list of `AnalyzedFunction` with name, package, file, line, side_effects list), `ComplexityResult` (functions: list of `FunctionComplexityData` with name, package, file, line, complexity), `CoverageResult` (functions: list of `FunctionCoverageData` with file, function, start_line, end_line, covered_stmts, total_stmts, percentage), `DiscoverResult` (source_files, test_files, framework), `TestMappingResult` (mappings: list of `AssertionMappingData`), `ClassifySignalsResult` (signals: list of signal data).
- [x] 2.2 Add JSON tags to all protocol types. Verify round-trip: marshal a canned response, unmarshal it, verify all fields survive.

## 3. External Provider Adapters

- [x] 3.1 Create `internal/adapter/complexity.go` with `ExternalComplexityProvider` struct implementing `crap.ComplexityProvider`. Holds a `*protocol.Client`. `Analyze` method calls `client.Call("complexity", ...)` and converts `ComplexityResult.Functions` to `[]crap.FunctionComplexity`.
- [x] 3.2 Create `internal/adapter/coverage.go` with `ExternalLineCoverageProvider` struct implementing `crap.LineCoverageProvider`. Holds a `*protocol.Client`. `Coverage` method calls `client.Call("coverage", ...)` and converts `CoverageResult.Functions` to `[]crap.FuncCoverage`.
- [x] 3.3 Create `internal/adapter/sideeffect.go` with `ExternalSideEffectAnalyzer` struct implementing `crap.SideEffectAnalyzer`. Holds a `*protocol.Client` and `Capabilities`. `Analyze` method calls `client.Call("analyze", ...)` and converts response to `[]taxonomy.AnalysisResult`. If `classify_signals` capability is true, also calls `classify_signals` and merges signal data.
- [x] 3.4 Create `internal/adapter/contract.go` with `ExternalContractCoverageProvider` struct implementing `crap.ContractCoverageProvider`. Holds a `*protocol.Client` and `Capabilities`. If `test_mapping` capability is true, calls `test_mapping` method and uses `quality.ComputeContractCoverage` to build the lookup function. If false, returns a no-op lookup (always false).
- [x] 3.5 Create `internal/adapter/session.go` with `Session` struct that manages the full protocol lifecycle: spawn binary, initialize, discover, run analysis pipeline (calling all required + optional methods based on capabilities), shutdown. Returns fully-constructed provider adapters ready to pass to `crap.Options`.
- [x] 3.6 Create `internal/adapter/adapter_test.go` with tests using the fake analyzer binary: (1) each adapter produces correct output from canned responses, (2) session lifecycle completes successfully, (3) optional method skipped when capability is false, (4) error propagation from required methods.

## 4. Analyzer Discovery

- [x] 4.1 Create `internal/adapter/discovery.go` with `Discover(analyzerFlag, language string, cfg *config.GazeConfig) (binary string, args []string, err error)`. Implements the three-tier discovery: CLI flag → config → PATH convention. Returns the binary path and args for spawning.
- [x] 4.2 Add `AnalyzersConfig` struct to `internal/config/config.go` with `map[string]AnalyzerEntry` where `AnalyzerEntry` has `Command string` and `Args []string`. Add `Analyzers AnalyzersConfig` field to `GazeConfig`. Update `DefaultConfig()`.
- [x] 4.3 Create `internal/adapter/discovery_test.go` with tests: (1) CLI flag overrides config, (2) config lookup works, (3) PATH fallback works, (4) no analyzer found returns empty (Go default).

## 5. CLI Integration

- [x] 5.1 Add `--analyzer` string flag and `--language` string flag to the `crap`, `quality`, and `report` commands in `cmd/gaze/main.go`. When `--analyzer` is set, construct `adapter.Session` and use external provider adapters. When not set, use existing Go providers. (`gaze analyze --analyzer` is deferred — see design.md D12.)
- [x] 5.2 Update `runCrap` in `cmd/gaze/main.go` to check for `--analyzer` flag. If set: call `adapter.Discover()`, create `adapter.Session`, get providers, pass to `crap.Options`. If not set: use `goprovider` as today.
- [x] 5.3 Update `runQuality` and the report pipeline similarly. (`runAnalyze` is deferred — see design.md D12.)
- [x] 5.4 Add integration test with fake analyzer: `go test -run TestCrapWithExternalAnalyzer ./cmd/gaze/...` — spawns the fake analyzer binary, runs `gaze crap --analyzer fake_analyzer`, verifies CRAP scores are computed from the fake data.

## 6. Non-Regression Verification

- [x] 6.1 Run `go test -race -count=1 -short ./...` — all tests MUST pass with zero failures.
- [x] 6.2 Run `go test -race -count=1 -run TestRunSelfCheck -timeout 30m ./cmd/gaze/...` — E2E self-check MUST produce identical output (verifies Go provider path is unchanged).
- [x] 6.3 Run `golangci-lint run` — MUST pass with no new warnings.

## 7. Documentation

- [x] 7.1 Update `README.md` with external analyzer section: how to use `--analyzer`, `.gaze.yaml` config, protocol overview for analyzer authors.
- [x] 7.2 Update `AGENTS.md` — add `internal/protocol/` and `internal/adapter/` to architecture section. Add to Recent Changes.
- [x] 7.3 Create `docs/protocol.md` — full protocol specification for external analyzer authors: JSON-RPC message format, method definitions, request/response schemas, capability negotiation, error handling, lifecycle diagram.
- [x] 7.4 Create GitHub issue in `unbound-force/website` for documentation updates: new CLI flags, multi-language support, analyzer development guide.

## 8. Constitution Alignment Verification

- [x] 8.1 Verify all four principles: Accuracy (scoring engine unchanged, mock analyzer validates protocol bridge), Minimal Assumptions (works with any language analyzer implementing the protocol, no analyzer needed for Go-only usage), Actionable Output (reports from external data use same formatting and fix strategies), Testability (protocol client testable with fake binary, adapters testable with mock protocol, end-to-end testable with fake analyzer).
<!-- spec-review: passed -->