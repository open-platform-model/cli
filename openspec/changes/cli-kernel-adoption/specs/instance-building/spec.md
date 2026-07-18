# Delta: instance-building (cli-kernel-adoption)

Loading/synthesis routes through the library kernel; the CLI-side synthetic-wrapper machinery is retired (enhancement 0006 D9; 0002 carryover). Behavioral requirements (values validation, concreteness, CUE-derived metadata, value-selection fallback, `mod vet` debugValues) are unchanged and stay in the main spec.

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Synthesis path resolves the catalog dep through the registry

**Reason**: The synthetic-wrapper CUE module (`loadSynthWrapper`) is deleted; kernel `SynthesizeInstance` resolves the `#ModuleInstance` schema through the kernel's schema cache (0002 carryover, 0006 D9).
**Migration**: No user-facing change — modules still declare their own deps; the wrapper's `opmodel.dev/core/v1alpha1@v1` pin ceases to exist.

### Requirement: Synth wrapper and user module load the same catalog version

**Reason**: There is no synth wrapper; schema/version binding is the kernel's responsibility (0006 D9).
**Migration**: Version-binding behavior is the kernel contract shipped by `library` (enhancement 0001).

### Requirement: `LoadInstanceFile()` loads a standalone `.cue` file with import resolution

**Reason**: Replaced by the kernel's source/instance-package loading (0006 D9).
**Migration**: Use kernel `LoadInstancePackage`/`LoadSourceFromFile` + `ProcessModuleInstance`.

### Requirement: `LoadModulePackage()` loads a local module CUE package

**Reason**: Replaced by kernel `LoadModulePackage` (0006 D9).
**Migration**: Call the kernel wrapper; `pkg/loader`'s copy is deleted.
