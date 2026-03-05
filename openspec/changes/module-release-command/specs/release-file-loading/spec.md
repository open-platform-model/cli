## ADDED Requirements

### Requirement: Release file loader detects release type and returns it alongside the value

A `LoadRelease()` function in `internal/loader/` SHALL load a `.cue` file, evaluate it, detect whether the top-level value is a `#ModuleRelease` or `#BundleRelease`, and return both the evaluated CUE value and a `ReleaseType` discriminator. The loader SHALL use `load.Instances()` so that CUE imports (including registry module imports) are resolved.

```go
// ReleaseType identifies the kind of release defined in a .cue file.
type ReleaseType int

const (
    ModuleRelease ReleaseType = iota
    BundleRelease
)

func LoadRelease(ctx *cue.Context, filePath string, registry string) (cue.Value, ReleaseType, string, error)
```

Type detection SHALL be performed by reading the `kind` field from the evaluated CUE value. Both `#ModuleRelease` (`kind: "ModuleRelease"`) and `#BundleRelease` (`kind: "BundleRelease"`) define `kind` as a concrete string literal in the OPM catalog.

#### Scenario: Load ModuleRelease file

- **WHEN** `LoadRelease()` is called with a `.cue` file containing `kind: "ModuleRelease"`
- **THEN** the loader SHALL return the evaluated value and `ReleaseType` of `ModuleRelease`

#### Scenario: Load BundleRelease file

- **WHEN** `LoadRelease()` is called with a `.cue` file containing `kind: "BundleRelease"`
- **THEN** the loader SHALL return the evaluated value and `ReleaseType` of `BundleRelease`

#### Scenario: Load release file with registry import

- **WHEN** `LoadRelease()` is called with a `.cue` file that imports a module from a registry
- **AND** the file defines `core.#ModuleRelease & { #module: importedModule, ... }`
- **THEN** the loader SHALL resolve the import, evaluate the CUE, and return the release value with `#module` filled

#### Scenario: Load release file without #module filled

- **WHEN** `LoadRelease()` is called with a `.cue` file where `#module` is not filled (left open)
- **THEN** the loader SHALL return the partially evaluated release value
- **AND** the caller SHALL be responsible for filling `#module` via `FillPath` (using `--module` flag)

#### Scenario: Load release file with invalid CUE

- **WHEN** `LoadRelease()` is called with a `.cue` file containing syntax errors
- **THEN** the loader SHALL return an error describing the CUE parse/evaluation failure

#### Scenario: Release file with unrecognised kind

- **WHEN** `LoadRelease()` is called with a `.cue` file where `kind` is absent or not a recognised value
- **THEN** the loader SHALL return an error: `"release file does not define a recognised release type (expected ModuleRelease or BundleRelease)"`

### Requirement: Release file naming convention

Release files SHALL follow the naming convention `<name>_release.cue` where `<name>` is the release identifier in kebab-case or snake_case. This is a convention, not enforced — the CLI SHALL accept any `.cue` file path.

#### Scenario: Conventional naming accepted

- **WHEN** `opm release build jellyfin_release.cue` is run
- **THEN** the CLI SHALL load and process the file normally

#### Scenario: Non-conventional naming accepted

- **WHEN** `opm release build my-custom-release-file.cue` is run
- **THEN** the CLI SHALL load and process the file normally without warning

### Requirement: Module injection via --module flag uses FillPath

When the `--module` flag is provided, the CLI SHALL load the module from the specified directory using `loader.LoadModule()` and inject it into the release value via `FillPath` at the `#module` path. This SHALL use the same `FillPath` mechanism as the existing `builder.Build()`.

#### Scenario: FillPath injection with --module

- **WHEN** a release file has `#module` unfilled
- **AND** `--module ./jellyfin` is provided
- **THEN** the CLI SHALL call `loader.LoadModule("./jellyfin")`
- **AND** fill `#module` with the loaded module's `Raw` CUE value via `FillPath`

#### Scenario: Module already imported, --module flag provided

- **WHEN** a release file imports and fills `#module` from a registry
- **AND** `--module ./jellyfin` is also provided
- **THEN** the `--module` flag SHALL take precedence and override the imported module

### Requirement: Release file pipeline skips module loading phase when #module is filled

When a release file is loaded with `#module` already filled (via import), the render pipeline SHALL skip the PREPARATION phase (module loading) and use the release's embedded module directly. The BUILD phase SHALL use the pre-filled release value rather than constructing one from scratch.

#### Scenario: Registry-imported module skips loader

- **WHEN** `opm release build release.cue` is run with a release file that imports its module
- **THEN** the pipeline SHALL NOT call `loader.LoadModule()`
- **AND** the pipeline SHALL proceed directly to validation, matching, and generation

#### Scenario: --module flag triggers module loading

- **WHEN** `opm release build release.cue --module ./path` is run
- **THEN** the pipeline SHALL call `loader.LoadModule("./path")`
- **AND** inject the result into the release before proceeding

### Requirement: Release metadata is computed by CUE evaluation

The release file's `metadata.uuid`, `metadata.labels`, and computed fields SHALL be derived by CUE evaluation of the `#ModuleRelease` definition, not by Go code. This is consistent with the existing builder behavior.

#### Scenario: UUID computed from release file

- **WHEN** a release file specifies `metadata.name: "jellyfin"` and `metadata.namespace: "media"`
- **AND** `#module` provides the module FQN
- **THEN** `metadata.uuid` SHALL be the deterministic UUID5 computed by CUE (SHA1 of OPMNamespace + FQN:name:namespace)

#### Scenario: Labels merged from module and release

- **WHEN** a release file is evaluated
- **THEN** `metadata.labels` SHALL contain both module-level labels and release-level OPM labels (`module-release.opmodel.dev/name`, `module-release.opmodel.dev/uuid`)
