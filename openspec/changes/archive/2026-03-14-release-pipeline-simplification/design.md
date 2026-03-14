## Context

`pkg/module/release.go` defines a single `Release` struct that carries both parse-time and process-time state. The struct has 6 fields, but only 4 are populated at construction (`Metadata`, `Module`, `RawCUE`, `Config`); the remaining 2 (`DataComponents`, `Values`) start as zero `cue.Value{}` and are set later by `ProcessModuleRelease`. The `RawCUE` field is mutated in-place at two points: once for `--module` injection and once for values filling. The `Config` field duplicates `Module.Config`.

The public rendering API (`ProcessModuleRelease`) currently does everything: config validation, values merging, values filling, concreteness checking, component finalization, matching, and execution — all in one function. There is no clear boundary between "prepare a release" and "render a release." The internal workflow layer (`internal/workflow/render`) calls `ProcessModuleRelease` as a black box but also does its own preparation work (module injection, values resolution, namespace override) by mutating the same `*module.Release` struct.

## Goals / Non-Goals

**Goals:**

- Simplify `module.Release` to four fields: `Metadata`, `Module`, `Spec`, `Values` — each with a clear invariant
- Introduce `ParseModuleRelease` in `pkg/module` to own the preparation boundary: validate, merge, fill, concrete, decode, construct
- Simplify `ProcessModuleRelease` in `pkg/render` to own the rendering boundary: finalize components, match, execute, return `*ModuleResult`
- Remove `Config` field from `Release` — use `Module.Config` directly
- Remove `DataComponents` field — finalized components are transient locals in processing
- Rename `RawCUE` → `Spec` for clarity
- Remove `ExecuteComponents()` method — no stored data components to return
- Keep `MatchComponents()` method on `Release` — it performs a meaningful `LookupPath` computation
- Update all consumers and tests

**Non-Goals:**

- Do not change matching, transform execution, or value precedence behavior
- Do not change bundle release types beyond field renames (bundle support is incomplete)
- Do not change the `RenderResult` type in `internal/workflow/render` — it already copies metadata by value
- Do not change `ValidateConfig` itself — only move where it is called

## Decisions

### Decision 1: Single `Release` type, not a phase split

Keep one `module.Release` struct with a clear invariant: when you have a `*module.Release`, it is fully prepared — `Spec` is concrete and complete, `Values` is concrete and merged, `Metadata` is decoded.

**Rationale**: The real semantic boundary in the pipeline is not "parsed vs processed" but "prepared release vs rendered output." A single type with a clear construction invariant is simpler than two types representing adjacent pipeline phases. The finalized executable components (the only field that would justify a second type) are transient processing data, not a domain concept worth storing.

**Alternative considered**: Two types (`ParsedRelease`, `ProcessedRelease`) — rejected because `Spec` is already values-filled and concrete after preparation, so the split would only add a `Components` field that is better kept as a local variable.

### Decision 2: `ParseModuleRelease` constructs the prepared release

A new public function `ParseModuleRelease` in `pkg/module` takes the raw release spec, module, and values, and returns a fully prepared `*module.Release`.

**Rationale**: This creates a clear preparation boundary. The function name uses "Parse" because it takes raw inputs and produces a typed domain object. It validates values against `Module.Config`, merges them, fills them into the spec, ensures concreteness, decodes metadata, and constructs the release. After `ParseModuleRelease`, the release is ready for rendering.

**Signature**:
```go
func ParseModuleRelease(ctx context.Context, spec cue.Value, mod Module, values []cue.Value) (*Release, error)
```

**Target skeleton**:
```go
// Release is a fully prepared module release ready for rendering.
type Release struct {
    // Metadata is the decoded release identity from the concrete release spec.
    Metadata *ReleaseMetadata

    // Module is the original module used to prepare the release.
    Module   Module

    // Spec is the concrete, values-filled #ModuleRelease CUE value.
    // Concrete (all regular fields resolved) but NOT finalized — CUE definition
    // fields (#resources, #traits, #blueprints) are preserved. Required by
    // MatchComponents() for component-transformer matching.
    // MUST NOT be passed to finalizeValue or v.Syntax(cue.Final()).
    Spec     cue.Value

    // Values is the concrete, merged values applied to the release.
    Values   cue.Value
}

// ParseModuleRelease validates values, fills them into the release spec,
// ensures the result is concrete, decodes metadata, and constructs Release.
func ParseModuleRelease(ctx context.Context, spec cue.Value, mod Module, values []cue.Value) (*Release, error)
```

