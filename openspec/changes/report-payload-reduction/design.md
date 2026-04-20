## Context

The `gaze report --ai=opencode` pipeline assembles a `ReportPayload` from four analysis steps (CRAP, Quality, Classify, Docscan), serializes the entire struct via `json.Marshal`, and pipes it as stdin to the AI adapter subprocess. The payload has grown to ~1.17M tokens (~3-4MB JSON), exceeding the 1M token context window of `claude-sonnet-4-6`.

The pipeline currently has zero payload size awareness — no filtering, truncation, deduplication, or budget enforcement exists between the analysis steps and the adapter.

The `ReportPayload` is a struct of four `json.RawMessage` fields, each containing the pretty-printed JSON from its respective step. The `ReportSummary` struct (tagged `json:"-"`) holds pre-extracted metrics but is never serialized to the adapter. Key size contributors:

| Step | Typical Size | Primary Bloat Source |
|------|-------------|---------------------|
| CRAP | 50-80 KB | `worst_crap`/`worst_gaze_crap`/`recommended_actions` duplicate `scores` entries |
| Quality | 200-500 KB | Full `SideEffect` objects (with nested `Classification.Signals`) embedded in `gaps`, `ambiguous_effects`, `discarded_returns` |
| Classify | 100-300 KB | Per-signal `source_file`, `excerpt`, `reasoning` fields |
| Docscan | 500 KB-3+ MB | Full raw text `content` of every `.md` file |

## Goals / Non-Goals

### Goals
- Reduce the AI adapter payload to fit within 1M tokens (~250KB JSON) with margin for system prompt and response
- Preserve all information the gaze-reporter agent needs to produce its formatted report (metrics, scores, quality assessments, classification labels)
- Keep `--format=json` output unchanged — full-fidelity JSON for machine consumers is unaffected
- Make the reduction testable with deterministic size assertions

### Non-Goals
- Changing the AI model or adapter — the fix should work with any model's context window
- Modifying the gaze-reporter agent prompt — it already works with the data it receives
- Adding pagination or multi-call splitting — a single compact payload is simpler and sufficient
- Changing the `docscan.DocumentFile` struct — the reduction operates on the serialized payload, not the scanner's data model

## Decisions

### D1: Compact serialization method on ReportPayload

**Decision**: Add a `CompactForAI() ([]byte, error)` method to `ReportPayload` that produces a reduced JSON representation. `runTextPath()` calls `CompactForAI()` instead of `json.Marshal(payload)`.

**Rationale**: This keeps the full `ReportPayload` intact for `--format=json` consumers (Observable Quality — machine-parseable output preserved). The compact method is a one-way projection that cannot accidentally affect the canonical JSON output. It is a pure function on the struct, testable in isolation (Testability).

**Alternative rejected**: Filtering at the step level (modifying `runCRAPStep`, `runQualityStep`, etc.) would couple step logic to the AI adapter's needs, violating the current separation where steps produce full-fidelity data and the runner decides how to deliver it.

### D2: Docscan reduction — paths only

**Decision**: The compact payload replaces the `docscan` field with an array of `{"path": "...", "priority": N}` objects — the `content` field is stripped entirely.

**Rationale**: The gaze-reporter agent has Read tool access and can fetch any file it needs. Embedding full file contents is wasteful and accounts for the single largest payload contribution. The reporter prompt instructs it to read documentation files as needed; it does not expect pre-loaded content in the payload.

### D3: Quality compaction — reference-based gaps

**Decision**: In the compact payload, Quality report `gaps`, `ambiguous_effects`, and `discarded_returns` fields are reduced to arrays of side effect IDs (strings) rather than full embedded `SideEffect` objects. The full side effect data is already available in the Classify step output.

**Rationale**: Each `SideEffect` object with its nested `Classification` and `Signals` array can be 500-2000 bytes. A test-target pair with 5 gaps embeds 2.5-10KB of duplicate data per pair. Replacing with ID references (`"se-a1b2c3d4"`) reduces this to ~80 bytes per pair while preserving the mapping.

