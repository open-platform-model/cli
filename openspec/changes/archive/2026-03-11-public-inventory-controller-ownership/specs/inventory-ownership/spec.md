## ADDED Requirements

### Requirement: Inventory provenance defines release ownership

The inventory contract SHALL define release ownership through a `createdBy` field with allowed values `cli` and `controller`. This field SHALL represent the original creator of the release and SHALL be used by tooling to determine whether a release may be mutated.

#### Scenario: CLI-owned release
- **WHEN** an inventory records `createdBy: "cli"`
- **THEN** the release SHALL be treated as CLI-managed

#### Scenario: Controller-owned release
- **WHEN** an inventory records `createdBy: "controller"`
- **THEN** the release SHALL be treated as controller-managed

### Requirement: Legacy inventories are treated as CLI-owned

If an inventory Secret does not contain `createdBy`, the system SHALL treat it as a legacy CLI-managed release for backward compatibility.

#### Scenario: Missing provenance on existing inventory
- **WHEN** an inventory Secret created before provenance support is read
- **AND** its `releaseMetadata` has no `createdBy` field
- **THEN** the release SHALL be treated as CLI-managed

### Requirement: Ownership is exclusive across tools

The CLI and any controller that uses the inventory contract SHALL treat ownership as exclusive. A tool MUST NOT silently take over a release whose inventory indicates it was created by the other tool.

#### Scenario: CLI sees controller-owned release
- **WHEN** the CLI loads an inventory with `createdBy: "controller"`
- **THEN** the CLI SHALL treat the release as not mutable by the CLI

#### Scenario: Controller sees CLI-owned release
- **WHEN** a controller loads an inventory with `createdBy: "cli"`
- **THEN** the controller SHALL treat the release as not mutable by the controller
