## Why

The `experiments/factory/` prototype validated a fundamentally simpler rendering architecture: CUE-native matching via `#MatchPlan` (eliminating Go-side component inspection), unified loading (eliminating the separate builder phase), `cue.Value`-based resources (eliminating eager K8s conversion), CUE-native auto-secrets via `#AutoSecrets` (eliminating Go-side injection), and native bundle support. The current CLI's `internal/builder/`, `internal/pipeline/`, `internal/loader/`, and `internal/core/{component,transformer,provider}` carry complexity that the factory proved unnecessary. This change promotes the factory engine to production, replaces the existing rendering system, moves shared types to `pkg/` for reusability, and fixes all tracked technical debt from `experiments/factory/DEBT.md`.

**SemVer**: MAJOR — internal package structure changes, `Pipeline` interface removed, `Resource` type changes from `*unstructured.Unstructured` to `cue.Value`.

## What Changes

- **BREAKING**: Replace `internal/pipeline/` with `pkg/engine/` — concrete `ModuleRenderer` and `BundleRenderer` structs; no `Pipeline` interface
- **BREAKING**: Replace `internal/builder/` — loading IS building; validation gates merge into `pkg/loader/`
- **BREAKING**: Replace `internal/loader/` with `pkg/loader/` — factory loading system with Module Gate and Bundle Gate validation, value finalization via `Syntax(cue.Final()) + BuildExpr`, release kind detection, and bundle release extraction
- **BREAKING**: Replace `core.Resource` — wraps `cue.Value` instead of `*unstructured.Unstructured`; gains conversion methods (`ToUnstructured()`, `MarshalYAML()`, `MarshalJSON()`, `ToMap()`) and accessor methods (`Kind()`, `Name()`, `Namespace()`, `GVK()`, `Labels()`, `Annotations()`, `APIVersion()`)
- **BREAKING**: Remove `internal/core/component/` — `Component` as a Go type is eliminated; CUE-native `#MatchPlan` replaces Go-side component field inspection for matching
- **BREAKING**: Remove `internal/core/transformer/` — types and execution logic absorbed into `pkg/engine/`
- **BREAKING**: Remove `internal/core/provider/` `Match()` method — CUE-native `#MatchPlan` in `v1alpha1/core/matcher/` replaces Go-side matching entirely
- Move all core types to `pkg/`: `module`, `modulerelease`, `provider`, `core` (resource, labels, weights), `errors`
- Add `pkg/bundle/` and `pkg/bundlerelease/` types from factory
- Remove `internal/builder/autosecrets.go` — superseded by CUE-layer `#AutoSecrets` discovery pipeline and `#OpmSecretsComponent` helper in `v1alpha1/`
- Merge factory validation gates (`validateConfig`) with CLI error parsing for structured `FieldError`/`ConflictError` output with file/line/path attribution
- Fix all 15 DEBT.md items during promotion: silent metadata decode errors, non-deterministic warnings, dead code, fail-fast/fail-slow asymmetry, fragile `isSingleResource` heuristic, dual Schema/DataComponents safety, and more
- Adapt all 23 downstream consumer files across `internal/cmd/mod/`, `internal/cmdutil/`, `internal/inventory/`, `internal/kubernetes/`, `internal/output/`
- Establish clean CUE boundary: nothing below `internal/cmdutil/` imports CUE packages; conversion from `cue.Value` to K8s-native types happens at the cmdutil layer

## Capabilities

### New Capabilities

- `engine-rendering`: CUE-native rendering engine with `ModuleRenderer` and `BundleRenderer`, two-phase render (CUE `#MatchPlan` match + `FillPath`/`injectContext` execute), structured `UnmatchedComponentsError` with per-transformer diagnostics
- `resource-conversion`: `Resource` type with `cue.Value` core and lazy conversion methods for YAML, JSON, `*unstructured.Unstructured`, and `map[string]any`; accessor methods for Kind, Name, Namespace, GVK, Labels, Annotations, APIVersion
- `validation-gates`: Module Gate and Bundle Gate system — validates consumer values against `#config` schema at load time; returns structured `FieldError` with file/line/path and `ConflictError` for cross-file conflicts; replaces the separate builder validation phase
- `pkg-types`: Exported type packages at `pkg/` level — `core`, `module`, `modulerelease`, `bundle`, `bundlerelease`, `provider`, `errors` — importable by external tools

