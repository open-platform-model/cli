# Module Synthetic Release

## Purpose

Defines how the CLI synthesizes a concrete `#ModuleRelease` directly from a module CUE package directory — without requiring an authored `release.cue`. The synthesis path lets module authors render their module to manifests using the module's own `debugValues` (or `-f` overrides), matching `cue eval` / `cue vet` ergonomics for the inner-loop.

## Requirements

### Requirement: Synthesize a `#ModuleRelease` from a module-package directory

The CLI SHALL synthesize a concrete `*ModuleRelease` from a module CUE package directory without requiring a `release.cue` file. The synthesis SHALL load the module as a whole CUE package (matching `cue eval`/`cue vet` semantics) and SHALL feed the produced `cue.Value` into the same downstream pipeline (`pkg/loader.LoadModuleReleaseFromValue`) used for release files and release packages.

#### Scenario: Module directory loads as a whole CUE package

- **WHEN** the synthesis function is called with a module-package directory containing multiple `.cue` files in the same package (e.g., `module.cue`, `components.cue`)
- **THEN** all files in the package SHALL be unified into a single CUE instance via `load.Instances(["."], &load.Config{Dir: modulePath})`
- **AND** no individual file path SHALL be accepted as the synthesis input

#### Scenario: `#ModuleRelease` schema is sourced via the registry

- **WHEN** the synthesis builds the wrapper
- **THEN** `#ModuleRelease` (and its transitive `core/v1alpha1` packages) SHALL be obtained from the OPM catalog by importing `mr "opmodel.dev/core/v1alpha1/modulerelease@v1"` in a small synthetic CUE module
- **AND** the wrapper SHALL apply `mr.#ModuleRelease` at the top level (the same shape used by real `releases/<env>/<module>/release.cue` files)
- **AND** the catalog dep in the synthetic `cue.mod/module.cue` SHALL be `opmodel.dev/core/v1alpha1@v1` (whole module, matching the pin shape used by real release modfiles)
- **AND** the catalog dep SHALL be resolved through CUE's standard registry/cache machinery (`CUE_REGISTRY` env + `~/.cache/cuelang/mod`)
- **AND** the synthesis SHALL NOT require the user's module to declare `opmodel.dev/core/v1alpha1/modulerelease@v1` as a dependency
- **AND** the synthesis SHALL NOT carry an embedded copy of the catalog schemas in the CLI binary

#### Scenario: Synth wrapper pins catalog at the user module's version

- **WHEN** the user's module's `cue.mod/module.cue` pins `opmodel.dev/core/v1alpha1@v1` at `vX.Y.Z`
- **THEN** the synthesis SHALL parse the user modfile via `mod/modfile.Parse` and reuse `vX.Y.Z` as the synth wrapper's pin for the same dep

#### Scenario: User module and synthetic wrapper share one CUE context

- **WHEN** the user's module value and the synthetic wrapper value are composed
- **THEN** both values SHALL be produced from the same `*cue.Context`
- **AND** composition SHALL use `Value.Unify` and `Value.FillPath`, not string-based CUE source generation

#### Scenario: No filesystem writes inside the user's module directory

- **WHEN** the synthesis runs
- **THEN** no files SHALL be created, modified, or left behind inside the module directory or its `cue.mod/`
- **AND** any temporary anchor directory used by the loader SHALL be removed before the command returns

### Requirement: Values selection mirrors `opm module vet`

The synthesis SHALL select values to fill the module's `#config` from the same sources `opm module vet` uses: `-f`/`--values` files merged in declaration order, falling back to the module's `debugValues` field when no `-f` flags are given.

#### Scenario: Values from `-f` flags

- **WHEN** the user invokes synthesis with one or more `-f` files
- **THEN** those files SHALL be loaded via `loader.LoadValuesFile` and unified in declaration order
- **AND** the resulting value SHALL be passed to `LoadModuleReleaseFromValue` as the values to fill into `#config`

#### Scenario: `debugValues` fallback

- **WHEN** the user invokes synthesis with no `-f` flag and the module defines a `debugValues` field
- **THEN** `debugValues` SHALL be used as the values input

#### Scenario: Neither values flag nor `debugValues`

- **WHEN** the user invokes synthesis with no `-f` flag and the module does not define `debugValues`
- **THEN** the CLI SHALL return an actionable error stating that the module must define `debugValues` or values must be supplied with `-f`

### Requirement: Synthetic release metadata defaults

The synthesis SHALL produce a `metadata.name` and `metadata.namespace` for the synthetic `#ModuleRelease` even when the user has not supplied them.

#### Scenario: Default name derived from module metadata

- **WHEN** the user does not pass `--name`
- **THEN** the synthetic `metadata.name` SHALL be `"<module.metadata.name>-debug"`

#### Scenario: Default namespace

- **WHEN** the user does not pass `--namespace`
- **THEN** the synthetic `metadata.namespace` SHALL be `"default"`

#### Scenario: Flag overrides

- **WHEN** the user passes `--name <n>` or `--namespace <ns>` (or both)
- **THEN** the synthetic `metadata.name` and/or `metadata.namespace` SHALL take the flag values

### Requirement: User module must declare a catalog dep

The synthesis SHALL require the user's module to declare `opmodel.dev/core/v1alpha1@v1` in its `cue.mod/module.cue` so that the synth wrapper can pin the same catalog version.

#### Scenario: Catalog dep present

- **WHEN** the user's module pins `opmodel.dev/core/v1alpha1@v1`
- **THEN** synthesis SHALL proceed using the same version pin

#### Scenario: Catalog dep missing

- **WHEN** the user's module declares no `opmodel.dev/core/v1alpha1@v1` dep
- **THEN** the CLI SHALL return an actionable error instructing the user to add the dep before building

### Requirement: Output banner distinguishes synthetic builds

The render output for a synthetic-release build SHALL be visually distinguishable from a real release-file build.

#### Scenario: Banner printed before render output

- **WHEN** synthesis runs and proceeds to render
- **THEN** the CLI SHALL print a banner naming the synthetic release and the source module before the render output (e.g., `Building synthetic release "<name>" for module "<module.metadata.name>"`)

### Requirement: Bundle release synthesis is not supported

The synthesis SHALL only produce `#ModuleRelease` values. Bundle directories or bundle-shaped inputs SHALL be rejected with a clear error.

#### Scenario: Bundle directory rejected

- **WHEN** the synthesis input directory contains a `#Bundle`/`#BundleRelease`-shaped CUE package instead of a `#Module`
- **THEN** the CLI SHALL return an error stating that bundle synthesis is not supported and pointing the user to `opm release build <file>` for bundle release files (when supported)
