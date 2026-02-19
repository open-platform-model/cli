## Context

The inventory Secret (`opm.<releaseName>.<releaseID>`) is written by `opm mod apply` and tracks the exact set of resources belonging to a release. It has existed since the inventory system was introduced and is always present for any OPM-managed release.

Currently, `mod status`, `mod delete`, and `mod diff` use the inventory as an optimization, falling back to a cluster-wide label-scan when the inventory is missing or when only `--release-name` is provided (for `status`). This fallback is incorrect:

- Label-scans pick up Kubernetes-generated children (e.g., `Endpoints`, `EndpointSlice`, `ReplicaSet`, `Pod`) that carry inherited labels but were never applied by OPM.
- Label-scans scan every API type on the server — slow and noisy.
- A release without an inventory Secret is not a valid OPM release. Providing partial/incorrect results is worse than a clear "not found" error.

**Key files:**
- `internal/cmd/mod/status.go`, `delete.go`, `diff.go` — command layer
- `internal/kubernetes/status.go`, `delete.go`, `diff.go` — business logic layer
- `internal/inventory/crud.go` — `GetInventory`, `FindInventoryByReleaseName`, `DiscoverResourcesFromInventory`
- `internal/cmdutil/flags.go` — `ReleaseSelectorFlags`

**Namespace resolution** already works correctly via `ResolveKubernetes`: flag → `OPM_NAMESPACE` env → config → `"default"`. The `--namespace` flag is currently documented as "required" but is not validated as such in code — it was simply a documentation issue. Removing the "required" language makes the existing behavior explicit.

## Goals / Non-Goals

**Goals:**
- Inventory is the sole resource discovery mechanism for `status`, `delete`, and `diff`.
- No inventory → `"release '<name>' not found in namespace '<ns>'"` error for `status` and `delete`.
- No inventory for `diff` → treat as "nothing deployed" (all rendered resources appear as `[new resource]`). This is the correct first-time-diff behavior.
- `--namespace` flag described as optional with config default.
- Integration tests updated to reflect inventory-only behavior.

**Non-Goals:**
- No new flags introduced.
- `DiscoverResources` is not deleted (still used by `tests/integration/deploy/main.go` for label verification).
- No change to `mod apply`.
- No change to the inventory data model or Secret format.

## Decisions

### Decision: No inventory = error for status/delete; no orphans for diff

**Alternatives considered:**
1. Keep label-scan as fallback — rejected because it returns incorrect results (inherited-label children, no "Missing" tracking).
2. Warn and return empty results — rejected because silently showing nothing is worse than a clear error.
3. Error for all three commands — rejected for `diff` because "no inventory" is a valid and common first-time state; all-added output is correct and useful.

**Chosen:** `status` and `delete` return `noResourcesFoundError` (which renders as `"release '<name>' not found in namespace '<ns>'"`) when no inventory exists. `diff` treats `nil` inventory as an empty live set (no orphans, all rendered resources are additions).

### Decision: `--release-name` path for status uses `FindInventoryByReleaseName`

`mod delete` already uses `FindInventoryByReleaseName` for the name-only path. `mod status` was inconsistent — it only did inventory-first when `--release-id` was provided. The fix makes `status` behave identically to `delete` for inventory lookup.

**Why not compute the UUID from the name?** The UUID is deterministic only if you know the module's FQN. `status` and `delete` don't have the module source — they are source-free commands. `FindInventoryByReleaseName` uses a label selector on the inventory Secret, which is the correct approach.

### Decision: Remove label-scan branches from the `kubernetes/` layer

The `InventoryLive` field on `StatusOptions` and `DeleteOptions`, and the `inventoryLive` parameter in `findOrphans`, change from "optional optimization" to "required input". The nil-check branches that called `DiscoverResources` are removed.

This makes the `kubernetes/` layer simpler and ensures label-scan cannot be accidentally re-introduced at the call site.

### Decision: Error message format

Use `"release '<name>' not found in namespace '<ns>'"` — clean, user-facing. Reuses the existing `noResourcesFoundError` type with a message tweak in the command layer. `--ignore-not-found` continues to suppress this error for CI teardown scripts.

### Decision: Scenario 6.6 (label-scan fallback integration test) removed

This scenario tested a behavior that no longer exists. Keeping it would require either deleting the assertion or making it test something other than what the scenario name implies. Removal is cleaner.

The deploy integration test (`tests/integration/deploy/main.go`) verifies that OPM labels are applied correctly. It currently uses `DiscoverResources` as a convenient way to retrieve labeled resources. This test should be updated to use `inventory.FindInventoryByReleaseName` + `inventory.DiscoverResourcesFromInventory` to stay consistent with the new invariant.

## Risks / Trade-offs

- **[Risk] Existing releases without an inventory Secret** → These cannot exist in a correctly-operated OPM environment. Any release applied with the current CLI has an inventory Secret. The risk is effectively zero for forward-looking usage. If a user somehow has a pre-inventory deployment, the clear error message guides them to re-apply.
- **[Risk] `DiscoverResources` remains exported but unused in command flow** → Mitigation: leave it exported for the deploy integration test. Mark for removal in a future cleanup pass if the test is refactored.
- **[Trade-off] Simpler code, stricter contract** → Removing the fallback makes the code simpler and the mental model cleaner. The cost is losing the ability to operate on ad-hoc labeled resources that have no inventory. This is the right trade-off for an inventory-first system.

## Migration Plan

No migration required. The change takes effect on next CLI build. Any `opm mod status` / `opm mod delete` call on a release that was applied with the current CLI will have an inventory Secret and will work correctly. First-time `opm mod diff` continues to show all resources as additions.
