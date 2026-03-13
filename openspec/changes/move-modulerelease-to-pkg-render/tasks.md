## 1. Move modulerelease types to pkg/render

- [x] 1.1 Create `pkg/render/modulerelease.go` from `internal/runtime/modulerelease/release.go`: change package to `render`, rename `ReleaseMetadata` to `ModuleReleaseMetadata`, update `NewModuleRelease` parameter type, update `ModuleRelease.Metadata` field type
- [x] 1.2 Note: `internal/runtime/modulerelease/` has no test files — no tests to move

## 2. Update internal callers — engine package

- [x] 2.1 Update `internal/engine/execute.go`: change import to `pkg/render`, replace `modulerelease.ModuleRelease` with `render.ModuleRelease`, replace `modulerelease.ReleaseMetadata` with `render.ModuleReleaseMetadata`
- [x] 2.2 Update `internal/engine/module_renderer.go`: change import to `pkg/render`, update type references
- [x] 2.3 Update `internal/engine/matchplan_test.go`: change import, update struct literals from `modulerelease.ReleaseMetadata` to `render.ModuleReleaseMetadata`

## 3. Update internal callers — bundlerelease package

- [x] 3.1 Update `internal/runtime/bundlerelease/release.go`: change import to `pkg/render`, update field type `Releases map[string]*render.ModuleRelease`

## 4. Update internal callers — releasefile package

- [x] 4.1 Update `internal/releasefile/get_release_file.go`: change import to `pkg/render`, update `NewModuleRelease` call, update `ReleaseMetadata` struct literals to `ModuleReleaseMetadata`

## 5. Update internal callers — releaseprocess package

- [x] 5.1 Update `internal/releaseprocess/module.go`: change import to `pkg/render`, update parameter types
- [x] 5.2 Update `internal/releaseprocess/synthesize.go`: change import to `pkg/render`, update `NewModuleRelease` call, update `ReleaseMetadata` struct literal to `ModuleReleaseMetadata`
- [x] 5.3 Update `internal/releaseprocess/module_test.go`: change import, update struct literals

## 6. Update internal callers — workflow packages

- [x] 6.1 Update `internal/workflow/render/types.go`: change import to `pkg/render`, update `ReleaseMetadata` → `ModuleReleaseMetadata` field type
- [x] 6.2 Update `internal/workflow/render/render.go`: change import to `pkg/render`, update type references
- [x] 6.3 Update `internal/workflow/render/values.go`: change import to `pkg/render`, update return types
- [x] 6.4 Update `internal/workflow/render/render_test.go`: change import, update struct literals
- [x] 6.5 Update `internal/workflow/render/log_output_test.go`: change import, update struct literals
- [x] 6.6 Update `internal/workflow/apply/apply_test.go`: change import, update struct literals

## 7. Update internal callers — cmd package

- [x] 7.1 Update `internal/cmd/module/verbose_output_test.go`: change import, update struct literals

## 8. Remove old package

- [x] 8.1 Delete `internal/runtime/modulerelease/` directory

## 9. Validation

- [x] 9.1 Run `task build` — confirm compilation succeeds
- [x] 9.2 Run `task test` — confirm all tests pass
- [x] 9.3 Run `task lint` — confirm linter passes

## 10. Commits

- [x] 10.1 Commit tasks 1.1–1.2, 2.1–2.3, 3.1, 4.1, 5.1–5.3, 6.1–6.6, 7.1, 8.1: `refactor(render): move ModuleRelease types to pkg/render, rename ReleaseMetadata to ModuleReleaseMetadata`
