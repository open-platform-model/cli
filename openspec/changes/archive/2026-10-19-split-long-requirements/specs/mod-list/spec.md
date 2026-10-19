## MODIFIED Requirements

### Requirement: List metadata extraction does not depend on inventory change history

`opm instance list` SHALL extract display metadata from each persisted instance inventory record: instance name from `instanceMetadata.name`, module name from `moduleMetadata.name`, version from `moduleMetadata.version`, instance ID from `instanceMetadata.uuid`, last applied time from `instanceMetadata.lastTransitionTime`, and age computed from `lastTransitionTime`.

#### Scenario: Display metadata sourced from persisted inventory

- **WHEN** the user runs `opm instance list -n production`
- **AND** a persisted instance inventory record exists
- **THEN** each row SHALL source instance name from `instanceMetadata.name`, module name/version from `moduleMetadata`, and owner from top-level `createdBy` (defaulting to `cli` for legacy inventories)

## ADDED Requirements

### Requirement: Owner display metadata comes from the top-level `createdBy` field

Owner display metadata SHALL come from the top-level `createdBy` field, defaulting to `cli` when that field is omitted for legacy inventories.
(Requirement: List metadata extraction does not depend on inventory change history)

#### Scenario: Owner defaults to cli for legacy inventories

- **WHEN** a persisted instance inventory record omits the top-level `createdBy` field (a legacy inventory)
- **THEN** owner display metadata SHALL come from the top-level `createdBy` field, defaulting to `cli`

### Requirement: `opm instance list` does not require inventory change-history metadata

`opm instance list` SHALL NOT require inventory change-history metadata such as latest change source version, raw values, or per-change timestamps. <!-- Was: release inventory record, releaseMetadata (enhancement 0002: hard rename with no alias window; instance files are `instance.cue`) -->
(Requirement: List metadata extraction does not depend on inventory change history)

#### Scenario: List works without change-history metadata

- **WHEN** `opm instance list` reads a persisted instance inventory record that holds no inventory change-history metadata such as latest change source version, raw values, or per-change timestamps
- **THEN** `opm instance list` SHALL NOT require that metadata
