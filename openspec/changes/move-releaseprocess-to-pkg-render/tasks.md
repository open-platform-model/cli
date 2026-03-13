## 1. Move releaseprocess files to pkg/render

- [ ] 1.1 Move `internal/releaseprocess/module.go` → `pkg/render/process.go`: change package to `render`, rename `ProcessModuleRelease` (keep name), update all intra-group type references to same-package (e.g., `*engine.ModuleRenderResult` → `*ModuleResult`, `*modulerelease.ModuleRelease` → `*ModuleRelease`, `match.Match` → `Match`)
- [ ] 1.2 Move `internal/releaseprocess/bundle.go` → `pkg/render/process_bundle.go`: change package to `render`, rename `ProcessBundleRelease` (keep name), update intra-group references
- [ ] 1.3 Move `internal/releaseprocess/synthesize.go` → `pkg/render/synthesize.go`: change package to `render`, rename `SynthesizeModuleRelease` → `SynthesizeModule`, update type references
- [ ] 1.4 Move `internal/releaseprocess/validate.go` → `pkg/render/validate.go` (or the file containing `ValidateConfig`): change package to `render`
- [ ] 1.5 Move `internal/releaseprocess/finalize.go` → `pkg/render/finalize.go` (unexported helpers): change package to `render`
- [ ] 1.6 Move `internal/releaseprocess/module_test.go` → `pkg/render/process_test.go`: change package to `render` (internal test — tests unexported helpers indirectly), remove imports of `internal/runtime/modulerelease` and `internal/runtime/bundlerelease` (now same package), replace `modulerelease.ModuleRelease` → `ModuleRelease`, `modulerelease.ReleaseMetadata` → `ModuleReleaseMetadata`, `bundlerelease.BundleRelease` → `BundleRelease`, `bundlerelease.BundleReleaseMetadata` → `BundleReleaseMetadata`. Note: `ProcessModuleRelease` and `ProcessBundleRelease` keep their names
- [ ] 1.7 Move `internal/releaseprocess/synthesize_test.go` → `pkg/render/synthesize_test.go`: change package to `render`, rename `SynthesizeModuleRelease` → `SynthesizeModule` in test calls, move `makeModuleWithComponents()` test helper along with it
- [ ] 1.8 Move `internal/releaseprocess/validate_test.go` → `pkg/render/validate_test.go`: change package to `render`, no function renames needed (`ValidateConfig` keeps its name)
- [ ] 1.9 Move `internal/releaseprocess/finalize_test.go` → `pkg/render/finalize_test.go`: change package to `render`, no renames needed (`finalizeValue` is unexported, keeps its name)

## 2. Update external callers

- [ ] 2.1 Update `internal/workflow/render/render.go`: change `releaseprocess.SynthesizeModuleRelease` → `render.SynthesizeModule`, `releaseprocess.ProcessModuleRelease` → `render.ProcessModuleRelease`
- [ ] 2.2 Update `internal/workflow/render/render_test.go`: change `releaseprocess.ValidateConfig` → `render.ValidateConfig`
- [ ] 2.3 Update `internal/workflow/render/validation_test.go`: change `releaseprocess.ValidateConfig` → `render.ValidateConfig`
- [ ] 2.4 Update `internal/cmd/module/vet.go`: change `releaseprocess.ValidateConfig` → `render.ValidateConfig`
- [ ] 2.5 Update `pkg/loader/validate_test.go`: change `internal/releaseprocess` import to `pkg/render`, update `releaseprocess.ValidateConfig` → `render.ValidateConfig`
- [ ] 2.6 Update `pkg/loader/validate_diag_test.go`: same as above

## 3. Remove old package

- [ ] 3.1 Delete `internal/releaseprocess/` directory entirely

## 4. Validation

- [ ] 4.1 Run `task build` — confirm compilation succeeds
- [ ] 4.2 Run `task test` — confirm all tests pass
- [ ] 4.3 Run `task lint` — confirm linter passes

## 5. Commits

- [ ] 5.1 Commit tasks 1.1–1.9, 2.1–2.6, 3.1: `refactor(render): move release processing to pkg/render, rename SynthesizeModuleRelease to SynthesizeModule`
