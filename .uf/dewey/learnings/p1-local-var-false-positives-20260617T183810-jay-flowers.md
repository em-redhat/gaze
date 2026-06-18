---
tag: p1-local-var-false-positives
author: jay-flowers
category: gotcha
created_at: 2026-06-17T18:38:10Z
identity: p1-local-var-false-positives-20260617T183810-jay-flowers
tier: draft
---

When designing scope-aware analysis in Go using the `types` package, the scope hierarchy is not as simple as `Universe → Package → Function → Body`. Go's type checker inserts a file scope between package and function, making depth-based heuristics unreliable for distinguishing parameters from body-local variables. The correct approach is `types.Object` pointer identity: collect the `types.Object` pointers for all signature-level variables (parameters, named returns, receiver) via `info.Defs`, then check if the resolved variable's object is in that set. This is what `collectSignatureVars` does in p1effects.go. For package-level variables, the two-level check `v.Parent().Parent() == types.Universe` still works because the file scope sits between package and function, not between package and Universe. The implementation pivoted from the spec's scope-depth approach to pointer identity after empirical testing revealed the file scope issue.
