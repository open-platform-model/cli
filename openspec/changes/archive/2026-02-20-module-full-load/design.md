## Context

`module.Load()` uses `load.Instances()` for AST inspection but stops short of CUE evaluation, leaving `Module.Components`, `Module.Config`, `Module.Values`, and `Module.Metadata.UUID` all zero-valued. `release.Builder.Build()` then calls `load.Instances()` a second time, generates a CUE AST overlay to compute a release UUID via the CUE `uuid` package, and reads `#opmReleaseMeta` to extract identity. This overlay approach is fragile (programmatic AST construction), redundant (double module load), and incorrect (the CLI's `opmNamespaceUUID` constant does not match `OPMNamespace` in `catalog/v0/core/common.cue`, producing wrong UUIDs). Separately, `build/component.Component` is byte-for-byte identical to `core.Component` but lives in a parallel package, creating duplication across 10 import sites.

## Goals / Non-Goals

**Goals:**

- `module.Load()` performs full CUE evaluation and returns a `*core.Module` with `Components` (schema-level), `Config`, `Values`, and an internal base `cue.Value` populated
- `core.Component` is extended to structurally mirror the `#Component` CUE schema (adds `ApiVersion`, `Kind`, `Metadata *ComponentMetadata`, `Blueprints`, `Spec`, `Validate()`)
- `build/component` package is deleted; all consumers use `core.Component`
- `release.Builder.Build()` accepts `*core.Module` and uses the already-evaluated base `cue.Value` — no second `load.Instances()` call
- CUE overlay (`overlay.go`, `generateOverlayAST()`) is deleted; release UUID is computed in Go using the correct `OPMNamespace`
- `OPMNamespace` is corrected to `11bc6112-a6e8-4021-bec9-b3ad246f9466` (matching `catalog/v0/core/common.cue`)
- Module metadata (`UUID`, `Labels`, `FQN`, `Version`) is read from the CUE evaluation result rather than computed separately

**Non-Goals:**

- Introducing `core.ModuleRelease` or changing the `BuiltRelease` output type (separate change)
- Changes to provider loading, transformer matching, or executor phases
- Namespace resolution precedence changes (handled in `namespace-resolution-precedence` change)
- Parallel/goroutine-safe CUE evaluation — all pipeline phases run single-threaded on one `*cue.Context`

## Decisions

### Decision 1: module.Load() does CUE eval using the instance already loaded for AST inspection

`inspectModule()` currently calls `load.Instances()` and reads `inst.Files` for AST walking. After this change, `Load()` passes `inst` to `cueCtx.BuildInstance(inst)` to get the base `cue.Value`, then calls `cueCtx.BuildInstance` only once per load. The base value is used to extract `#config`, `values`, `#components`, and module metadata fields.

**Why**: `load.Instances()` is already paid for AST inspection. Reusing the same instance for `BuildInstance` avoids any additional disk I/O. Separating the two calls (one for AST, one for build) would require loading twice.

**Alternative considered**: Call `load.Instances()` a second time in `Load()` with a clean config for CUE eval. Rejected — redundant, same cost as the existing double-load in the pipeline.

### Decision 2: Module.value stored as unexported field with package-level accessor

`core.Module` gains an unexported `value cue.Value` field set by `module.Load()`. An exported accessor `CUEValue() cue.Value` is added, consistent with the existing `PkgName() string` / `SetPkgName()` pattern. `release.Builder.Build()` calls `mod.CUEValue()` to obtain the base value for `FillPath` and `Unify` operations.

**Why**: The base `cue.Value` is a load-time implementation detail. Exporting it as a struct field would encourage callers to depend on the raw CUE graph; an accessor communicates that it is set by `Load()` and read by `Build()`, not for general use. Unexported field + accessor is the established pattern in this codebase (`pkgName`).

**Alternative considered**: Return `(mod *core.Module, baseValue cue.Value, error)` from `module.Load()`, threading `baseValue` separately into `Build()`. Rejected — callers would need to track two values instead of one, recreating the same manual state management the module type is meant to eliminate.

### Decision 3: core.Component is extended to mirror #Component CUE schema

`core.Component` gains `ApiVersion string`, `Kind string`, `Metadata *ComponentMetadata`, `Blueprints map[string]cue.Value`, and `Spec cue.Value`. The existing flat `Name`, `Labels`, `Annotations` fields move into `ComponentMetadata`. The existing `Value cue.Value` field is retained as the full component CUE value.

```go
type Component struct {
    ApiVersion string               `json:"apiVersion"`
    Kind       string               `json:"kind"`
    Metadata   *ComponentMetadata   `json:"metadata"`
    Resources  map[string]cue.Value `json:"#resources"`
    Traits     map[string]cue.Value `json:"#traits"`
    Blueprints map[string]cue.Value `json:"#blueprints"`
    Spec       cue.Value            `json:"spec"`
    Value      cue.Value            `json:"value"`
}

type ComponentMetadata struct {
    Name        string            `json:"name"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
}
```

`Spec` captures the merged component spec (`spec: close({ _allFields })` in CUE) — the user-facing fields from all attached resources, traits, and blueprints. Before user values it is the schema; after `FillPath` it is concrete.

**Why**: `core.Component` already imports `cuelang.org/go/cue` for `cue.Value` fields. Extending it to match `#Component` makes the Go type self-describing and eliminates the need to remember which CUE fields correspond to which Go fields. `Spec` explicitly represents what the executor consumes; having it extracted avoids repeated `LookupPath("spec")` calls throughout the pipeline.

