## MODIFIED Requirements

### Requirement: The resulting instance must be fully concrete
The builder SHALL validate that the `#ModuleInstance` value is fully concrete after injection, and return an error if any field remains abstract or unresolved. When the values leave required `#config` values unset, the error SHALL hold one finding for each unset value at `values.<field>`, whether a component reads the value or not, and SHALL NOT hold a finding at the place inside a component that reads it; the cli SHALL print these findings as the kernel returns them.

#### Scenario: Incomplete values leave instance non-concrete
- **WHEN** the provided values do not satisfy all required fields in `#config`
- **THEN** the builder SHALL return an error identifying which fields are not concrete

#### Scenario: Every unset required value is named

- **WHEN** a render runs with values that leave two required `#config` values unset, one that a component reads and one that no component reads
- **THEN** the command exits 2 and prints two findings, `values.<field>` for each
- **AND** no finding names a path under `components`

#### Scenario: Fully provided values produce a concrete instance
- **WHEN** all required fields in `#config` are satisfied by the selected values
- **THEN** the builder SHALL return a `*core.ModuleInstance` where all components are concrete and ready for matching
