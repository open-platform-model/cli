## 1. Create `internal/loader/` package

- [ ] 1.1 Create `internal/loader/loader.go` with package declaration and `Load(ctx *cue.Context, modulePath, registry string) (*core.Module, error)` signature
- [ ] 1.2 Implement path resolution step: call `mod.ResolvePath()` and return error on failure
- [ ] 1.3 Implement CUE instance loading: set `CUE_REGISTRY` env var when registry non-empty (with `defer Unsetenv`), call `load.Instances([]string{"."}, cfg)`, return descriptive error if no instances or `inst.Err != nil`
- [ ] 1.4 Implement CUE evaluation: call `cueCtx.BuildInstance(inst)`, return wrapped error if `baseValue.Err() != nil`
- [ ] 1.5 Implement metadata extraction: copy `extractModuleMetadata` helper from legacy loader; extract name, fqn (with `apiVersion` fallback), version, uuid, defaultNamespace, labels into `core.ModuleMetadata`
- [ ] 1.6 Implement `#config` extraction: `LookupPath(cue.ParsePath("#config"))` → set `mod.Config` if exists
- [ ] 1.7 Implement `values` extraction: `LookupPath(cue.ParsePath("values"))` → set `mod.Values` if exists
- [ ] 1.8 Implement `#components` extraction: `LookupPath(cue.ParsePath("#components"))` → call `core.ExtractComponents`, set `mod.Components` if exists
- [ ] 1.9 Set `mod.Raw = baseValue` (requires `pipeline-core-raw` to land first; use `mod.SetCUEValue(baseValue)` as temporary fallback if not yet merged)
- [ ] 1.10 Add `output.Debug(...)` call on success logging path, name, fqn, version, defaultNamespace, component count

## 2. Write tests for `internal/loader/`

- [ ] 2.1 Add table-driven test `TestLoad_PathResolution` covering: relative path resolves, non-existent path returns error, missing `cue.mod/` returns error
- [ ] 2.2 Add test `TestLoad_Success` using an existing fixture module (e.g., `tests/fixtures/`) — assert all `core.Module` fields are populated and `mod.Validate()` passes
- [ ] 2.3 Add test `TestLoad_PartialMetadata` — use a minimal fixture with only `metadata.name` set; assert absent fields are zero values and no error is returned
- [ ] 2.4 Add test `TestLoad_NoComponents` — assert `mod.Components` is nil when `#components` is absent in the module

## 3. Validation

- [ ] 3.1 Run `task fmt` and fix any formatting issues
- [ ] 3.2 Run `task test` and confirm all tests pass
