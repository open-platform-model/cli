## ADDED Requirements

### Requirement: Module and ModuleRelease identifiers are distinct

Internal Go identifiers (function names, type names, field names, variable names) SHALL accurately reflect whether they operate on a Module (a CUE definition directory) or a ModuleRelease (a concrete deployed instance). Identifiers that represent release concepts SHALL use "Release" in their name, not "Module".

This requirement applies to internal packages only. User-facing terminal output (success messages, log prefixes, command descriptions) is governed by the UX principle that users think in terms of modules and is explicitly excluded.

#### Scenario: Release-scoped function names

- **WHEN** a function operates on or returns ModuleRelease data (release name, release ID, release namespace, release status)
- **THEN** its Go identifier SHALL include "Release" rather than "Module"

#### Scenario: Release-scoped type names

- **WHEN** a type's primary role is to represent or track a ModuleRelease concept
- **THEN** its Go identifier SHALL include "Release" or another accurate descriptor rather than "Module"

#### Scenario: Inventory types unambiguous

- **WHEN** an inventory type has fields for both the source module name and the release name
- **THEN** the field names SHALL be distinct and unambiguous (e.g., `ModuleName` vs `ReleaseName`), not both named `Name` or `Namespace`

#### Scenario: Dead types removed

- **WHEN** a type is defined but never instantiated or type-asserted anywhere in the codebase
- **THEN** it SHALL be removed to prevent false expectations for future contributors

#### Scenario: JSON backward compatibility preserved

- **WHEN** a Go field is renamed for clarity
- **THEN** its JSON struct tag SHALL remain unchanged to preserve deserialization of existing serialized data (e.g., existing inventory Secrets on clusters)
