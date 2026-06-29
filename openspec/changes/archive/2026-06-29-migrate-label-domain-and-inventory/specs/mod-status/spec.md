## MODIFIED Requirements

<!-- enhancement 0002 D6/D9/D10 — X3-deferred to X4 per D-X3.6 (single capability owner). Restates the requirements whose normative text changes under the rename: the inventory-discovery selector flags (--instance-name/--instance-id) and the "instance inventory record" noun. Per-scenario flag-string swaps in unrestated requirements ride the archive spec-sync / hygiene pass. -->

### Requirement: Status discovers resources via ownership inventory

The `opm mod status` command SHALL read the persisted instance inventory record for the instance to discover its tracked resources. If `--instance-id` is provided, it SHALL use `inventory.GetInventory` (direct GET by name, with UUID label fallback). If only `--instance-name` is provided, it SHALL use `inventory.FindInventoryByInstanceName` (inventory-record lookup by instance-name label). Once the inventory is found, it SHALL perform one targeted GET per tracked entry via `inventory.DiscoverResourcesFromInventory`. It MUST NOT require module source or re-rendering. It MUST NOT use a cluster-wide label-scan to discover workload resources. <!-- Was: release inventory record, --release-id/--release-name (0002 D8/D-X4.2) -->

#### Scenario: Discover by instance ID

- **WHEN** the user runs `opm mod status --instance-id <uuid> -n production`
- **THEN** the command SHALL resolve the instance inventory record via `inventory.GetInventory`
- **AND** SHALL perform one targeted GET per tracked entry

#### Scenario: Discover by instance name

- **WHEN** the user runs `opm mod status --instance-name my-app -n production`
- **THEN** the command SHALL resolve the instance inventory record by instance-name label
- **AND** SHALL NOT require module source or re-rendering

### Requirement: Status header does not depend on inventory change history

The status output SHALL display a metadata header above the resource table containing instance name, namespace, aggregate health status, and a resource summary. Module version and ownership metadata SHALL come from the persisted instance inventory record (`instanceMetadata`, `moduleMetadata`, `createdBy`) when present. The command SHALL NOT require inventory change-history metadata such as source version, raw values, or per-change timestamps, and it MUST NOT require module source or re-rendering. <!-- Was: release name, release inventory record, releaseMetadata (0002 D8/D9) -->

#### Scenario: Header sourced from persisted inventory

- **WHEN** the user runs `opm mod status --instance-name my-app -n production`
- **AND** a persisted instance inventory record exists
- **THEN** the header SHALL display the instance name, namespace, aggregate health, and a resource summary sourced from `instanceMetadata`/`moduleMetadata`/`createdBy`