**Alternative considered**: Keep flat `Name`/`Labels`/`Annotations` on `Component`, add new fields alongside. Rejected — flattening is inconsistent with every other domain type in `core` (`Module`, `Provider`, `Transformer` all use `Metadata` sub-types) and inconsistent with the CUE schema.

### Decision 4: Component.Validate() is a structural receiver method

`Validate() error` on `*Component` checks: `Metadata != nil`, `Metadata.Name != ""`, `len(Resources) > 0`, `Value.Exists()`. It does NOT check CUE concreteness — that is a separate concern (`IsConcrete() bool` checks `Value.Validate(cue.Concrete(true)) == nil`).

**Why**: Structural validation at extraction time (both in `Load()` and `Build()`) guards against malformed components reaching the matcher or executor. Keeping structural and concreteness checks separate lets `Load()` call `Validate()` without requiring concrete values (schema-level components are valid but not concrete). `Build()` calls both `Validate()` and `IsConcrete()`.

### Decision 5: Component extraction lives in internal/core/ as a package-level function

`core.ExtractComponents(v cue.Value) (map[string]*Component, error)` is a package-level function in `internal/core/`. Both `module.Load()` (schema-level extraction from `#components`) and `release.Builder.Build()` (concrete extraction from the filled component value) call it. The existing `extractComponentsFromDefinition` and `extractComponent` in `release/metadata.go` are replaced by this function and deleted.

**Why**: `core` already imports `cuelang.org/go/cue` for `cue.Value` fields; adding extraction logic introduces no new dependency. Both `build/module` and `build/release` import `core` — putting extraction in `core` gives both packages access without creating cross-`build/` sub-package dependencies. The extraction logic is pure `cue.Value` → `*Component` transformation with no pipeline-specific concerns.

**Alternative considered**: Keep extraction in `build/release/metadata.go` and have `build/module` import `build/release`. Rejected — `build/module` is a loader; importing `build/release` would create a confusing dependency direction (loaders should not depend on builders).

**Alternative considered**: Repurpose `build/component/` as an extraction-functions package. Rejected — the package exists solely as a workaround for the type duplication. With the type consolidated into `core`, the package has no identity and should be deleted entirely.

### Decision 6: build/component/ package is deleted entirely

`internal/build/component/types.go` is the only file in the package. With `core.Component` as the canonical type, the package has no remaining purpose. All 10 import sites are updated to use `core.Component`.

**Why**: Dead packages invite confusion about which type to use. A clean deletion with updated import sites is unambiguous.

### Decision 7: release.Builder.Build() takes *core.Module, drops load.Instances()

`Build(modulePath string, opts Options, valuesFiles []string)` becomes `Build(mod *core.Module, opts Options, valuesFiles []string)`. Inside `Build()`:

- `mod.CUEValue()` provides the base `cue.Value` (already built by `Load()`)
- User values files are loaded via `b.loadValuesFile()` and unified with the base value
- Values are validated against `mod.Config` (already extracted by `Load()`)
- `FillPath("#config", values)` produces the concrete release value
- `core.ExtractComponents(concreteValue.LookupPath("#components"))` extracts concrete components

`load.Instances()`, `generateOverlayAST()`, `formatNode()`, `detectPackageName()` are all deleted from `Build()`. `Options.PkgName` field is removed (the package name is no longer needed post-overlay-removal).

**Values selection**: when `--values` is provided, those values are used directly and the module's `values` field is ignored. When no `--values` is provided, the module's `values` field (stored in `mod.Values` by `Load()`) serves as the config input. In both cases the selected values are passed whole to `FillPath("#config", selectedValues)` — there is no merging of user values with module-level values.

**Why**: The overlay's only purpose was to inject `#opmReleaseMeta` (UUID + labels) into the CUE namespace so CUE could compute identity. With UUID computation moved to Go and module metadata read from the evaluation result, the overlay has no remaining function. Eliminating it removes the most complex and fragile part of the build phase.

