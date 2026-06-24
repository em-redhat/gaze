## Context

Gaze's CRAP pipeline (`crap.Analyze`) and quality pipeline (`BuildContractCoverageFunc`) are monolithically coupled to Go-specific tooling. The universal scoring core (`Formula`, `ClassifyQuadrant`, `computeScores`, `buildSummary`, `assignFixStrategy`, `ComputeContractCoverage`, `ComputeOverSpecification`) is already language-agnostic -- it operates on string-typed effect enums, float64 coverage percentages, and integer complexity values. The coupling is entirely in the *producers* of this data, not the *consumers*.

Relevant existing patterns:
- `crap.Options` uses an options struct pattern for configurable behavior (`internal/crap/crap.go`)
- `ContractCoverageFunc func(pkg, function string) (ContractCoverageInfo, bool)` already exists as a callback abstraction in `crap.Options` -- proving the pattern works for the quality pipeline
- `pipelineStepFuncs` in `internal/aireport/runner.go` uses function-valued dependency injection for testing
- `BuildSSAFunc` in `quality.Options` provides injection for SSA construction

The data flow has a clear boundary:

```
[GO-SPECIFIC PRODUCERS]                    [UNIVERSAL SCORING CORE]
                                           
gocyclo.Analyze()          ──┐             
                             ├─→ []FunctionComplexity ─→ computeScores()
go test -coverprofile      ──┤                            ├─→ Formula()
ParseCoverProfile()        ──┘             ├─→ coverMaps ─┘   ClassifyQuadrant()
                                           │                  assignFixStrategy()
analysis.LoadAndAnalyze()  ──┐             │                  buildSummary()
classify.Classify()        ──┤             │
quality.Assess()           ──┤             │
BuildContractCoverageFunc()──┘─→ ContractCoverageFunc ──────┘
```

## Goals / Non-Goals

### Goals
- Extract four provider interfaces at the natural boundary between Go-specific producers and universal scoring
- Provide default Go implementations wrapping existing code
- Validate interface layer with mock providers using synthetic data
- Maintain byte-identical output with default providers (zero regression)
- Enable Phase 2 (external analyzer protocol) without further refactoring of the scoring core

