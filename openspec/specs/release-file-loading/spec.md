## Purpose

Defines the contract for loading standalone `.cue` release files via `pkg/loader`. Release files are self-contained CUE files that declare a `#ModuleRelease` or `#BundleRelease` with inline values and optional CUE import references to modules from a registry. This loading path is used by `opm release` commands and contrasts with the module-directory path used by `opm mod` commands.

## Requirements

### Requirement: Release file loader lives in `pkg/loader/`

A `LoadReleaseFile()` function in `pkg/loader/` (file: `pkg/loader/release_file.go`) SHALL load a `.cue` file, evaluate it with CUE import resolution, and return the evaluated CUE value plus the resolve directory. Type detection is performed by the existing `DetectReleaseKind()` function which is already in `pkg/loader/module_release.go`.

```go
// LoadReleaseFile loads a #ModuleRelease or #BundleRelease from a standalone .cue file.
// CUE imports (including registry module references) are resolved via load.Instances()
// using the file's parent directory for cue.mod resolution.
func LoadReleaseFile(ctx *cue.Context, filePath string, registry string) (cue.Value, string, error)
```

The `DetectReleaseKind()` function is called by the caller after `LoadReleaseFile()` returns:

```go
releaseVal, resolveDir, err := loader.LoadReleaseFile(cueCtx, filePath, registry)
// ...
kind, err := loader.DetectReleaseKind(releaseVal)  // already exists in pkg/loader
```

#### Scenario: Load ModuleRelease file

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file containing `kind: "ModuleRelease"`
- **THEN** the loader SHALL return the evaluated value and resolve directory
- **AND** a subsequent call to `DetectReleaseKind()` SHALL return `"ModuleRelease"`

#### Scenario: Load BundleRelease file

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file containing `kind: "BundleRelease"`
- **THEN** the loader SHALL return the evaluated value
- **AND** a subsequent call to `DetectReleaseKind()` SHALL return `"BundleRelease"`

#### Scenario: Load release file with registry import

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file that imports a module from a registry
- **AND** the file defines `core.#ModuleRelease & { #module: importedModule, ... }`
- **THEN** the loader SHALL resolve the import, evaluate the CUE, and return the release value with `#module` filled

#### Scenario: Load release file without `#module` filled

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file where `#module` is not filled (left open)
- **THEN** the loader SHALL return the partially evaluated release value without error
- **AND** the caller SHALL be responsible for filling `#module` via `FillPath` (using `--module` flag)

#### Scenario: Load release file with invalid CUE

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file containing syntax errors
- **THEN** the loader SHALL return an error describing the CUE parse/evaluation failure

#### Scenario: Release file with unrecognised kind

- **WHEN** `LoadReleaseFile()` returns a value where `kind` is absent or unrecognised
- **AND** `DetectReleaseKind()` is called on that value
- **THEN** `DetectReleaseKind()` SHALL return an error: `"unknown release kind: ..."` or `"release package has no 'kind' field"`

### Requirement: Release file naming convention

Release files SHALL follow the naming convention `<name>_release.cue` where `<name>` is the release identifier in kebab-case or snake_case. This is a convention, not enforced — the CLI SHALL accept any `.cue` file path.

#### Scenario: Conventional naming accepted

- **WHEN** `opm release build jellyfin_release.cue` is run
- **THEN** the CLI SHALL load and process the file normally

#### Scenario: Non-conventional naming accepted

- **WHEN** `opm release build my-custom-release-file.cue` is run
- **THEN** the CLI SHALL load and process the file normally without warning

### Requirement: Module injection via `--module` flag uses `LoadModulePackage()` + FillPath

When the `--module` flag is provided, the CLI SHALL load the module from the specified directory using `pkg/loader.LoadModulePackage()` and inject it into the release value via `FillPath` at the `#module` path. This replaces the deleted `loader.LoadModule()`.

```go
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error)
```

#### Scenario: FillPath injection with `--module`

- **WHEN** a release file has `#module` unfilled
- **AND** `--module ./jellyfin` is provided
- **THEN** the CLI SHALL call `loader.LoadModulePackage("./jellyfin")`
- **AND** fill `#module` with the returned CUE value via `releaseVal.FillPath(cue.MakePath(cue.Def("module")), modVal)`

#### Scenario: Module already imported, `--module` flag provided

- **WHEN** a release file imports and fills `#module` from a registry
- **AND** `--module ./jellyfin` is also provided
- **THEN** the `--module` flag SHALL take precedence — FillPath overwrites the imported value

### Requirement: Release file uses `LoadModuleReleaseFromValue()` for validation and extraction

After loading and optional `#module` injection, the CLI SHALL call the existing `pkg/loader.LoadModuleReleaseFromValue()` to run the Module Gate, concreteness check, metadata extraction, and value finalization. This function is agnostic to how the CUE value was loaded — it works identically for both module-directory and release-file inputs.

#### Scenario: Registry-imported module goes through Module Gate

- **WHEN** `opm release build release.cue` is run with a release file that imports its module
- **THEN** `LoadReleaseFile()` returns the evaluated value with `#module` filled
- **AND** `LoadModuleReleaseFromValue()` runs the Module Gate (validates `values` against `#module.#config`)
- **AND** `LoadModuleReleaseFromValue()` extracts metadata, finalizes, and returns `*ModuleRelease`
- **AND** `loader.LoadModule()` is NOT called (it no longer exists)

#### Scenario: `--module` flag triggers local module loading then Module Gate

- **WHEN** `opm release build release.cue --module ./path` is run
- **THEN** `loader.LoadModulePackage("./path")` returns the module CUE value
- **AND** `FillPath` injects it into the release value
- **AND** `LoadModuleReleaseFromValue()` runs normally on the filled value

### Requirement: Release metadata is computed by CUE evaluation

The release file's `metadata.uuid`, `metadata.labels`, and computed fields SHALL be derived by CUE evaluation of the `#ModuleRelease` definition via `LoadModuleReleaseFromValue()`. This is consistent with the module-directory path.

#### Scenario: UUID computed from release file

- **WHEN** a release file specifies `metadata.name: "jellyfin"` and `metadata.namespace: "media"`
- **AND** `#module` provides the module FQN
- **THEN** `metadata.uuid` SHALL be the deterministic UUID5 computed by CUE

#### Scenario: Labels merged from module and release

- **WHEN** a release file is evaluated
- **THEN** `metadata.labels` SHALL contain both module-level labels and release-level OPM labels (`module-release.opmodel.dev/name`, `module-release.opmodel.dev/uuid`)
