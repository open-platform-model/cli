## Purpose

Defines the contract for loading standalone `.cue` release files and parsing them into barebones release objects for the release-processing pipeline.

## Requirements

### Requirement: Release file loader lives in `pkg/loader/`

A `LoadReleaseFile()` function in `pkg/loader/` (file: `pkg/loader/release_file.go`) SHALL load a `.cue` file, evaluate it with CUE import resolution, and return the evaluated CUE value plus the resolve directory. An internal parse-only `GetReleaseFile()` function SHALL load and inspect an absolute `release.cue` path, detect whether it represents a `ModuleRelease` or `BundleRelease`, and return a barebones release object without validating values. `GetReleaseFile()` SHALL require release metadata itself to be concrete, but it SHALL NOT require a concrete `#module` or `#bundle` reference unless the release metadata depends on it.

```go
func LoadReleaseFile(ctx *cue.Context, filePath string, registry string) (cue.Value, string, error)
func GetReleaseFile(filePath string) (*FileRelease, error)
```

The CLI MAY call `LoadReleaseFile()` and `GetReleaseFile()` as separate stages, but `GetReleaseFile()` is also allowed to perform its own file loading internally as part of parse-only release extraction.

#### Scenario: Load ModuleRelease file
- **WHEN** `LoadReleaseFile()` is called with a `.cue` file containing `kind: "ModuleRelease"`
- **THEN** the loader SHALL return the evaluated value and resolve directory
- **AND** `GetReleaseFile()` SHALL also be able to parse the same release file path into a barebones module release object

#### Scenario: Load BundleRelease file
- **WHEN** `LoadReleaseFile()` is called with a `.cue` file containing `kind: "BundleRelease"`
- **THEN** the loader SHALL return the evaluated value
- **AND** `GetReleaseFile()` SHALL also be able to parse the same release file path into a barebones bundle release object

#### Scenario: Load release file with registry import
- **WHEN** `LoadReleaseFile()` is called with a `.cue` file that imports a module from a registry
- **AND** the file defines `core.#ModuleRelease & { #module: importedModule, ... }`
- **THEN** the loader SHALL resolve the import, evaluate the CUE, and return the release value with `#module` filled

#### Scenario: Load release file without `#module` filled
- **WHEN** `LoadReleaseFile()` is called with a `.cue` file where `#module` is not filled (left open)
- **THEN** the loader SHALL return the partially evaluated release value without error
- **AND** `GetReleaseFile()` SHALL still return a barebones module release object
- **AND** the caller SHALL be responsible for later filling `#module` when required

#### Scenario: Load release file with invalid CUE
- **WHEN** `LoadReleaseFile()` is called with a `.cue` file containing syntax errors
- **THEN** the loader SHALL return an error describing the CUE parse/evaluation failure

#### Scenario: Release file with unrecognised kind
- **WHEN** `GetReleaseFile()` is called for a release file whose kind is absent or unrecognised
- **THEN** it SHALL return an error describing the unsupported release kind

### Requirement: Module injection via `--module` flag uses `LoadModulePackage()` + FillPath

When the `--module` flag is provided, the CLI SHALL load the module from the specified directory using `pkg/loader.LoadModulePackage()` and inject it into the module release's raw CUE value via `FillPath` at the `#module` path before module-release processing begins.

#### Scenario: FillPath injection with `--module`
- **WHEN** a release file has `#module` unfilled
- **AND** `--module ./jellyfin` is provided
- **THEN** the CLI SHALL call `loader.LoadModulePackage("./jellyfin")`
- **AND** fill `#module` on the raw release CUE value before processing the release

#### Scenario: Module already imported, `--module` flag provided
- **WHEN** a release file imports and fills `#module` from a registry
- **AND** `--module ./jellyfin` is also provided
- **THEN** the `--module` flag SHALL take precedence by overwriting the raw `#module` value before processing

### Requirement: Release metadata must be concrete during parse-only extraction

The release file's computed fields such as `metadata.uuid` and merged metadata labels SHALL be concrete and decodable during `GetReleaseFile()` so parse-time extraction can populate the authoritative Go metadata before later processing begins.

#### Scenario: UUID is available during parse-only extraction
- **WHEN** a module release is parsed from a release file whose metadata is fully concrete
- **THEN** `GetReleaseFile()` SHALL decode the concrete `metadata` into the returned release object

#### Scenario: Parse-only extraction fails when release metadata is not concrete
- **WHEN** `GetReleaseFile()` parses a release file whose computed metadata depends on unresolved inputs and is therefore not concrete
- **THEN** it SHALL return an error describing that release metadata must be concrete
