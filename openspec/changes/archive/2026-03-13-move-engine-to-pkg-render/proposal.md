## Why

The `internal/engine` package is the CUE render pipeline — the core of OPM's value proposition. It transforms module components through provider transformers to produce Kubernetes resources. The controller needs this exact logic. Moving it to `pkg/render/` makes it importable by both CLI and controller without duplication.

## What Changes

- Move `ModuleRenderer`, `BundleRenderer`, result types, and transform execution logic from `internal/engine/` into `pkg/render/`
- **Rename types**: `ModuleRenderer` → `Module`, `BundleRenderer` → `Bundle`, `ModuleRenderResult` → `ModuleResult`, `BundleRenderResult` → `BundleResult`, `NewModuleRenderer` → `NewModule`, `NewBundleRenderer` → `NewBundle`, `.Render()` → `.Execute()`
- **Replace `charmbracelet/log`**: convert the 2 `log.Warn` calls to append to `ModuleResult.Warnings` instead
- `match_alias.go` is already deleted in change 1 (no aliases allowed)
- Update all external callers

## Capabilities

### New Capabilities

### Modified Capabilities

- `engine-rendering`: The rendering engine moves from `internal/engine` to `pkg/render` with type renames. `charmbracelet/log` dependency is removed; metadata decode warnings are surfaced through the `Warnings` slice on result types.

## Impact

- **Files moved**: All files from `internal/engine/` → `pkg/render/` (engine.go, module_renderer.go, bundle_renderer.go, execute.go, context.go, component.go + tests). `match_alias.go` already deleted in change 1.
- **Dependency removed**: `github.com/charmbracelet/log` no longer imported by the render package
- **External callers updated** (3 files — `engine.MatchPlan`/`engine.MatchResult` references already updated to `render.*` in change 1):
  - `internal/workflow/render/types.go`: `engine.ComponentSummary` → `render.ComponentSummary`
  - `internal/workflow/render/render.go`: indirect via releaseprocess return types
  - `internal/cmd/module/verbose_output_test.go`: `engine.ComponentSummary` → `render.ComponentSummary`
  - `internal/releaseprocess/module.go`: `engine.NewModuleRenderer` → `render.NewModule`, `engine.ModuleRenderResult` → `render.ModuleResult`
- **SemVer**: MINOR (new public API surface)
