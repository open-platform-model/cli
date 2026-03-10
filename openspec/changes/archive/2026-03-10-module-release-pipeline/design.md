## Context

The current release pipeline grew by accretion. Release-file loading in `pkg/loader`, synthesized module-release construction in `pkg/loader/module_as_release.go`, match-plan construction in `pkg/engine`, and command orchestration in `internal/cmdutil/render.go` all carry overlapping knowledge about values resolution, config validation, kind detection, component extraction, and rendering. The result is a pipeline with weak stage boundaries:

```text
current release flow

load value(s)
   │
   ├─ detect kind
   ├─ validate config
   ├─ fill values into release/module
   ├─ extract metadata
   ├─ finalize components
   ├─ build match plan in CUE
   └─ execute transforms in engine

ownership today
  pkg/loader           -> loading + validation + extraction
  pkg/engine           -> matching + execution
  internal/cmdutil     -> orchestration + file-resolution policy
```

This coupling makes several things harder than they should be:

- `BundleRelease` support is constrained by eager module-centric validation before the pipeline has cleanly split by release kind
- release-file parsing cannot be reused independently because the loader implicitly validates and concretizes
- matching logic is hidden behind CUE evaluation, so behavior is harder to test and reason about from Go
- the release structs themselves do not reflect the real stages the pipeline moves through (`raw`, `validated values`, `concrete components`, `finalized data components`)

The change introduces an explicit parse -> process -> execute model. Parsing is kept internal because release-file handling is a CLI concern. Processing and matching become explicit public package APIs because they represent reusable release-domain behavior rather than command glue.

## Goals / Non-Goals

**Goals:**

- Introduce an internal parse-only `GetReleaseFile` API that reads an absolute release file path and returns a barebones `ModuleRelease` or `BundleRelease` without validating values
- Redesign `modulerelease.ModuleRelease` and `bundlerelease.BundleRelease` so they expose the fields needed by later processing stages: `RawCUE`, `Config`, `Values`, and finalized component/release state
- Add public processing APIs that own value validation, concreteness, component extraction/finalization, Go-side matching, and rendering orchestration
- Reimplement the logic in `catalog/v1alpha1/core/matcher/matcher.cue` in Go while preserving exact matching semantics and deterministic diagnostics
- Refactor `pkg/engine` so it executes a precomputed match plan rather than building the match plan internally
- Keep the current CLI user-facing behavior intact while changing the internal architecture beneath it

**Non-Goals:**

- Fully implement bundle rendering in this change; `ProcessBundleRelease` only needs a gate-validating stub to establish the API shape
- Replace all remaining CUE evaluation in the render pipeline; transform execution still uses CUE `#transform`
- Change command UX, flag names, or output formats beyond what is needed to preserve the current behavior
- Introduce a general-purpose release parsing API for external consumers; parsing remains internal because release-file handling is a CLI concern

## Decisions

### 1. Split the pipeline into parse, process, and execute stages

The new architecture adopts three explicit stages:

```text
parse (internal)
  GetReleaseFile(abs release.cue)
        │
        ▼
barebones release
        │
process (public)
  ValidateConfig
  ProcessModuleRelease / ProcessBundleRelease
        │
        ▼
concrete release + match plan
        │
execute (public engine)
  ModuleRenderer.Render(..., matchPlan)
```

This matches the actual responsibilities better than the current package split:

- parsing decides what kind of release file exists and extracts raw values plus concrete release metadata
- processing owns value handling and release concretization
- engine owns transform execution and resource collection

Alternative considered: keep `pkg/loader` as the owner of the full flow and add more helper methods there. Rejected because it preserves the current ambiguity: `loader` would continue to parse, validate, concretize, and partially render rather than having a single concern.

### 2. Keep release-file parsing internal and allow partial extraction

`GetReleaseFile` is intentionally internal because release-file handling is a CLI concern. It accepts an already-resolved absolute file path and performs only parse-time responsibilities:

- load the file into CUE
- determine whether it contains a `ModuleRelease` or `BundleRelease`
- construct a barebones Go release object
- extract concrete release metadata, `RawCUE`, and `Config`
- tolerate unresolved `#module` / `#bundle` references only when release metadata is already concrete

The permissive behavior is deliberate, but only for the module or bundle payload. Release metadata itself must already be concrete at parse time so `GetReleaseFile` can decode the authoritative Go metadata once and reuse it through later stages. Current release-file flows may still allow later `--module` injection so long as release metadata does not depend on unresolved `#module` fields.

Alternative considered: require `#module` or `#bundle` to be concrete during parsing. Rejected because it collapses parsing back into validation and would force a second special-case pre-processing path for local-module injection.

### 3. Redesign release structs around pipeline stages

The release structs are changed to represent the lifecycle of processing rather than only the final extracted state.

`ModuleRelease` will carry:

- `Metadata`
- `Module`
- `RawCUE cue.Value`
- `DataComponents cue.Value`
- `Config cue.Value`
- `Values cue.Value`

`BundleRelease` will carry:

- `Metadata`
- `Bundle`
- `RawCUE cue.Value`
- `Releases map[string]*modulerelease.ModuleRelease`
- `Config cue.Value`
- `Values cue.Value`

This shape lets later stages attach validated values and finalized components without losing the original raw release expression.

Alternative considered: keep the old hidden `schema` and `dataComponents` fields and expose more receiver methods. Rejected because it hides too much of the pipeline state and keeps the important processing stages implicit.

### 4. Matching uses concrete non-finalized components; execution uses finalized data components

This is the most important technical distinction in the design.

