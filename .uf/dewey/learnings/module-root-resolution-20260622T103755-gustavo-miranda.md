---
tag: module-root-resolution
author: gustavo-miranda
category: pattern
created_at: 2026-06-22T10:37:55Z
identity: module-root-resolution-20260622T103755-gustavo-miranda
tier: draft
---

When fixing os.Getwd()-as-module-root bugs in Go CLI tools, the comprehensive approach is to extract a shared FindModuleRoot(startDir) function to a loader/utility package that walks up the directory tree looking for go.mod. The key insight is that the bug scope is always larger than initially reported — in gaze, what appeared as a single gaze-crap bug actually affected 9 call sites across 4 files (cmd/gaze/main.go, internal/crap/contract.go, internal/crap/coverage.go, internal/aireport/runner_steps.go). The pattern is to push os.Getwd() calls to the CLI entry points and thread moduleDir as a parameter through all internal functions. Functions like loadGazeConfigBestEffort that called os.Getwd() internally should instead accept moduleDir as a parameter (the cmd/gaze/main.go version already had this correct pattern). Defense-in-depth diagnostics (per-entry stderr warnings plus error on 0/N resolution) catch edge cases beyond the walk-up fix like corrupted profiles or wrong modules.
