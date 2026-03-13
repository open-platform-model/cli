## Context

`internal/engine` is the CUE render pipeline. It takes processed releases + match plans and produces `[]*core.Resource`. After changes 1-3, all its intra-group dependencies (match, modulerelease, bundlerelease) are already in `pkg/render/`. The engine itself has one CLI dependency: 2 `charmbracelet/log.Warn` calls for non-fatal metadata decode errors.

## Goals / Non-Goals

**Goals:**
- Move all engine logic to `pkg/render/`
- Rename types per the agreed naming convention
- Remove `charmbracelet/log` dependency by converting warn calls to `Warnings` slice entries
- `match_alias.go` already deleted in change 1 — no aliases allowed anywhere

**Non-Goals:**
- Change rendering behavior
- Refactor the execute pipeline

## Decisions

### Type renames
| Old | New | Rationale |
|-----|-----|-----------|
| `ModuleRenderer` | `Module` | `render.Module` reads well, avoids redundancy |
| `BundleRenderer` | `Bundle` | Same |
| `NewModuleRenderer()` | `NewModule()` | Follows type name |
| `NewBundleRenderer()` | `NewBundle()` | Follows type name |
| `ModuleRenderResult` | `ModuleResult` | `render.ModuleResult` — "Render" is implied by package |
| `BundleRenderResult` | `BundleResult` | Same |
| `.Render()` | `.Execute()` | Avoids `render.Module.Render()` redundancy |

### Replace charmbracelet/log with warnings slice
The engine currently calls `log.Warn("failed to decode metadata", ...)` in 2 places during transform execution when a rendered resource's metadata can't be decoded. These are non-fatal — the resource is still usable.

**New behavior**: Append a warning string to `ModuleResult.Warnings` instead. The caller (CLI or controller) decides how to surface the warning.

**Rationale**: The `Warnings` slice already exists on the result types. This keeps `pkg/render/` free of any logging framework dependency.

### match_alias.go already deleted
`internal/engine/match_alias.go` was deleted in change 1. All callers that previously referenced `engine.MatchPlan`, `engine.MatchResult`, etc. were updated to use `render.MatchPlan`, `render.MatchResult` directly. No aliases exist anywhere in the codebase.

## Risks / Trade-offs

- **[Medium risk] Rename breadth**: Multiple type renames in one change. Each external caller needs updating. Use `go build ./...` to catch all.
- **[Low risk] Warning behavior change**: Metadata decode errors previously went to stderr via `log.Warn`. Now they're silently collected in the result. Callers that want to display them must check `result.Warnings`. The CLI's workflow/render already handles warnings from this slice.
