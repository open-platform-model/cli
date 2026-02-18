## MODIFIED Requirements

### Requirement: Diff categorizes resources into three states

The command SHALL categorize each resource into one of three states: modified (exists both locally and on cluster with differences), added (exists locally but not on cluster), or orphaned (exists on cluster with OPM labels but not in local render). When an inventory Secret exists, orphan detection SHALL use inventory set-difference (entries in previous inventory not in current render) with targeted GETs to verify live state. When no inventory exists, orphan detection SHALL fall back to label-based discovery via `DiscoverResources()`.

#### Scenario: Resource exists on cluster but not in local render

- **WHEN** a resource was previously applied (has OPM labels) but is no longer produced by the local render
- **THEN** `opm mod diff` SHALL display the resource as orphaned with a message indicating it will be removed on next apply

#### Scenario: New resource in local render

- **WHEN** a resource is produced by the local render but does not exist on the cluster
- **THEN** `opm mod diff` SHALL display the resource as a new addition

#### Scenario: Orphan detection with inventory

- **WHEN** an inventory Secret exists for the release
- **THEN** orphans SHALL be computed as inventory entries not present in the current rendered set
- **AND** each orphan SHALL be verified via a targeted GET (missing resources are excluded)

#### Scenario: Orphan detection without inventory (fallback)

- **WHEN** no inventory Secret exists for the release
- **THEN** orphan detection SHALL fall back to `DiscoverResources()` label-scan
- **AND** a debug log message SHALL indicate "No inventory found, falling back to label-based discovery"
