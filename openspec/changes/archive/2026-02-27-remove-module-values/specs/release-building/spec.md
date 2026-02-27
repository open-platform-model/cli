## MODIFIED Requirements

### Requirement: Value selection falls back to module defaults when no files are given
The builder SHALL discover and load `values.cue` from the module directory (`mod.ModulePath`) when no `--values` files are provided. The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` SHALL be completely ignored. If neither `--values` files nor `values.cue` exist, the builder SHALL return an error.

#### Scenario: No values files, values.cue exists in module directory
- **WHEN** no `--values` files are provided
- **AND** a `values.cue` file exists in the module directory
- **THEN** the builder SHALL load `values.cue`, extract the `values` field, and use it for injection

#### Scenario: No values files, no values.cue
- **WHEN** no `--values` files are provided
- **AND** no `values.cue` file exists in the module directory
- **THEN** the builder SHALL return an error indicating values must be provided via `values.cue` or `--values`

#### Scenario: --values files provided, values.cue exists
- **WHEN** `--values` files are provided
- **AND** a `values.cue` file also exists in the module directory
- **THEN** the builder SHALL use ONLY the `--values` files and completely ignore `values.cue`

#### Scenario: Multiple --values files are unified
- **WHEN** more than one values file is provided via `--values`
- **THEN** the builder SHALL unify all files together before injection

### Requirement: Builder accepts a loaded module and produces a concrete release
The builder SHALL accept a fully-loaded `*core.Module` and release options (name, namespace, optional values files) and return a concrete `*core.ModuleRelease` with all fields populated. The builder SHALL NOT depend on `Module.Values` being set.

#### Scenario: Successful build with default values
- **WHEN** a loaded module is provided and no values files are given
- **AND** `values.cue` exists in the module directory
- **THEN** the builder SHALL return a concrete `*core.ModuleRelease` with metadata, components, and values all set

#### Scenario: Successful build with external values files
- **WHEN** one or more `--values` files are provided
- **THEN** the builder SHALL load and unify those files and use them as the sole values source
