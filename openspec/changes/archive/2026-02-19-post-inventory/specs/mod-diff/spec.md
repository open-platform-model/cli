## MODIFIED Requirements

### Requirement: Diff categorizes resources into three states

The command SHALL categorize each resource into one of three states: modified (exists both locally and on cluster with differences), added (exists locally but not on cluster), or orphaned (exists on cluster per inventory but not in local render). When an inventory Secret exists, orphan detection SHALL use inventory set-difference: entries in the previous inventory not present in the current render, verified with targeted GETs. When no inventory exists, orphan detection SHALL return an empty set — all rendered resources SHALL be shown as additions. The command MUST NOT fall back to a cluster-wide label-scan at any point.

#### Scenario: Module not yet deployed shows all as additions

- **WHEN** a module has never been applied to the cluster
- **AND** therefore no inventory Secret exists
- **THEN** `opm mod diff` SHALL show every rendered resource as an addition (`[new resource]`)
- **AND** no orphans SHALL be reported

#### Scenario: Orphan detection with inventory

- **WHEN** an inventory Secret exists for the release
- **THEN** orphans SHALL be computed as inventory entries not present in the current rendered set
- **AND** each orphan candidate SHALL be verified via a targeted GET (missing resources on cluster are excluded from orphan list)

## REMOVED Requirements

### Requirement: Orphan detection without inventory (fallback)

**Reason**: Label-scan returns incorrect results (inherited-label children), is slow, and violates the invariant that inventory is the authoritative record. "No inventory" correctly means "nothing previously deployed" — returning no orphans is accurate.

**Migration**: No action required. The behavior when no inventory exists (all resources as additions) is correct and unchanged for first-time diffs.