```text
concrete RawCUE
  └─ components
      ├─ preserves #resources / #traits / #blueprints
      └─ required for matching

finalized DataComponents
  └─ plain data-only component map
      └─ required for FillPath into #transform
```

The Go matcher must inspect `#resources` and `#traits`, which are lost when components are finalized into data-only values. Therefore `ProcessModuleRelease` must derive two views from the same concrete release:

1. `schemaComponents := concreteRaw.LookupPath("components")` for `match.Match`
2. `DataComponents := finalize(schemaComponents)` for engine execution

Alternative considered: have matching run on `DataComponents`. Rejected because finalization removes the definition fields that the matching rules depend on.

### 5. Move matching out of CUE and into a dedicated public Go package

`pkg/match` will own the match-plan types and the `Match()` function. Its behavior will mirror `catalog/v1alpha1/core/matcher/matcher.cue` exactly:

- for every component and transformer pair, compute missing labels/resources/traits
- mark a pair as matched only when all three missing lists are empty
- compute `Unmatched` for components with zero matching transformers
- compute `UnhandledTraits` from component traits not covered by any matched transformer's `requiredTraits` or `optionalTraits`
- provide deterministic `MatchedPairs()`, `NonMatchedPairs()`, and `Warnings()` helpers

This makes the behavior explicit and testable in Go while preserving the current semantics.

Alternative considered: keep CUE `#MatchPlan` as the implementation and just wrap it behind a new Go API. Rejected because the user goal is to simplify the codebase by moving matching logic into Go, and a wrapper would preserve the current indirection without reducing conceptual weight.

### 6. Make `ValidateConfig` return merged values plus structured validation errors

The processing layer needs both the validation result and the merged values. The API therefore returns the unified concrete values and a structured config error:

```text
ValidateConfig(schema, []values)
   ├─ unify values together first
   ├─ fail if value files conflict
   ├─ validate merged values against schema
   └─ return merged value on success
```

The error stays structured so CLI display code can continue printing grouped diagnostics separately from the validation logic.

Alternative considered: return only `error` and force callers to re-unify values later. Rejected because it duplicates work and makes the processing pipeline more error-prone.

### 7. `pkg/engine` becomes execution-focused

`pkg/engine` will no longer construct the match plan. Instead, `ProcessModuleRelease` computes the plan and passes it into the renderer. The engine remains responsible for:

- iterating matched pairs
- filling `#component` with finalized component data
- filling `#context` from release/component metadata
- evaluating `#transform.output`
- decoding output resources
- collecting warnings and execution errors

This narrows engine responsibility to transform execution rather than pipeline orchestration.

Alternative considered: keep matching in `ModuleRenderer.Render()` and call the new Go matcher internally. Rejected because it would still make the engine own too many stages and weaken the clarity of the new parse/process/execute split.

### 8. Split module and bundle processing result types

`ProcessModuleRelease` and `ProcessBundleRelease` will not be forced into a single shared result type.

- `ProcessModuleRelease` returns the normal module render result
- `ProcessBundleRelease` returns a bundle-oriented result type later; for now it is a validating stub

This avoids a premature abstraction that would blur the difference between module-level and bundle-level rendering.

Alternative considered: create one common `RenderResult` shared by both processors. Rejected because bundle rendering naturally aggregates multiple module releases and has different result-shape needs.

## Risks / Trade-offs

- [Behavior drift between Go matcher and CUE matcher] -> Build parity tests directly from the current matcher semantics and reuse existing match-plan result helpers to keep diagnostics identical
- [Concrete metadata may reject some late-injection release files] -> Require release metadata to be concrete at parse time and keep unresolved `#module` / `#bundle` support only for payload fields that do not affect release metadata
- [Transition period with old and new loader/engine APIs may create duplication] -> Rewire command paths incrementally, then delete or reduce legacy entry points once the new flow is proven
- [Bundle processing scope could expand unexpectedly] -> Keep `ProcessBundleRelease` as a gate-validating stub and explicitly defer full bundle rendering behavior
- [Release struct changes may cascade into many tests] -> Update specs and tests in the same change so the new type shape becomes the single supported contract
- [Concreteness handling may move errors to a different stage] -> Preserve user-facing diagnostics by keeping structured validation errors and reusing existing pretty-printing paths in `internal/cmdutil`

## Migration Plan

1. Introduce the new release struct fields and internal parse-only `GetReleaseFile`
2. Add `pkg/releaseprocess.ValidateConfig` and new processing APIs
3. Implement `pkg/match.Match` with deterministic plan helpers and parity tests
4. Refactor `pkg/engine` so `ModuleRenderer.Render()` executes a supplied match plan
5. Rewire `internal/cmdutil/render.go` to use parse -> process -> execute for release files and synthesized releases
6. Leave `ProcessBundleRelease` as a validating stub so callers can adopt the API shape before full bundle rendering lands
7. Remove or thin-wrap legacy loader functions once all command entry points use the new pipeline

Rollback is straightforward during development because this is an internal architectural refactor: if the new processing flow proves incorrect, commands can continue using the existing `loader` + `engine` path until parity is restored.

## Open Questions

- Should `GetReleaseFile` use one wrapper type (`FileRelease`) or two dedicated internal functions behind one dispatcher? The wrapper type currently appears simplest, but both are viable.
- Should `ModuleRelease.RawCUE` be replaced in-place with the concrete filled value after processing, or should the original raw value be preserved separately for debugging? The current direction favors replacing it after successful processing to keep the active state simple.
- Should the old `pkg/loader.ValidateConfig` name be preserved as a forwarding shim for compatibility during migration, or should the codebase switch directly to `pkg/releaseprocess.ValidateConfig`?
