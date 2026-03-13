## 1. Move bundlerelease types to pkg/render

- [ ] 1.1 Create `pkg/render/bundlerelease.go` from `internal/runtime/bundlerelease/release.go`: change package to `render`, update `Releases` field type from `*modulerelease.ModuleRelease` to `*ModuleRelease` (same package now), remove `//nolint:revive` stutter comment, remove the `modulerelease` import
- [ ] 1.2 Note: `internal/runtime/bundlerelease/` has no test files — no tests to move

## 2. Update internal callers

- [ ] 2.1 Update `internal/engine/bundle_renderer.go`: change import to `pkg/render`, update `*bundlerelease.BundleRelease` → `*render.BundleRelease`
- [ ] 2.2 Update `internal/engine/matchplan_test.go`: change import, update struct literals
- [ ] 2.3 Update `internal/releasefile/get_release_file.go`: change import to `pkg/render`, update field types and struct literals for `BundleRelease` and `BundleReleaseMetadata`
- [ ] 2.4 Update `internal/releaseprocess/bundle.go`: change import to `pkg/render`, update parameter type
- [ ] 2.5 Update `internal/releaseprocess/module_test.go`: change import, update struct literals

## 3. Remove old package

- [ ] 3.1 Delete `internal/runtime/bundlerelease/` directory
- [ ] 3.2 Remove `internal/runtime/` directory if now empty

## 4. Validation

- [ ] 4.1 Run `task build` — confirm compilation succeeds
- [ ] 4.2 Run `task test` — confirm all tests pass
- [ ] 4.3 Run `task lint` — confirm linter passes

## 5. Commits

- [ ] 5.1 Commit tasks 1.1, 2.1–2.5, 3.1–3.2: `refactor(render): move BundleRelease types to pkg/render`
