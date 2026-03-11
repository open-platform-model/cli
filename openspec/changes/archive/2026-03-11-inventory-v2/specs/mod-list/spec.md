## MODIFIED Requirements

### Requirement: List command discovers releases via persisted ownership inventory

The `opm mod list` command SHALL continue to discover deployed releases via persisted inventory records, but it SHALL NOT require inventory change-history fields to identify the current owned resource set for a release.

#### Scenario: List works with ownership-only inventory

- **WHEN** the user runs `opm mod list` in a namespace containing releases with ownership-only inventory
- **THEN** the command SHALL still discover those releases and evaluate their health from the tracked resource set

### Requirement: List metadata extraction does not depend on inventory change history

The list command SHALL NOT require inventory change-history metadata (latest change source version, raw values, timestamp history) as part of the public inventory contract. Release and module display metadata SHALL come from the persisted release inventory record (`releaseMetadata`, `moduleMetadata`, `createdBy`) or be omitted until richer release status exists.

#### Scenario: List remains functional with ownership-only inventory

- **WHEN** a release has ownership-only inventory and no latest change entry
- **THEN** `opm mod list` SHALL still be able to display the release and compute health from tracked resources

#### Scenario: List reads deployed module version from module metadata

- **WHEN** a persisted release inventory record includes `moduleMetadata.version`
- **THEN** `opm mod list` SHALL use that field for deployed module version display instead of inventory change history