### Decision 8: OPMNamespace is corrected and release UUID is computed in Go

`const OPMNamespace = "11bc6112-a6e8-4021-bec9-b3ad246f9466"` is added to `internal/core/labels.go`. A helper `ComputeReleaseUUID(fqn, name, namespace string) string` uses `uuid.NewSHA1(uuid.MustParse(OPMNamespace), []byte(fqn+":"+name+":"+namespace))`, matching the CUE formula `uid.SHA1(OPMNamespace, "\(fqn):\(name):\(namespace)")` in `catalog/v0/core/module_release.cue`.

**Why**: UUID v5 (SHA1-based) is deterministic given the same inputs and namespace. Moving computation to Go with the correct namespace produces identical values to what CUE would compute, while eliminating the overlay. The old constant (`c1cbe76d-...`) was derived from `uuid.NewSHA1(uuid.NameSpaceDNS, "opmodel.dev")` — different from the catalog's fixed value, silently producing wrong UUIDs.

**Migration**: All previously stored release UUIDs (in Kubernetes inventory secrets) are invalidated. This is acceptable: the project is pre-production and stored inventory is test data only.

### Decision 9: Module metadata is read from CUE evaluation, not recomputed in Go

After `BuildInstance()`, `Load()` reads `metadata.uuid`, `metadata.fqn`, `metadata.version`, and `metadata.labels` directly from the evaluated `cue.Value`. `Module.Metadata.UUID` is therefore whatever CUE computes via `uid.SHA1(OPMNamespace, "\(fqn):\(version)")` — no Go-side computation needed for module identity.

`Load()` also reads the module's `values` field (if present) and stores it in `Module.Values` as a `cue.Value`. This field holds the module author's suggested config inputs — a plain struct like `values: { image: "nginx:1.28.2", replicas: 1 }`. It is used by `Build()` as the fallback config input when no `--values` flag is provided, and may be displayed by `opm mod status`. It plays no role in the build flow when user values are explicitly supplied.

**Why**: Duplicating the computation formula in Go would create two sources of truth. Reading from the CUE value ensures the CLI always agrees with the schema, including when the formula changes in the catalog. The CUE evaluation already does the work; Go just reads the result.

### Decision 10: extractReleaseMetadata() is rewritten to use Go values, not #opmReleaseMeta

`extractReleaseMetadata()` in `release/metadata.go` is rewritten to construct `core.ReleaseMetadata` from `opts.Name`, `opts.Namespace`, `mod.Metadata.FQN`, `mod.Metadata.Version`, `core.ComputeReleaseUUID(...)`, and computed labels — without reading from any `#opmReleaseMeta` CUE path. The `#opmReleaseMeta` path no longer exists in the evaluated value (overlay deleted).

**Why**: `#opmReleaseMeta` was an overlay-injected synthetic CUE definition. With the overlay removed, the metadata fields are available directly from `mod.Metadata` and `opts`. Reading from Go values is simpler and type-safe.

## Experiment Validation

All runtime-critical design claims were validated in `experiments/module-full-load/` before implementation. The experiment is a fully detached package — it imports only `cuelang.org/go/cue`, `cuelang.org/go/cue/load`, and `github.com/google/uuid`, with no references to `internal/`.

> **Simplifications in the experiment**: The test module (`experiments/module-full-load/testdata/test_module/`) uses a `defaultValues` field and adds CUE `*` defaults to all `#config` fields (e.g., `image: string | *"nginx:latest"`). Neither of these exists in production modules. The experiment's purpose is to validate CUE mechanics (BuildInstance, FillPath, LookupPath, UUID formula), not to replicate the exact production module shape. Findings that depend on `#config` having defaults (e.g., schema-level components being concrete) are experiment artefacts and do not affect the production design.