### Decision 3: `ProcessModuleRelease` returns `*ModuleResult`

`ProcessModuleRelease` is the single public rendering entrypoint. It accepts a prepared `*module.Release` and a `*provider.Provider`, and returns `*render.ModuleResult`. Matching and execution happen internally — callers do not orchestrate pipeline phases.

**Rationale**: The stated goal is that the public pipeline starts and ends with `ProcessModuleRelease`. Returning an intermediate type would leak pipeline structure back into callers.

**Signature**:
```go
func ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider) (*ModuleResult, error)
```

**Target skeleton**:
```go
// ProcessModuleRelease renders a prepared release with the given provider.
func ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider) (*ModuleResult, error)

// Execute runs matched transformers against the provided component views and
// returns rendered resources, component summaries, and warnings.
func (r *Module) Execute(
    ctx context.Context,
    rel *module.Release,
    schemaComponents cue.Value,
    dataComponents cue.Value,
    plan *MatchPlan,
) (*ModuleResult, error)
```

### Decision 4: `Config` is not duplicated on `Release`

The `Config` field is removed from `Release`. Callers access the config schema via `Release.Module.Config`.

**Rationale**: `Config` was always derived from `Module.Config` (or `#module.#config`). Storing it separately created two sources of truth. `ParseModuleRelease` reads `mod.Config` directly during validation.

### Decision 5: Finalized components are transient

The values-filled and finalized components (`finalizeValue` output) are local variables inside `ProcessModuleRelease`, not stored on `Release`.

**Rationale**: No downstream consumer needs the finalized components after rendering is complete. `ModuleResult` already carries the rendered `[]*core.Resource` and `[]ComponentSummary`. Storing finalized components would add a field that only `ProcessModuleRelease` internals read.

### Decision 6: `MatchComponents()` stays on `Release`

`Release.MatchComponents()` returns `rel.Spec.LookupPath(cue.ParsePath("components"))` — the schema-preserving component view used for matching. It stays as a method because it encapsulates a CUE path lookup.

**Rationale**: The method preserves CUE definition fields (`#resources`, `#traits`, `#blueprints`) needed for matching. This is distinct from the finalized/constraint-free component view used for execution. Keeping it as a method documents the intent and hides the path detail.

**Target skeleton**:
```go
// MatchComponents returns the schema-preserving components value used for
// matching. The returned value keeps definition fields such as #resources,
// #traits, and #blueprints.
func (r *Release) MatchComponents() cue.Value {
    return r.Spec.LookupPath(cue.ParsePath("components"))
}
```

### Decision 7: `Values` stored on `Release`

The merged, validated, concrete values are stored as `Release.Values`. This field is populated by `ParseModuleRelease` during construction.

**Rationale**: `Values` is useful for downstream inspection, debugging, and potential future use in the rendering pipeline. It also makes the release self-describing — you can see exactly what values were applied.

### Decision 8: `Spec` is concrete but NOT finalized

`Release.Spec` passes `cue.Concrete(true)` validation but retains CUE definition fields (`#resources`, `#traits`, `#blueprints`). These are two distinct CUE concepts:

- **Concrete** (`cue.Concrete(true)`) — all regular fields have definite values. Read-only check. Definitions survive.
- **Finalized** (`cue.Final()` / `finalizeValue`) — constraints stripped, definitions removed, pure data. Definitions do NOT survive.

`MatchComponents()` depends on definitions surviving in `Spec`. The matching pipeline (`match.go`) accesses `cue.Def("resources")` and `cue.Def("traits")` on each component to build the match plan. If `Spec` were finalized, these lookups would silently return nothing and matching would break — components would appear to have no resources or traits.