### Non-Goals
- Taxonomy generalization (renaming `GoVersion` to `LanguageVersion`, adding language-specific effect types) -- deferred to protocol spec
- Classification signal provider interface (external analyzers return pre-classified results, keeping Go's 5-signal model internal)
- New CLI flags (`--analyzer`, `--language`) -- Phase 2 scope
- JSON-RPC protocol implementation -- Phase 2 scope
- Changes to `internal/analysis/`, `internal/classify/`, or `internal/quality/` package APIs -- these are wrapped, not modified
- Report format changes -- output is byte-identical

## Decisions

### D1: Interfaces defined in `internal/crap/provider.go`

The interfaces reference `ContractCoverageInfo`, `FuncCoverage`, and other types that live in the `crap` package. Defining interfaces in a separate `internal/provider/` package would create an import cycle (`provider` imports `crap` for types, `crap` imports `provider` for interfaces).

Options evaluated:

| Option | Pros | Cons |
|--------|------|------|
| **Interfaces in `internal/crap/`** | No import cycle; types already available; minimal file churn | Interfaces sit next to Go-specific code |
| Interfaces in `internal/provider/` with type copies | Clean separation | Type duplication; conversion boilerplate; maintenance burden |
| Shared types package | Maximum separation | Over-engineering for Phase 1; adds indirection |

**Decision**: Define interfaces and `FunctionComplexity` in `internal/crap/provider.go`. When Phase 2 introduces `internal/protocol/`, interfaces can be promoted to a shared package if import cycles arise. Pragmatism over purity for Phase 1.

This aligns with **Composability First** (Principle II) -- the interfaces are independently usable by any provider implementation without requiring the provider to import Go-specific analysis packages.

### D2: Go adapters in `internal/provider/goprovider/`

Go-specific implementations live in a separate sub-package, making the Go analysis a pluggable module:

```
internal/provider/goprovider/
  complexity.go    -- wraps gocyclo.Analyze()
  coverage.go      -- wraps generateCoverProfile + ParseCoverProfile
  sideeffect.go    -- wraps analysis.LoadAndAnalyze + classify.Classify
  contract.go      -- wraps BuildContractCoverageFunc logic
```

This creates a clean import direction: `goprovider` imports `crap` (for interface types), `analysis`, `classify`, `quality`, `loader`, `gocyclo`. The `crap` package imports nothing from `goprovider` -- it only knows about the interfaces.

**Decision**: `internal/provider/goprovider/` holds all Go-specific adapter code. Default Go providers are constructed at call sites (`cmd/gaze/main.go`, `internal/aireport/runner.go`) and passed via `crap.Options`, keeping `crap` free of `goprovider` imports.

### D3: Nil-means-default pattern for backward compatibility

Default Go provider construction happens at **call sites** (the CLI layer), not inside `crap.Analyze`. This keeps the `crap` package free of transitive Go-specific imports:

```go
// In cmd/gaze/main.go (runCrap):
opts := crap.Options{
    ComplexityProvider:   goprovider.NewComplexityProvider(),
    LineCoverageProvider: goprovider.NewLineCoverageProvider(stderr),
    // ContractCoverageProvider set separately if needed
}
```

The `crap.Analyze` function requires non-nil providers — callers must construct them. This achieves true import decoupling: `internal/crap/` imports only `internal/taxonomy/` and standard library types, never `goprovider`, `analysis`, `classify`, `quality`, `loader`, or `gocyclo`.

For backward compatibility during transition, if a provider field is nil, `crap.Analyze` returns an error indicating the provider is required. Existing callers (`runCrap`, `runReport`) are updated to construct providers explicitly.

**Decision**: Provider fields in `crap.Options` are required interface values. Callers construct the appropriate provider for their context (Go callers use `goprovider`, future external analyzers use protocol adapters). This follows the established dependency injection pattern but inverts the nil-default — callers are explicit about their data sources.

**Note**: Phase 1 achieves true import decoupling — `internal/crap/` has zero transitive dependency on Go-specific analysis packages. This is a stronger separation than the alternative of nil-means-default inside `Analyze`, which would re-introduce transitive coupling through `goprovider` imports.

### D4: `FunctionComplexity` replaces `gocyclo.Stat` in `computeScores`

`gocyclo.Stat` carries `go/token.Position` which pulls in Go-specific imports. The replacement type carries identical data as plain fields:

```go
type FunctionComplexity struct {
    Package    string `json:"package"`
    Function   string `json:"function"`
    File       string `json:"file"`
    Line       int    `json:"line"`
    Complexity int    `json:"complexity"`
}
```

The conversion `gocyclo.Stat -> FunctionComplexity` happens inside `GoComplexityProvider.Analyze()`. The `computeScores` function receives language-neutral data.

Field mapping: `stat.PkgName` -> `Package`, `stat.FuncName` -> `Function`, `stat.Pos.Filename` -> `File`, `stat.Pos.Line` -> `Line`, `stat.Complexity` -> `Complexity`.

### D5: SideEffectAnalyzer returns pre-classified results

External analyzers handle both side effect detection AND classification. The `SideEffectAnalyzer.Analyze()` method returns `[]taxonomy.AnalysisResult` with `Classification` already attached to each `SideEffect`.

For the Go implementation, this means `GoSideEffectAnalyzer.Analyze()` calls `analysis.LoadAndAnalyze()` followed by `classify.Classify()` internally. Go's 5-signal classification model (interface satisfaction, visibility, callers, naming, godoc) stays internal to the Go adapter.

This simplifies the interface surface -- one method, one return type -- and avoids a separate `ClassificationSignalProvider` interface. External analyzers in Phase 2 can implement classification using whatever signals make sense for their language.

**Note**: `SideEffectAnalyzer` is intentionally NOT a field on `crap.Options`. It is consumed only by `ContractCoverageProvider` implementations as a composition dependency (the quality pipeline needs side effect data as input). `crap.Analyze` never calls side effect analysis directly — it receives pre-computed contract coverage via the `ContractCoverageProvider` lookup function. Adding `SideEffectAnalyzer` to `crap.Options` would create a misleading API surface where callers could set it but it would be ignored by `Analyze`.

### D6: ContractCoverageProvider encapsulates the quality pipeline

The current `BuildContractCoverageFunc` is the deepest coupling point. It orchestrates:
1. Package path resolution
2. Side effect analysis + classification
3. Test package loading
4. SSA construction
5. Target inference
6. Assertion detection + mapping
7. Contract coverage computation

The `ContractCoverageProvider.Build()` method returns:
- A lookup function: `func(pkg, function string) (ContractCoverageInfo, bool)`
- A list of degraded package paths: `[]string`
- An error

The Go implementation wraps the existing `BuildContractCoverageFunc` logic. This is the highest-leverage interface -- it encapsulates the entire quality pipeline behind a simple lookup function.

### D7: `ContractCoverageFunc` and `SSADegradedPackages` remain in Options

For backward compatibility during transition, the existing `ContractCoverageFunc` and `SSADegradedPackages` fields in `crap.Options` are kept. When `ContractCoverageProvider` is set, it takes precedence. When nil, the system falls back to `ContractCoverageFunc`/`SSADegradedPackages` (existing behavior). This prevents breaking any current callers during the transition. Both fields should be marked with `// Deprecated:` GoDoc comments pointing to `ContractCoverageProvider`. Removal is deferred to Phase 2.

### D8: `buildCoverMap` and `coverMaps` remain in `crap/analyze.go`

The `coverMaps` type and `buildCoverMap` function MUST remain in `crap/analyze.go`. They convert provider output (`[]FuncCoverage`) into the internal lookup format consumed by `computeScores`. The `LineCoverageProvider` returns `[]FuncCoverage`; the consumer (`crap.Analyze`) builds its own lookup structures. This is a consumer-side concern, not a provider concern.

### D9: `resolvePatterns` stays in `crap/analyze.go` as shared utility

The `resolvePatterns` function converts package patterns to absolute file paths. It is used by both `GoComplexityProvider` and `GoLineCoverageProvider`. Rather than duplicating it or exporting it from `goprovider`, it remains in `crap/analyze.go` as an exported utility (`ResolvePatterns`) that both Go providers can call. The `testFileRegexp` moves to `goprovider/complexity.go` since it is specific to Go complexity analysis.

### D10: Tasks 2.3 and 2.4 are atomic

Changing `computeScores` to accept `[]FunctionComplexity` (Task 2.3) and updating `crap.Analyze` to produce `FunctionComplexity` via providers (Task 2.4) MUST be done in the same commit. An intermediate state where `computeScores` accepts `FunctionComplexity` but `Analyze` still produces `gocyclo.Stat` would create a compile error.

## Risks / Trade-offs

### R1: Interface indirection overhead

**Risk**: Provider interface dispatch adds a virtual call per invocation.

**Mitigation**: Interface dispatch in Go is ~2ns. These interfaces are called once per analysis run (not per function or per assertion). Zero measurable performance impact. Benchmark verification in polish phase.

### R2: Increased code surface area

**Risk**: Three new packages, ~10 new files, adapter boilerplate.

**Mitigation**: The adapters are thin wrappers. The total new code is small relative to the coupling it eliminates. The alternative (keeping monolithic coupling) blocks multi-language support entirely.

### R3: Call-site provider construction

**Risk**: Every caller of `crap.Analyze` must now construct providers explicitly. This is slightly more verbose than the current API.

**Mitigation**: There are only two production call sites (`cmd/gaze/main.go:runCrap` and `internal/aireport/runner.go`). Both already construct `crap.Options` with multiple fields. Adding provider construction is a minor change. The benefit (true import decoupling of the `crap` package) outweighs the verbosity cost.

### R4: `isGeneratedFile` filtering

**Risk**: Generated file detection currently reads Go file headers. After refactoring, `computeScores` receives `FunctionComplexity` with a file path but doesn't know the language.

**Mitigation**: The `FunctionComplexity` type carries the file path. The `isGeneratedFile` check stays in `computeScores` as-is -- it reads file headers looking for `// Code generated`. External analyzers can pre-filter generated files or the check can be generalized in Phase 2. No behavior change for Go analysis.

### R5: Import cycle evolution

**Risk**: As gaze grows, the interface definitions in `crap/provider.go` may need to move to a standalone package to avoid cycles with new consumers.

**Mitigation**: The interfaces are defined in a single file with clear boundaries. Promoting them to `internal/provider/provider.go` in Phase 2 is a mechanical refactoring (move file, update imports). The Phase 1 placement is pragmatic, not permanent.