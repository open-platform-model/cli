## Context

The OPM CLI uses `k8s.io/client-go` v0.35.0 to interact with Kubernetes clusters. Two issues affect user experience:

1. **Uncontrolled warning output**: When `ServerGroupsAndResources()` enumerates API resources, the API server sends HTTP Warning headers for deprecated resources (e.g., `v1/ComponentStatus`, `v1/Endpoints`). The default `rest.WarningHandler` logs these via klog, bypassing charmbracelet/log formatting.

2. **Delete race condition**: Discovery finds resources with OPM labels, including auto-managed resources like `Endpoints` and `EndpointSlice` that inherit labels from their parent `Service`. When deletion proceeds, the parent is deleted first, K8s garbage-collects the children, and OPM gets 404 errors trying to delete already-gone resources.

**Current flow:**
```
ServerGroupsAndResources() → enumerates ALL API versions
  → API server sends Warning headers
  → klog outputs raw "warnings.go:107" lines
  
DiscoverResources() → finds all labeled resources including Endpoints
Delete() → attempts to delete everything → 404 on GC'd children
```

## Goals / Non-Goals

**Goals:**
- Route K8s API warnings through charmbracelet/log for consistent formatting
- Allow users to configure warning behavior (warn/debug/suppress) via config.cue
- Reduce API calls by using `ServerPreferredResources()` instead of `ServerGroupsAndResources()`
- Prevent 404 errors by filtering out controller-managed resources during delete/diff

**Non-Goals:**
- Adding CLI flags for warning suppression (config-only for simplicity)
- Changing how `status` command handles owned resources (may show them for visibility)
- State tracking for applied resources (that's a larger future enhancement)

## Decisions

### Decision 1: Custom WarningHandler routing through charmbracelet/log

**Choice**: Implement `opmWarningHandler` that satisfies `rest.WarningHandler` interface and routes to `output.Warn()` or `output.Debug()` based on config.

**Rationale**: 
- Maintains consistent log formatting across the CLI
- Leverages existing verbose mode (`--verbose` enables debug output)
- No new dependencies

**Alternatives considered**:
- `rest.NoWarnings{}` — too blunt, loses useful deprecation info
- Custom klog hook — fragile, klog internals could change

### Decision 2: Three-level config for warning behavior

**Choice**: `log.kubernetes.apiWarnings: "warn" | "debug" | "suppress"`

**Rationale**:
- `"warn"` (default): Show warnings through normal log channel
- `"debug"`: Only visible with `--verbose`, reduces noise for most users
- `"suppress"`: Drop entirely for CI/automation where warnings are noise

**Alternatives considered**:
- Boolean `suppressWarnings` — too coarse, no middle ground
- Pattern-based filtering — too complex for the benefit

### Decision 3: Use ServerPreferredResources() for discovery

**Choice**: Replace `ServerGroupsAndResources()` with `ServerPreferredResources()`.

**Rationale**:
- Returns only the preferred version for each resource type
- Fewer API calls, less warning surface area
- Matches what `kubectl api-resources` uses
- Still handles `IsGroupDiscoveryFailedError` the same way

**Alternatives considered**:
- Keep current API — more warning noise, no benefit
- Manual version filtering — complex and error-prone

### Decision 4: ExcludeOwned option for discovery filtering

**Choice**: Add `ExcludeOwned bool` to `DiscoveryOptions`. When true, skip resources where `len(ownerReferences) > 0`.

**Rationale**:
- Correct by design: auto-managed resources ALWAYS have ownerReferences
- Generalizes to all controller-managed children (Endpoints, EndpointSlice, ReplicaSet, Pod, etc.)
- Zero maintenance burden — no hardcoded skip lists
- Simple implementation — check after List, before append

**Alternatives considered**:
- Hardcoded kind skip list — brittle, needs ongoing maintenance
- Applied-resource tracking — correct but requires significant state management infrastructure

### Decision 5: Config schema structure

**Choice**: `log.kubernetes` is non-optional (has default), `log` remains optional.

```cue
log?: {
    timestamps: bool | *true
    kubernetes: {
        apiWarnings: "warn" | "debug" | "suppress" | *"warn"
    }
}
```

**Rationale**:
- `log?` already optional in current schema — users can omit entire section
- `kubernetes` within `log` always has a value because `apiWarnings` has a default
- CUE fills in defaults automatically when user omits fields

## Risks / Trade-offs

**[Risk] ServerPreferredResources() might miss resources in non-preferred versions**  
→ Mitigation: This matches kubectl behavior. Users with resources in old API versions likely have bigger problems. The ownerReference filter is the real safety net.

**[Risk] ExcludeOwned could skip user-created resources with ownerReferences**  
→ Mitigation: Unlikely in OPM context — OPM creates top-level resources, not children of other resources. If someone manually sets ownerReferences on OPM-managed resources, they're explicitly opting into cascade behavior.

**[Risk] Config migration for existing users**  
→ Mitigation: New field has a default (`"warn"`), so existing configs work unchanged. No breaking change.

**[Trade-off] `status` command will show different resources than `delete` previews**  
→ Accepted: `status` showing owned resources (like Pods) could be useful for health visibility. Can revisit if confusing.
