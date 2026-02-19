## REMOVED Requirements

### Requirement: Release-id based resource discovery

**Reason**: The `DiscoverResources()` function and its label-scan mechanism have been removed. All discovery now uses inventory-based targeted GETs.

**Migration**: No user action required. The `--release-id` flag continues to work on `mod delete` and `mod status` commands, but now uses inventory lookup (`inventory.GetInventory()` with fallback to label-scan for the inventory Secret only) rather than cluster-wide resource scanning.

---

### Requirement: Dual-strategy discovery with union

**Reason**: Label-scan discovery has been removed, eliminating the need for dual-strategy (release-id + name+namespace union) discovery.

**Migration**: Commands now use inventory-first resolution: lookup inventory Secret by release-id or release-name, then perform targeted GETs for resources listed in that inventory. This is more efficient (N GETs vs hundreds of API calls) and more accurate (only shows what OPM actually applied).

---

### Requirement: Release-id selector builder

**Reason**: The `BuildReleaseIDSelector()` function has been removed along with the label-scan discovery system.

**Migration**: N/A - internal implementation detail. The inventory system uses label selectors only to find the inventory Secret itself, not to scan all resource types.

---

### Requirement: Delete with --release-id flag

**Reason**: This requirement is partially retained but implementation has changed. The `--release-id` flag still exists and works, but no longer uses `DiscoverResources()`.

**Migration**: No user-facing changes. `opm mod delete --release-id <uuid>` continues to work. Internally, it now calls `inventory.GetInventory()` which uses the release-id to find the inventory Secret, then discovers resources from that inventory.

---

### Requirement: Status and diff use dual-strategy discovery

**Reason**: Dual-strategy discovery has been removed. Commands now use inventory-only discovery.

**Migration**: No user action required. `mod status` and `mod diff` continue to function. Internally, they read the inventory Secret (using release-id if available in rendered metadata) and perform targeted GETs for tracked resources.

---

### Requirement: Identity display in status output

**Reason**: This requirement is being retained and moved to the `mod-status` spec where it belongs conceptually. The `identity-based-discovery` spec was focused on the now-removed label-scan mechanism; identity display is a status command concern.

**Migration**: No code changes. Module ID and Release ID continue to be displayed in `mod status` output. This is purely a documentation reorganization.
