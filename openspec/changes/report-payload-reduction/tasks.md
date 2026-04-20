## 1. Compact payload types and method

- [x] 1.1 Add compact intermediate types to `internal/aireport/payload.go`: `compactPayload`, `compactDocscanEntry`, `compactQualityReport` (with `GapIDs`, `AmbiguousEffectIDs`, `DiscardedReturnIDs` instead of full objects), `compactContractCoverage` (with ID arrays), `compactClassifyResult`, `compactClassification` (without Signals), `compactCRAPReport` (without worst offender lists), `compactSummary` (unexported, mirroring `ReportSummary` with JSON tags).
- [x] 1.2 Implement `CompactForAI() ([]byte, error)` on `ReportPayload`. Unmarshal each `json.RawMessage` field into the full type, project into the compact type, marshal the compact struct with `json.Marshal` (compact, no indentation). Handle nil fields (step failures) by passing them through as `null`. Handle empty arrays (`[]`) distinctly from nil (`null`). If any unmarshal/marshal fails, return the error directly (no fallback to full payload).
- [x] 1.3 Add dual-path documentation comment on `ReportPayload` noting that new fields must be considered for both `json.Marshal` (full) and `CompactForAI()` (compact) serialization paths.
- [x] 1.4 Add test: all steps failed (all four fields nil, all errors populated) produces valid JSON with null step fields and populated errors.
- [x] 1.5 Add test: step succeeds with empty data (docscan returns `[]`) produces `[]` in compact output, not `null`.

## 2. Docscan content stripping

- [x] 2.1 In `CompactForAI()`, unmarshal the `Docscan` field into `[]docscan.DocumentFile`, project each entry to `compactDocscanEntry{Path, Priority}` (dropping `Content`), marshal back.
- [x] 2.2 Add test: given a docscan payload with 3 files each having 10KB content, `CompactForAI()` docscan output contains paths and priorities only, no content field present.

## 3. Quality data compaction

- [x] 3.1 In `CompactForAI()`, unmarshal the `Quality` field, replace `ContractCoverage.Gaps []SideEffect` with `ContractCoverage.GapIDs []string` (extracted from each SideEffect's `ID` field). Same for `ContractCoverage.DiscardedReturns` → `ContractCoverage.DiscardedReturnIDs` and `QualityReport.AmbiguousEffects` → `QualityReport.AmbiguousEffectIDs`. Preserve `ContractCoverage.GapHints`, `ContractCoverage.DiscardedReturnHints`, and all scalar fields.
- [x] 3.2 Omit `quality_summary.worst_coverage_tests` from the compact quality output. Preserve `total_tests`, `average_contract_coverage`, `total_over_specifications`, `assertion_detection_confidence`, `ssa_degraded`, `ssa_degraded_packages`.
- [x] 3.3 Add test: given a quality payload with 3 gaps each as full SideEffect objects, compact output has `contract_coverage.gap_ids: ["se-a1b2c3d4", ...]` and no `contract_coverage.gaps` field. Verify `contract_coverage.gap_hints` are preserved.
- [x] 3.4 Add test: given a quality payload with 2 discarded returns, compact output has `contract_coverage.discarded_return_ids` and no `contract_coverage.discarded_returns` field.
- [x] 3.5 Add test: given a quality payload with 2 ambiguous effects, compact output has `ambiguous_effect_ids` and no `ambiguous_effects` field.
- [x] 3.6 Add test: Classify step failed (nil) but Quality has gaps with IDs — `gap_ids` are still emitted correctly.

## 4. Classify signal stripping

- [x] 4.1 In `CompactForAI()`, unmarshal the `Classify` field, walk each result's side effects and use compact classification struct without Signals field. Preserve `Label`, `Confidence`, `Reasoning`.
- [x] 4.2 Add test: given a classify payload with signals and reasoning, compact output has classification with label, confidence, and reasoning but no signals array.

## 5. CRAP summary deduplication

- [x] 5.1 In `CompactForAI()`, unmarshal the `CRAP` field, nil out `Summary.WorstCrap`, `Summary.WorstGazeCrap`, and `Summary.RecommendedActions`. Preserve `Scores` array and all other summary fields.
- [x] 5.2 Add test: given a CRAP payload with 150 scores and worst offender lists, compact output has 150 scores and summary without worst/recommended fields.

## 6. Wire compact payload into text path

- [x] 6.1 In `runTextPath()` (`internal/aireport/runner.go`), replace `json.Marshal(payload)` with `payload.CompactForAI()`. No changes to `runJSONPath()`. Add stderr diagnostic: `fmt.Fprintf(stderr, "Payload: %d bytes (compact)\n", len(compactBytes))`.
- [x] 6.2 Add test: `runTextPath` sends compact payload to adapter (verify via fake adapter that receives stdin). Assert: (1) docscan content absent, (2) no `signals` arrays in classify data, (3) no `worst_crap`/`worst_gaze_crap` in CRAP summary, (4) top-level `summary` field present.

## 7. Size budget verification

- [x] 7.1 Add test: construct a synthetic `ReportPayload` representing ~200 functions, ~100 test-target pairs, ~30 doc files with defined fixture parameters (15KB content per doc, 3-5 effects per function, 2-4 signals with 100-char excerpts per effect, 2 gaps per test-target pair). Assert `CompactForAI()` output is under 300KB.
- [x] 7.2 Add test: construct the same synthetic payload and verify `json.Marshal(payload)` produces the full uncompacted output (docscan content present, signals present, worst offender lists present) — confirming `--format=json` is unaffected.

## 8. Reporter prompt audit

- [x] 8.1 Read `.opencode/agents/gaze-reporter.md` and check for references to fields absent from the compact payload (`signals`, `content` on docscan, `worst_crap`, `worst_gaze_crap`, `recommended_actions`, `worst_coverage_tests`, `gaps` as objects). If references exist, update the prompt to use compact field names or add guidance about using Read tool for detailed data.

## 9. Constitution alignment verification

- [x] 9.1 Verify Testability: confirm all new functions (`CompactForAI`, compact type projections) are tested with synthetic inputs, no subprocess or filesystem dependencies. 100% branch coverage of compact method.
- [x] 9.2 Verify Composability: confirm no new external dependencies introduced (`go mod tidy` produces no changes).

## 10. CI and integration

- [x] 10.1 Run `go test -race -count=1 -short ./...` — all tests pass.
- [x] 10.2 Run `golangci-lint run` — no new lint issues.
- [x] 10.3 Run `go build ./cmd/gaze && ./gaze report ./... --format=json --coverprofile=coverage.out > /dev/null` — verify full JSON output succeeds (smoke test for Observable Quality).
<!-- spec-review: passed -->
<!-- code-review: passed -->
