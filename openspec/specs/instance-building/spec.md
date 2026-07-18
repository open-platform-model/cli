## Purpose

Defines the contract for loading and building a concrete `*moduleinstance.ModuleInstance` via the `pkg/loader` package. There is no separate builder phase — loading IS building, consistent with the `promote-factory-engine` architecture. The loader is responsible for value selection, schema validation (Module Gate), CUE-native evaluation of metadata and labels, and concreteness verification.

## Requirements

### Requirement: Loader validates consumer values and produces a concrete ModuleInstance

The CLI SHALL produce validated, concrete instances exclusively through the `library` kernel. The three loading entry points map onto kernel calls:

1. **Module-directory path**: kernel `LoadModulePackage` + `SynthesizeInstance` — used by `opm mod`/`opm module` commands. Accepts a directory containing a module CUE package.
2. **Standalone instance file**: kernel instance-package loading (`LoadInstancePackage`/`LoadSourceFromFile`) + `ProcessModuleInstance` — used by `opm instance` commands. Accepts a `.cue` file with CUE import resolution.
3. **Module-package synthesis**: kernel `SynthesizeInstance` — used by `opm instance build <dir>` and `opm module build`. Accepts a module package directory (no `instance.cue`); the kernel unifies inputs against the resolved `#ModuleInstance` schema and lets CUE derive uuid, components, auto-secrets, and standard labels.

All paths run the kernel's Module Gate equivalent (`ValidateModuleValues*` / `ProcessModuleInstance` concreteness enforcement), producing a `*module.Instance`. The CLI SHALL NOT carry its own `LoadModuleInstanceFromValue` pipeline.

#### Scenario: Successful load from module directory

- **WHEN** the module-directory path loads a directory containing a module package and values
- **THEN** kernel synthesis returns a `*module.Instance` with all fields populated

#### Scenario: Successful load from instance file

- **WHEN** the instance-file path loads a `.cue` file where the module reference resolves via CUE import
- **THEN** kernel processing returns a `*module.Instance` with all fields populated (including auto-secrets derived by CUE)

#### Scenario: Successful synthesis from a module-package directory

- **WHEN** synthesis runs against a module-package directory with `-f` values or the module's `debugValues`
- **THEN** kernel `SynthesizeInstance` returns a `*module.Instance` whose kind is `ModuleInstance`

#### Scenario: Module Gate catches type mismatch

- **WHEN** consumer values contain a field with the wrong type
- **THEN** the kernel validation SHALL surface a structured config error identifying the offending field

### Requirement: Values are validated against the module config schema before injection
The builder SHALL validate the selected values against the module's `#config` schema and return a descriptive error if they do not conform.

#### Scenario: Values match schema
- **WHEN** the selected values satisfy all constraints in `#config`
- **THEN** injection proceeds without error

#### Scenario: Values violate schema
- **WHEN** the selected values contain a field that violates a `#config` constraint (wrong type, out-of-range, missing required field)
- **THEN** the builder SHALL return an error identifying the offending field and the constraint that was violated

### Requirement: Instance metadata and labels are derived by CUE evaluation
The builder SHALL load `#ModuleInstance` from `opmodel.dev/core@v1` (resolved from the module's own dependency cache) and inject the module, instance name, namespace, and values via `FillPath`. UUID, labels, and derived metadata fields SHALL be computed by CUE evaluation, not by Go code.

#### Scenario: UUID is deterministic
- **WHEN** the same module, instance name, and namespace are provided
- **THEN** the resulting `ModuleInstance.Metadata.UUID` SHALL be identical across builds

#### Scenario: Labels are populated from CUE evaluation
- **WHEN** the instance is built successfully
- **THEN** `ModuleInstance.Metadata.Labels` SHALL contain all expected OPM labels as evaluated by `#ModuleInstance`

#### Scenario: Core v1 schema loaded
- **WHEN** the builder loads the core schema
- **THEN** it SHALL load `opmodel.dev/core@v1` (not `opmodel.dev/core@v0`)
- **THEN** error messages SHALL reference `opmodel.dev/core@v1`

### Requirement: The resulting instance must be fully concrete
The builder SHALL validate that the `#ModuleInstance` value is fully concrete after injection, and return an error if any field remains abstract or unresolved.

#### Scenario: Incomplete values leave instance non-concrete
- **WHEN** the provided values do not satisfy all required fields in `#config`
- **THEN** the builder SHALL return an error identifying which fields are not concrete

#### Scenario: Fully provided values produce a concrete instance
- **WHEN** all required fields in `#config` are satisfied by the selected values
- **THEN** the builder SHALL return a `*core.ModuleInstance` where all components are concrete and ready for matching

### Requirement: Value selection falls back to module defaults when no files are given

When no `--values` files are provided, the builder SHALL discover values using the following priority:

1. When `instance.cue` is present: auto-discover `values.cue` from the module directory (existing behavior)
2. When `instance.cue` is absent and `debugValues` is defined in the module: use `debugValues` as the values source
3. When neither `instance.cue` nor `values.cue` nor `debugValues` is available: return a descriptive error

The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` and `debugValues` SHALL both be ignored.

When using `LoadInstanceFile()` (instance-file path), the `values` field is inline in the instance CUE file itself. There is no `values.cue` fallback — the instance file is self-contained.

#### Scenario: No values file, `values.cue` exists in module directory

- **WHEN** `LoadInstancePackage()` is called with no explicit values file
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

### Requirement: `opm mod vet` uses `debugValues` by default

The `opm mod vet` command SHALL use the module's `debugValues` field as the values source when no `-f` flag is provided. This validation SHALL happen in the module vet command itself rather than through `cmdutil.RenderRelease()`.

#### Scenario: `debugValues` used when no `-f` flag

- **WHEN** `opm mod vet` is run without `-f` flags
- **THEN** the module's `debugValues` field is extracted and used as the values source
- **AND** the vet output shows "debugValues" as the values source

#### Scenario: `-f` flag overrides `debugValues`

- **WHEN** `opm mod vet` is run with one or more `-f` flags
- **THEN** the explicit values files are used
- **AND** `debugValues` is ignored

#### Scenario: `debugValues` is `_` (unconstrained)

- **WHEN** `opm mod vet` is run without `-f` flags
- **AND** the module's `debugValues` field is `_` (open/unconstrained, not filled by the author)
- **THEN** `opm mod vet` returns an error: "debugValues is not concrete — module must provide complete test values"
