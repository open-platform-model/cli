## 1. Receiver Methods on core.Module

- [x] 1.1 Add `ResolvePath() error` to `internal/core/module.go` — resolve `ModulePath` to absolute path, verify directory exists, verify `cue.mod/` present; mutate `ModulePath` in-place on success
- [x] 1.2 Add `Validate() error` to `internal/core/module.go` — check `ModulePath != ""`, `Metadata != nil`, `Metadata.Name != ""`; no CUE concreteness checks (FQN omitted — only available after Phase 2 CUE evaluation)
- [x] 1.3 Add unit tests for `ResolvePath()` in `internal/core/module_test.go` — valid path, non-existent directory, missing `cue.mod/`, relative-to-absolute conversion
- [x] 1.4 Add unit tests for `Validate()` in `internal/core/module_test.go` — fully populated passes, nil Metadata fails, empty Name fails, non-concrete CUE values pass, FQN not checked

## 2. module.Load() Constructor

- [x] 2.1 Add `Load(cueCtx *cue.Context, modulePath, registry string) (*core.Module, error)` to `internal/build/module/loader.go` — construct `core.Module{ModulePath: modulePath}`, call `mod.ResolvePath()`, run AST inspection via internal `inspectModule`, populate `Metadata.Name`, `Metadata.DefaultNamespace`, and store `PkgName` on the module
- [x] 2.2 Decide pkgName storage: added unexported `pkgName string` field to `core.Module` with a `PkgName() string` accessor and `SetPkgName(string)` setter; wired into `release.Build()` via `release.Options.PkgName` through `mod.PkgName()`
- [x] 2.3 Add unit tests for `module.Load()` — valid module returns populated `*core.Module` with resolved path and metadata; invalid path returns error; module without string-literal name returns `*core.Module` with empty `Metadata.Name`

## 3. Remove Superseded Code

- [x] 3.1 Delete standalone `ResolvePath(modulePath string) (string, error)` function from `internal/build/module/loader.go`
- [x] 3.2 Delete `ExtractMetadata(cueCtx, modulePath, registry)` function from `internal/build/module/loader.go`
- [x] 3.3 Delete `MetadataPreview` type from `internal/build/module/types.go`
- [x] 3.4 `Inspection` type kept exported in `types.go` (used internally by `Load()`/`inspectModule()`; `InspectModule` public function removed and replaced by unexported `inspectModule`; `Builder.InspectModule()` delegate removed; tests migrated to `module.Load()`)

## 4. Wire Into Pipeline

- [x] 4.1 Update `pipeline.Render()` PREPARATION phase to call `module.Load(p.cueCtx, opts.ModulePath, opts.Registry)` instead of `module.ResolvePath()` + `p.releaseBuilder.InspectModule()` + `module.ExtractMetadata()` fallback
- [x] 4.2 Call `mod.Validate()` after `module.Load()` returns; return fatal error if validation fails
- [x] 4.3 Update release name derivation: read `mod.Metadata.Name` instead of `moduleMeta.Name`
- [x] 4.4 Update namespace resolution: read `mod.Metadata.DefaultNamespace` instead of `moduleMeta.DefaultNamespace`
- [x] 4.5 Pass `mod.PkgName()` to `release.Build()` via `release.Options.PkgName`
- [x] 4.6 Remove `p.releaseBuilder.InspectModule()` call and all references to `module.MetadataPreview` in pipeline; `Builder.CueContext()` method also removed (only used for old ExtractMetadata fallback)

## 5. Verify

- [x] 5.1 Run `task test` — all existing tests pass with no behavior change
- [x] 5.2 Run `task check` — fmt + vet + test all green; lint issues are all pre-existing (34 issues, none introduced by this change)
- [x] 5.3 Manual smoke test: `opm mod build` on a test fixture module produces identical output before and after
