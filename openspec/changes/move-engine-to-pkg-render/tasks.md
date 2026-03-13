## 1. Move engine files to pkg/render

- [ ] 1.1 Move `internal/engine/module_renderer.go` → `pkg/render/module_renderer.go`: change package to `render`, rename `ModuleRenderer` → `Module`, `NewModuleRenderer` → `NewModule`, `ModuleRenderResult` → `ModuleResult`, rename `.Render()` → `.Execute()`, remove imports of `internal/match` and `internal/runtime/modulerelease` (now same package)
- [ ] 1.2 Move `internal/engine/bundle_renderer.go` → `pkg/render/bundle_renderer.go`: change package to `render`, rename `BundleRenderer` → `Bundle`, `NewBundleRenderer` → `NewBundle`, `BundleRenderResult` → `BundleResult`, rename `.Render()` → `.Execute()`, remove imports of `internal/match`, `internal/runtime/modulerelease`, `internal/runtime/bundlerelease` (now same package)
- [ ] 1.3 Move `internal/engine/execute.go` → `pkg/render/execute.go`: change package to `render`, replace the 2 `log.Warn` calls with appending to a warnings accumulator, remove `charmbracelet/log` import, thread warnings back into `ModuleResult.Warnings`
- [ ] 1.4 Move `internal/engine/context.go` → `pkg/render/context.go` (if exists): change package to `render`
- [ ] 1.5 Move `internal/engine/component.go` → `pkg/render/component.go` (if exists): change package to `render`
- [ ] 1.6 Move engine test files to `pkg/render/`, update package declarations and imports
- [ ] 1.7 Note: `match_alias.go` was already deleted in change 1 — nothing to do here

## 2. Update external callers

- [ ] 2.1 Update `internal/workflow/render/types.go`: change `engine.ComponentSummary` → `render.ComponentSummary` (note: `engine.MatchPlan` was already updated to `render.MatchPlan` in change 1)
- [ ] 2.2 Update `internal/workflow/render/render.go`: update any `engine.*` references to `render.*`, adjust for `ModuleRenderResult` → `ModuleResult` rename
- [ ] 2.3 Update `internal/cmd/module/verbose_output_test.go`: change `engine.ComponentSummary` → `render.ComponentSummary` (note: `engine.MatchPlan`/`engine.MatchResult` already updated to `render.*` in change 1)
- [ ] 2.4 Update `internal/releaseprocess/module.go`: change `engine.NewModuleRenderer` → `render.NewModule`, `engine.ModuleRenderResult` → `render.ModuleResult`, `engine.ModuleRenderer.Render` → `render.Module.Execute`
- [ ] 2.5 Update `internal/releaseprocess/bundle.go`: change `engine.NewBundleRenderer` → `render.NewBundle`, `engine.BundleRenderResult` → `render.BundleResult`, `.Render()` → `.Execute()`

## 3. Remove old package

- [ ] 3.1 Delete `internal/engine/` directory entirely

## 4. Validation

- [ ] 4.1 Run `task build` — confirm compilation succeeds
- [ ] 4.2 Run `task test` — confirm all tests pass
- [ ] 4.3 Run `task lint` — confirm linter passes
- [ ] 4.4 Verify `charmbracelet/log` does not appear in `pkg/render/` imports: `grep -r "charmbracelet/log" pkg/render/`

## 5. Commits

- [ ] 5.1 Commit tasks 1.1–1.2, 1.4–1.7, 2.1–2.5, 3.1: `refactor(render): move engine renderers to pkg/render with type renames`
- [ ] 5.2 Commit task 1.3: `refactor(render): replace charmbracelet/log with warnings slice in execute`
