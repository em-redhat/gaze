# Unbound Force — managed by uf init

@AGENTS.md
@.opencode/agents/cobalt-crush-dev.md

## Convention Packs

@.opencode/uf/packs/default.md
@.opencode/uf/packs/severity.md
@.opencode/uf/packs/content.md
@.opencode/uf/packs/go.md

## Review Agents (read on-demand)

When performing code review, read the applicable
Divisor agent from .opencode/agents/:
- divisor-guard.md — intent drift, constitution
- divisor-architect.md — structure, patterns, DRY
- divisor-adversary.md — security, error handling
- divisor-testing.md — test quality, assertions
- divisor-sre.md — operations, performance

## Active Technologies
- Go 1.25+ (per `go.mod` directive) + Standard library only (no new dependencies). Existing: `gopkg.in/yaml.v3` (config), `encoding/json` (report output) (039-baseline-gazecrap-threshold)
- N/A — no persistence changes (039-baseline-gazecrap-threshold)

## Recent Changes
- 039-baseline-gazecrap-threshold: Added Go 1.25+ (per `go.mod` directive) + Standard library only (no new dependencies). Existing: `gopkg.in/yaml.v3` (config), `encoding/json` (report output)
