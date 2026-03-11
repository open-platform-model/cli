## Purpose

Defines the public reusable inventory package boundary so non-CLI components can consume the same inventory contract as the CLI.

## Requirements

### Requirement: Inventory contract is exposed as a public package

The reusable inventory contract SHALL be available from a public `pkg/inventory` package so non-CLI components can consume the same inventory types and Secret serialization contract as the CLI.

#### Scenario: Controller imports public inventory package

- **WHEN** a future controller needs to read or write OPM inventory Secrets
- **THEN** it SHALL be able to import `pkg/inventory` without importing CLI command packages

### Requirement: Public inventory package contains only reusable contract concerns

The public inventory package SHALL expose the inventory model, naming helpers, serialization helpers, provenance helpers, and pure change-history or identity helpers. It MUST NOT require CLI output packages or CLI-specific command dependencies.

#### Scenario: Public inventory package stays decoupled from CLI output

- **WHEN** a consumer imports `pkg/inventory`
- **THEN** it SHALL NOT be required to import CLI output or command packages

### Requirement: Public inventory package preserves the existing wire contract

Moving inventory helpers to `pkg/inventory` SHALL NOT change the inventory Secret name format, label selector contract, serialized data keys, or JSON field names used by existing inventories.

#### Scenario: Existing inventory remains readable after package move

- **WHEN** an inventory Secret written before the package move is read through `pkg/inventory`
- **THEN** it SHALL deserialize successfully without migration