### Modified Capabilities

- `render-pipeline`: 5-phase pipeline replaced by direct engine calls; `Pipeline` interface removed; `RenderResult` restructured to use new types
- `build`: Separate builder phase eliminated; values validation, FillPath chain, and component extraction all happen inside loader
- `loader-api`: New loader handles module releases, bundle releases, provider loading, value finalization (`Syntax(cue.Final()) + BuildExpr`), and validation gates; `LoadModule` replaced by `LoadReleasePackage` + `LoadModuleReleaseFromValue`
- `core`: `Resource` type changes to `cue.Value`-based; label constants and GVK weights move to `pkg/core/`
- `core-module`: Moves to `pkg/module/`; dead `pkgName` field removed
- `core-modulerelease`: Moves to `pkg/modulerelease/`; dual `Schema`/`DataComponents` replaced with typed accessors `MatchComponents()`/`ExecuteComponents()`; value/pointer embedding inconsistency fixed
- `core-provider`: Moves to `pkg/provider/`; Go-side `Match()` removed; becomes thin CUE value wrapper
- `core-transformer`: Eliminated as standalone package; execution logic moves to `pkg/engine/execute.go`
- `core-component`: Eliminated entirely — CUE-native matching needs no Go-side component struct
- `core-component-extraction`: Eliminated — `ExtractComponents()` no longer needed
- `component-matching`: Go-side label/resource/trait matching replaced by CUE-native `#MatchPlan`
- `provider-match`: Go-side `Provider.Match()` replaced by CUE `#MatchPlan` evaluation
- `transformer-match-plan-execute`: Go-side `TransformerMatchPlan.Execute()` replaced by `pkg/engine/execute.go`
- `errors-domain`: Moves to `pkg/errors/`; gains `ConfigError` and gate-related error types from factory
- `auto-secrets-injection`: Go-side `autosecrets.go` removed; CUE-layer `#AutoSecrets` + `#OpmSecretsComponent` in `v1alpha1/` handles discovery, grouping, and component injection declaratively
- `release-building`: `builder.Build()` absorbed into new loader pipeline
- `cmdutil`: `RenderRelease()` adapted to call engine directly; handles `cue.Value` to `Unstructured` conversion at this layer
- `module-loading`: `LoadModule()` absorbed into new release-centric loader

## Impact

**Packages removed** (10): `internal/builder/`, `internal/pipeline/`, `internal/loader/`, `internal/core/component/`, `internal/core/transformer/`, `internal/core/provider/`, `internal/core/module/`, `internal/core/modulerelease/`, `internal/core/resource.go`, `internal/errors/`

**Packages created** (8): `pkg/core/`, `pkg/module/`, `pkg/modulerelease/`, `pkg/bundle/`, `pkg/bundlerelease/`, `pkg/provider/`, `pkg/loader/`, `pkg/engine/`

**Packages adapted** (5): `internal/cmd/mod/`, `internal/cmdutil/`, `internal/inventory/`, `internal/kubernetes/`, `internal/output/`

**Files affected**: ~23 production files, ~5 test files with struct literals requiring full rewrite

**Dependencies**: No new external dependencies. CUE SDK (`cuelang.org/go`) and K8s libraries (`k8s.io/apimachinery`) already in `go.mod`.

**CUE boundary**: After migration, nothing below `internal/cmdutil/` imports CUE packages. `internal/kubernetes/` and `internal/inventory/` receive only `*unstructured.Unstructured` and primitive Go types.
