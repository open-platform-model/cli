## 1. Delete standalone files

- [x] 1.1 Delete `internal/cmd/module/build.go`
- [x] 1.2 Delete `internal/cmd/module/build_test.go`
- [x] 1.3 Delete `pkg/render/synthesize.go`
- [x] 1.4 Delete `pkg/render/synthesize_test.go`

## 2. Remove module build from command registration

- [x] 2.1 Remove `NewModuleBuildCmd` registration from `internal/cmd/module/mod.go` (remove the `c.AddCommand(NewModuleBuildCmd(cfg))` line and update the command description)

## 3. Remove FromModule workflow and supporting code

- [x] 3.1 Remove `FromModule` function from `internal/workflow/render/render.go`
- [x] 3.2 Remove `resolveModulePath` function from `internal/workflow/render/render.go`
- [x] 3.3 Remove `ReleaseOpts` type from `internal/workflow/render/types.go`
- [x] 3.4 Remove `hasReleaseFile` function from `internal/workflow/render/types.go`
- [x] 3.5 Remove `loadModuleReleaseForRender` function from `internal/workflow/render/values.go`
- [x] 3.6 Remove `LoadModuleReleaseForTest` function from `internal/workflow/render/values.go`
- [x] 3.7 Clean up unused imports in edited files

## 4. Remove orphaned tests

- [x] 4.1 Remove `TestRenderModule_NilConfig` from `internal/workflow/render/render_test.go`
- [x] 4.2 Remove `TestRenderModule_RejectsReleasePackagePath` from `internal/workflow/render/render_test.go`
- [x] 4.3 Remove `TestLoadModuleReleaseForRender_UsesReleaseNameOverride` from `internal/workflow/render/render_test.go`
- [x] 4.4 Clean up unused imports in test file

## 5. Update stale comment

- [x] 5.1 Update comment on `renderPreparedModuleRelease` in `internal/workflow/render/render.go` — remove reference to `FromModule` (currently says "shared execution tail for both FromModule and FromReleaseFile")

## 6. Update documentation

- [x] 6.1 Remove `opm module build` entry from `docs/roadmap.md` ("CLI commands: working today" section, line 32)
- [x] 6.2 Remove open TODO items referencing `opm mod build` from `TODO.md` (lines 34, 36–59, 120, 130)

## 7. Validation

- [x] 7.1 Run `task build` — verify binary compiles
- [x] 7.2 Run `task test` — verify all remaining tests pass
- [x] 7.3 Run `task lint` — verify linter passes
