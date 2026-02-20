## Context

`internal/build/module/loader.go` currently uses a two-pass approach to populate `core.Module`:

1. **AST pass** (`inspectModule` → `ExtractMetadataFromAST`): walks raw CUE file AST to extract `metadata.name` and `metadata.defaultNamespace` as string literals only. Also captures `inst.PkgName`.
2. **Evaluation pass** (`extractModuleMetadata`): uses the fully evaluated `cue.Value` to extract `fqn`, `version`, `identity`, and `labels` via `LookupPath`.

The split exists because `load.Instances()` is called inside `inspectModule`, which doubles as the loader for the `*build.Instance`. But the AST scanning of `name`/`defaultNamespace` is unnecessary — those fields evaluate to concrete strings just like `fqn` and `version`, and they can be read from the evaluated value with the same `LookupPath` pattern already in use.

The result is two files (`inspector.go`, `types.go`) and an `Inspection` intermediate type that exist solely to hold results from the AST walk — which is then immediately superseded by evaluation results.

## Goals / Non-Goals

**Goals:**

- Collapse all metadata extraction into a single post-evaluation pass using `LookupPath`
- Delete `inspector.go`, `types.go`, and the `Inspection` struct
- Inline `load.Instances()` directly into `Load` (it was only called from `inspectModule`)
- Remove the `"cuelang.org/go/cue/build"` import from `loader.go` (no longer needed in exported surface)
- Update `Validate()` error message to remove the "string literal" language
- Update tests to reflect that computed metadata names now resolve correctly

**Non-Goals:**

- Changes to `core.Module` fields or their types
- Changes to calling code outside `internal/build/module`, `internal/core`, and `internal/build/release`

## Decisions

### Use `LookupPath` + `.String()` for name and defaultNamespace

**Decision**: Extract `metadata.name` and `metadata.defaultNamespace` from the evaluated `cue.Value` using the same `LookupPath` pattern already used for `fqn`, `version`, `identity`.

**Rationale**: CUE evaluation makes all concrete string fields available uniformly. There is no behavioral difference between extracting a literal string and a computed string — both evaluate to `string` in Go. The AST walk was an optimization that doesn't apply (evaluation runs regardless), and it silently dropped valid computed names.

**Alternative considered**: Keep AST walk for `name` as a validation gate (enforce literal-only). Rejected — the constraint was undocumented and surprising to module authors whose names happened to be computed. CUE's own type system is the right enforcement layer (if the field is not concrete, `.String()` fails and the field is left empty, which `Validate()` catches).

### Inline `load.Instances()` into `Load`

**Decision**: Remove `inspectModule` as a separate function. Call `load.Instances()` directly in `Load`, extract `inst.PkgName`, then call `BuildInstance()`.

**Rationale**: `inspectModule` was only ever called from `Load`. Separating it added indirection without providing a reusable API. The resulting code is linear and easier to follow.

### Extend `extractModuleMetadata` to cover name and defaultNamespace

**Decision**: Add `metadata.name` and `metadata.defaultNamespace` lookups to the existing `extractModuleMetadata` helper, keeping the extraction logic in one place.

**Rationale**: All scalar metadata fields follow the same pattern. Consolidating into one function makes the extraction surface obvious and easy to extend.

### Extract Component.Spec from evaluated value

**Decision**: In `extractComponent()`, add a `LookupPath(cue.ParsePath("spec"))` block after `#traits` extraction, assigning the result to `comp.Spec` if it exists.

**Rationale**: `Spec` is declared on `Component` and covered by the `core-component` spec scenario ("Extracted component has Spec value"). The field was consistently skipped — the test fixture defines `spec: { container: {...}, replicas: ... }` inside the `web` component, but no code ever read it. Adding the lookup follows the identical pattern already used for `#resources` and `#traits`.

**Alternative considered**: None — the omission is unambiguous.

### Initialize and extract Component.Blueprints

**Decision**: Add `Blueprints: make(map[string]cue.Value)` to the struct literal in `extractComponent()`, and add a `#blueprints` extraction block using the same pattern as `#traits`.

**Rationale**: `Traits` is initialized to an empty map even when absent; `Blueprints` was not initialized at all, leaving it `nil`. The inconsistency makes callers that range over `Blueprints` panic if no blueprints are defined. Consistent initialization (always a non-nil map, possibly empty) matches the design intent and the `Traits` precedent.

### Add IsConcrete() gate in Build()

**Decision**: After `core.ExtractComponents()` in `builder.go:Build()`, iterate all extracted components and return `fmt.Errorf("component %q is not concrete after value injection", name)` for any where `!comp.IsConcrete()`.

**Rationale**: The spec (`core-component/spec.md`, scenario "Build() gates concrete-only operations on IsConcrete()") requires `Build()` itself to return the error. Currently, the concreteness check happens later in `pipeline.go` via `rel.Validate()`, but the spec assigns responsibility to `Build()`. Placing the check here is also more diagnostic: the error names the specific component, rather than surfacing a generic CUE validation error downstream.

**Alternative considered**: Leave it in `pipeline.go:Validate()`. Rejected — the spec is explicit, and early failure with a named component is better UX.

### Add UUID to test fixture and TestLoad_ValidModule

**Decision**: Add `identity: "test-uuid-1234"` to `testdata/test-module/module.cue` and add `assert.NotEmpty(t, mod.Metadata.UUID)` to `TestLoad_ValidModule`.

**Rationale**: The UUID extraction path (`metadata.identity` → `meta.UUID`) was silently uncovered. The existing test structure already asserts `FQN` and `Version` from the same evaluation pass; adding `UUID` follows the same pattern and closes the gap.

## Risks / Trade-offs

**Computed names now resolve** → Previously, a module with `name: _base + "-suffix"` produced `Name == ""` which caused `Validate()` to fail with a confusing "string literal" message. After this change, the name resolves correctly. This is a fix, not a risk, but test assertions need updating.

**No behavioral change for well-formed modules** → Modules with literal `metadata.name` (the majority) see identical behavior. The evaluated string is the same as the AST-extracted string.

**IsConcrete gate is a new early-exit path in Build()** → Modules that were previously reaching the pipeline's `Validate()` step with non-concrete components will now fail earlier in `Build()`. The error message is more specific. No well-formed module is affected.

## Open Questions

None. Scope is bounded to `internal/build/module`, `internal/core`, and `internal/build/release`.
