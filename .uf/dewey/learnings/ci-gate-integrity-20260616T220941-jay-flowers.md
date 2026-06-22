---
tag: ci-gate-integrity
author: jay-flowers
category: gotcha
created_at: 2026-06-16T22:09:41Z
identity: ci-gate-integrity-20260616T220941-jay-flowers
tier: draft
---

When changing a Go type from `int` to `*int` across a pipeline, the blast radius extends beyond the obvious production files to include compact/serialization structs and all test files that construct the struct. In the ci-gate-integrity change, the `GazeCRAPload` type change from `int` to `*int` required updates to `crapStepResult`, `ReportSummary`, `compactSummary`, `ThresholdResult.Actual`, and 5+ test files (threshold_test.go, pipeline_internal_test.go, payload_test.go, compact_test.go, main_test.go). The spec review council caught the compact.go omission before implementation, saving a compilation failure during implementation. Lesson: when planning a type change, grep for all struct literals containing the field name across the entire codebase, not just the struct definition.
