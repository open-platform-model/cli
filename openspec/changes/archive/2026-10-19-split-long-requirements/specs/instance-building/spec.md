## REMOVED Requirements

### Requirement: Loader validates consumer values and produces a concrete ModuleInstance
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI produces validated, concrete instances exclusively through the library kernel", "Every synthesis path lets CUE derive the uuid, components, auto-secrets and standard labels", "Every path runs the kernel's shape gate and concreteness enforcement", "The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance", "The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance", "The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir".

### Requirement: Value selection falls back to module defaults when no files are given
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "With no `--values` files the builder discovers values by a three-step priority", "The builder never reads values from `Module.Values` and `--values` files make `values.cue` and `debugValues` ignored", "An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback".

## ADDED Requirements

### Requirement: The CLI produces validated, concrete instances exclusively through the library kernel

The CLI SHALL produce validated, concrete instances exclusively through the `library` kernel. The CLI SHALL NOT carry its own `LoadModuleInstanceFromValue` pipeline and SHALL NOT reach for a raw-value loading tier.

#### Scenario: Module Gate catches type mismatch

- **WHEN** consumer values contain a field with the wrong type
- **THEN** the kernel validation SHALL surface a structured config error identifying the offending field

### Requirement: Every synthesis path lets CUE derive the uuid, components, auto-secrets and standard labels

In every synthesis path the kernel SHALL unify inputs against the resolved `#ModuleInstance` schema and let CUE derive uuid, components, auto-secrets, and standard labels. (Requirements: The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance; The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance; The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir)

#### Scenario: Synthesis derives uuid, components, auto-secrets and standard labels

- **WHEN** a synthesis path runs
- **THEN** the kernel SHALL unify inputs against the resolved `#ModuleInstance` schema and let CUE derive uuid, components, auto-secrets, and standard labels

### Requirement: Every path runs the kernel's shape gate and concreteness enforcement

All paths SHALL run the kernel's shape gate and concreteness enforcement, producing a `*module.Instance`. (Requirements: The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance; The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance; The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir)

#### Scenario: Every path runs the shape gate and concreteness enforcement

- **WHEN** any loading path produces an instance
- **THEN** the kernel SHALL run its shape gate and concreteness enforcement and produce a `*module.Instance`

### Requirement: The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance

The loading entry points SHALL map onto kernel acquire verbs:

1. **Module-directory path**: kernel `AcquireModuleFromDir` + `SynthesizeInstance`, used by `opm module build` and `opm module apply` with a local directory. Accepts a directory containing a module CUE package; the acquire stages the directory as the module's source, so synthesis builds inside the module's own root. Its values are kernel sources: the `-f` files, or the module's `debugValues` rendered as one source by the CLI.

#### Scenario: Successful load from module directory

- **WHEN** the module-directory path acquires a directory containing a module package and values
- **THEN** kernel synthesis returns a `*module.Instance` with all fields populated

#### Scenario: Successful synthesis from a module-package directory

- **WHEN** synthesis runs against a module-package directory with `-f` values or the module's `debugValues`
- **THEN** kernel `SynthesizeInstance` returns a `*module.Instance` whose kind is `ModuleInstance`

### Requirement: The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance

The loading entry points SHALL map onto kernel acquire verbs:

2. **Published-module path**: kernel `AcquireModuleFromRegistry` + `SynthesizeInstance`, used by `opm module build` and `opm module apply` with a published module path. The module is fetched at the resolved version; values are selected exactly as for the module-directory path.
(Requirement: The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance)

#### Scenario: Successful synthesis from a published module

- **WHEN** the published-module path acquires a module version from the registry with `-f` values or the module's `debugValues`
- **THEN** kernel `SynthesizeInstance` returns a `*module.Instance` whose kind is `ModuleInstance`

### Requirement: The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir

The loading entry points SHALL map onto kernel acquire verbs:

3. **Instance package**: kernel `AcquireInstanceFromDir` on an instance package directory (a named directory, or the directory of a named `.cue` file), with any `-f` files passed as trailing values sources (`LoadSourceFromFile`), used by `opm instance` commands. The package may be a standalone CUE module or a package inside another CUE module.

#### Scenario: Successful load from instance file

- **WHEN** the instance-package path acquires the package directory of a `.cue` file where the module reference resolves via CUE import
- **THEN** kernel acquisition returns a `*module.Instance` with all fields populated (including auto-secrets derived by CUE)

### Requirement: With no `--values` files the builder discovers values by a three-step priority

When no `--values` files are provided, the builder SHALL discover values using the following priority:

1. When `instance.cue` is present: auto-discover `values.cue` from the module directory (existing behavior)
2. When `instance.cue` is absent and `debugValues` is defined in the module: use `debugValues` as the values source
3. When neither `instance.cue` nor `values.cue` nor `debugValues` is available: return a descriptive error

#### Scenario: No values file, `values.cue` exists in module directory

- **WHEN** `AcquireInstanceFromDir` acquires an instance package with no trailing values source
- **AND** `values.cue` exists in the module directory
- **THEN** `values.cue` is loaded alongside `instance.cue` as part of the CUE instance

#### Scenario: No values files, no values.cue, debugValues defined

- **WHEN** no `--values` files are provided
- **AND** no `values.cue` file exists in the module directory
- **AND** the module defines a concrete `debugValues` field
- **THEN** the builder SHALL use `debugValues` as the values source

#### Scenario: No values files, no values.cue, no debugValues

- **WHEN** no `--values` files are provided
- **AND** no `values.cue` file exists in the module directory
- **AND** the module has no `debugValues` field
- **THEN** the builder SHALL return an error indicating the user must provide values via `values.cue`, `debugValues`, or `--values`

### Requirement: The builder never reads values from `Module.Values` and `--values` files make `values.cue` and `debugValues` ignored

The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` and `debugValues` SHALL both be ignored.

#### Scenario: Multiple `--values` files are unified

- **WHEN** more than one values file is provided via `--values`
- **THEN** the builder SHALL unify all files together before injection

### Requirement: An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback

When an instance package is acquired from its directory (`AcquireInstanceFromDir`, the instance-file path), the `values` field is inline in the instance CUE file itself. There SHALL be no `values.cue` fallback: the instance package is self-contained.

#### Scenario: Instance file is self-contained

- **WHEN** `AcquireInstanceFromDir` acquires the package holding an instance file
- **THEN** the `values` field is read from the instance CUE file's inline definition
- **AND** no `values.cue` file is searched for or loaded
