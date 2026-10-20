## Purpose

Defines how the CLI synthesizes a concrete `#ModuleInstance` directly from a module CUE package directory — without requiring an authored `instance.cue`. The synthesis path lets module authors render their module to manifests using the module's own `debugValues` (or `-f` overrides), matching `cue eval` / `cue vet` ergonomics for the inner-loop.

## Requirements

### Requirement: Values selection mirrors `opm module vet`

The synthesis SHALL select values to fill the module's `#config` from the same sources `opm module vet` uses, through the same resolver (`render.ResolveModuleValues`): `-f`/`--values` files as file-backed kernel values sources in declaration order, falling back to the module's `debugValues` rendered as one source when no `-f` flags are given.

#### Scenario: Values from `-f` flags

- **WHEN** the user invokes synthesis with one or more `-f` files
- **THEN** each file SHALL become a kernel values source attributed to that file, in declaration order
- **AND** the sources SHALL be validated against the module's `#config` through the kernel before synthesis, so a violation or a conflict is attributed to the file it came from
- **AND** the sources SHALL be passed to the kernel's `SynthesizeInstance` as the values to fill into `#config`

#### Scenario: `debugValues` fallback

- **WHEN** the user invokes synthesis with no `-f` flag and the module defines a `debugValues` field
- **THEN** `debugValues` SHALL be rendered as the single values source, attributed to the module's `debugValues`

#### Scenario: Neither values flag nor `debugValues`

- **WHEN** the user invokes synthesis with no `-f` flag and the module does not define `debugValues`
- **THEN** the CLI SHALL return an actionable error stating that the module must define `debugValues` or values must be supplied with `-f`

### Requirement: Synthetic instance metadata defaults

The synthesis SHALL produce a `metadata.name` and `metadata.namespace` for the synthetic `#ModuleInstance` even when the user has not supplied them. The default name SHALL be a valid instance name for every valid module name: a module name is a CUE package name, which may carry `_` but never `-`, while an instance name is a DNS label, which may carry `-` but never `_`.

#### Scenario: Default name derived from module metadata

- **WHEN** the user does not pass `--name`
- **THEN** the synthetic `metadata.name` SHALL be `"<module.metadata.name>-debug"` with every `_` in the module name replaced by `-`

#### Scenario: Multi-word module name

- **WHEN** a module named `my_app` is synthesized without `--name`
- **THEN** the synthetic `metadata.name` SHALL be `"my-app-debug"`
- **AND** synthesis SHALL NOT refuse the name

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
- **THEN** the CLI SHALL return an error stating that bundle synthesis is not supported and pointing the user to `opm instance build <file>` for bundle instance files (when supported)

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
