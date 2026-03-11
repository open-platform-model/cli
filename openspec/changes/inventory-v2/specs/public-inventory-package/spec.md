## MODIFIED Requirements

### Requirement: Inventory contract is exposed as a public package

The reusable inventory contract SHALL be available from a public `pkg/inventory` package so non-CLI components can consume the same ownership inventory types and helpers as the CLI.

#### Scenario: Controller imports public inventory package
- **WHEN** a future controller needs to compute stale resources or persist current owned resources in release status
- **THEN** it SHALL be able to import `pkg/inventory` without importing CLI command packages or storage-specific history types

### Requirement: Public inventory package contains only reusable ownership concerns

The public inventory package SHALL expose ownership inventory types, identity helpers, stale-set helpers, and any pure digest/sort helpers needed to reason about the current owned resource set. It MUST NOT require CLI output packages, CLI-specific command dependencies, or history-bearing storage types.

#### Scenario: Public inventory package stays decoupled from storage history
- **WHEN** a consumer imports `pkg/inventory`
- **THEN** it SHALL NOT be required to import change-history types, storage codecs, or source/version metadata types to use the ownership contract

### Requirement: Storage representation does not define the public contract

Inventory persistence MAY use a Secret or another storage mechanism, but the storage representation SHALL NOT define the core public inventory API.

#### Scenario: Persisted inventory remains independent from storage shape
- **WHEN** the CLI or a future controller persists an ownership inventory
- **THEN** the public ownership inventory API SHALL remain independent from the details of that storage representation
