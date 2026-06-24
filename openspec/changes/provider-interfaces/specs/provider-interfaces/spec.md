## ADDED Requirements

### Requirement: ComplexityProvider Interface

The `crap` package MUST define a `ComplexityProvider` interface with an `Analyze` method that accepts package patterns and a root directory, and returns `[]FunctionComplexity` -- a language-neutral representation of per-function cyclomatic complexity. The `FunctionComplexity` type MUST carry package, function, file, line, and complexity fields with JSON tags. The `FunctionComplexity` type MUST NOT import `go/token`, `go/ast`, `go/types`, `go/packages`, `go/ssa`, or `github.com/fzipp/gocyclo`.

#### Scenario: Go complexity provider returns identical data

- **GIVEN** the `GoComplexityProvider` wrapping `gocyclo.Analyze()`
- **WHEN** `Analyze` is called with Go package patterns
- **THEN** the returned `[]FunctionComplexity` contains identical package, function, file, line, and complexity values as the current `[]gocyclo.Stat` output

#### Scenario: Mock complexity provider with synthetic data

- **GIVEN** a `MockComplexityProvider` configured with 3 functions of known complexities (1, 10, 25)
- **WHEN** `crap.Analyze` runs with this provider
- **THEN** the resulting CRAP scores match `Formula(complexity, coverage)` for each function

#### Scenario: Complexity provider returns error

- **GIVEN** a `ComplexityProvider` that returns an error
- **WHEN** `crap.Analyze` runs with this provider
- **THEN** `crap.Analyze` MUST return the error with context wrapping and produce no scores

---

### Requirement: LineCoverageProvider Interface

The `crap` package MUST define a `LineCoverageProvider` interface with a `Coverage` method that accepts package patterns, a root directory, and an optional cover profile path, and returns `[]FuncCoverage`. When a cover profile path is provided, the provider MUST use it instead of generating coverage data. The existing `FuncCoverage` type MUST be reused without modification.

#### Scenario: Go coverage provider with pre-generated profile

- **GIVEN** the `GoLineCoverageProvider` and a pre-generated `coverage.out` file
- **WHEN** `Coverage` is called with the cover profile path
- **THEN** the returned `[]FuncCoverage` matches `ParseCoverProfile` output for the same file

#### Scenario: Go coverage provider generates coverage

- **GIVEN** the `GoLineCoverageProvider` and no cover profile path
- **WHEN** `Coverage` is called
- **THEN** the returned `[]FuncCoverage` is non-empty, each entry has a non-empty File and Function, and Coverage values are between 0.0 and 100.0

#### Scenario: Coverage provider returns error

- **GIVEN** a `LineCoverageProvider` that returns an error
- **WHEN** `crap.Analyze` runs with this provider
- **THEN** `crap.Analyze` MUST return the error with context wrapping and produce no scores

#### Scenario: Go coverage provider handles partial profile

- **GIVEN** the `GoLineCoverageProvider` and `go test` exits non-zero but produces a partial coverage profile
- **WHEN** `Coverage` is called
- **THEN** the provider MUST recover partial coverage data and return it with no error, logging a warning to Stderr

---

### Requirement: SideEffectAnalyzer Interface

The `crap` package MUST define a `SideEffectAnalyzer` interface with an `Analyze` method that accepts a package path and returns `[]taxonomy.AnalysisResult` with `Classification` already attached to each `SideEffect`. Implementations MUST handle both side effect detection and classification internally.

#### Scenario: Go side effect analyzer returns classified results

- **GIVEN** the `GoSideEffectAnalyzer` wrapping `analysis.LoadAndAnalyze` and `classify.Classify`
- **WHEN** `Analyze` is called with a Go package path
- **THEN** every `SideEffect` in the returned results has a non-zero `Classification` with a valid `Label` (contractual, ambiguous, or incidental)

#### Scenario: Side effect analyzer returns error for invalid package

- **GIVEN** a `SideEffectAnalyzer` and an invalid package path
- **WHEN** `Analyze` is called
- **THEN** an error MUST be returned and the results MUST be nil

Note: `SideEffectAnalyzer` is NOT a field on `crap.Options`. It is consumed only by `ContractCoverageProvider` implementations as a composition dependency. See design decision D5.

---

### Requirement: ContractCoverageProvider Interface

The `crap` package MUST define a `ContractCoverageProvider` interface with a `Build` method that accepts package patterns and a root directory, and returns a contract coverage lookup function `func(pkg, function string) (ContractCoverageInfo, bool)`, a list of degraded package paths `[]string`, and an error. The lookup function MUST return `(info, true)` when quality data exists for the given package and function, and `(zero, false)` otherwise.

#### Scenario: Go contract coverage provider wraps quality pipeline

