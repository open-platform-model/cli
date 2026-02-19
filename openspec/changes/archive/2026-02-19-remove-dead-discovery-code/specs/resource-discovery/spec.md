## REMOVED Requirements

### Requirement: Label selector construction

**Reason**: Label-scan discovery via `DiscoverResources()` has been removed. All commands now use inventory-based discovery exclusively, which performs targeted GETs per inventory entry rather than API-wide label scans.

**Migration**: No user action required. Commands (`status`, `delete`, `diff`) continue to work with `--release-name` and `--release-id` flags, but now resolve resources through the inventory Secret rather than cluster-wide label queries.

---

### Requirement: Inventory Secret excluded from workload discovery

**Reason**: The `DiscoverResources()` function that required this exclusion filter has been removed.

**Migration**: N/A - inventory-based discovery (`DiscoverResourcesFromInventory`) inherently never includes the inventory Secret in results, as it only fetches resources explicitly listed in the inventory entries.

---

### Requirement: Inventory-based resource discovery

**Reason**: This requirement is being moved to the `release-inventory` spec where it belongs conceptually. The `resource-discovery` spec originally covered both label-scan and inventory-based approaches; with label-scan removed, inventory discovery is better documented as part of the inventory system itself.

**Migration**: No code changes. The implementation remains in `internal/inventory/discover.go` and is unchanged. This is purely a documentation reorganization.
