## 1. pkg/loader — SynthesizeModuleRelease

- [ ] 1.1 Create `pkg/loader/module_as_release.go` with `SynthesizeModuleRelease(cueCtx, modVal, valuesVal, releaseName, namespace)` function
- [ ] 1.2 Implement Module Gate in `SynthesizeModuleRelease`: call `validateConfig(modVal.LookupPath("#config"), valuesVal, "module", releaseName)`
- [ ] 1.3 Fill `#config` with values: `filledMod := modVal.FillPath(cue.ParsePath("#config"), valuesVal)`, check `filledMod.Err()`
- [ ] 1.4 Extract schema components: `schemaComps := filledMod.LookupPath(cue.ParsePath("#components"))`, error if not exists
- [ ] 1.5 Wrap components in synthetic schema value: `cueCtx.CompileString("{}")` + `FillPath("components", schemaComps)` so `MatchComponents()` finds `"components"`
- [ ] 1.6 Finalize components via `finalizeValue(cueCtx, schemaComps)` → `dataComponents`
- [ ] 1.7 Decode module metadata: `modVal.LookupPath(cue.ParsePath("metadata")).Decode(&modMeta)` into `module.ModuleMetadata`
- [ ] 1.8 Construct `ReleaseMetadata{Name: releaseName, Namespace: namespace}` (UUID left empty)
- [ ] 1.9 Return `modulerelease.NewModuleRelease(relMeta, module.Module{Metadata: modMeta, Raw: modVal}, syntheticSchema, dataComponents)`

## 2. pkg/loader — Tests for SynthesizeModuleRelease

- [ ] 2.1 Add unit tests in `pkg/loader/module_as_release_test.go` covering: success with valid module+debugValues, Module Gate failure on invalid values, non-existent `#components` field, correct release name and namespace in output
- [ ] 2.2 Verify `MatchComponents()` returns a valid, iterable CUE value from the synthetic schema
- [ ] 2.3 Verify `ExecuteComponents()` returns a fully concrete, constraint-free value (`Validate(cue.Concrete(true))` passes)

## 3. internal/cmdutil — RenderRelease synthesis branch

- [ ] 3.1 Add `hasReleaseFile(modulePath string) bool` helper in `internal/cmdutil/render.go` (os.Stat check for `release.cue`)
- [ ] 3.2 Add synthesis branch at the top of the `pkg cue.Value` loading block in `RenderRelease`: `if !hasReleaseFile(modulePath) { ... }`
- [ ] 3.3 In synthesis branch: when `DebugValues && len(opts.Values) == 0`, load module package, extract `debugValues`, validate concreteness, error with `"no release.cue found — add debugValues to module or use -f <values-file>"` when absent
- [ ] 3.4 In synthesis branch: when `len(opts.Values) > 0`, load values from first `-f` file via `loader.LoadValuesFile`
- [ ] 3.5 In synthesis branch: resolve `releaseName` (opts.ReleaseName → `module.metadata.name` → `filepath.Base(modulePath)`)
- [ ] 3.6 In synthesis branch: resolve `moduleNamespace` — use `module.metadata.defaultNamespace` when `k8sConfig.Namespace.Source` is neither `SourceFlag` nor `SourceEnv`, else use `k8sConfig.Namespace.Value`
- [ ] 3.7 In synthesis branch: call `loader.SynthesizeModuleRelease(cueCtx, modVal, valuesVal, releaseName, moduleNamespace)` → `*ModuleRelease`; handle error with `PrintValidationError`
- [ ] 3.8 Refactor common tail (namespace override, `LoadProvider`, `engine.Render`, resource conversion) to be shared by both synthesis and normal paths — extract to avoid duplication if needed, or use `rel` variable set in each branch
- [ ] 3.9 Ensure synthesis branch applies the existing post-synthesis namespace override: `if s == SourceFlag || s == SourceEnv { rel.Metadata.Namespace = namespace }`

## 4. internal/cmd/mod — build and apply

- [ ] 4.1 In `internal/cmd/mod/build.go`: add `DebugValues: len(rf.Values) == 0` to `cmdutil.RenderReleaseOpts` in `runBuild`
- [ ] 4.2 In `internal/cmd/mod/apply.go`: add `DebugValues: len(rf.Values) == 0` to `cmdutil.RenderReleaseOpts` in `runApply`

## 5. Validation

- [ ] 5.1 Manual smoke test: `opm mod build .` in `examples/modules/jellyfin` produces valid manifests without a `release.cue`
- [ ] 5.2 Manual smoke test: `opm mod build . -f <values-file>` in a bare module directory works with explicit values
- [ ] 5.3 Manual smoke test: `opm mod build .` in a directory WITH `release.cue` but no `-f` uses `debugValues` (regression: existing vet behavior now consistent with build)
- [ ] 5.4 Manual smoke test: `opm mod build . -f values.cue` in a directory WITH `release.cue` still uses `-f` values (no regression)
- [ ] 5.5 Run `task test:unit` — all loader and cmdutil tests pass
- [ ] 5.6 Run `task fmt && task lint` — no formatting or lint issues
- [ ] 5.7 Run `task test` — full test suite passes
