## REMOVED Requirements

### Requirement: Discovery uses preferred API resources

**Reason**: The `DiscoverResources()` function that enumerated API resources has been removed. Inventory-based discovery performs targeted GETs and does not need to enumerate all API types on the cluster.

**Migration**: N/A - internal implementation detail. Commands continue to work without change. No cluster-wide API enumeration occurs.

---

### Requirement: ExcludeOwned option filters controller-managed resources

**Reason**: The `ExcludeOwned` option was part of `DiscoveryOptions` for the now-removed `DiscoverResources()` function. Inventory-based discovery inherently doesn't include controller-managed children (like auto-created Endpoints or EndpointSlices) because OPM never applies them and therefore never adds them to the inventory.

**Migration**: No user action required. Delete and diff commands continue to exclude controller-managed resources automatically because those resources are not tracked in the inventory Secret.

---

### Requirement: Delete command excludes owned resources

**Reason**: This requirement is still satisfied, but the mechanism has changed. Instead of using `ExcludeOwned: true` during label-scan discovery, delete now uses inventory-based discovery which only includes resources that OPM explicitly applied.

**Migration**: No user action required. `opm mod delete` continues to delete only OPM-managed resources and ignores auto-created children like Endpoints and EndpointSlice.

---

### Requirement: Diff command excludes owned resources

**Reason**: Same as delete - the requirement is still satisfied via inventory-based discovery rather than the `ExcludeOwned` filter.

**Migration**: No user action required. `opm mod diff` continues to show diffs only for resources OPM manages, not for auto-created children.