- **GIVEN** the `GoContractCoverageProvider` wrapping `BuildContractCoverageFunc`
- **WHEN** `Build` is called
- **THEN** the returned lookup function produces identical `ContractCoverageInfo` values as the current `BuildContractCoverageFunc` output

#### Scenario: Mock contract coverage with ambiguous effects

- **GIVEN** a `MockContractCoverageProvider` returning 0% coverage with reason `"all_effects_ambiguous"`
- **WHEN** `crap.Analyze` runs with this provider
- **THEN** the resulting `Score.ContractCoverageReason` field carries `"all_effects_ambiguous"` through to output

#### Scenario: Contract coverage provider returns error

- **GIVEN** a `ContractCoverageProvider` whose `Build` method returns an error
- **WHEN** `crap.Analyze` runs with this provider
- **THEN** `crap.Analyze` MUST continue without GazeCRAP scores (graceful degradation), using only line coverage for CRAP computation, consistent with the current behavior when `ContractCoverageFunc` is nil

---

### Requirement: Provider Fields in crap.Options

`crap.Options` MUST include provider fields: `ComplexityProvider`, `LineCoverageProvider`, and `ContractCoverageProvider`. Callers MUST construct and provide these providers explicitly. The `SideEffectAnalyzer` interface is intentionally excluded from `crap.Options` — it is consumed only by `ContractCoverageProvider` implementations as a composition dependency (see D5). The existing `ContractCoverageFunc` and `SSADegradedPackages` fields MUST be preserved for backward compatibility with `// Deprecated:` GoDoc comments. When `ContractCoverageProvider` is set, it MUST take precedence over `ContractCoverageFunc`.

#### Scenario: Callers provide Go providers

- **GIVEN** `crap.Options` with `ComplexityProvider` and `LineCoverageProvider` set to Go provider implementations
- **WHEN** `crap.Analyze` runs on a Go package
- **THEN** output is byte-identical to the current implementation

#### Scenario: Provider precedence over ContractCoverageFunc

- **GIVEN** `crap.Options` with both `ContractCoverageProvider` and `ContractCoverageFunc` set
- **WHEN** `crap.Analyze` runs
- **THEN** `ContractCoverageProvider.Build()` is called and `ContractCoverageFunc` is ignored

---

### Requirement: computeScores Type Change

The `computeScores` function MUST accept `[]FunctionComplexity` instead of `[]gocyclo.Stat`. The function MUST produce identical `[]Score` output for identical input data. The `isGeneratedFile` filtering logic MUST remain functional using the `File` field from `FunctionComplexity`.

#### Scenario: computeScores with FunctionComplexity

- **GIVEN** a `[]FunctionComplexity` with 5 functions and matching coverage data
- **WHEN** `computeScores` runs
- **THEN** the returned `[]Score` contains CRAP scores matching `Formula(complexity, coverage)` for each function, with correct quadrant classifications

---

### Requirement: Interface Import Isolation

The file containing provider interface definitions (`internal/crap/provider.go`) MUST NOT import `go/ast`, `go/types`, `go/packages`, `go/ssa`, or `github.com/fzipp/gocyclo`. Interface definitions MUST reference only standard library types, `internal/taxonomy` types, and `internal/crap` types.

#### Scenario: Provider file has no Go-specific imports

- **GIVEN** the file `internal/crap/provider.go`
- **WHEN** its import list is inspected
- **THEN** it contains zero imports from `go/ast`, `go/types`, `go/packages`, `go/ssa`, or `github.com/fzipp/gocyclo`

---

## MODIFIED Requirements

### Requirement: crap.Analyze internal dispatch

`crap.Analyze` MUST use the provider fields in `Options` for complexity, coverage, and contract coverage data. Callers MUST construct providers and pass them via `Options`. The external behavior (output, exit codes, error messages) MUST remain identical when Go providers are used. (Previously: `crap.Analyze` directly called `gocyclo.Analyze()`, `generateCoverProfile()`, `ParseCoverProfile()`, and used `ContractCoverageFunc` from Options.)

#### Scenario: Go providers produce identical output

- **GIVEN** `crap.Options` with Go provider implementations constructed by the caller
- **WHEN** `crap.Analyze` runs on a Go package
- **THEN** output MUST be byte-identical to the pre-refactoring implementation

#### Scenario: Provider dispatch with custom provider

- **GIVEN** `crap.Options` with `ComplexityProvider` set to a mock returning 3 functions
- **WHEN** `crap.Analyze` runs
- **THEN** the mock provider's `Analyze` method MUST be called instead of `gocyclo.Analyze()`

## REMOVED Requirements

None -- this is a pure refactoring with zero behavioral changes.