### D4: Signal stripping in Classify output

**Decision**: In the compact payload, `Classification.Signals` arrays are omitted entirely. The top-level `Classification.Label`, `Classification.Confidence`, and `Classification.Reasoning` fields are preserved.

**Rationale**: Individual signals (`source`, `weight`, `source_file`, `excerpt`, `reasoning` per signal) are debugging/audit data. The reporter agent uses `Label` and `Confidence` for report formatting — it never references individual signals. This is the second-largest contributor to Classify step size.

### D5: CRAP summary deduplication

**Decision**: In the compact payload, `summary.worst_crap`, `summary.worst_gaze_crap`, and `summary.recommended_actions` are omitted. The full `scores` array is preserved (it already contains all data needed to derive these subsets).

**Rationale**: These three fields duplicate 20-40 entries from the `scores` array. The reporter agent can identify worst offenders from the scores directly. Similarly, `quality_summary.worst_coverage_tests` is omitted from the Quality output.

### D6: Include ReportSummary in compact output

**Decision**: The compact payload includes a top-level `"summary"` field with the `ReportSummary` values (CRAPload, GazeCRAPload, AvgContractCoverage, classification counts, SSA degradation info).

**Rationale**: Currently `ReportSummary` is `json:"-"` and never reaches the adapter. Including it gives the reporter agent immediate access to pre-extracted key metrics without parsing raw step JSON. This is a small addition (~200 bytes) that significantly aids report formatting.

## Risks / Trade-offs

### Risk: Reporter agent output quality degrades

The compact payload omits verbose signal details and raw doc contents. If the reporter agent's prompt evolves to reference these fields, reports may silently lose detail.

**Mitigation**: The reporter prompt is scaffolded and tool-owned — changes go through `gaze init` and are version-tracked. A breaking prompt change would surface during development. Additionally, the reporter has Read tool access to fetch any file or detail it needs on demand.

### Risk: Compact payload size still exceeds limit for very large projects

Projects with thousands of functions could still produce large CRAP `scores` arrays even after deduplication.

**Mitigation**: The `scores` array uses compact JSON (no indentation) and each entry is ~200 bytes. Even 2000 functions would produce ~400KB — well within the 1M token budget. If a project exceeds this, a future enhancement could filter to CRAPload-only functions (Q3+Q4), but this is not needed now.

### Trade-off: Two serialization paths

Adding `CompactForAI()` means `ReportPayload` now has two serialization paths — `json.Marshal` (full) and `CompactForAI()` (compact). Any new fields added to the payload must be considered for both paths.

**Accepted**: The cost is modest — `CompactForAI()` is explicit about what it includes, and new fields default to excluded from the compact path (safe by default). A comment on `ReportPayload` will document the dual-path requirement.

### D7: CompactForAI error handling — fail hard

**Decision**: If `CompactForAI()` returns an error, `runTextPath` fails the command. There is no fallback to `json.Marshal(payload)`.

**Rationale**: The full payload already causes a hard failure (API rejection with "prompt is too long"). Falling back to it would produce the same error with an additional misleading retry. Failing fast on compact errors surfaces bugs in the compaction logic immediately.

## Coverage Strategy

All new code is covered by unit tests with synthetic `ReportPayload` inputs. No integration or e2e tests are required — `CompactForAI()` is a pure data transformation with no I/O, subprocess, or filesystem dependencies.

- **Unit tests**: Each compaction behavior (docscan stripping, quality ID extraction, signal stripping, CRAP deduplication, summary inclusion) has a dedicated test with explicit assertions on the compact JSON output.
- **Size budget test**: A representative synthetic payload (~200 functions, ~100 test-target pairs, ~30 doc files with defined field sizes) verifies the compact output stays under 300KB.
- **Full-fidelity preservation test**: A test verifies `json.Marshal(payload)` output is unaffected by the existence of `CompactForAI()`.
- **Edge case tests**: Empty payloads (all steps nil), mixed success/failure payloads, and zero-length step outputs are tested.
- **Target**: 100% branch coverage of `CompactForAI()` and all compact type projection logic.
