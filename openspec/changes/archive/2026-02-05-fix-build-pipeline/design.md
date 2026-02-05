# Design: Fix Build Pipeline

## Context

OPM modules use CUE definitions (`#components`, `#values`) to define reusable schemas. The current module loader in `internal/build/module.go` looks for concrete fields (`components`, `values`) which don't exist in standard OPM modules.

The OPM type system has two key types:

- `#Module`: Developer-authored blueprint with definitions (`#components`, `#values`)
- `#ModuleRelease`: Deployment instance that wraps a module and forces concreteness

Currently, developers must create a separate `#ModuleRelease` file to use `opm mod build`, which is cumbersome during development.

## Goals / Non-Goals

**Goals:**

- Enable `opm mod build` to work with standard `#Module` definitions without requiring a release file
- Surface clear validation errors when required fields are missing
- Maintain compatibility with explicit `#ModuleRelease` files (if provided, use them)

**Non-Goals:**

- Changing the OPM type system or module authoring patterns
- Supporting partial/incomplete modules (all required fields must be satisfiable)
- Persisting the synthesized release to disk

## Decisions

### Decision 1: Synthesize `#ModuleRelease` in CUE, not Go

**Choice**: Build a CUE expression that wraps the loaded module in `#ModuleRelease`, then evaluate it.

**Rationale**:

- Leverages CUE's type system for validation (required fields, constraints)
- Produces clear CUE error messages when validation fails
- Keeps the CLI aligned with OPM's CUE-first philosophy

**Alternatives considered**:

- *Go-based extraction*: Manually look for `#components` path and extract. Rejected because it bypasses CUE validation and would require reimplementing constraint checking in Go.

### Decision 2: Inject release wrapper via CUE unification

**Choice**: Create a CUE value that unifies the module with a release template:

```cue
import "opmodel.dev/core@v0"

_release: core.#ModuleRelease & {
    metadata: {
        name:      "<from-module>"
        namespace: "<from-cli-flag>"
    }
    #module: <loaded-module-value>
}
```

**Rationale**:

- CUE unification naturally forces `#components` → `components` via the release schema
- Validation errors come from CUE with proper source locations
- The `#module` field connects to the loaded module value

### Decision 3: Extract components from `_release.components`

**Choice**: After unification, extract components from the concrete `components` field on the release.

**Rationale**:

- The `#ModuleRelease` schema defines `components: #module.#components`
- This path is guaranteed to be concrete after successful unification
- Consistent with how the render pipeline expects component data

### Decision 4: Require `--namespace` flag when synthesizing

**Choice**: The `--namespace` flag becomes required for `opm mod build` when no explicit release file exists.

**Rationale**:

- `#ModuleRelease.metadata.namespace` is required by the schema
- Explicit is better than implicit (Principle VII)
- Aligns with deployment semantics (you must know where you're deploying)

**Alternatives considered**:

- *Default to "default" namespace*: Rejected because it could lead to accidental deployments to wrong namespace.
- *Infer from kubeconfig context*: Rejected because build should be hermetic and not depend on cluster state.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| CUE evaluation overhead | Minimal - already loading module via CUE. One additional unification step. |
| Error message clarity | CUE errors include source locations. Wrap with user-friendly context in CLI. |
| Breaking change if existing modules rely on `components` field | MINOR version bump. Document in changelog. Most modules use `#components`. |
| Registry dependency for `opmodel.dev/core@v0` | Already required for module imports. No new dependency. |

## Implementation Approach

1. **Detect module type**: Check if loaded value satisfies `#ModuleRelease` (has `components` field). If yes, use directly. If no, synthesize.

2. **Build release expression**: Construct CUE that imports core and wraps module:

   ```go
   releaseExpr := fmt.Sprintf(`
   import "opmodel.dev/core@v0"
   _release: core.#ModuleRelease & {
       metadata: name: %q
       metadata: namespace: %q
       #module: _
   }
   `, moduleName, namespace)
   ```

3. **Unify and validate**: Unify the release expression with the loaded module value. CUE validates all constraints.

4. **Extract components**: Read `_release.components` for the concrete component map.

5. **Handle errors**: Wrap CUE validation errors with actionable guidance (e.g., "missing required field 'container.name' in component 'web'").
