# Module Release Synthesis

## ADDED Requirements

### Requirement: Synthesize ModuleRelease from Module

The module loader SHALL automatically wrap a `#Module` in a synthetic `#ModuleRelease` when the loaded CUE value does not already contain a concrete `components` field.

#### Scenario: Module without release file

- **WHEN** user runs `opm mod build --namespace test ./my-module`
- **AND** the module defines `#components` but not `components`
- **THEN** the loader synthesizes a `#ModuleRelease` wrapping the module
- **AND** components are extracted from the release's concrete `components` field

#### Scenario: Module with explicit release file

- **WHEN** user runs `opm mod build --namespace test ./my-module`
- **AND** the module directory contains a file that produces a `#ModuleRelease` with concrete `components`
- **THEN** the loader uses the existing release directly without synthesis

### Requirement: Namespace resolution with precedence

The CLI SHALL resolve the target namespace using the following precedence (highest to lowest):

1. `--namespace` flag
2. `#Module.metadata.defaultNamespace` (if defined by module author)

The CLI SHALL require either `--namespace` flag OR `defaultNamespace` to be present when synthesizing.

#### Scenario: Namespace from flag overrides defaultNamespace

- **WHEN** user runs `opm mod build --namespace production ./my-module`
- **AND** the module defines `metadata.defaultNamespace: "dev"`
- **THEN** the synthesized release has `metadata.namespace: "production"`

#### Scenario: Namespace from defaultNamespace when flag omitted

- **WHEN** user runs `opm mod build ./my-module` without `--namespace`
- **AND** the module defines `metadata.defaultNamespace: "dev"`
- **THEN** the synthesized release has `metadata.namespace: "dev"`

#### Scenario: Missing namespace with no defaultNamespace

- **WHEN** user runs `opm mod build ./my-module` without `--namespace`
- **AND** the module does NOT define `metadata.defaultNamespace`
- **THEN** the CLI returns an error indicating `--namespace` is required or `defaultNamespace` should be set in the module

### Requirement: CUE unification for synthesis

The loader SHALL synthesize the release by unifying the loaded module with the `core.#ModuleRelease` schema, injecting the module as `#module` and namespace from resolved value.

#### Scenario: Valid module synthesizes successfully

- **WHEN** the module satisfies all required constraints
- **THEN** CUE unification succeeds
- **AND** the release contains concrete `components` derived from `#module.#components`

#### Scenario: Module with missing required fields

- **WHEN** the module has incomplete required fields (e.g., missing `container.name`)
- **THEN** CUE unification fails with validation error
- **AND** the error message includes the field path and constraint that failed

### Requirement: Clear validation error messages

The CLI SHALL wrap CUE validation errors with actionable guidance, including the component name, field path, and what value is expected.

#### Scenario: Missing required field error

- **WHEN** synthesis fails due to missing `spec.container.name` in component `web`
- **THEN** error message includes: component name (`web`), field path (`spec.container.name`), and that a string value is required

#### Scenario: Constraint violation error

- **WHEN** synthesis fails due to constraint violation (e.g., `replicas: -1` when `>=1` required)
- **THEN** error message includes the field, actual value, and constraint that was violated

### Requirement: Extract module metadata from release

The loader SHALL extract module metadata (`name`, `version`, `labels`) from the synthesized release's `metadata` field, which inherits from the wrapped module.

#### Scenario: Metadata inheritance

- **WHEN** synthesis succeeds
- **THEN** `LoadedModule.Name` equals the module's `metadata.name`
- **AND** `LoadedModule.Version` equals the module's `metadata.version`
- **AND** `LoadedModule.Namespace` equals the resolved namespace (flag or defaultNamespace)
