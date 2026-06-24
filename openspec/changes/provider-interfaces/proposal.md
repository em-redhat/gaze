## Why

Gaze's core value proposition -- side effect detection, contract classification, CRAP scoring, test quality assessment -- is language-agnostic. The taxonomy, scoring formulas, classification engine, reporting, TUI, and AI pipeline all work on abstract data structures. However, the implementation is tightly coupled to Go-specific tooling at four critical points:

1. **Complexity**: `gocyclo.Analyze()` is called directly in `crap/analyze.go`
2. **Line coverage**: `go test -coverprofile` is spawned directly in `crap/analyze.go`
3. **Side effect analysis**: `analysis.LoadAndAnalyze()` + `classify.Classify()` use `go/ast`, `go/types`, `go/ssa`
4. **Contract coverage**: `BuildContractCoverageFunc()` orchestrates the entire quality pipeline using Go-specific SSA, AST, and type system APIs

This coupling prevents multi-language support (Issue #95, the external analyzer protocol for snake-eyes/Python). Before a JSON-RPC protocol can be added, the Go-specific analysis must be separated from the universal scoring engine behind clean interfaces.

This change extracts four provider interfaces at the natural seam between "language-specific data acquisition" and "universal scoring/reporting." It is Phase 1 of Issue #95 -- a pure refactoring with zero user-facing changes that makes Phase 2 (the protocol itself) possible.

## What Changes

### Provider Interface Extraction

Four interfaces are defined in `internal/crap/provider.go`:

| Interface | Replaces | Boundary |
|-----------|----------|----------|
| `ComplexityProvider` | Direct `gocyclo.Analyze()` call | `crap/analyze.go:123` |
| `LineCoverageProvider` | `generateCoverProfile` + `ParseCoverProfile` | `crap/analyze.go:97-133` |
| `SideEffectAnalyzer` | `analysis.LoadAndAnalyze` + `classify.Classify` | `crap/contract.go:68-80` |
| `ContractCoverageProvider` | `BuildContractCoverageFunc` (entire quality pipeline) | `crap/contract.go:36` |

### Go Provider Adapters

Default Go implementations live in `internal/provider/goprovider/`, wrapping existing code behind the new interfaces. When provider fields in `crap.Options` are nil, these Go defaults are used -- preserving identical behavior.

### Mock Providers for Validation

Mock implementations in `internal/provider/mockprovider/` validate that the interface layer works with synthetic (non-Go) data. This proves the universal scoring core operates correctly on externally-supplied data before Phase 2 introduces real external analyzers.

### Internal Type Change

`computeScores` in `crap/analyze.go` accepts `[]FunctionComplexity` (a new language-neutral struct) instead of `[]gocyclo.Stat`. The new type carries identical information (package, function, file, line, complexity) without `go/token` imports.

## Capabilities

### New Capabilities
- `provider-interfaces`: Four Go interfaces (`ComplexityProvider`, `LineCoverageProvider`, `SideEffectAnalyzer`, `ContractCoverageProvider`) that decouple language-specific analysis from universal scoring
- `function-complexity-type`: Language-neutral `FunctionComplexity` struct replacing `gocyclo.Stat` in the scoring pipeline
- `go-provider-adapters`: Default Go implementations wrapping existing analysis code behind provider interfaces
- `mock-provider-testing`: Mock provider implementations proving the interface layer works with synthetic data

### Modified Capabilities
- `crap.Options`: Extended with optional provider fields (nil = use Go default)
- `crap.Analyze`: Calls provider methods instead of direct Go-specific functions when providers are set
- `computeScores`: Accepts `[]FunctionComplexity` instead of `[]gocyclo.Stat`

### Removed Capabilities
- None -- this is a pure refactoring. All existing behavior is preserved.

## Impact

- **New packages**: `internal/provider/goprovider/` (~4 files), `internal/provider/mockprovider/` (~2 files)
- **New file**: `internal/crap/provider.go` (interface definitions + `FunctionComplexity` type)
- **Modified files**: `internal/crap/analyze.go` (provider dispatch), `internal/crap/crap.go` (Options fields), `internal/crap/analyze_internal_test.go` (type change)
- **Backward compatible**: Zero user-facing changes. Identical CLI behavior, identical output formats, identical exit codes. No new CLI flags, no new configuration.
- **No new external dependencies**: All interfaces reference existing types from `internal/taxonomy` and `internal/crap`.

## Constitution Alignment

Assessed against the Gaze project constitution (`.specify/memory/constitution.md` v1.3.0).

### I. Accuracy

**Assessment**: PASS

This is a pure refactoring -- no scoring logic, no detection logic, and no classification logic changes. The universal scoring core (`Formula`, `ClassifyQuadrant`, `computeScores`, `buildSummary`, `assignFixStrategy`) is untouched. Output is byte-identical when using default Go providers. Full test suite passage (unit, integration, E2E self-check) validates zero regression.

### II. Minimal Assumptions

**Assessment**: PASS

No user-facing changes. No new CLI flags, no new configuration options, no setup requirements. The refactoring is invisible to users. Interface definitions make zero assumptions about the data source language -- they accept universal types (strings, integers, taxonomy structs).

### III. Actionable Output

**Assessment**: PASS

Output formats are unchanged. JSON and text reports are byte-identical. No new metrics or output fields. This is infrastructure, not user-facing output. Cross-run comparability is preserved.

### IV. Testability

**Assessment**: PASS

Each provider interface is independently testable via mock implementations. The universal scoring core can be tested with synthetic data for the first time (currently impossible because `crap.Analyze` is monolithically coupled to gocyclo and `go test`). Mock provider tests validate formula correctness, quadrant classification, and fix strategy assignment with known inputs. Coverage strategy: unit tests for Go provider adapters, mock provider validation tests, existing test suite for regression detection.