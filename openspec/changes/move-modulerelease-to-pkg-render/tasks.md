## 1. Move modulerelease types to pkg/render

- [ ] 1.1 Create `pkg/render/modulerelease.go` from `internal/runtime/modulerelease/release.go`: change package to `render`, rename `ReleaseMetadata` to `ModuleReleaseMetadata`, update `NewModuleRelease` parameter type, update `ModuleRelease.Metadata` field type
- [ ] 1.2 Note: `internal/runtime/modulerelease/` has no test files — no tests to move

## 2. Update internal callers — engine package

- [ ] 2.1 Update `internal/engine/execute.go`: change import to `pkg/render`, replace `modulerelease.ModuleRelease` with `render.ModuleRelease`, replace `modulerelease.ReleaseMetadata` with `render.ModuleReleaseMetadata`
- [ ] 2.2 Update `internal/engine/module_renderer.go`: change import to `pkg/render`, update type references
- [ ] 2.3 Update `internal/engine/matchplan_test.go`: change import, update struct literals from `modulerelease.ReleaseMetadata` to `render.ModuleReleaseMetadata`

## 3. Update internal callers — bundlerelease package

- [ ] 3.1 Update `internal/runtime/bundlerelease/release.go`: change import to `pkg/render`, update field type `Releases map[string]*render.ModuleRelease`

## 4. Update internal callers — releasefile package

- [ ] 4.1 Update `internal/releasefile/get_release_file.go`: change import to `pkg/render`, update `NewModuleRelease` call, update `ReleaseMetadata` struct literals to `ModuleReleaseMetadata`

## 5. Update internal callers — releaseprocess package

- [ ] 5.1 Update `internal/releaseprocess/module.go`: change import to `pkg/render`, update parameter types
- [ ] 5.2 Update `internal/releaseprocess/synthesize.go`: change import to `pkg/render`, update `NewModuleRelease` call, update `ReleaseMetadata` struct literal to `ModuleReleaseMetadata`
- [ ] 5.3 Update `internal/releaseprocess/module_test.go`: change import, update struct literals

## 6. Update internal callers — workflow packages

- [ ] 6.1 Update `internal/workflow/render/types.go`: change import to `pkg/render`, update `ReleaseMetadata` → `ModuleReleaseMetadata` field type
- [ ] 6.2 Update `internal/workflow/render/render.go`: change import to `pkg/render`, update type references
- [ ] 6.3 Update `internal/workflow/render/values.go`: change import to `pkg/render`, update return types
- [ ] 6.4 Update `internal/workflow/render/render_test.go`: change import, update struct literals
- [ ] 6.5 Update `internal/workflow/render/log_output_test.go`: change import, update struct literals
- [ ] 6.6 Update `internal/workflow/apply/apply_test.go`: change import, update struct literals

## 7. Update internal callers — cmd package

- [ ] 7.1 Update `internal/cmd/module/verbose_output_test.go`: change import, update struct literals

## 8. Remove old package

- [ ] 8.1 Delete `internal/runtime/modulerelease/` directory

## 9. Validation

- [ ] 9.1 Run `task build` — confirm compilation succeeds
- [ ] 9.2 Run `task test` — confirm all tests pass
- [ ] 9.3 Run `task lint` — confirm linter passes

## 10. Commits

- [ ] 10.1 Commit tasks 1.1–1.2, 2.1–2.3, 3.1, 4.1, 5.1–5.3, 6.1–6.6, 7.1, 8.1: `refactor(render): move ModuleRelease types to pkg/render, rename ReleaseMetadata to ModuleReleaseMetadata`
