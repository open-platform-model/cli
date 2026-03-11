## ADDED Requirements

### Requirement: Status header displays release ownership

The `opm mod status` command SHALL display release ownership derived from inventory provenance in the metadata header.

#### Scenario: Header shows controller ownership
- **WHEN** the user runs `opm mod status` for a release whose inventory records `createdBy: "controller"`
- **THEN** the metadata header SHALL include `Owner: controller`

#### Scenario: Header shows legacy CLI ownership
- **WHEN** the user runs `opm mod status` for a release whose inventory has no `createdBy`
- **THEN** the metadata header SHALL include `Owner: cli`

### Requirement: Status warns for non-CLI-managed releases

When the CLI reads a controller-managed release, `opm mod status` SHALL surface a warning that the release is controller-managed and cannot be mutated by the CLI.

#### Scenario: Controller-managed warning
- **WHEN** the user runs `opm mod status` for a controller-managed release
- **THEN** the command SHALL display a warning indicating that the release is controller-managed
- **AND** the command SHALL still show the release status information
