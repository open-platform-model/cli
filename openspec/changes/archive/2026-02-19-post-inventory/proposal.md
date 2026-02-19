## Why

`opm mod status`, `delete`, and `diff` currently fall back to a label-scan across all API types when no inventory Secret is found or when only `--release-name` is provided. This produces incorrect results (noise from Kubernetes-generated children like `Endpoints`/`EndpointSlice`), is slow, and violates the invariant that the inventory Secret is the authoritative record of a release. The fix closes this gap now that the inventory system is stable and all releases created by OPM will have one.

## What Changes

- **`mod status`**: Always uses inventory-first discovery for both `--release-name` and `--release-id`. No inventory → `"release not found"` error. **Removes label-scan fallback.**
- **`mod delete`**: No inventory → `"release not found"` error. **Removes label-scan fallback.**
- **`mod diff`**: No inventory → treated as "nothing deployed" (all resources shown as `[new resource]`). **Removes label-scan fallback.** This is correct behavior for a first-time diff.
- **`--namespace` flag**: No longer required for `status` and `delete`. Falls back to config default → `OPM_NAMESPACE` env → `"default"` (already handled by `ResolveKubernetes`). Flag description updated.
- **`kubernetes.GetReleaseStatus`**, **`kubernetes.Delete`**, **`kubernetes.findOrphans`**: Remove label-scan branches. `InventoryLive` is now the sole resource source.
- **Integration test**: Scenario 6.6 (label-scan fallback) removed. Deploy integration test updated to use inventory-first discovery.
- **Docstrings**: Remove "discovered via OPM labels" language throughout.

## Capabilities

### New Capabilities

None. This is a behavior correction, not a new capability.

### Modified Capabilities

- `mod-status`: Discovery changes from label-scan-first to inventory-only. Error behavior when release not found changes from "no resources found" to "release not found".
- `mod-delete`: Same as above.
- `mod-diff`: Orphan detection changes from label-scan fallback to empty-orphan-set when no inventory exists.

## Impact

- **`internal/cmd/mod/status.go`**: Inventory lookup for both selector paths; error on missing inventory.
- **`internal/cmd/mod/delete.go`**: Error on missing inventory; remove fallback flow.
- **`internal/cmd/mod/diff.go`**: Remove fallback debug messages; nil inventory = no orphans.
- **`internal/kubernetes/status.go`**: Remove `DiscoverResources` call from `GetReleaseStatus`.
- **`internal/kubernetes/delete.go`**: Remove `DiscoverResources` call from `Delete`.
- **`internal/kubernetes/diff.go`**: Remove `DiscoverResources` call from `findOrphans`.
- **`internal/cmdutil/flags.go`**: Update `--namespace` description.
- **`tests/integration/inventory-ops/main.go`**: Remove scenario 6.6.
- **`tests/integration/deploy/main.go`**: Update label-scan call to inventory-first.

SemVer: **PATCH** — behavior correction. Users with a valid OPM-managed release (which always has an inventory Secret) see no change. Users relying on the label-scan fallback for releases without an inventory were already getting incorrect results.
