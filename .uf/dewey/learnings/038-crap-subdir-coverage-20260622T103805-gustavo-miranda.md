---
tag: 038-crap-subdir-coverage
author: gustavo-miranda
category: gotcha
created_at: 2026-06-22T10:38:05Z
identity: 038-crap-subdir-coverage-20260622T103805-gustavo-miranda
tier: draft
---

During spec review of the 038-crap-subdir-coverage feature, the review found a call site that the original issue analysis missed: runDocscan at cmd/gaze/main.go:767 also uses os.Getwd() as a proxy for the module root. The lesson is that when an issue reports N affected call sites, always grep for ALL os.Getwd() calls in the codebase and evaluate each one — the actual count is typically higher than what the issue author identified. In this case the issue found 5 sites, jflowers' review council found 5, but a thorough codebase scan found 10 total (9 needing fixes, 1 correct as-is in scaffold.go). The spec review process caught the gap before implementation.
