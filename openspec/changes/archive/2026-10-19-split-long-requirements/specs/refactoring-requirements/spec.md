## MODIFIED Requirements

### Requirement: Module and ModuleInstance identifiers are distinct

Internal Go identifiers (function names, type names, field names, variable names) SHALL accurately reflect whether they operate on a Module (a CUE definition directory) or a ModuleInstance (a concrete deployed instance). Identifiers that represent instance concepts SHALL use "Instance" in their name, not "Module". <!-- Was: ModuleRelease / "Release" (renamed to instance, #97) -->

#### Scenario: Instance-scoped function names

- **WHEN** a function operates on or returns ModuleInstance data (instance name, instance UUID, instance namespace, instance status)
- **THEN** its Go identifier SHALL include "Instance" rather than "Module"

#### Scenario: Instance-scoped type names

- **WHEN** a type's primary role is to represent or track a ModuleInstance concept
- **THEN** its Go identifier SHALL include "Instance" or another accurate descriptor rather than "Module"

#### Scenario: Inventory types unambiguous

- **WHEN** an inventory type has fields for both the source module name and the instance name
- **THEN** the field names SHALL be distinct and unambiguous (e.g., `ModuleName` vs `InstanceName`), not both named `Name` or `Namespace`

#### Scenario: Dead types removed

- **WHEN** a type is defined but never instantiated or type-asserted anywhere in the codebase
- **THEN** it SHALL be removed to prevent false expectations for future contributors

#### Scenario: JSON backward compatibility preserved

- **WHEN** a Go field is renamed for clarity
- **THEN** its JSON struct tag SHALL remain unchanged to preserve deserialization of existing serialized data (e.g., existing `ModuleInstance` records on clusters)

## ADDED Requirements

### Requirement: The identifier rule applies to internal packages only

The requirement "Module and ModuleInstance identifiers are distinct" SHALL apply to internal packages only. User-facing terminal output (success messages, log prefixes, command descriptions) is governed by the UX principle that users think in terms of modules and is explicitly excluded.

#### Scenario: User-facing terminal output is excluded

- **WHEN** user-facing terminal output (success messages, log prefixes, command descriptions) is written
- **THEN** it is governed by the UX principle that users think in terms of modules and is explicitly excluded
