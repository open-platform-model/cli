## MODIFIED Requirements

### Requirement: Loader validates consumer values and produces a concrete ModuleRelease

The `pkg/loader` package SHALL provide the full pipeline from CUE file loading through to a validated, concrete `*modulerelease.ModuleRelease`. There is no separate builder phase — loading IS building, consistent with the `promote-factory-engine` architecture.

The loader SHALL support three loading entry points:

1. **Module-directory path** (`LoadReleasePackage` + `LoadModuleReleaseFromValue`): used by `opm mod` commands. Accepts a directory containing `release.cue` + `values.cue`.
2. **Standalone release file** (`LoadReleaseFile` + `LoadModuleReleaseFromValue`): used by `opm release` commands. Accepts a single `.cue` file with CUE import resolution.
3. **Module-package synthesis** (`SynthesizeModuleReleaseFromPackage` + `LoadModuleReleaseFromValue`): used by `opm release build <dir>` and `opm module build`. Accepts a directory containing a module CUE package (no `release.cue`), composes a `#ModuleRelease` wrapper from a synthetic CUE module pinned at the user-module's catalog version, and feeds the result into the same downstream pipeline.

All three paths feed into the same `LoadModuleReleaseFromValue()` function which runs the Module Gate (validate values against `#module.#config`), concreteness check, metadata extraction, and value finalization.

#### Scenario: Successful load from module directory (existing behavior)

- **WHEN** `LoadReleasePackage()` is called with a module directory containing `release.cue` and `values.cue`
- **THEN** it returns a concrete `cue.Value` ready for `LoadModuleReleaseFromValue()`
- **AND** `LoadModuleReleaseFromValue()` returns a `*ModuleRelease` with all fields populated

#### Scenario: Successful load from release file

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file where `#module` is already filled via CUE import
- **THEN** it returns a concrete `cue.Value`
- **AND** `LoadModuleReleaseFromValue()` returns a `*ModuleRelease` with all fields populated (including auto-secrets handled by CUE `#AutoSecrets`)

#### Scenario: Successful synthesis from a module-package directory

- **WHEN** `SynthesizeModuleReleaseFromPackage()` is called with a module-package directory and either `-f` values or the module's `debugValues`
- **THEN** it returns a `cue.Value` shaped as a `#ModuleRelease` with `#module` filled by the loaded module value and `metadata.name`/`metadata.namespace` filled with caller-supplied or default synthetic values
- **AND** `LoadModuleReleaseFromValue()` returns a `*ModuleRelease` with all fields populated (including auto-secrets handled by CUE `#AutoSecrets`)

#### Scenario: Module Gate catches type mismatch

- **WHEN** consumer values contain a field with the wrong type
- **THEN** `LoadModuleReleaseFromValue()` returns a `*ConfigError` with structured `FieldError` details

#### Scenario: Auto-secrets are handled by CUE (no Go injection)

- **WHEN** a module's `#config` contains `#Secret` fields and concrete secret values are provided
- **THEN** the CUE `#AutoSecrets` mechanism in the loader automatically discovers and groups secrets
- **AND** the resulting `*ModuleRelease` contains the `opm-secrets` component
- **AND** no Go-side auto-secrets injection code is required

## ADDED Requirements

### Requirement: Synthesis path resolves the catalog dep through the registry

The synthesis entry point SHALL declare `opmodel.dev/core/v1alpha1@v1` as a dependency in the synthetic wrapper's `cue.mod/module.cue` and SHALL let CUE's loader resolve it via the standard registry/cache path (`CUE_REGISTRY` + `~/.cache/cuelang/mod`). The user's module SHALL NOT be required to add `opmodel.dev/core/v1alpha1/modulerelease@v1` as its own dependency.

#### Scenario: Synthesis works for an unpublished local module

- **WHEN** the user invokes `SynthesizeModuleReleaseFromPackage()` with a module that has never been published to a registry but does declare `opmodel.dev/core/v1alpha1@v1` in its own `cue.mod/module.cue`
- **THEN** synthesis SHALL succeed
- **AND** the user's module package SHALL be loaded directly from disk (not via the registry)

#### Scenario: Synthesis reuses the local CUE module cache

- **WHEN** synthesis runs a second time for the same catalog version
- **THEN** the catalog dep SHALL be served from `~/.cache/cuelang/mod` without a network fetch

### Requirement: Synth wrapper and user module load the same catalog version

The synthesis SHALL guarantee that the synthetic wrapper and the user's module resolve `opmodel.dev/core/v1alpha1@v1` to the same registry artifact, so that the `#Module` definition the user satisfies is the same `#Module` definition the wrapper's `#ModuleRelease.#module!: module.#Module` constraint expects.

#### Scenario: Catalog version copied from user modfile

- **WHEN** the user's `cue.mod/module.cue` pins `opmodel.dev/core/v1alpha1@v1` at version `vX.Y.Z`
- **THEN** the synthetic wrapper's `cue.mod/module.cue` SHALL pin the same dep at the same `vX.Y.Z`

#### Scenario: User module declares no catalog dep

- **WHEN** the user's `cue.mod/module.cue` declares no `opmodel.dev/core/v1alpha1@v1` dep
- **THEN** synthesis SHALL return a `DetailError` instructing the user to add the dep
