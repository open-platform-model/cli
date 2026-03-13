## Why

The `internal/match` package implements the component-to-transformer matching algorithm — pure business logic with zero CLI dependencies. It needs to be importable by the Kubernetes controller (poc-controller) to share the same matching logic. This is the first step in a series of changes that will create a unified `pkg/render/` package containing the full render pipeline, shared between CLI and controller.

## What Changes

- Move `internal/match/` into the new `pkg/render/` package
- All exported types (`MatchPlan`, `MatchResult`, `MatchedPair`, `NonMatchedPair`) and the `Match()` function become part of `pkg/render/`
- Update all internal callers to import from `pkg/render/` instead of `internal/match/`
- This change creates the `pkg/render/` package that subsequent changes will build upon

## Capabilities

### New Capabilities

- `pkg-render-match`: The component-to-transformer matching algorithm is publicly importable from `pkg/render/`. External consumers (e.g., the Kubernetes controller) can call `render.Match()` and use `render.MatchPlan` without depending on CLI internals.

### Modified Capabilities

- `component-matching`: The matching implementation moves from `internal/match` to `pkg/render/`. No behavioral change — same algorithm, same types, new location.

## Impact

- **Files moved**: `internal/match/match.go`, `internal/match/match_test.go` → `pkg/render/match.go`, `pkg/render/match_test.go`
- **File deleted**: `internal/engine/match_alias.go` — type aliases are forbidden; delete and update all consumers directly
- **Internal callers updated** (7 files):
  - `internal/engine/execute.go` — `match.MatchPlan` → `render.MatchPlan`, `match.MatchedPair` → `render.MatchedPair`
  - `internal/engine/module_renderer.go` — `match.MatchPlan` → `render.MatchPlan`
  - `internal/engine/bundle_renderer.go` — `match.Match()` → `render.Match()`
  - `internal/releaseprocess/module.go` — `match.Match()` → `render.Match()`
  - `internal/workflow/render/types.go` — `engine.MatchPlan` → `render.MatchPlan` (was consuming alias)
  - `internal/cmd/module/verbose_output_test.go` — `engine.MatchPlan` → `render.MatchPlan`, `engine.MatchResult` → `render.MatchResult` (was consuming alias)
- **No breaking public API changes** — `internal/match` was never importable externally
- **SemVer**: MINOR (new public API surface)
