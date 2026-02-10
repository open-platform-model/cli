## Context

The CLI currently identifies deployed resources using three labels stamped at apply time: `managed-by`, `module name`, and `module namespace`. Delete, status, and diff discover resources by scanning the cluster with a label selector built from these three values. This works but provides no secondary identification — if labels are tampered with or the module is renamed, resources become orphaned.

The catalog schema change (companion `deterministic-release-identity` change) adds computed `metadata.identity` fields to `#Module` and `#ModuleRelease` using CUE's `uuid.SHA1` builtin. These are UUID v5 values, deterministic from the module's FQN/version and the release's FQN/name/namespace. This CLI change consumes those identities and uses them for labeling and discovery.

**Current data flow:**

```text
CUE module → build.ModuleLoader → build.ReleaseBuilder → build.Executor → RenderResult
                                        │
                                        ├─ ReleaseMetadata { Name, Namespace, Version, FQN, Labels }
                                        │
                                        └─ (identity not extracted)

RenderResult.Module (ModuleMetadata) → kubernetes.Apply → injectLabels
```

**Key constraint:** `kubernetes.Apply` currently receives `build.ModuleMetadata`, not `build.ReleaseMetadata`. The release builder produces both, but only `ModuleMetadata` is propagated through `RenderResult.Module` to the apply path.

## Goals / Non-Goals

**Goals:**

- Stamp `module-release.opmodel.dev/uuid` and `module.opmodel.dev/uuid` labels on all applied resources
- Enable dual-strategy discovery: release-id label + name/namespace labels, unioned and deduplicated
- Add `--release-id` flag to `mod delete` as an alternative to `--name`
- Display identity in `mod status` output
- Maintain full backwards compatibility with pre-identity resources

**Non-Goals:**

- Modifying the CUE catalog schemas (separate change)
- Implementing inventory ConfigMap or state storage
- Implementing ModuleRelease CR with ownerReferences
- Computing UUIDs in Go (they come from CUE evaluation)

## Decisions

### Decision 1: Extend ModuleMetadata rather than passing ReleaseMetadata to Apply

**Choice:** Add `Identity` and `ReleaseIdentity` fields to `build.ModuleMetadata`, extracted during release building, and propagated through the existing `RenderResult.Module` path.

**Alternative considered:** Change `kubernetes.Apply` signature to accept `build.ReleaseMetadata` instead of or in addition to `build.ModuleMetadata`.

**Rationale:** `ModuleMetadata` is already the established bridge between the build pipeline and the kubernetes package. It flows through `RenderResult.Module` and is consumed by apply, status, and diff. Adding two string fields to it is minimal. Changing the `Apply` signature would cascade through all call sites (production + integration tests) and blur the separation between build and kubernetes packages for no structural gain.

```go
type ModuleMetadata struct {
    Name      string
    Namespace string
    Version   string
    Labels    map[string]string
    Components []string
    // New fields:
    Identity        string  // Module identity UUID (from #Module.metadata.identity)
    ReleaseIdentity string  // Release identity UUID (from metadata.labels)
}
```

### Decision 2: Conditional label injection (empty = skip)

**Choice:** Only inject identity labels when the value is non-empty. If the module was built from an older catalog version that doesn't compute identity, the labels are simply omitted.

**Rationale:** This mirrors the existing pattern for version and component labels (`if meta.Version != ""`, `if res.Component != ""`). It makes the feature backwards-compatible without branching logic — the zero value is the off switch.

### Decision 3: Union discovery with UID-based deduplication

**Choice:** When both release-id and name+namespace are available, run two separate `List` calls per API resource type, merge results, and deduplicate by Kubernetes resource UID (`metadata.uid`).

**Alternative considered:** Single label selector with OR logic. Kubernetes label selectors don't support OR across different keys — you'd need two queries anyway.

**Alternative considered:** Query by release-id only, skip name+namespace. This would break for pre-identity resources that don't have the release-id label.

**Rationale:** Two queries per resource type has negligible overhead (the API server handles both efficiently via label indexes). UID-based deduplication is O(n) and foolproof — UIDs are guaranteed unique within a cluster.

```text
DiscoverResources(ctx, client, DiscoveryOptions{
    ModuleName:  "blog",        // may be empty
    Namespace:   "default",     // required
    ReleaseID:   "a1b2c3...",   // may be empty
})

Strategy:
  If ReleaseID != "" → query by release-id selector
  If ModuleName != "" → query by name+namespace selector
  Union both result sets, deduplicate by UID
```

