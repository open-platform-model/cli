## Purpose

Defines how the CLI synthesizes a concrete `#ModuleInstance` directly from a module CUE package directory — without requiring an authored `instance.cue`. The synthesis path lets module authors render their module to manifests using the module's own `debugValues` (or `-f` overrides), matching `cue eval` / `cue vet` ergonomics for the inner-loop.

## Requirements

### Requirement: Synthesize a `#ModuleInstance` from a module-package directory

The CLI SHALL synthesize a concrete instance from a module CUE package directory without requiring an `instance.cue` file, via kernel `SynthesizeInstance`. The synthesis SHALL load the module as a whole CUE package (matching `cue eval`/`cue vet` semantics) and pass it, with resolved values and synthetic metadata, to the kernel; the kernel unifies against the resolved `#ModuleInstance` schema so uuid, components, auto-secrets, and standard labels derive in CUE. The produced instance SHALL have `kind: "ModuleInstance"` — the synthesis SHALL NOT apply `#ModuleRelease` and SHALL NOT import `opmodel.dev/core/v1alpha1/modulerelease@v1`.

#### Scenario: Module directory loads as a whole CUE package

- **WHEN** synthesis is called with a module-package directory containing multiple `.cue` files in the same package
- **THEN** all files in the package SHALL be loaded as a single CUE instance via the kernel's module-package loading
- **AND** no individual file path SHALL be accepted as the synthesis input

#### Scenario: Emitted kind is ModuleInstance

- **WHEN** `opm module build` or `opm instance build <dir>` synthesizes and renders
- **THEN** the built instance SHALL carry `kind: "ModuleInstance"`
- **AND** no production code path SHALL reference `#ModuleRelease`

#### Scenario: No synthetic wrapper module

- **WHEN** synthesis runs
- **THEN** no temporary CUE module (synthetic `cue.mod/module.cue`) SHALL be created
- **AND** no files SHALL be created, modified, or left behind inside the module directory

#### Scenario: One CUE context

- **WHEN** the module value and synthesized instance are composed
- **THEN** both SHALL be produced from the kernel's single `*cue.Context`

### Requirement: Values selection mirrors `opm module vet`

The synthesis SHALL select values to fill the module's `#config` from the same sources `opm module vet` uses: `-f`/`--values` files merged in declaration order, falling back to the module's `debugValues` field when no `-f` flags are given.

#### Scenario: Values from `-f` flags

- **WHEN** the user invokes synthesis with one or more `-f` files
- **THEN** those files SHALL be loaded via `loader.LoadValuesFile` and unified in declaration order
- **AND** the resulting value SHALL be passed to `LoadModuleInstanceFromValue` as the values to fill into `#config`

#### Scenario: `debugValues` fallback

- **WHEN** the user invokes synthesis with no `-f` flag and the module defines a `debugValues` field
- **THEN** `debugValues` SHALL be used as the values input

#### Scenario: Neither values flag nor `debugValues`

- **WHEN** the user invokes synthesis with no `-f` flag and the module does not define `debugValues`
- **THEN** the CLI SHALL return an actionable error stating that the module must define `debugValues` or values must be supplied with `-f`

### Requirement: Synthetic instance metadata defaults

The synthesis SHALL produce a `metadata.name` and `metadata.namespace` for the synthetic `#ModuleInstance` even when the user has not supplied them.

#### Scenario: Default name derived from module metadata

- **WHEN** the user does not pass `--name`
- **THEN** the synthetic `metadata.name` SHALL be `"<module.metadata.name>-debug"`

#### Scenario: Default namespace

- **WHEN** the user does not pass `--namespace`
- **THEN** the synthetic `metadata.namespace` SHALL be `"default"`

#### Scenario: Flag overrides

- **WHEN** the user passes `--name <n>` or `--namespace <ns>` (or both)
- **THEN** the synthetic `metadata.name` and/or `metadata.namespace` SHALL take the flag values

### Requirement: Output banner distinguishes synthetic builds

The render output for a synthetic-instance build SHALL be visually distinguishable from a real instance-file build.

#### Scenario: Banner printed before render output

- **WHEN** synthesis runs and proceeds to render
- **THEN** the CLI SHALL print a banner naming the synthetic instance and the source module before the render output (e.g., `Building synthetic instance "<name>" for module "<module.metadata.name>"`)

### Requirement: Bundle release synthesis is not supported

The synthesis SHALL only produce `#ModuleInstance` values. Bundle directories or bundle-shaped inputs SHALL be rejected with a clear error.

#### Scenario: Bundle directory rejected

- **WHEN** the synthesis input directory contains a `#Bundle`/`#BundleRelease`-shaped CUE package instead of a `#Module`
- **THEN** the CLI SHALL return an error stating that bundle synthesis is not supported and pointing the user to `opm release build <file>` for bundle release files (when supported)
