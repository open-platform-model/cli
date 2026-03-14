## 1. Simplify `module.Release` type in `pkg/module`

- [ ] 1.1 Rename `RawCUE` field to `Spec` in `pkg/module/release.go`
- [ ] 1.2 Remove `DataComponents` field from `Release`
- [ ] 1.3 Remove `Config` field from `Release`
- [ ] 1.4 Remove `NewRelease` constructor
- [ ] 1.5 Remove `ExecuteComponents()` method
- [ ] 1.6 Keep `MatchComponents()` method — update to read from `Spec` instead of `RawCUE`
- [ ] 1.7 Ensure `Values` field exists on `Release` (already present, keep it)

## 2. Add `ParseModuleRelease` in `pkg/module`

- [ ] 2.1 Add `ParseModuleRelease(ctx context.Context, spec cue.Value, mod Module, values []cue.Value) (*Release, error)` to `pkg/module/release.go` (or a new file `pkg/module/parse.go`)
- [ ] 2.2 Implement: call `render.ValidateConfig(mod.Config, values, "module", name)` to validate and merge values
- [ ] 2.3 Implement: fill merged values into `spec` via `spec.FillPath(cue.ParsePath("values"), merged)`
- [ ] 2.4 Implement: validate filled spec with `cue.Concrete(true)`
- [ ] 2.5 Implement: decode `*ReleaseMetadata` from filled spec's `metadata` field
- [ ] 2.6 Implement: construct and return `*Release{Metadata, Module, Spec, Values}`

## 3. Simplify `ProcessModuleRelease` in `pkg/render`

- [ ] 3.1 Change `ProcessModuleRelease` signature: accept `(ctx context.Context, rel *module.Release, p *provider.Provider)`, return `(*ModuleResult, error)` in `pkg/render/process_modulerelease.go`
- [ ] 3.2 Remove `ValidateConfig` call from `ProcessModuleRelease`
- [ ] 3.3 Remove values filling / mutation logic (`mr.Values = merged`, `mr.RawCUE = mr.RawCUE.FillPath(...)`, `mr.DataComponents = ...`)
- [ ] 3.4 Read schema components via `rel.MatchComponents()`
- [ ] 3.5 Derive finalized components via `finalizeValue(p.Data.Context(), schemaComponents)` as a local variable
- [ ] 3.6 Compute match plan via `Match(schemaComponents, p)`
- [ ] 3.7 Execute via `NewModule(p).Execute(ctx, rel, schemaComponents, dataComponents, plan)` (or adjusted signature)
- [ ] 3.8 Return `*ModuleResult`
- [ ] 3.9 Update `pkg/render/process_test.go`: construct `*module.Release` with prepared fields, assert on returned `*ModuleResult`

## 4. Update module renderer and execution

- [ ] 4.1 Update `Module.Execute()` signature in `pkg/render/module_renderer.go`: accept `schemaComponents` and `dataComponents` as arguments instead of reading from `rel.ExecuteComponents()` and `rel.MatchComponents()`
- [ ] 4.2 Update `Module.Execute()` body: use argument values instead of method calls on release
- [ ] 4.3 Update `executeTransforms` in `pkg/render/execute.go`: field access from `rel.RawCUE` to `rel.Spec` where needed
- [ ] 4.4 Update `executePair` in `pkg/render/execute.go`: same field access updates
- [ ] 4.5 Update `injectContext` in `pkg/render/execute.go`: metadata reads stay as `rel.Metadata.X` and `rel.Module.Metadata.X` (no change needed if already direct)
- [ ] 4.6 Update `pkg/render/matchplan_test.go`: update release construction and `Execute()` call sites

## 5. Update `internal/releasefile`

- [ ] 5.1 Update `FileRelease` struct in `internal/releasefile/get_release_file.go`: replace `Module *module.Release` with raw parse fields (spec `cue.Value`, best-effort module info)
- [ ] 5.2 Update `bareModuleRelease()`: return raw parse data instead of `*module.Release`
- [ ] 5.3 Remove `NewRelease` call from `bareModuleRelease()`
- [ ] 5.4 Update `internal/releasefile/get_release_file_test.go`: adjust assertions to match new `FileRelease` shape

## 6. Update orchestration in `internal/workflow/render`

- [ ] 6.1 Update `FromReleaseFile()` in `internal/workflow/render/render.go`: extract raw spec and module info from `FileRelease`
- [ ] 6.2 Apply `--module` injection on raw spec `cue.Value` (not on `Release.RawCUE`)
- [ ] 6.3 Build `module.Module` from available module data
- [ ] 6.4 Call `module.ParseModuleRelease(ctx, spec, mod, valuesVals)` to get `*module.Release`
- [ ] 6.5 Apply namespace override on `rel.Metadata.Namespace` if needed
- [ ] 6.6 Call `render.ProcessModuleRelease(ctx, rel, p)` to get `*render.ModuleResult`
- [ ] 6.7 Update `renderPreparedModuleRelease` or inline its logic into `FromReleaseFile`
- [ ] 6.8 Update result assembly: read `rel.Metadata` and `rel.Module.Metadata` directly
- [ ] 6.9 Update `resolveReleaseValues`: change `rel.RawCUE` references to raw spec `cue.Value`
- [ ] 6.10 Update `internal/workflow/render/render_test.go`: adjust release construction and field references

## 7. Update bundle types

- [ ] 7.1 Rename `RawCUE` → `Spec` on `bundle.Release` in `pkg/bundle/release.go` for consistency
- [ ] 7.2 Update `ProcessBundleRelease` in `pkg/render/process_bundlerelease.go` for field rename
- [ ] 7.3 Update `Bundle.Execute()` in `pkg/render/bundle_renderer.go` for field access changes

## 8. Validation

- [ ] 8.1 Run `task test` — all existing tests pass
- [ ] 8.2 Run `task lint` — no lint issues
- [ ] 8.3 Run `task build` — clean build
