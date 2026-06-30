---
tag: baseline-comparison
author: jay-flowers
category: pattern
created_at: 2026-06-23T17:55:13Z
identity: baseline-comparison-20260623T175513-jay-flowers
tier: draft
---

When adding a new threshold field to the gaze baseline comparison system (internal/crap/compare.go), the change must be threaded through three types: BaselineConfig (config.go, YAML), CompareOptions (compare.go, in-memory), and ComparisonSummary (crap.go, JSON output). The CLI layer (cmd/gaze/main.go loadAndCompare) wires config to options, and buildComparisonSummary propagates options to summary. This three-type pipeline — config struct, options struct, output struct — is the established pattern for all comparison parameters (epsilon, new_function_threshold). The report layer has three separate sites that evaluate new-function violation status: buildComparisonSummary (counting), WriteComparisonJSON (JSON status assignment), and writeComparisonNewFunctions (text formatting). All three must be updated consistently when adding new violation criteria. The nil-guard pattern for GazeCRAP (s.GazeCRAP != nil && *s.GazeCRAP > threshold) must be used at every evaluation site because GazeCRAP is optional (*float64).
