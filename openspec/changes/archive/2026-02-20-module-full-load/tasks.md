## 1. internal/core — UUID constants and helpers

- [x] 1.1 Add `OPMNamespace = "11bc6112-a6e8-4021-bec9-b3ad246f9466"` constant to `internal/core/labels.go`, replacing the old `opmNamespaceUUID` constant
- [x] 1.2 Add `ComputeReleaseUUID(fqn, name, namespace string) string` function to `internal/core/labels.go` using `uuid.NewSHA1(uuid.MustParse(OPMNamespace), []byte(fqn+":"+name+":"+namespace))`
- [x] 1.3 Write unit tests for `ComputeReleaseUUID`: same inputs → same UUID, different name/namespace → different UUIDs, result is UUID v5

## 2. internal/core — Component type extension

- [x] 2.1 Introduce `ComponentMetadata` struct (`Name string`, `Labels map[string]string`, `Annotations map[string]string`) in `internal/core/`
- [x] 2.2 Extend `core.Component` with `ApiVersion string`, `Kind string`, `Metadata *ComponentMetadata`, `Blueprints map[string]cue.Value`, `Spec cue.Value`; move existing flat `Name`/`Labels`/`Annotations` fields into `ComponentMetadata`
- [x] 2.3 Add `Validate() error` receiver on `*core.Component`: checks `Metadata != nil`, `Metadata.Name != ""`, `len(Resources) > 0`, `Value.Exists()`
- [x] 2.4 Add `IsConcrete() bool` receiver on `*core.Component`: returns `Value.Validate(cue.Concrete(true)) == nil`
- [x] 2.5 Add `core.ExtractComponents(v cue.Value) (map[string]*Component, error)` package-level function; iterate hidden fields, call `comp.Validate()` per component
- [x] 2.6 Write unit tests for `Validate()`, `IsConcrete()`, and `ExtractComponents()` using inline CUE or fixture values

## 3. internal/build/component — deletion

- [x] 3.1 Update all 10 `build/component.Component` import sites to use `core.Component` (search for `build/component` imports across `internal/`)
- [x] 3.2 Delete `internal/build/component/` package directory

## 4. internal/core — Module CUE value accessor

- [x] 4.1 Add unexported `value cue.Value` field to `core.Module` struct
- [x] 4.2 Add `CUEValue() cue.Value` exported accessor
- [x] 4.3 Add unexported `setCUEValue(v cue.Value)` setter used by `module.Load()`
- [x] 4.4 Update `core.Module.Validate()` to also check `Metadata.FQN != ""` and `mod.CUEValue().Exists()`

## 5. internal/build/module — full CUE evaluation in Load()

- [x] 5.1 After AST inspection in `Load()` / `inspectModule()`, call `cueCtx.BuildInstance(inst)` on the same instance; surface evaluation errors immediately
- [x] 5.2 Extract `#config` → `mod.Config` and `values` field → `mod.Values` from the evaluated base value (zero value if absent, no error)
- [x] 5.3 Extract `metadata.fqn`, `metadata.uuid`, `metadata.version`, `metadata.labels` from the evaluated base value → `mod.Metadata`
- [x] 5.4 Call `core.ExtractComponents()` on `#components` value → `mod.Components`; call `mod.SetCUEValue()` with the base value
- [x] 5.5 Update or add unit tests for the extended `Load()` using test fixture modules in `internal/build/` testdata

## 6. internal/build/release — remove overlay, rewrite Build()

- [x] 6.1 Delete `internal/build/release/overlay.go` (removes `generateOverlayAST()`, `formatNode()`)
- [x] 6.2 Remove `PkgName` field from `release.Options`; remove `detectPackageName()` from `builder.go`
- [x] 6.3 Change `Build()` signature from `Build(modulePath string, opts Options, valuesFiles []string)` to `Build(mod *core.Module, opts Options, valuesFiles []string)`
- [x] 6.4 Add precondition guard at the top of `Build()`: return error if `mod.CUEValue().Exists()` is false
- [x] 6.5 Remove `load.Instances()` and `cueCtx.BuildInstance()` calls from `Build()`; obtain base value via `mod.CUEValue()`
- [x] 6.6 Implement values selection in `Build()`: use `--values` files if provided, otherwise fall back to `mod.Values`; validate selected values against `mod.Config` via `Unify` before `FillPath`
- [x] 6.7 Apply selected values via `base.FillPath(cue.MakePath(cue.Def("config")), selectedValues)`; call `core.ExtractComponents()` on the concrete `#components` value; gate on `IsConcrete()` per component
- [x] 6.8 Rewrite `extractReleaseMetadata()` to construct `core.ReleaseMetadata` from `mod.Metadata`, `opts`, and `core.ComputeReleaseUUID(mod.Metadata.FQN, opts.Name, opts.Namespace)` — no `#opmReleaseMeta` path
- [x] 6.9 Update or add unit tests for `Build()` with the new signature and values-selection logic

## 7. internal/build/pipeline — wire updated signatures

- [x] 7.1 Update `pipeline.Render()` to pass `mod *core.Module` to `Build(mod, ...)` instead of `modulePath string`
- [x] 7.2 Remove `opts.PkgName` propagation from pipeline; remove any remaining references to the deleted field

## 8. Validation

- [x] 8.1 Run `task test` — all tests pass
- [x] 8.2 Run `task check` — fmt, vet, and test all pass (pre-existing lint warnings not introduced by this change)
