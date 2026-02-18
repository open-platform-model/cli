## Context

OPM currently applies resources via server-side apply with OPM labels (`app.kubernetes.io/managed-by`, `module-release.opmodel.dev/name`, etc.) but stores no record of which resources were applied. Discovery relies on scanning all API types with label selectors. This is the source of orphaned resources on rename, no pruning, slow discovery, and false positives (#16 Endpoints in delete).

The existing codebase has:

- `internal/build/` — render pipeline producing `[]*build.Resource` with `GVK()`, `Name()`, `Namespace()`, `Component` accessors
- `internal/kubernetes/` — apply (SSA), delete, diff, status, discovery via label selectors
- `internal/cmd/` — cobra commands orchestrating pipeline + kubernetes operations
- `pkg/weights/` — resource ordering by GVK weight

The inventory Secret will be one additional Kubernetes object per release, stored in the same namespace as the release resources. RFC-0001 provides the full specification.

## Goals / Non-Goals

**Goals:**

- Record the exact set of applied resources per release in a K8s Secret
- Enable automatic pruning of stale resources after apply
- Provide fast, precise resource discovery for diff/delete/status (N targeted GETs instead of scanning all API types)
- Maintain full backward compatibility — all commands fall back to label-scan when no inventory exists
- Support change history with configurable retention for rollback investigation
- Prevent destructive operations from component renames via safety checks

**Non-Goals:**

- CRD-based inventory (future migration path exists via `kind: ModuleRelease` / `apiVersion: core.opmodel.dev/v1alpha1` in metadata, but this implementation uses a plain Secret)
- Rollback command (`opm mod rollback`) — history is stored for investigation, not automated rollback
- Concurrent apply coordination — optimistic concurrency on the inventory Secret detects conflicts but does not resolve them
- Solving partial apply to cluster (#17) — write-nothing-on-failure keeps the inventory consistent, but resources already applied to the cluster before a failure remain

## Decisions

### 1. New `internal/inventory/` package

**Decision:** All inventory logic (types, serialization, digest, change ID, history, CRUD) lives in a new `internal/inventory/` package.

**Why over splitting across packages:** Keeps inventory as a self-contained domain. The `internal/kubernetes/` package stays focused on raw K8s operations (apply, delete, discover). The inventory package imports `internal/build` (for `*build.Resource`) and `internal/kubernetes` (for `*Client` in CRUD), but neither imports inventory — no import cycle risk.

**Why over putting everything in `internal/kubernetes/`:** The inventory has its own data model, serialization format, and business logic (stale set, change ID, history pruning) that is conceptually separate from K8s API operations. Mixing them would bloat the kubernetes package.

### 2. Typed `corev1.Secret` for inventory CRUD

**Decision:** Use `k8s.io/api/core/v1.Secret` (typed client) for inventory read/write, not `unstructured.Unstructured`.

**Why:** The inventory Secret has a fixed, known schema. Typed access gives compile-time safety for field access, cleaner serialization via `StringData`/`Data` fields, and is idiomatic for K8s controllers working with well-known types. The existing `kubernetes.Client` struct already carries a `Clientset` field (`kubernetes.Interface`) which provides `CoreV1().Secrets()`.

**Why not unstructured:** Would be consistent with the existing discovery/apply code that uses `dynamic.Interface`, but adds unnecessary type assertions and map access for a resource we fully control. The unstructured approach is appropriate for user-defined resources where the schema is unknown.

**Trade-off:** Adds `k8s.io/api` as a direct dependency (already indirect via client-go).

### 3. `opmodel.dev/component: inventory` label (new label key)

**Decision:** The inventory Secret carries `opmodel.dev/component: inventory` as a distinguishing label. This is a **new label key** distinct from `component.opmodel.dev/name` (`LabelComponentName`).

**Why two label keys:** `component.opmodel.dev/name` is set by CUE transformers on application resources and carries the component name (e.g., `"app"`, `"cache"`). `opmodel.dev/component` is an OPM infrastructure label that categorizes the type of OPM-managed object. The inventory Secret is not a component output — it is OPM infrastructure. Using the same key would conflate two different concepts.

**Constants in `discovery.go`:**

```
LabelComponent           = "opmodel.dev/component"
labelComponentInventory  = "inventory"
```

### 4. Deterministic manifest digest with 5-key total ordering

**Decision:** Resources are sorted by (weight, group, kind, namespace, name) before serialization and hashing. Go's `json.Marshal` handles map key ordering. SHA256 of the concatenated JSON with newline separators.

**Why 5 keys:** Weight alone is non-deterministic — multiple resources can share a weight (e.g., Deployment and StatefulSet both at 100). Adding group, kind, namespace, name guarantees a unique position since no two resources in a valid deployment share GVK + namespace + name.

**Why also fix `pipeline.go` sort:** The pipeline currently uses weight-only sorting. Upgrading to 5-key ordering makes `opm mod build` output deterministic as a free side benefit. The sort function is shared between digest computation and pipeline output.

### 5. Change ID = SHA1(path + version + values + digest)

**Decision:** Four inputs to the change ID, not just the manifest digest.

**Why all four:** The manifest digest alone would suffice for detecting content changes, but including `path` and `version` ensures module upgrades always appear as new changes — even if the rendered output is identical (e.g., a no-op version bump). Including `values` ensures that explicitly setting a default is recorded as a distinct change.

**Format:** `change-sha1-<8hex>` (first 8 hex chars of SHA1). Short enough for Secret keys, collision-resistant for practical history sizes (10 entries).

### 6. Values stored as resolved CUE string

**Decision:** The `values` field in a `ChangeEntry` stores the resolved/unified CUE value after the build pipeline unifies all values sources, serialized as a CUE-formatted string.

**Why not raw file content:** Raw file content doesn't capture the effect of `--values` flag overrides. The resolved value is what was actually applied — this is what matters for change detection and debugging. Consistent with how Timoni stores values.

### 7. Full PUT semantics for inventory writes

**Decision:** Read inventory -> modify in memory -> write entire Secret back. Not server-side apply or JSON patch.

**Why:** The inventory Secret is fully owned by OPM. There are no other writers. Full PUT is simple, atomic, and avoids the complexity of merge strategies. Optimistic concurrency via `resourceVersion` from the read detects any concurrent modification.

**Why not SSA:** Server-side apply is designed for shared ownership. The inventory has a single owner (the OPM CLI). SSA adds field management overhead for no benefit.

### 8. Create-then-prune ordering

**Decision:** New resources are applied before stale resources are deleted.

**Why:** This briefly doubles resource count but ensures the new state is running before the old state is removed. If pruning were done first, there would be a window where the application is missing resources. For stateful workloads (PVCs, StatefulSets), this ordering prevents data loss.

**Trade-off:** Transient resource doubling. For most modules (3-20 resources), this is negligible.

### 9. Write-nothing-on-failure

**Decision:** If any resource fails to apply, the inventory is NOT updated and stale resources are NOT pruned.

**Why:** This keeps the inventory consistent with the cluster state. On the next retry, the same stale set will be computed and the same apply will be attempted. The inventory only advances when the full apply succeeds.

**Trade-off:** Resources that were successfully applied before the failure remain on the cluster but are not recorded. On retry, they will be re-applied (idempotent via SSA) and then recorded.

### 10. Graceful degradation via label-scan fallback

**Decision:** All commands (diff, delete, status) first attempt inventory-based discovery. If no inventory Secret is found, they fall back to `DiscoverResources()` label-scan with a debug log message.

**Why:** Ensures backward compatibility with modules applied before the inventory feature. Users upgrading OPM don't need to re-apply all modules. On their next `opm mod apply`, the inventory Secret is created and subsequent operations use it.

### 11. `InventorySecret.resourceVersion` as unexported field

**Decision:** The `resourceVersion` from the K8s Secret is stored as an unexported field on `InventorySecret`, accessible via a `ResourceVersion() string` method.

**Why:** Prevents callers from setting it directly. Only `UnmarshalFromSecret` populates it (from the actual K8s object). `WriteInventory` reads it for optimistic concurrency. This makes it impossible to accidentally write a stale or fabricated version.

## Risks / Trade-offs

**[etcd size limit]** The inventory Secret stores change history. 10 changes at ~2-5KB each = ~20-50KB, well within etcd's 1MB Secret limit. Modules with hundreds of resources could approach the limit at high `--max-history` values. **Mitigation:** Default `--max-history=10` keeps size manageable. History pruning removes oldest entries when the cap is exceeded.

**[Inventory-cluster divergence]** If a user manually deletes resources from the cluster without using OPM, the inventory will reference resources that no longer exist. **Mitigation:** Pruning treats 404 as success (resource already gone). Status and diff detect missing resources via targeted GET failures.

**[First-time apply with existing resources]** If resources matching the render output already exist (applied by another tool or a previous OPM version without inventory), the first inventory write will record them. However, any resources NOT in the current render will be invisible to future pruning. **Mitigation:** Pre-apply existence check on first install warns about untracked resources. Label-scan fallback ensures delete/status still find all labeled resources.

**[Concurrent apply]** Two `opm mod apply` runs against the same release will conflict on the inventory Secret write (optimistic concurrency failure). **Mitigation:** Clear error message with retry guidance. This is the correct behavior — concurrent applies to the same release are inherently unsafe.

**[Secret name length]** K8s Secret names are limited to 253 characters. `opm.<name>.<releaseID>` where releaseID is a UUID (36 chars) and name could be long. **Mitigation:** Module names are typically short (3-30 chars). The format `opm.` + name + `.` + UUID = ~45 chars minimum, well within limits.
