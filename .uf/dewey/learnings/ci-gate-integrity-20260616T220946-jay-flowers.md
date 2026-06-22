---
tag: ci-gate-integrity
author: jay-flowers
category: pattern
created_at: 2026-06-16T22:09:46Z
identity: ci-gate-integrity-20260616T220946-jay-flowers
tier: draft
---

The spec review council consistently caught a shared finding across 4 of 5 reviewers: the `ThresholdResult.Actual` type needed to change from `int` to `*int` to represent the "metric unavailable" case. This was a design gap where the spec specified behavior ("append a FAIL result with name GazeCRAPload unavailable") without specifying how the struct would represent it. When multiple reviewers independently identify the same issue, it's a strong signal of a real design gap, not a false positive. The council review added about 2 minutes but prevented an implementation impasse that would have required ad-hoc design decisions.
