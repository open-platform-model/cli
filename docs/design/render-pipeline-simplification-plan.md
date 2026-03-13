# render pipeline simplification plan

## Goal

Reduce orchestration complexity around `release build` and `release apply` without changing user-visible behavior, render semantics, or Kubernetes functionality.

This plan focuses on making the current pipeline smaller, easier to read, and easier to extend by removing duplicate orchestration, tightening package ownership, and clarifying phase boundaries.

## Non-goals

- Do not change the matching algorithm.
- Do not change the transformer execution model.
- Do not change value precedence rules.
- Do not change inventory, pruning, or apply behavior.
- Do not finish bundle-release support as part of this simplification work.

## Current pain points

- The render workflow has two large entrypoints that converge late and duplicate the same tail logic.
- `ModuleRelease` currently acts as both parsed input and mutable phase state, which makes invariants harder to track.
- Release-file parsing constructs partially initialized render-layer types, which blurs package boundaries.
- Values resolution is spread across multiple branches.
- Several small helper layers only forward calls and add navigation overhead without adding policy.
- A few pure duplicates exist, such as apply summary formatting and resource conversion loops.

## Desired shape

The code should read as three clear stages:

1. Prepare input
2. Process/render
3. Emit/apply

The preparation stage should be mode-specific (`module` source vs `release.cue` source), while processing and emission should be shared.

---

## Improvement 1: Collapse the two render entrypoints

### What it is

`internal/workflow/render/render.go` currently has two main flows:

- `Release(...)` for module-source rendering
- `ReleaseFile(...)` for release-file rendering

They prepare inputs differently, but once they have a `*pkgrender.ModuleRelease` and values, they do the same work:

- apply namespace override
- load the provider
- call `pkg/render.ProcessModuleRelease(...)`
- convert resources to `*unstructured.Unstructured`
- assemble the workflow `Result`

### What would change

Extract one shared helper for the common tail, conceptually:

```go
renderPreparedModuleRelease(
    ctx context.Context,
    rel *pkgrender.ModuleRelease,
    values []cue.Value,
    provider *provider.Provider,
    namespaceOverride string,
) (*Result, error)
```

The two public entrypoints remain, but they become thin preparation adapters:

- `Release(...)` prepares from module source, then calls the common helper
- `ReleaseFile(...)` prepares from a release file, then calls the common helper

### Why

- Removes duplicated orchestration and error wrapping.
- Makes render behavior consistent across entry modes.
- Makes the file easier to follow because only preparation differs.
- Creates a natural place to plug in future bundle rendering once supported.

### Expected code impact

- `internal/workflow/render/render.go` becomes shorter and more obviously staged.
- Resource conversion and `Result` assembly exist in one place.
- Tests can target the shared tail separately from input preparation.

---

## Improvement 2: Introduce a single preparation phase for `ModuleRelease`

### What it is

Today the logic that answers “how do I get a renderable `ModuleRelease`?” is split across:

- `internal/workflow/render/render.go`
- `internal/workflow/render/values.go`
- `internal/releasefile/get_release_file.go`

### What would change

Create explicit preparation helpers with a common output shape:

```go
type PreparedModuleRelease struct {
    Release *pkgrender.ModuleRelease
    Values  []cue.Value
}
```

Then use two preparation functions:

- `PrepareFromModulePath(...)`
- `PrepareFromReleaseFile(...)`

Each helper owns all mode-specific setup:

- loading source CUE
- release-name override
- `debugValues` behavior
- values-file loading
- local `--module` injection when applicable
- module default namespace fallback where applicable

### Why

- Centralizes the most important orchestration boundary.
- Reduces the amount of state threaded across multiple files.
- Makes it easier to explain and test the render pipeline.
- Avoids having to remember which package handles which preparation detail.

### Expected code impact

- Fewer branches in the main render functions.
- More focused tests around module preparation vs release-file preparation.
- Easier reuse for `vet`, `build`, `diff`, and `apply`.

---

## Improvement 3: Reduce mutation and phase ambiguity in `ModuleRelease`

### What it is

`pkg/render/modulerelease.go` currently stores both:

- parsed input state
- processing state created later in the pipeline

The most confusing field is `RawCUE`, because it starts as parse-time release CUE and later becomes a values-filled concrete release value.

### What would change

Two possible levels of cleanup:

#### Option A: Low-risk field clarification

Keep the struct, but split semantic phases into clearer fields, for example:

- `ParsedCUE`
- `ConcreteCUE`

Keep `DataComponents` and `Values`, but make the phase transitions explicit.

#### Option B: Full phase split

Split runtime state into stage-specific types, for example:

- `ParsedModuleRelease`
- `ProcessedModuleRelease`

The processing function would accept the parsed type and return the processed type or final render result.

### Why

- Clarifies invariants at every phase.
- Reduces subtle mental overhead when reading the code.
- Makes future refactors safer because phase boundaries become explicit.
- Prevents accidental reuse of a field whose meaning changed earlier in the flow.

