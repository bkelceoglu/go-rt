---
name: gort-architect
description: >-
  Use this skill whenever adding or refactoring plugins, strategies, multi-turn agentic
  providers (Crescendo, GOAT, Hydra), template injectors, or graders in the go-rt project.
  Ensures 100% alignment with docs/MASTER_PLAN.md and shared brain files.
---

# GO-RT Architect Skill & Runbook

When implementing features in `~/codes/go-rt`, follow this exact workflow so that code produced in `agy` (CLI) and `antigravity-ide` is indistinguishable.

## 1. Pre-Flight Checklist
1. Read [`docs/brain/CONTEXT.md`](../../../docs/brain/CONTEXT.md) and [`docs/brain/TASKS.md`](../../../docs/brain/TASKS.md).
2. Check [`docs/MASTER_PLAN.md`](../../../docs/MASTER_PLAN.md) for the exact specification of the target package.

## 2. Component Recipes

### A. Adding a New Redteam Plugin
1. If the plugin is template + rubric based (e.g., `sql-injection`, `bola`, `bfla`, `rbac`, `ssrf`, `shell-injection`):
   - Create `embed/plugins/<plugin_id>.yaml` with `id`, `severity`, `metric`, `template`, and `rubric`.
   - Register it in `internal/plugin/registry.go`.
2. If the plugin requires custom generation logic (e.g., `intent`, `pii`, `harmbench`):
   - Implement `domain.Plugin` in `internal/plugin/<name>.go` composing `plugin.BaseGenerator`.
   - Keep each helper function $\le 15$ lines.

### B. Adding a Static Strategy (`base64`, `rot13`, `leetspeak`, `homoglyph`)
1. Implement a pure transformation function `func(input string) string` in `internal/strategy/static.go`.
2. Wrap it with `NewStaticStrategy(id, metricSuffix, transformFn)` which:
   - Clones `TestCase` and `Vars`.
   - Stores original `Vars[injectVar]` in `Metadata.OriginalText`.
   - Replaces `Vars[injectVar]` with transformed payload.
   - Appends `/MetricSuffix` to each `Assertion.Metric`.

### C. Adding a Multi-Turn Agentic Strategy (`crescendo`, `goat`, `hydra`)
1. Implement `domain.Provider` in `internal/provider/agentic/<name>.go` as a **Decorator** wrapping `Target domain.Provider`.
2. Use `ConversationMemory` with `RollbackLastTurn()` support for backtracking on target refusals.
3. In `internal/strategy/multi_turn.go`, attach the decorator to `TestCase.TargetOverride`.

## 3. Post-Flight Verification
1. Run `make check` (`go fmt ./... && go vet ./... && go test -race ./...`).
2. Mark completed items in [`docs/brain/TASKS.md`](../../../docs/brain/TASKS.md) and update [`docs/brain/CONTEXT.md`](../../../docs/brain/CONTEXT.md).
