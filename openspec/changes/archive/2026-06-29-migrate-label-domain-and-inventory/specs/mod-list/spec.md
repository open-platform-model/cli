## MODIFIED Requirements

<!-- enhancement 0002 D6/D9 — X3-deferred to X4 per D-X3.6 (single capability owner). Restates the requirements whose normative text changes under the instance-noun rename: instance discovery and display metadata. Unchanged table-formatting/sorting behavior rides the archive spec-sync. -->

### Requirement: List command discovers instances via persisted ownership inventory

The `opm mod list` command SHALL discover all deployed module instances by listing persisted instance inventory records in the target namespace. It SHALL use the `ListInventories` function from the inventory package. It MUST NOT require module source, re-rendering, knowledge of specific instance names, or inventory change-history fields to identify the current owned resource set for an instance. <!-- Was: "discovers releases", "release inventory records", "release names" (0002 D9) -->

#### Scenario: List discovers via ListInventories

- **WHEN** the user runs `opm mod list -n production`
- **THEN** the command SHALL list deployed module instances via `ListInventories` for `production`
- **AND** SHALL NOT require module source or re-rendering

### Requirement: List metadata extraction does not depend on inventory change history

The command SHALL extract display metadata from each persisted instance inventory record: instance name from `instanceMetadata.name`, module name from `moduleMetadata.name`, version from `moduleMetadata.version`, instance ID from `instanceMetadata.uuid`, last applied time from `instanceMetadata.lastTransitionTime`, and age computed from `lastTransitionTime`. Owner display metadata SHALL come from the top-level `createdBy` field, defaulting to `cli` when that field is omitted for legacy inventories. The command SHALL NOT require inventory change-history metadata such as latest change source version, raw values, or per-change timestamps. <!-- Was: release inventory record, releaseMetadata (0002 D8/D9) -->

#### Scenario: Display metadata sourced from persisted inventory

- **WHEN** the user runs `opm mod list -n production`
- **AND** a persisted instance inventory record exists
- **THEN** each row SHALL source instance name from `instanceMetadata.name`, module name/version from `moduleMetadata`, and owner from top-level `createdBy` (defaulting to `cli` for legacy inventories)