### Expected code impact

- Better naming in the render engine.
- Fewer comments needed to explain lifecycle semantics.
- Easier debugging of release preparation vs release processing.

### Recommendation

Start with Option A first. It gives most of the readability gain at lower risk.

---

## Improvement 4: Simplify the release-file layer

### What it is

`internal/releasefile/get_release_file.go` currently parses a release file and directly constructs a partially initialized `render.ModuleRelease` or `render.BundleRelease`.

That means a parsing package is already constructing runtime render-state objects.

### What would change

Replace the current “bare render object” output with a smaller parsed DTO, such as:

```go
type ParsedReleaseFile struct {
    Path        string
    Kind        Kind
    RawCUE      cue.Value
    Metadata    cue.Value
    ModuleCUE   cue.Value
    BundleCUE   cue.Value
    Config      cue.Value
}
```

The render preparation phase then translates that DTO into a proper `ModuleRelease` or `BundleRelease`.

### Why

- Keeps parsing responsibilities separate from runtime orchestration.
- Avoids passing partially initialized render objects across package boundaries.
- Makes release-file loading easier to test in isolation.
- Improves long-term flexibility if release-file semantics change.

### Expected code impact

- Cleaner separation between `internal/releasefile` and `pkg/render`.
- Fewer “best effort” init patterns in the parse layer.
- More explicit assembly of render state in one place.

---

## Improvement 5: Consolidate values resolution

### What it is

Values currently come from several sources depending on entry mode:

- explicit `--values`
- sibling `values.cue`
- inline `values`
- `debugValues` for module-source flows

The rules are spread across several functions.

### What would change

Move each mode's values logic behind one explicit helper and make precedence obvious in code. For example:

- `resolveModuleValues(...)`
- `resolveReleaseFileValues(...)`

Each helper should:

- return `[]cue.Value`
- own all relevant error messages
- document precedence in one place

### Why

- Values are central to the whole render path.
- Centralizing the policy prevents subtle behavior drift.
- It becomes much easier to answer “where do values come from in this mode?”

### Expected code impact

- Smaller render orchestration functions.
- Better unit tests for precedence and failure modes.
- More consistent user-facing errors.

---

## Improvement 6: Extract the repeated resource-conversion loop

### What it is

Both render flows convert `engineResult.Resources` into `[]*unstructured.Unstructured` using the same loop.

### What would change

Extract one helper, conceptually:

```go
func toUnstructuredResources(resources []*core.Resource) ([]*unstructured.Unstructured, error)
```

### Why

- Pure duplication.
- A small cleanup with immediate readability benefit.
- Keeps conversion-related error wrapping consistent.

### Expected code impact

- A few fewer repeated lines.
- Easier future changes if conversion behavior evolves.

---

## Improvement 7: Remove wrapper-only output forwarding layers

### What it is

`internal/workflow/render/output.go` currently forwards directly into `internal/cmdutil/manifest_output.go`.

### What would change

Choose one owner for manifest emission:

- either commands call `cmdutil` directly
- or workflow owns emission and `cmdutil` stops wrapping it

The current extra forwarding layer should be removed if it adds no policy.

### Why

- The wrapper does not currently simplify anything.
- It adds navigation hops when reading the code.
- Fewer thin layers make ownership clearer.

### Expected code impact

- Slightly flatter structure.
- Easier tracing from command to output implementation.

---

## Improvement 8: Unify build-command execution logic

### What it is

`internal/cmd/module/build.go` and `internal/cmd/release/build.go` follow the same pattern:

- parse output format
- resolve Kubernetes config
- call the render workflow
- show warnings/output
- write manifests

### What would change

Extract a shared build execution helper and keep only input-mode-specific flag wiring in each command file.

For example, the helper could accept:

- a callback that returns `(*render.Result, error)`
- output-format and split-output parameters

### Why

- Command files should focus on CLI surface, not repeated orchestration.
- It reduces drift between the two build paths.
- Shared behavior changes only need one code edit.

### Expected code impact

- Smaller command implementations.
- Better consistency between `module build` and `release build`.

---

## Improvement 9: Deduplicate apply summary formatting

### What it is

`FormatApplySummary` exists in more than one place.

### What would change

Keep a single implementation in the package that owns apply behavior and delete the duplicate.

### Why

- Duplicate helpers are maintenance noise.
- This is a trivial simplification with no behavior risk.

### Expected code impact

- Slightly smaller utility surface.
- No more drift risk for summary formatting.

---

## Improvement 10: Clarify ownership of manifest output code

### What it is

Manifest-output responsibilities are spread across:

- `internal/output/manifest.go`
- `internal/output/split.go`
- `internal/cmdutil/manifest_output.go`
- `internal/workflow/render/output.go`

### What would change

Pick one ownership model and align the packages to it.

#### Option A: `internal/output` owns all emission

- `internal/output` writes stdout and split-file output
- command/workflow code only prepares arguments

