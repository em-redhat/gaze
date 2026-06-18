---
tag: p1-local-var-false-positives
author: jay-flowers
category: pattern
created_at: 2026-06-17T18:38:21Z
identity: p1-local-var-false-positives-20260617T183821-jay-flowers
tier: draft
---

The spec review council's most valuable contribution to the p1-local-var-false-positives change was identifying coverage gaps: named return variables and receiver field access were mentioned in the design as externally observable but had no test scenarios. Three of four reviewers independently flagged the named return gap, confirming it was a real omission. Adding these test fixtures (NamedReturnMapWrite and WriteToStructMap with a Container struct) validated two distinct code paths — the FuncType scope detection for named returns and the SelectorExpr unwrapping in unwrapToIdent. The code review then caught that the positive-path tests for these fixtures had shallower assertions than the existing tests — missing tier and description verification. This two-phase review (spec then code) catches different classes of issues: spec review finds missing coverage, code review finds assertion quality.
