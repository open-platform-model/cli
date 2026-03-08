## MODIFIED Requirements

### Requirement: Loader validates consumer values and produces a concrete ModuleRelease

The `pkg/loader` package SHALL provide the full pipeline from CUE file loading through to a validated, concrete `*modulerelease.ModuleRelease`. There is no separate builder phase — loading IS building, consistent with the `promote-factory-engine` architecture.

The loader SHALL support two loading entry points:

1. **Module-directory path** (`LoadReleasePackage` + `LoadModuleReleaseFromValue`): used by `opm mod` commands. Accepts a directory containing `release.cue` + `values.cue`.
2. **Standalone release file** (`LoadReleaseFile` + `LoadModuleReleaseFromValue`): used by `opm release` commands. Accepts a single `.cue` file with CUE import resolution.

Both paths feed into the same `LoadModuleReleaseFromValue()` function which runs the Module Gate (validate values against `#module.#config`), concreteness check, metadata extraction, and value finalization.

#### Scenario: Successful load from module directory (existing behavior)

- **WHEN** `LoadReleasePackage()` is called with a module directory containing `release.cue` and `values.cue`
- **THEN** it returns a concrete `cue.Value` ready for `LoadModuleReleaseFromValue()`
- **AND** `LoadModuleReleaseFromValue()` returns a `*ModuleRelease` with all fields populated

#### Scenario: Successful load from release file

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file where `#module` is already filled via CUE import
- **THEN** it returns a concrete `cue.Value`
- **AND** `LoadModuleReleaseFromValue()` returns a `*ModuleRelease` with all fields populated (including auto-secrets handled by CUE `#AutoSecrets`)

#### Scenario: Release file with `--module` override

- **WHEN** `LoadReleaseFile()` returns a value where `#module` is not concrete
- **AND** the caller fills `#module` via `FillPath` using `LoadModulePackage()`
- **THEN** `LoadModuleReleaseFromValue()` successfully returns a `*ModuleRelease`

#### Scenario: Module Gate catches type mismatch

- **WHEN** consumer values contain a field with the wrong type
- **THEN** `LoadModuleReleaseFromValue()` returns a `*ConfigError` with structured `FieldError` details

#### Scenario: Auto-secrets are handled by CUE (no Go injection)

- **WHEN** a module's `#config` contains `#Secret` fields and concrete secret values are provided
- **THEN** the CUE `#AutoSecrets` mechanism in the loader automatically discovers and groups secrets
- **AND** the resulting `*ModuleRelease` contains the `opm-secrets` component
- **AND** no Go-side auto-secrets injection code is required

### Requirement: Value selection falls back to `values.cue` for module-directory builds

When using `LoadReleasePackage()` (module-directory path), the loader SHALL use `values.cue` from the module directory when no explicit values file is provided. If neither an explicit values file nor `values.cue` exists, the loader SHALL return an error.

When using `LoadReleaseFile()` (release-file path), the `values` field is inline in the release CUE file itself. There is no `values.cue` fallback — the release file is self-contained.

#### Scenario: No values file, `values.cue` exists in module directory

- **WHEN** `LoadReleasePackage()` is called with no explicit values file
- **AND** `values.cue` exists in the module directory
- **THEN** `values.cue` is loaded alongside `release.cue` as part of the CUE instance

#### Scenario: Release file is self-contained

- **WHEN** `LoadReleaseFile()` is called
- **THEN** the `values` field is read from the release CUE file's inline definition
- **AND** no `values.cue` file is searched for or loaded

#### Scenario: No values file, no `values.cue` (module-directory build)

- **WHEN** `LoadReleasePackage()` is called with no explicit values file
- **AND** no `values.cue` file exists in the module directory
- **THEN** the loader returns an error indicating values must be provided via `values.cue` or `--values`

## ADDED Requirements

### Requirement: `LoadReleaseFile()` loads a standalone `.cue` file with import resolution

The `pkg/loader` package SHALL export `LoadReleaseFile()` in `pkg/loader/release_file.go`. This function loads a standalone `.cue` release file using `load.Instances()` with the file's parent directory for `cue.mod` resolution, enabling CUE registry module imports.

```go
func LoadReleaseFile(ctx *cue.Context, filePath string, registry string) (cue.Value, string, error)
```

#### Scenario: Release file with registry import resolves successfully

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file that imports a module from `opmodel.dev/modules/jellyfin@v1`
- **AND** the file's parent directory contains a `cue.mod/module.cue` declaring the dependency
- **THEN** the import is resolved, the module is unified into `#module`, and the evaluated value is returned

#### Scenario: Release file without `cue.mod/` fails with clear error

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file in a directory with no `cue.mod/` ancestor
- **THEN** the loader returns an error describing the missing module configuration

### Requirement: `LoadModulePackage()` loads a local module for `--module` flag injection

The `pkg/loader` package SHALL export `LoadModulePackage()` in `pkg/loader/release_file.go`. This function loads a module CUE package from a local directory and returns the raw `cue.Value` for `FillPath` injection into a release value. This replaces the deleted `internal/loader.LoadModule()` for this specific use case.

```go
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error)
```

#### Scenario: Local module loaded for `--module` injection

- **WHEN** `LoadModulePackage()` is called with a valid module directory
- **THEN** it returns the evaluated `cue.Value` of the module package
- **AND** the caller can inject it via `releaseVal.FillPath(cue.MakePath(cue.Def("module")), modVal)`

### Requirement: `opm mod vet` uses `debugValues` by default

The `opm mod vet` command SHALL use the module's `debugValues` field as the values source when no `-f` flag is provided. The extraction SHALL happen in `internal/cmdutil/render.go` via the `DebugValues bool` field on `RenderReleaseOpts`. The loader SHALL expose a `LoadReleasePackageWithValue()` variant that accepts a pre-loaded `cue.Value` instead of a values file path.

#### Scenario: `debugValues` used when no `-f` flag

- **WHEN** `opm mod vet` is run without `-f` flags
- **THEN** `RenderRelease()` is called with `DebugValues: true`
- **AND** the module's `debugValues` field is extracted and used as the values source
- **AND** the vet output shows "debugValues" as the values source

#### Scenario: `-f` flag overrides `debugValues`

- **WHEN** `opm mod vet` is run with one or more `-f` flags
- **THEN** `DebugValues` is `false` and the explicit values files are used
- **AND** `debugValues` is ignored

#### Scenario: `debugValues` is `_` (unconstrained)

- **WHEN** `opm mod vet` is run without `-f` flags
- **AND** the module's `debugValues` field is `_` (open/unconstrained, not filled by the author)
- **THEN** `RenderRelease()` returns an error: "debugValues is not concrete — module must provide complete test values"
