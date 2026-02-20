## Context

`Builder.Build()` in `internal/build/release/builder.go` currently embeds three distinct validation operations directly in its body (steps 4b, 4c, and 7), and returns the build-internal `BuiltRelease` type. This means the pipeline cannot call validation as a discrete step — it is invisible inside `Build()`, and any caller that wants to skip or substitute a validation step cannot do so.

`BuiltRelease` is a build-package-private type (`internal/build/release/types.go`) with no identity outside that package. The pipeline stores it as `rel` and passes it directly to the executor. Meanwhile `core.ModuleRelease` exists as the canonical domain type for the same concept but is currently unused in this flow.

The validation logic itself lives in `internal/build/release/validation.go` as package-level functions (`validateValuesAgainstConfig`, `collectAllCUEErrors`) that take raw `cue.Value` arguments. They are straightforward field-walking operations on values the `ModuleRelease` already owns — no external CUE context is required because `cue.Value` carries its own runtime reference.

## Goals / Non-Goals

**Goals:**

- Add `ValidateValues() error` and `Validate() error` as receiver methods on `core.ModuleRelease`
- Make `release.Build()` return `*core.ModuleRelease` instead of `*BuiltRelease`
- Move the two validation calls out of `Builder.Build()` and into the pipeline as explicit `rel.ValidateValues()` → `rel.Validate()` steps
- Remove the `BuiltRelease` type once `core.ModuleRelease` replaces it

**Non-Goals:**

- Changing validation logic or error types — only ownership moves
- Modifying how the CUE overlay, values unification, or component extraction works
- Changing `build/pipeline.go` beyond the BUILD phase (phases 3–6 untouched)
- Consolidating `build/component.Component` → `core.Component` (separate change)

## Decisions

### Decision 1: Validation methods operate on already-populated cue.Value fields

`ValidateValues()` needs to walk `rel.Values` against `rel.Module.Config`. Both are `cue.Value` fields on `ModuleRelease`. CUE values carry an implicit reference to the `cue.Context` that compiled them, so all CUE operations (`Fields()`, `Allows()`, `Unify()`, `Validate()`) work directly on the stored values without needing a context parameter.

**Alternative considered**: Pass `*cue.Context` as a method parameter. Rejected — the context is already embedded in the values themselves; passing it explicitly would be redundant and would force the caller to hold a context reference just to call validation.

### Decision 2: Builder.Build() extracts values before returning

Currently `Builder.Build()` calls `validateValuesAgainstConfig(configDef, valuesVal)` mid-method, using locally-scoped `cue.Value` variables. After this change, those same values must be on the returned `*core.ModuleRelease` so the receiver methods can access them.

`ModuleRelease.Values` already exists as a field (the end-user values). `ModuleRelease.Module.Config` holds the schema. The builder already extracts both into the `BuiltRelease` flow; it simply needs to populate these fields on `*core.ModuleRelease` instead.

The builder continues to do all CUE loading work. It returns a `*core.ModuleRelease` with `Values` and `Module.Config` populated. The receiver methods then perform validation using those stored values.

### Decision 3: Remove validation steps 4b and 7 from Builder.Build()

Steps 4b (`validateValuesAgainstConfig`) and 7 (concrete component check) are removed from `Build()`. The pipeline becomes responsible for calling them via receiver methods immediately after `Build()` returns. Step 4c (`collectAllCUEErrors` on the full tree) stays in `Build()` because it is a loading-time structural check, not a validation gate the caller needs to control.

**Rationale**: Steps 4b and 7 are the validations the pipeline spec explicitly calls out as discrete pipeline phases. Keeping 4c in `Build()` is correct because a module that fails CUE validation cannot produce a usable release regardless — it is a fatal construction error, not a validatable condition.

### Decision 4: BuiltRelease removed; core.ModuleRelease is the single type

`BuiltRelease` is a structural duplicate of `core.ModuleRelease` with different field names (`Components` using `build/component.Component` instead of `core.Component`). The two types will diverge permanently if left in place.

`core.ModuleRelease.Components` uses `map[string]*core.Component`. Since `core.Component` and `build/component.Component` are currently identical structs, `Build()` can copy or cast them without data loss. This does not depend on the separate `core.Component` consolidation change.

**Alternative considered**: Keep `BuiltRelease` as an internal builder output and add a `.ToModuleRelease()` conversion method. Rejected — conversion adds indirection with no benefit; the builder can populate `*core.ModuleRelease` directly.

### Decision 5: pipeline.go BUILD phase becomes three explicit steps

```go
// BEFORE (all hidden inside Build())
rel, err := p.releaseBuilder.Build(modulePath, releaseOpts, opts.Values)

// AFTER (three visible steps)
rel, err := release.Build(p.cueCtx, mod, releaseOpts, opts.Values)  // → *core.ModuleRelease
if err := rel.ValidateValues(); err != nil { return nil, err }
if err := rel.Validate(); err != nil { return nil, err }
```

The pipeline phases map directly to the spec: PREPARATION (module load) → BUILD (release construction + explicit validation gates) → MATCHING → GENERATE.

## Risks / Trade-offs

**Risk: Callers of Builder.Build() that relied on implicit validation** → There is one caller (`pipeline.go`). It is updated in this change to call the receiver methods explicitly. No external callers exist.

**Risk: Concurrent test suites that mock BuiltRelease** → Tests in `internal/build/transform/executor_test.go` and `context_annotations_test.go` reference `release.ReleaseMetadata` directly (already showing LSP errors indicating partial migration is in progress). These tests will need updating to use `core.ReleaseMetadata` and `*core.ModuleRelease`. This is expected cleanup, not new breakage.

**Risk: ValidateValues() on a ModuleRelease with nil Module.Config** → If `Module.Config` is the zero `cue.Value` (not populated), `LookupPath` calls will return non-existent values. `ValidateValues()` must guard: if `!rel.Module.Config.Exists()` or `!rel.Values.Exists()`, return nil (nothing to validate). This mirrors the current `if configDef.Exists() && valuesVal.Exists()` guard in `Build()`.

## Migration Plan

1. Add `ValidateValues()` and `Validate()` methods to `internal/core/module_release.go` — move logic from `release/validation.go` functions
2. Update `release/builder.go`: remove steps 4b and 7; return `*core.ModuleRelease` populated from the existing extraction logic; keep step 4c
3. Remove `release/types.go` `BuiltRelease` type (or deprecate as an alias until all tests are updated)
4. Update `build/pipeline.go` BUILD phase to call receiver methods after `Build()` returns
5. Update affected tests: `executor_test.go`, `context_annotations_test.go`
6. Run `task check` to confirm no regressions

Each step is independently compilable — the Go type system will surface all callers that need updating at step 3.

## Open Questions

None — decisions above are sufficient to begin implementation.
