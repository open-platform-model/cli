## ADDED Requirements

### Requirement: List command displays release ownership

The `opm mod list` command SHALL expose release ownership derived from inventory provenance. Table outputs SHALL include an OWNER column, and structured outputs SHALL include an `owner` field.

#### Scenario: Table output shows controller ownership
- **WHEN** the user runs `opm mod list`
- **AND** a release inventory records `createdBy: "controller"`
- **THEN** the release row SHALL display `controller` in the OWNER column

#### Scenario: Legacy inventory shows CLI ownership
- **WHEN** the user runs `opm mod list`
- **AND** a release inventory has no `createdBy`
- **THEN** the release row SHALL display `cli` in the OWNER column
