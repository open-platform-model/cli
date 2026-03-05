## MODIFIED Requirements

### Requirement: Builder accepts a loaded module and produces a concrete release

The builder SHALL accept a fully-loaded `*core.Module` and release options (name, namespace, optional values files) and return a concrete `*core.ModuleRelease` with all fields populated. The builder SHALL NOT depend on `Module.Values` being set. When the module's `#config` contains `#Secret` fields, the builder SHALL additionally inject an auto-generated `opm-secrets` component into the release's component map.

The builder SHALL also support an alternate code path where a pre-evaluated `#ModuleRelease` CUE value is provided (from a release file). In this case, the builder SHALL skip the `FillPath` construction of `#ModuleRelease` and instead validate and extract metadata and components from the pre-filled value.

#### Scenario: Successful build from module (existing behavior)

- **WHEN** a loaded module is provided and values files are given
- **THEN** the builder SHALL construct `#ModuleRelease` via FillPath and return a concrete `*core.ModuleRelease`

#### Scenario: Successful build from pre-filled release value

- **WHEN** a pre-evaluated `#ModuleRelease` CUE value is provided (from a release file with `#module` already filled)
- **THEN** the builder SHALL skip the FillPath construction
- **AND** validate concreteness of the provided value
- **AND** extract metadata, components, and autoSecrets from the CUE evaluation
- **AND** return a concrete `*core.ModuleRelease`

#### Scenario: Pre-filled release with --module override

- **WHEN** a pre-evaluated `#ModuleRelease` CUE value is provided
- **AND** a `--module` flag provides a local module
- **THEN** the builder SHALL fill `#module` via FillPath with the local module's Raw value
- **AND** proceed with validation and extraction

#### Scenario: Build with secrets produces release containing opm-secrets component

- **WHEN** a loaded module with `#Secret` fields in `#config` is provided with concrete secret values
- **THEN** the builder SHALL return a concrete `*core.ModuleRelease`
- **AND** the release's components SHALL include `"opm-secrets"` with the correct `#resources` FQN

### Requirement: Value selection falls back to module defaults when no files are given

The builder SHALL discover and load `values.cue` from the module directory (`mod.ModulePath`) when no `--values` files are provided and the build is from a module directory (not from a release file). The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` SHALL be completely ignored. If neither `--values` files nor `values.cue` exist, the builder SHALL return an error.

When building from a release file, the values SHALL come from the release file's `values` field. The `values.cue` fallback SHALL NOT apply to release-file builds.

#### Scenario: No values files, values.cue exists in module directory

- **WHEN** no `--values` files are provided during a module-directory build
- **AND** a `values.cue` file exists in the module directory
- **THEN** the builder SHALL load `values.cue`, extract the `values` field, and use it for injection

#### Scenario: Release file build uses inline values

- **WHEN** building from a release file
- **THEN** the builder SHALL use the `values` field from the release CUE value
- **AND** SHALL NOT search for or load `values.cue` from any directory

#### Scenario: No values files, no values.cue (module-directory build)

- **WHEN** no `--values` files are provided during a module-directory build
- **AND** no `values.cue` file exists in the module directory
- **THEN** the builder SHALL return an error indicating values must be provided via `values.cue` or `--values`

## ADDED Requirements

### Requirement: Builder supports debugValues as values source

The builder SHALL accept a `DebugValues` option. When set, the builder SHALL use the module's `debugValues` field as the values source instead of loading from `values.cue` or `--values` files. This is used by `opm mod vet`.

#### Scenario: debugValues used when option is set

- **WHEN** the builder is invoked with `DebugValues: true`
- **AND** the module defines a non-empty `debugValues` field
- **THEN** the builder SHALL extract `debugValues` from the module's CUE value
- **AND** use it as the values source for the `#ModuleRelease` FillPath injection

#### Scenario: debugValues is empty and option is set

- **WHEN** the builder is invoked with `DebugValues: true`
- **AND** the module's `debugValues` field is `_` (open/unconstrained)
- **THEN** the builder SHALL return an error indicating debugValues is not defined in the module
