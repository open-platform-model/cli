## MODIFIED Requirements

### Requirement: Loader validates consumer values and produces a concrete ModuleInstance

The CLI SHALL produce validated, concrete instances exclusively through the `library` kernel. The loading entry points map onto kernel acquire verbs:

1. **Module-directory path**: kernel `AcquireModuleFromDir` + `SynthesizeInstance`, used by `opm module build` and `opm module apply` with a local directory. Accepts a directory containing a module CUE package; the acquire stages the directory as the module's source, so synthesis builds inside the module's own root. Its values are kernel sources: the `-f` files, or the module's `debugValues` rendered as one source by the CLI.
2. **Published-module path**: kernel `AcquireModuleFromRegistry` + `SynthesizeInstance`, used by `opm module build` and `opm module apply` with a published module path. The module is fetched at the resolved version; values are selected exactly as for the module-directory path.
3. **Instance package**: kernel `AcquireInstanceFromDir` on an instance package directory (a named directory, or the directory of a named `.cue` file), with any `-f` files passed as trailing values sources (`LoadSourceFromFile`), used by `opm instance` commands. The package may be a standalone CUE module or a package inside another CUE module.

In every synthesis path the kernel unifies inputs against the resolved `#ModuleInstance` schema and lets CUE derive uuid, components, auto-secrets, and standard labels. All paths run the kernel's shape gate and concreteness enforcement, producing a `*module.Instance`. The CLI SHALL NOT carry its own `LoadModuleInstanceFromValue` pipeline and SHALL NOT reach for a raw-value loading tier.

#### Scenario: Successful load from module directory

- **WHEN** the module-directory path acquires a directory containing a module package and values
- **THEN** kernel synthesis returns a `*module.Instance` with all fields populated

#### Scenario: Successful load from instance file

- **WHEN** the instance-package path acquires the package directory of a `.cue` file where the module reference resolves via CUE import
- **THEN** kernel acquisition returns a `*module.Instance` with all fields populated (including auto-secrets derived by CUE)

#### Scenario: Successful synthesis from a module-package directory

- **WHEN** synthesis runs against a module-package directory with `-f` values or the module's `debugValues`
- **THEN** kernel `SynthesizeInstance` returns a `*module.Instance` whose kind is `ModuleInstance`

#### Scenario: Module Gate catches type mismatch

- **WHEN** consumer values contain a field with the wrong type
- **THEN** the kernel validation SHALL surface a structured config error identifying the offending field

#### Scenario: Successful synthesis from a published module

- **WHEN** the published-module path acquires a module version from the registry with `-f` values or the module's `debugValues`
- **THEN** kernel `SynthesizeInstance` returns a `*module.Instance` whose kind is `ModuleInstance`