#### Option B: workflow owns emission policy

- workflow decides split/stdout behavior
- `internal/output` remains a lower-level serializer/formatter package

### Why

- The current code works, but ownership is fuzzy.
- Clear ownership reduces cognitive load when tracing output behavior.
- It simplifies future changes like alternate output targets.

### Expected code impact

- Fewer helper hops.
- Easier “how does YAML get written?” trace.

### Recommendation

Option A is simpler. Keep emission in `internal/output` and let other layers call it directly.

---

## Improvement 11: Load the provider earlier and pass it explicitly

### What it is

Provider selection and loading currently happens late in render orchestration.

### What would change

Resolve and load the provider once during preparation/orchestration, then pass a concrete `*provider.Provider` into the common render helper.

This changes the shape from:

- prepare release
- load provider later
- process release

to:

- prepare release
- prepare provider
- execute render with ready inputs

### Why

- Makes dependencies explicit.
- Reduces repeated provider-loading logic.
- Better matches the mental model that render execution should run on fully resolved inputs.

### Expected code impact

- Cleaner shared render helper signatures.
- Easier tests because provider setup is outside the execution core.

---

## Improvement 12: Encapsulate config-loader registry side effects

### What it is

`internal/config/loader.go` temporarily mutates `CUE_REGISTRY` during config loading.

### What would change

If the CUE SDK allows it, move to a non-global configuration path.

If not, wrap the env-mutation behavior in one narrow helper with:

- explicit setup
- explicit restore behavior
- no duplicated environment handling elsewhere

### Why

- Process-global env mutation increases surprise.
- Side effects should be easy to identify and isolate.
- Even if behavior remains unchanged, encapsulation reduces complexity.

### Expected code impact

- Cleaner config-loading flow.
- Easier future test setup.
- Better separation between registry resolution and CUE instance loading.

### Recommendation

Treat this as a follow-up simplification after the render/orchestration cleanup.

---

## Improvement 13: Make incomplete bundle support clearly internal or clearly deferred

### What it is

Bundle render types exist, but the main CLI path does not support bundle release rendering yet.

### What would change

Choose one of these approaches:

#### Option A: Keep but isolate

- keep bundle engine types
- document them as not yet part of the active CLI path
- avoid routing current simplification work through them

#### Option B: Remove premature orchestration scaffolding

- remove or reduce unused paths until actual bundle implementation begins

### Why

- Partial feature scaffolding increases code surface area.
- It adds conceptual weight to the codebase even when not used.
- Clear status helps future contributors understand what is real vs planned.

### Expected code impact

- Better signal about currently supported behavior.
- Less confusion while refactoring the active `ModuleRelease` path.

### Recommendation

Prefer Option A for now: keep the types, but clearly treat them as not-yet-wired CLI internals.

---

## Improvement 14: Standardize the pipeline vocabulary

### What it is

The codebase already naturally forms three phases:

1. prepare
2. process/render
3. emit/apply

But those stages are not always reflected clearly in naming.

### What would change

Use consistent naming for helpers and comments:

- `Prepare...`
- `Process...`
- `Write...` / `Apply...`

Avoid names that hide stage semantics when a function is actually phase-specific.

### Why

- Naming is part of simplification.
- Consistent stage language reduces the need to re-learn local conventions in each file.
- This makes the code easier to teach, review, and document.

### Expected code impact

- Better readability.
- Fewer ambiguous helper names.
- Easier onboarding for new contributors.

---

## Recommended execution order

To keep the work low-risk and incremental, apply the changes in this order:

1. Extract the common render tail from `internal/workflow/render/render.go`
2. Extract the repeated resource-conversion helper
3. Deduplicate `FormatApplySummary`
4. Remove wrapper-only output forwarding layers
5. Unify build-command execution logic
6. Consolidate values resolution into explicit preparation helpers
7. Refactor the release-file layer to return parsed DTOs instead of partial render objects
8. Reduce `ModuleRelease` phase ambiguity
9. Encapsulate config-loader registry side effects
10. Reassess bundle scaffolding boundaries

This order starts with the highest-value, lowest-risk changes and defers structural type changes until the orchestration shape is cleaner.

## Success criteria

- The render workflow reads as prepare -> process -> emit/apply.
- Module-source and release-file rendering share one common execution tail.
- Value precedence rules remain unchanged but live in fewer places.
- No package returns partially initialized render-state objects unless that is its explicit responsibility.
- Duplicate helpers and forwarding-only layers are removed.
- Reading the `release apply` flow requires fewer file jumps.

## Summary

The main simplification opportunity is not in the CUE execution engine itself. The strongest gains come from reducing orchestration duplication and tightening package boundaries around input preparation.

The code already has a good conceptual foundation:

- modules define intent
- releases define concrete deployment instances
- providers define transformer behavior
- the engine turns those into concrete resources

This plan keeps that model intact while making the surrounding code easier to navigate, easier to test, and easier to extend.
