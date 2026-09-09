## MODIFIED Requirements

### Requirement: Loader validates consumer values and produces a concrete ModuleInstance

The CLI SHALL produce validated, concrete instances exclusively through the `library` kernel. The three loading entry points map onto kernel acquire verbs:

1. **Module-directory path**: kernel `AcquireModuleFromDir` + `SynthesizeInstance` — used by `opm mod`/`opm module` commands. Accepts a directory containing a module CUE package; the acquire stages the directory as the module's source, so synthesis builds inside the module's own root.
2. **Standalone instance file**: kernel `AcquireInstanceFromDir` on the file's package directory, with any `-f` files passed as trailing values sources (`LoadSourceFromFile`) — used by `opm instance` commands. Accepts a `.cue` file with CUE import resolution.
3. **Module-package synthesis**: kernel `SynthesizeInstance` — used by `opm instance build <dir>` and `opm module build`. Accepts a module package directory (no `instance.cue`); its values are kernel sources — the `-f` files, or the module's `debugValues` rendered as one source by the CLI — and the kernel unifies inputs against the resolved `#ModuleInstance` schema and lets CUE derive uuid, components, auto-secrets, and standard labels.

All paths run the kernel's shape gate and concreteness enforcement, producing a `*module.Instance`. The CLI SHALL NOT carry its own `LoadModuleInstanceFromValue` pipeline and SHALL NOT reach for a raw-value loading tier.

#### Scenario: Successful load from module directory

- **WHEN** the module-directory path acquires a directory containing a module package and values
- **THEN** kernel synthesis returns a `*module.Instance` with all fields populated

#### Scenario: Successful load from instance file

- **WHEN** the instance-file path acquires the package directory of a `.cue` file where the module reference resolves via CUE import
- **THEN** kernel acquisition returns a `*module.Instance` with all fields populated (including auto-secrets derived by CUE)

#### Scenario: Successful synthesis from a module-package directory

- **WHEN** synthesis runs against a module-package directory with `-f` values or the module's `debugValues`
- **THEN** kernel `SynthesizeInstance` returns a `*module.Instance` whose kind is `ModuleInstance`

#### Scenario: Module Gate catches type mismatch

- **WHEN** consumer values contain a field with the wrong type
- **THEN** the kernel validation SHALL surface a structured config error identifying the offending field

### Requirement: Value selection falls back to module defaults when no files are given

When no `--values` files are provided, the builder SHALL discover values using the following priority:

1. When `instance.cue` is present: auto-discover `values.cue` from the module directory (existing behavior)
2. When `instance.cue` is absent and `debugValues` is defined in the module: use `debugValues` as the values source
3. When neither `instance.cue` nor `values.cue` nor `debugValues` is available: return a descriptive error

The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` and `debugValues` SHALL both be ignored.

When using `LoadInstanceFile()` (instance-file path), the `values` field is inline in the instance CUE file itself. There is no `values.cue` fallback — the instance file is self-contained.

#### Scenario: No values file, `values.cue` exists in module directory

- **WHEN** `AcquireInstanceFromDir` acquires an instance package with no trailing values source
- **AND** `values.cue` exists in the module directory
- **THEN** `values.cue` is loaded alongside `instance.cue` as part of the CUE instance

#### Scenario: Instance file is self-contained

- **WHEN** `LoadInstanceFile()` is called
- **THEN** the `values` field is read from the instance CUE file's inline definition
- **AND** no `values.cue` file is searched for or loaded

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

#### Scenario: Multiple `--values` files are unified

- **WHEN** more than one values file is provided via `--values`
- **THEN** the builder SHALL unify all files together before injection
