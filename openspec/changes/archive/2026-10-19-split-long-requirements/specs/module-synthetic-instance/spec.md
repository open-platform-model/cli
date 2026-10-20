## REMOVED Requirements

### Requirement: Synthesize a `#ModuleInstance` from a module-package directory
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI synthesizes a concrete instance from a module CUE package directory via kernel `SynthesizeInstance`", "The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE", "The produced instance has kind `ModuleInstance` and the synthesis does not apply `#ModuleRelease`".

### Requirement: Synthesize a `#ModuleInstance` from a published module
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI synthesizes a concrete instance from a published module through the kernel and writes nothing to disk but CUE's module cache", "Values selection, metadata defaults and the output banner of a published module are the same as for a module-package directory".

## ADDED Requirements

### Requirement: The CLI synthesizes a concrete instance from a module CUE package directory via kernel `SynthesizeInstance`

The CLI SHALL synthesize a concrete instance from a module CUE package directory without requiring an `instance.cue` file, via kernel `SynthesizeInstance`. Only the `opm module` commands (`build`, `apply` and `vet`) synthesize; `opm instance build` never does.

#### Scenario: Only the opm module commands synthesize

- **WHEN** `opm instance build` runs
- **THEN** `opm instance build` never synthesizes

### Requirement: The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE

The synthesis SHALL load the module as a whole CUE package (matching `cue eval`/`cue vet` semantics) and pass it, with resolved values and synthetic metadata, to the kernel; the kernel unifies against the resolved `#ModuleInstance` schema so uuid, components, auto-secrets, and standard labels derive in CUE.

#### Scenario: Module directory loads as a whole CUE package

- **WHEN** synthesis is called with a module-package directory containing multiple `.cue` files in the same package
- **THEN** all files in the package SHALL be loaded as a single CUE instance via the kernel's module-package loading
- **AND** no individual file path SHALL be accepted as the synthesis input

#### Scenario: No synthetic wrapper module

- **WHEN** synthesis runs
- **THEN** no temporary CUE module (synthetic `cue.mod/module.cue`) SHALL be created
- **AND** no files SHALL be created, modified, or left behind inside the module directory

#### Scenario: One CUE context

- **WHEN** the module value and synthesized instance are composed
- **THEN** both SHALL be produced from the kernel's single `*cue.Context`

### Requirement: The produced instance has kind `ModuleInstance` and the synthesis does not apply `#ModuleRelease`

The produced instance SHALL have `kind: "ModuleInstance"`; the synthesis SHALL NOT apply `#ModuleRelease` and SHALL NOT import `opmodel.dev/core/v1alpha1/modulerelease@v1`.

#### Scenario: Emitted kind is ModuleInstance

- **WHEN** `opm module build` or `opm module apply` synthesizes and renders
- **THEN** the built instance SHALL carry `kind: "ModuleInstance"`
- **AND** no production code path SHALL reference `#ModuleRelease`

### Requirement: The CLI synthesizes a concrete instance from a published module through the kernel and writes nothing to disk but CUE's module cache

The CLI SHALL synthesize a concrete instance from a published module without a local copy of its source: the module is acquired from the registry through the kernel (`AcquireModuleFromRegistry`) at the version the `published-module-resolution` capability selects, then passed to kernel `SynthesizeInstance` exactly as a module acquired from a directory is. The synthesis SHALL write nothing to disk other than CUE's own module cache.

#### Scenario: Published module synthesizes

- **WHEN** `opm module build opmodel.dev/modules/web_app --version 1.0.4` runs
- **THEN** the built instance SHALL carry `kind: "ModuleInstance"` and embed version `1.0.4` of the module
- **AND** no file SHALL be created in the working directory

### Requirement: Values selection, metadata defaults and the output banner of a published module are the same as for a module-package directory

Values selection, metadata defaults and the output banner SHALL be the same as for a module-package directory.
(Requirements: The CLI synthesizes a concrete instance from a published module through the kernel and writes nothing to disk but CUE's module cache; Values selection mirrors `opm module vet`; Synthetic instance metadata defaults; Output banner distinguishes synthetic builds)

#### Scenario: debugValues of a published module

- **WHEN** a published module is synthesized with no `-f` flag and the module defines `debugValues`
- **THEN** `debugValues` SHALL be rendered as the single values source, attributed to the module's `debugValues`
