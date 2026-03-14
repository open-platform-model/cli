## 1. Move ModuleRelease types to pkg/module

- [x] 1.1 Create `pkg/module/release.go` with `ModuleRelease`, `ReleaseMetadata`, `NewRelease()`, `MatchComponents()`, and `ExecuteComponents()` — change package to `module`, update internal import of `module.Module` to just `Module`
- [x] 1.2 Delete `pkg/render/modulerelease.go`

## 2. Move BundleRelease types to pkg/bundle

- [x] 2.1 Create `pkg/bundle/release.go` with `BundleRelease`, `ReleaseMetadata` — change package to `bundle`, add import of `pkg/module` for `*module.Release` in `Releases` field, update internal import of `bundle.Bundle` to just `Bundle`
- [x] 2.2 Delete `pkg/render/bundlerelease.go`

## 3. Update pkg/render imports

- [x] 3.1 Update `pkg/render/process_modulerelease.go` — import `pkg/module`, reference `*module.Release` and `*module.ReleaseMetadata`
- [x] 3.2 Update `pkg/render/process_bundlerelease.go` — import `pkg/bundle`, reference `*bundle.Release` and `*bundle.ReleaseMetadata`
- [x] 3.3 Update `pkg/render/execute.go` — import `pkg/module`, reference `*module.Release`
- [x] 3.4 Update `pkg/render/module_renderer.go` and `pkg/render/bundle_renderer.go` — update type references
- [x] 3.5 Update `pkg/render/matchplan_test.go` — update type references in tests

## 4. Update internal/ consumer imports

- [x] 4.1 Update `internal/releasefile/get_release_file.go` — change `render.ModuleRelease` → `module.Release`, `render.BundleRelease` → `bundle.Release`
- [x] 4.2 Update `internal/workflow/render/types.go` and `internal/workflow/render/values.go` — change `pkgrender.ModuleRelease` → `module.Release`
- [x] 4.3 Update `internal/workflow/render/render.go` — update type references
- [x] 4.4 Update test files: `internal/workflow/render/render_test.go`, `internal/workflow/render/log_output_test.go`, `internal/workflow/apply/apply_test.go`, `internal/cmd/module/verbose_output_test.go`

## 5. Validation

- [x] 5.1 Run `task build` — verify compilation
- [x] 5.2 Run `task test` — verify all tests pass
- [x] 5.3 Run `task lint` — verify linter passes