**Rationale**: The constraint-free (finalized) component view is only needed for transformer execution — it is derived transiently inside `ProcessModuleRelease` via `finalizeValue(schemaComponents)` and passed as a local variable. It is never stored on `Release` and never derived from `Spec` directly. `Spec` must remain un-finalized for the lifetime of the `Release`.

**Invariant**: `Release.Spec` MUST NOT be passed to `finalizeValue` or `v.Syntax(cue.Final())`.

## Risks / Trade-offs

**[Risk: `--module` injection mutates raw spec before `ParseModuleRelease`]** → Accepted. The internal workflow layer fills `#module` into the raw spec via `FillPath` before calling `ParseModuleRelease`. This is a preparation-phase mutation on the raw `cue.Value`, not on a `*module.Release`. Once `ParseModuleRelease` constructs the release, no further mutations occur.

**[Risk: `GetReleaseFile` return type change]** → Medium risk. `GetReleaseFile` currently returns `*FileRelease` containing `*module.Release`. It will need to return raw parse data instead, since `*module.Release` now requires validated values to construct. The `FileRelease` struct and `bareModuleRelease` helper need updating.

**[Risk: Bundle `Releases` map entries]** → Low risk. `bundle.Release.Releases` stays `map[string]*module.Release`. Bundle rendering is unimplemented (`ProcessBundleRelease` returns "not implemented yet"). When bundle support is implemented, each map entry will need to go through `ParseModuleRelease` before rendering.

**[Trade-off: `ParseModuleRelease` does more than "parse"]** → Accepted. The name "Parse" slightly undersells the function (it also validates, merges, fills, and checks concreteness). But "Parse" is conventional for "take raw inputs, produce typed output." Alternatives like `PrepareModuleRelease` or `BuildModuleRelease` were considered but Parse aligns better with the domain vocabulary.

**[Trade-off: `ProcessModuleRelease` no longer accepts `[]cue.Value`]** → Feature, not cost. Callers must validate and merge values before calling `ProcessModuleRelease`. This makes the preparation/rendering boundary explicit and prevents `ProcessModuleRelease` from accumulating preparation responsibilities.

## API Sketch

The intended end state is:

```go
package module

// Release is a fully prepared module release ready for rendering.
type Release struct {
    // Metadata is the decoded release identity from the concrete release spec.
    Metadata *ReleaseMetadata

    // Module is the original module used to prepare the release.
    Module   Module

    // Spec is the concrete, values-filled #ModuleRelease CUE value.
    // Concrete (all regular fields resolved) but NOT finalized — CUE definition
    // fields (#resources, #traits, #blueprints) are preserved. Required by
    // MatchComponents() for component-transformer matching.
    // MUST NOT be passed to finalizeValue or v.Syntax(cue.Final()).
    Spec     cue.Value

    // Values is the concrete, merged values applied to the release.
    Values   cue.Value
}

// ParseModuleRelease validates values, fills them into the release spec,
// ensures the result is concrete, decodes metadata, and constructs Release.
func ParseModuleRelease(ctx context.Context, spec cue.Value, mod Module, values []cue.Value) (*Release, error)

// MatchComponents returns the schema-preserving components value used for
// matching. The returned value keeps definition fields such as #resources,
// #traits, and #blueprints.
func (r *Release) MatchComponents() cue.Value {
    return r.Spec.LookupPath(cue.ParsePath("components"))
}
```

```go
package render

// ProcessModuleRelease renders a prepared release with the given provider.
func ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider) (*ModuleResult, error)

// Execute runs matched transformers against the provided component views and
// returns rendered resources, component summaries, and warnings.
func (r *Module) Execute(
    ctx context.Context,
    rel *module.Release,
    schemaComponents cue.Value,
    dataComponents cue.Value,
    plan *MatchPlan,
) (*ModuleResult, error)
```

`finalizeValue(...)`, `Match(...)`, `executeTransforms(...)`, `executePair(...)`, and `injectContext(...)` remain implementation details inside `pkg/render`.
