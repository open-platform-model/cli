## 1. Receiver Methods on core.Module

- [ ] 1.1 Add `ResolvePath() error` to `internal/core/module.go` — resolve `ModulePath` to absolute path, verify directory exists, verify `cue.mod/` present; mutate `ModulePath` in-place on success
- [ ] 1.2 Add `Validate() error` to `internal/core/module.go` — check `ModulePath != ""`, `Metadata != nil`, `Metadata.Name != ""`, `Metadata.FQN != ""`; no CUE concreteness checks
- [ ] 1.3 Add unit tests for `ResolvePath()` in `internal/core/module_test.go` — valid path, non-existent directory, missing `cue.mod/`, relative-to-absolute conversion
- [ ] 1.4 Add unit tests for `Validate()` in `internal/core/module_test.go` — fully populated passes, nil Metadata fails, empty Name fails, empty FQN fails, non-concrete CUE values pass

## 2. module.Load() Constructor

- [ ] 2.1 Add `Load(cueCtx *cue.Context, modulePath, registry string) (*core.Module, error)` to `internal/build/module/loader.go` — construct `core.Module{ModulePath: modulePath}`, call `mod.ResolvePath()`, run AST inspection via existing `InspectModule` internally, populate `Metadata.Name`, `Metadata.DefaultNamespace`, and store `PkgName` on the module
- [ ] 2.2 Decide pkgName storage: add unexported `pkgName string` field to `core.Module` with a `PkgName() string` accessor, or return it as a second value from `Load()`; wire into `release.Build()` via `release.Options.PkgName`
- [ ] 2.3 Add unit tests for `module.Load()` — valid module returns populated `*core.Module` with resolved path and metadata; invalid path returns error; module without string-literal name returns `*core.Module` with empty `Metadata.Name`

## 3. Remove Superseded Code

- [ ] 3.1 Delete standalone `ResolvePath(modulePath string) (string, error)` function from `internal/build/module/loader.go`
- [ ] 3.2 Delete `ExtractMetadata(cueCtx, modulePath, registry)` function from `internal/build/module/loader.go`
- [ ] 3.3 Delete `MetadataPreview` type from `internal/build/module/types.go`
- [ ] 3.4 Verify `Inspection` type is still needed internally by `Load()`; if only used inside `Load()`, consider inlining

## 4. Wire Into Pipeline

- [ ] 4.1 Update `pipeline.Render()` PREPARATION phase to call `module.Load(p.cueCtx, opts.ModulePath, opts.Registry)` instead of `module.ResolvePath()` + `p.releaseBuilder.InspectModule()` + `module.ExtractMetadata()` fallback
- [ ] 4.2 Call `mod.Validate()` after `module.Load()` returns; return fatal error if validation fails
- [ ] 4.3 Update release name derivation: read `mod.Metadata.Name` instead of `moduleMeta.Name`
- [ ] 4.4 Update namespace resolution: read `mod.Metadata.DefaultNamespace` instead of `moduleMeta.DefaultNamespace`
- [ ] 4.5 Pass `mod.PkgName()` (or equivalent) to `release.Build()` via `release.Options.PkgName`
- [ ] 4.6 Remove `p.releaseBuilder.InspectModule()` call and any remaining references to `module.MetadataPreview` in pipeline

## 5. Verify

- [ ] 5.1 Run `task test` — all existing tests must pass with no behavior change
- [ ] 5.2 Run `task check` — fmt + vet + test all green
- [ ] 5.3 Manual smoke test: `opm mod build` on a test fixture module produces identical output before and after