| Decision | Test file | Key tests |
|---|---|---|
| 1 — single load, AST + BuildInstance from same inst | `single_load_test.go` | `TestSingleLoad_ASTAndBuildFromSameInst`, `TestSingleLoad_InstIsReusable`, `TestSingleLoad_ASTFilesPopulatedBeforeBuild` |
| 4+5 — schema-level component extraction, Validate/IsConcrete | `schema_components_test.go` | `TestSchemaExtract_ComponentsExist`, `TestSchemaExtract_IterateComponents`, `TestSchemaExtract_ResourcesIterable`, `TestSchemaExtract_TraitsOptional`, `TestSchemaExtract_Validate_Structural` |
| 7 — FillPath replaces overlay; base value immutable and reusable | `build_without_overlay_test.go`, `user_values_test.go` | `TestBuildNoOverlay_FillConfigProducesConcrete`, `TestBuildNoOverlay_BaseValueUnchanged`, `TestBuildNoOverlay_MultipleReleases`, `TestBuildNoOverlay_ValuesValidation`, `TestUserValues_ApplyToModule`, `TestUserValues_OverridesSchemaDefaults` |
| 8 — OPMNamespace corrected; UUID formula deterministic | `uuid_parity_test.go` | `TestUUID_OldNamespaceDiffersFromNew`, `TestUUID_ReleaseUUIDDeterministic`, `TestUUID_OldFormulaProducesDifferentUUID`, `TestUUID_ModuleAndReleaseUUIDDontCollide` |
| 9 — metadata readable from evaluated value | `metadata_from_eval_test.go` | `TestMetadata_NameReadable`, `TestMetadata_UUIDReadable`, `TestMetadata_FQNReadable`, `TestMetadata_LabelsIterable`, `TestMetadata_ConfigExtractable` |
| User values file flow (external values, validation against #config) | `user_values_test.go` | `TestUserValues_LoadExternalFile`, `TestUserValues_ValidateAgainstConfig`, `TestUserValues_InvalidField_Rejected` |

## Risks / Trade-offs

**OPMNamespace correction breaks all existing UUIDs** — any release inventory stored in Kubernetes (secrets keyed by release UUID) becomes stale. Mitigation: pre-production codebase; test clusters can be reset. Documented as intentional breaking change in proposal.

**Module.value couples Load() and Build() via an internal field** — `Build()` now depends on `module.Load()` having been called (otherwise `mod.CUEValue()` returns a zero `cue.Value`). Mitigation: `Build()` checks `mod.CUEValue().Exists()` and returns a clear error if zero. `Load()` is the only constructor for `*core.Module` in the pipeline.

**CUE eval in Load() may fail for modules with non-concrete metadata** — if `metadata.fqn` or `metadata.uuid` cannot be evaluated (e.g., computed from a missing import), `BuildInstance()` will return errors. Mitigation: errors from `BuildInstance()` are surfaced immediately with the CUE error details, same as existing behavior in `Build()`.

**core.ExtractComponents on schema-level values may return non-concrete Resources/Traits** — `cue.Value` fields in schema-level components are constraint expressions, not concrete values. Mitigation: this is intentional and documented; schema-level components are for matching (labels, FQN presence) only. Concrete components are extracted by `Build()` after `FillPath`. `Component.IsConcrete()` distinguishes the two.

**10 import sites require mechanical update** — changing `component.Component` to `core.Component` across 10 files is low-risk but high-surface-area. Mitigation: the type fields are identical (pre-extension); only the package qualifier changes. Test suite will catch any missed sites.

## Migration Plan

1. Add `OPMNamespace` constant and `ComputeReleaseUUID()` to `internal/core/labels.go`
2. Extend `core.Component` with new fields; add `ComponentMetadata`; add `Validate()` and `IsConcrete()` methods
3. Add `core.ExtractComponents()` package-level function to `internal/core/`
4. Update all 10 `build/component.Component` import sites to `core.Component`; delete `internal/build/component/`
5. Add `value cue.Value` unexported field + `CUEValue()` / `setCUEValue()` accessors to `core.Module`
6. Extend `module.Load()` / `inspectModule()`: call `cueCtx.BuildInstance(inst)`, extract `#config` → `Module.Config`, `values` field → `Module.Values`, `#components` → `Module.Components` via `core.ExtractComponents()`, and `metadata.*` → `Module.Metadata`; call `comp.Validate()` per extracted component
7. Change `release.Builder.Build()` signature to `Build(mod *core.Module, ...)`: remove `load.Instances()`, `generateOverlayAST()`, `detectPackageName()`; rewrite `extractReleaseMetadata()` from Go values; call `core.ExtractComponents()` on concrete value
8. Delete `internal/build/release/overlay.go`; remove `Options.PkgName`
9. Update `pipeline.Render()` to pass `mod` to `Build(mod, ...)`; remove `opts.PkgName` propagation
10. Run `task test` — verify no behavior change for modules with valid metadata

Rollback: Steps 1–10 are reversible by reverting the affected files. The UUID namespace change cannot be rolled back transparently for any stored inventory — full cluster reset required.

## Open Questions

~~Should `core.ExtractComponents()` be a method on `cue.Value` via a wrapper, or a free-standing function?~~ **Resolved**: free-standing function. The experiment confirms this is natural at every call site — both `Load()` and `Build()` call it with a `cue.Value` obtained from `LookupPath`, not from a receiver. See `experiments/module-full-load/schema_components_test.go:extractComponents`.

~~Should `Build()` call `mod.Validate()` as a precondition?~~ **Resolved**: yes. `Build()` checks `mod.CUEValue().Exists()` and returns a clear error if zero (indicating `Load()` was not called or failed). This is captured in Migration Plan step 7.
