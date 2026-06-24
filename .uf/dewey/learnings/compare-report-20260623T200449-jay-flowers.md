---
tag: compare-report
author: jay-flowers
category: pattern
created_at: 2026-06-23T20:04:49Z
identity: compare-report-20260623T200449-jay-flowers
tier: draft
---

The writeComparisonDeltaTable function in internal/crap/compare_report.go uses a conditional rendering pattern for optional data: it first scans all matching deltas to detect whether any have GazeCRAP data (hasGaze flag), then uses one of two rendering branches — a two-row format when GazeCRAP exists (function name on its own line, then indented GazeCRAP and CRAP metric rows) or the original single-row format when no GazeCRAP data is present. This pattern keeps backward compatibility while extending the display. The 80-column terminal width constraint from AGENTS.md was the key architectural driver — adding inline columns would have exceeded 108 chars, so the two-row format was chosen to stay under 80. The width compliance test scopes its assertion to the comparison section only (after the "--- Baseline Comparison" header), not the preceding WriteText output which has its own width management.
