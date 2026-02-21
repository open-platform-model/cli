## ADDED Requirements

### Requirement: Builder accepts a loaded module and produces a concrete release
The builder SHALL accept a fully-loaded `*core.Module` and release options (name, namespace, optional values files) and return a concrete `*core.ModuleRelease` with all fields populated.

#### Scenario: Successful build with default values
- **WHEN** a loaded module with default values is provided and no values files are given
- **THEN** the builder SHALL return a concrete `*core.ModuleRelease` with metadata, components, and values all set

#### Scenario: Successful build with external values files
- **WHEN** one or more `--values` files are provided
- **THEN** the builder SHALL load and unify those files and use them in place of the module's default values

#### Scenario: Multiple values files are unified
- **WHEN** more than one values file is provided
- **THEN** the builder SHALL unify all files together before injection, with later files taking precedence over earlier ones

### Requirement: Values are validated against the module config schema before injection
The builder SHALL validate the selected values against the module's `#config` schema and return a descriptive error if they do not conform.

#### Scenario: Values match schema
- **WHEN** the selected values satisfy all constraints in `#config`
- **THEN** injection proceeds without error

#### Scenario: Values violate schema
- **WHEN** the selected values contain a field that violates a `#config` constraint (wrong type, out-of-range, missing required field)
- **THEN** the builder SHALL return an error identifying the offending field and the constraint that was violated

### Requirement: Release metadata and labels are derived by CUE evaluation
The builder SHALL load `#ModuleRelease` from `opmodel.dev/core@v0` (resolved from the module's own dependency cache) and inject the module, release name, namespace, and values via `FillPath`. UUID, labels, and derived metadata fields SHALL be computed by CUE evaluation, not by Go code.

#### Scenario: UUID is deterministic
- **WHEN** the same module, release name, and namespace are provided
- **THEN** the resulting `ModuleRelease.Metadata.UUID` SHALL be identical across builds

#### Scenario: Labels are populated from CUE evaluation
- **WHEN** the release is built successfully
- **THEN** `ModuleRelease.Metadata.Labels` SHALL contain all expected OPM labels as evaluated by `#ModuleRelease`

### Requirement: The resulting release must be fully concrete
The builder SHALL validate that the `#ModuleRelease` value is fully concrete after injection, and return an error if any field remains abstract or unresolved.

#### Scenario: Incomplete values leave release non-concrete
- **WHEN** the provided values do not satisfy all required fields in `#config`
- **THEN** the builder SHALL return an error identifying which fields are not concrete

#### Scenario: Fully provided values produce a concrete release
- **WHEN** all required fields in `#config` are satisfied by the selected values
- **THEN** the builder SHALL return a `*core.ModuleRelease` where all components are concrete and ready for matching

### Requirement: Value selection falls back to module defaults when no files are given
The builder SHALL use the module's embedded default values (from `values.cue`, stored in `mod.Values`) when no external values files are provided.

#### Scenario: No values files, module has defaults
- **WHEN** no `--values` files are provided and the module has a non-empty `mod.Values`
- **THEN** the builder SHALL use `mod.Values` as the input for injection

#### Scenario: No values files, module has no defaults
- **WHEN** no `--values` files are provided and `mod.Values` is absent
- **THEN** the builder SHALL return an error indicating that values must be provided
