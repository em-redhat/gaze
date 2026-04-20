## Why

The `gaze report --ai=opencode` CI step fails because the combined JSON payload exceeds the AI model's 1M token context window. The payload reached ~1.17M tokens on run [#24633674351](https://github.com/unbound-force/gaze/actions/runs/24633674351), causing two consecutive "prompt is too long" errors from the API and a non-zero exit code that fails the CI job.

The payload grows with every new function, test, and documentation file added to the project. Without intervention, this failure is permanent and worsening — the AI quality report step will never pass again on `main`.

Root causes identified via pipeline analysis:

1. **Docscan embeds full file contents** — every `.md` file's raw text is serialized into the payload, contributing hundreds of KB to several MB depending on spec/doc volume.
2. **Quality step embeds full SideEffect objects** (with nested Classification and Signals arrays) in `gaps`, `ambiguous_effects`, and `discarded_returns` — duplicating data already present in the Classify step output.
3. **Summary fields duplicate scores** — `worst_crap`, `worst_gaze_crap`, `recommended_actions`, and `worst_coverage_tests` re-embed full objects already in their parent arrays.
4. **Classification signals include verbose debugging fields** — `source_file`, `excerpt`, and `reasoning` per signal are useful for human debugging but not for AI report formatting.
5. **No payload size budget** — the pipeline has zero awareness of output size; there is no cap, filter, or truncation on any step's contribution.

## What Changes

Introduce a payload reduction layer between the four pipeline steps and the AI adapter. The layer filters, deduplicates, and truncates step outputs to produce a compact "AI-ready" payload that preserves all information the reporter agent needs for formatting while staying well within context window limits.

## Capabilities

### New Capabilities
- `payload budget enforcement`: The pipeline enforces a maximum payload size before sending to the AI adapter. The compact output MUST stay under 300KB, providing ~4x margin below the 1M token context window and leaving ample room for the system prompt and model response.
- `docscan content exclusion`: Docscan output sent to the AI adapter includes file paths and priority only — full file contents are stripped. The AI reporter already has Read tool access to fetch file contents on demand.
- `quality data compaction`: Quality reports sent to the AI use side effect IDs referencing the Classify output instead of re-embedding full SideEffect objects with Classification and Signals.
- `signal stripping`: Classification signals (the per-signal `source_file`, `excerpt`, and `reasoning` fields) are omitted from the AI payload. The top-level `reasoning` summary on each Classification is preserved.

### Modified Capabilities
- `ReportPayload serialization`: A new `CompactForAI() ([]byte, error)` method on `ReportPayload` produces the reduced payload. The existing `json.Marshal` path for `--format=json` is unchanged — full-fidelity JSON output is preserved for machine consumers.
- `runTextPath()`: Uses `CompactForAI()` instead of `json.Marshal(payload)` when sending to an AI adapter.

### Removed Capabilities
- None. Full JSON output (`--format=json`) remains unchanged. Only the AI adapter input path is affected.

## Impact

- **Files modified**: `internal/aireport/payload.go` (new compact method), `internal/aireport/runner.go` (call compact in text path), `internal/aireport/runner_steps.go` (optional: docscan content stripping at source)
- **CI**: The "Gaze quality report" step will succeed again once the payload fits within the 1M token window
- **AI report quality**: Minimal impact — the reporter agent receives the same summary metrics, CRAP scores, quality reports, and classification labels. Verbose signal details and raw doc contents were never referenced in the reporter's output format.
- **JSON output consumers**: Zero impact — `--format=json` continues to emit the full unmodified payload

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: PASS

The change preserves all artifact-based communication. The `ReportPayload` struct remains the single interface between pipeline steps and the AI adapter. The compact method is a projection of the same data — no new coupling between steps is introduced.

### II. Composability First

**Assessment**: PASS

Gaze remains independently installable and usable without AI adapters (`--format=json` is unaffected). The compaction logic is self-contained within the `aireport` package and introduces no new external dependencies.

### III. Observable Quality

**Assessment**: PASS

Machine-parseable JSON output (`--format=json`) is unchanged. The AI adapter receives a subset of the same data — all metrics, scores, and classifications are preserved. Provenance metadata (versions, locations) is maintained in the compact output.

### IV. Testability

**Assessment**: PASS

`CompactForAI()` is a pure transformation (struct in, bytes out) testable in isolation with synthetic payloads. No external services, subprocess calls, or filesystem access required. Size assertions can enforce the budget invariant in unit tests.
