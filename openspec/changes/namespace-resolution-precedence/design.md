## Context

Namespace resolution currently happens in two separate places:

1. **`config.ResolveKubernetes()`** — resolves `--namespace` > `OPM_NAMESPACE` > `config.kubernetes.namespace` > `"default"` into a `ResolvedField{Value, Source}` before the module is loaded
2. **`pipeline.resolveNamespace()`** — chooses between the pre-resolved namespace and `module.metadata.defaultNamespace`

The problem: by the time the pipeline sees the namespace, it's already been collapsed to a single string via `K8sConfig.Namespace.Value`. The source information (`ResolvedField.Source`) that tracks whether the value came from a flag, env var, config, or default is available in `cmdutil.RenderRelease()` but is discarded before reaching the pipeline. The pipeline therefore cannot distinguish "user explicitly set this" from "this is the config fallback" and cannot correctly insert `module.metadata.defaultNamespace` at step 3.

The fix is minimal: pass the namespace source through to the pipeline so it can conditionally override with the module's default namespace.

## Goals / Non-Goals

**Goals:**

- Insert `module.metadata.defaultNamespace` as step 3 in namespace resolution for build-pipeline commands
- Pipeline distinguishes user-explicit namespace (flag/env) from config/default fallback
- `GlobalConfig` is never mutated
- Non-pipeline commands (`mod delete`, `mod status`) are unaffected

**Non-Goals:**

- Changing the `config.ResolveKubernetes()` resolver or `ResolvedField` type
- Changing the `--namespace` flag behavior or `OPM_NAMESPACE` env var behavior
- Adding new CLI flags

## Decisions

### Decision 1: Pass namespace source via RenderOptions, not the full ResolvedField

`RenderOptions` gains a `NamespaceSource config.Source` field alongside the existing `Namespace string`. The pipeline uses this to decide whether to override with `module.metadata.defaultNamespace`.

**Why**: The pipeline only needs to know *where* the namespace came from — not the full `ResolvedField` with shadowed values. Passing just `Source` keeps `RenderOptions` simple and avoids a dependency from `build/types.go` to `config.ResolvedField`. The `Source` type is a plain `string` alias (`"flag"`, `"env"`, `"config"`, `"default"`) which is already defined in `internal/config/`.

**Alternative considered**: Pass the full `ResolvedField` struct. Rejected — `RenderOptions` belongs to the `build` package and should not take a hard dependency on config-layer types for a single field.

**Alternative considered**: Add a boolean `NamespaceExplicit bool` instead of the source. Rejected — less informative and less extensible; the source enum is already available and carries more useful debugging information.

### Decision 2: Pipeline overrides namespace only when source is config or default

The pipeline's `resolveNamespace()` method applies this logic after the module is loaded:

```text
if namespace source is flag or env:
    use opts.Namespace as-is (user was explicit)
else if module.Metadata.DefaultNamespace is non-empty:
    use module.Metadata.DefaultNamespace (step 3)
else:
    use opts.Namespace as-is (config or default fallback, step 4)
```

This is a simple conditional override — no new precedence chain function needed. The existing 3-step resolution in `config.ResolveKubernetes()` is unchanged; the pipeline just applies one additional override in a specific condition.

**Why**: Minimal change. The config resolver continues to work as before for all commands. Only the pipeline applies the module-specific override. Non-pipeline commands never see this logic.

### Decision 3: cmdutil.RenderRelease passes source from K8sConfig

`cmdutil.RenderRelease()` already has `opts.K8sConfig.Namespace.Source`. The change is one line:

```go
renderOpts := build.RenderOptions{
    // ...existing fields...
    Namespace:       namespace,
    NamespaceSource: opts.K8sConfig.Namespace.Source,  // new
}
```

**Why**: The source information is already available at the call site. No upstream changes needed.

### Decision 4: Remove the "default" hardcoded fallback from NamespaceRequiredError

The current pipeline has a `NamespaceRequiredError` that fires when `resolveNamespace()` returns `""`. With the 4-step precedence chain, this error should only occur if all four steps produce an empty string — which means `config.kubernetes.namespace` is unset and the module has no `defaultNamespace`. The `"default"` hardcoded fallback in `config.ResolveKubernetes()` (line 213 of resolver.go) already prevents this in practice. No change needed to the error handling — it remains as a defensive guard.

### Decision 5: ExtractMetadata removal is handled by core-module-receiver-methods

The proposal mentions removing `ExtractMetadata` and `MetadataPreview`. This removal is already captured in the `core-module-receiver-methods` change (Decision 4 in that design). This change depends on `core-module-receiver-methods` being complete — specifically `module.Load()` returning a `*core.Module` with `Metadata.DefaultNamespace` populated from AST inspection.

## Risks / Trade-offs

**Behavioral change for modules with `defaultNamespace`** — modules that define `metadata.defaultNamespace` and were previously deployed without `--namespace` will now use the module's namespace instead of `config.kubernetes.namespace`. This changes the release UUID (namespace is an input to UUID5 computation), which means the first deploy after this change will create a new release identity rather than updating the existing one. Mitigated by: this only affects the narrow case where `defaultNamespace` is set and no explicit namespace is provided.

**`build` package imports `config.Source`** — `RenderOptions` will import the `Source` type from `internal/config/`. This is a new dependency direction (`build` → `config`) but only for a type alias (`type Source string`). The dependency is lightweight and justified by avoiding duplication.

**Alternative to avoid the import**: Define `NamespaceSource string` directly in `build/types.go` as a plain string and document the expected values. This avoids the import but loses type safety. Worth considering if the `build` → `config` dependency feels wrong.

## Migration Plan

1. Add `NamespaceSource` field to `RenderOptions` in `internal/build/types.go`
2. Update `cmdutil.RenderRelease()` to pass `K8sConfig.Namespace.Source` into `RenderOptions.NamespaceSource`
3. Update `pipeline.resolveNamespace()` to accept source and module default namespace; apply the conditional override logic
4. Update `pipeline.Render()` PREPARATION phase to call `resolveNamespace()` with the new arguments after `module.Load()` returns
5. Verify `task test` passes
6. Add tests for the 4-step precedence scenarios in the spec

Rollback: revert `build/types.go`, `cmdutil/render.go`, and `build/pipeline.go`. The config resolver is untouched.

## Open Questions

- **`build` importing `config.Source`**: Should `RenderOptions.NamespaceSource` use `config.Source` directly, or should the field be a plain `string` to avoid the import? The typed version is safer but introduces a dependency direction. Lean toward using `config.Source` since it's a simple type alias.
