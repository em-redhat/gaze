---
tag: ci-gate-integrity
author: jay-flowers
category: pattern
created_at: 2026-06-16T22:09:50Z
identity: ci-gate-integrity-20260616T220950-jay-flowers
tier: draft
---

When fixing CI gate integrity bugs where "no data" was silently reported as "pass", the most important architectural decision was using `*int` consistently throughout the pipeline to distinguish nil (unavailable) from zero (computed value). The `gaze crap` command already used `*int` for `crap.Summary.GazeCRAPload`, but the report pipeline converted it to plain `int` at the boundary, losing the nil signal. The fix was to propagate `*int` through all intermediate structs: `crapStepResult`, `ReportSummary`, `compactSummary`, and `ThresholdResult.Actual`. The key insight: if the source type is `*int`, every intermediate struct in the pipeline must also be `*int`, or you'll lose the nil signal at whichever boundary does the `*int`-to-`int` conversion.
