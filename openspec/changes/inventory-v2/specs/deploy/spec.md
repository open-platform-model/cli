## MODIFIED Requirements

### Requirement: mod apply uses ownership inventory for pruning

`opm mod apply` SHALL use the current ownership inventory to compute stale resources (previous owned set minus current rendered set). It SHALL NOT require inventory change-history fields to perform stale-set computation or pruning.

#### Scenario: Apply computes stale set from ownership inventory

- **WHEN** a previous ownership inventory tracks resources `A`, `B`, and `C`
- **AND** the current render contains `A` and `B`
- **THEN** `C` SHALL be considered stale and eligible for pruning

### Requirement: mod apply writes current release inventory record after successful apply

After all resources are successfully applied, `opm mod apply` SHALL persist the current release inventory record for the release. The persisted form SHALL store top-level `createdBy`, `releaseMetadata`, `moduleMetadata`, and the current owned resource set directly instead of a history-bearing inventory shape.

#### Scenario: Successful apply persists current release inventory record

- **WHEN** `opm mod apply` successfully applies all rendered resources
- **THEN** the persisted release inventory record SHALL record the current owned resource entries for that release
- **AND** the record SHALL preserve top-level `createdBy`, `releaseMetadata`, and `moduleMetadata`
- **AND** the ownership inventory in that record SHALL be sufficient for later prune and resource enumeration

### Requirement: mod apply stores deployed module version in module metadata

When `opm mod apply` persists the release inventory record, it SHALL store the deployed module version in `moduleMetadata.version` rather than relying on inventory change history.

#### Scenario: Apply persists deployed module version in module metadata

- **WHEN** `opm mod apply` persists a release inventory record for a versioned module
- **THEN** the persisted record SHALL contain that deployed module version under `moduleMetadata.version`

### Requirement: mod delete uses ownership inventory for resource enumeration

`opm mod delete` SHALL use the ownership inventory stored in the persisted release inventory record to enumerate resources for deletion when it exists. If no persisted inventory exists, the command MAY fall back to legacy discovery behavior.

#### Scenario: Delete with ownership inventory

- **WHEN** running `opm mod delete` and a persisted release inventory record exists
- **THEN** only resources listed in that record's ownership inventory SHALL be deleted