### Decision 4: DiscoverResources takes an options struct

**Choice:** Replace the current `DiscoverResources(ctx, client, moduleName, namespace)` signature with `DiscoverResources(ctx, client, opts DiscoveryOptions)` where opts includes optional `ReleaseID`.

**Rationale:** The function already takes 4 params. Adding release-id as a 5th positional parameter is fragile. An options struct is idiomatic Go for optional parameters and aligns with how `DeleteOptions` and `ApplyOptions` already work. This is a breaking change to the internal API but not the CLI surface.

### Decision 5: --name becomes conditionally required on delete

**Choice:** `mod delete` requires `--namespace` always, plus at least one of `--name` or `--release-id`. Cobra's `MarkFlagRequired("name")` is replaced with manual validation in `RunE`.

**Rationale:** Cobra doesn't support "at least one of" natively. Manual validation with a clear error message ("either --name or --release-id is required") is straightforward and matches user expectations.

### Decision 6: Extract identity from CUE labels (not computed in Go)

**Choice:** In `extractMetadata`, read both identity values from the CUE evaluation output:

- `metadata.identity` — the module identity UUID (extracted via `LookupPath`)
- `metadata.labels."module-release.opmodel.dev/uuid"` — the release identity UUID (extracted from the labels map)

**Rationale:** The CUE catalog schemas compute both identity values and inject the release-id as a label on the module. By extracting from labels rather than computing in Go, we:

1. Maintain a single source of truth (CUE catalog is authoritative)
2. Avoid duplicating UUID v5 computation logic across languages
3. Eliminate the risk of Go/CUE UUID computation drift
4. Keep the CLI as a pure consumer of catalog-produced values

```go
// In extractMetadata:

// Module identity from metadata.identity
if v := concreteModule.LookupPath(cue.ParsePath("metadata.identity")); v.Exists() {
    if str, err := v.String(); err == nil {
        metadata.Identity = str
    }
}

// Extract labels first
if labelsVal := concreteModule.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
    // ... iterate and populate metadata.Labels
}

// Release identity from labels (set by catalog schema transformer)
if rid, ok := metadata.Labels["module-release.opmodel.dev/uuid"]; ok {
    metadata.ReleaseIdentity = rid
}
```

**Alternative considered:** Compute the release identity in Go using UUID v5. This was rejected because:

- It duplicates logic that exists in CUE
- Requires maintaining the OPM namespace UUID constant in sync across languages
- Adds a dependency on the `github.com/google/uuid` package
- Creates risk of computation drift if the input format ever changes

The OPM namespace UUID constant (`c1cbe76d-5687-5a47-bfe6-83b081b15413`) is retained in the Go `identity` package for documentation purposes, but Go does not use it for computation.

## Risks / Trade-offs

**[Risk] Dual-query discovery doubles API server calls per resource type.**
→ Mitigation: Negligible impact. Each query is a server-side label-indexed lookup. The existing full-scan already queries every API resource type. Two queries per type instead of one is immeasurable in practice.

**[Risk] Pre-identity resources won't have release-id labels after re-apply.**
→ Mitigation: They will. Re-running `mod apply` with the new CLI stamps the identity labels on existing resources via server-side apply. The labels are additive.

**[Risk] `--name` is no longer `MarkFlagRequired` — users might forget both flags.**
→ Mitigation: Clear validation error message. `--namespace` remains required since we always scope to a namespace.

**[Risk] Old modules without catalog schema updates won't have release-id in labels.**
→ Mitigation: When the label is absent, `ReleaseIdentity` is empty string, and the identity label is not injected (Decision 2). Discovery falls back to name+namespace selector. Full backwards compatibility.

## Migration Plan

1. **Catalog change lands first** — publishes updated schemas with `metadata.identity` on `#Module` and release-id label injection
2. **CLI change lands** — reads identity from CUE output, stamps labels, enables dual-discovery
3. **Existing deployments:** `opm mod apply` re-run stamps identity labels on existing resources via server-side apply. No special migration command needed.
4. **Rollback:** Remove the identity label injection. Discovery falls back to name+namespace only (existing code path). No data loss.

## Open Questions

1. **Should `opm mod list` (future TODO) use the release-id label for discovery?** — Likely yes, but that's a separate change. The label infrastructure built here enables it.
2. **Should the release-id be shown in `mod apply` output?** — Probably in verbose mode only, to avoid cluttering normal output.
