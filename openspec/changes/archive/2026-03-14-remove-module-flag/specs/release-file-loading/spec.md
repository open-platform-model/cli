## REMOVED Requirements

### Requirement: Module injection via `--module` flag uses `LoadModulePackage()` + FillPath

**Reason**: The `--module` flag created 4 field mutations on a partially-constructed `*module.Release` before rendering. Removing it eliminates this mutation path and enforces a single module resolution path (CUE imports). The CLI is pre-1.0.

**Migration**: Use CUE imports in the release file to fill `#module`. For local development, set up a local CUE registry or use relative import paths.

## MODIFIED Requirements

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

#### Scenario: #module not filled produces error in render pipeline
- **WHEN** the release file does not import or fill `#module`
- **AND** no `--module` flag exists
- **THEN** the render pipeline SHALL exit with an error indicating `#module` is not filled and the user must import a module

#### Scenario: Load release file with invalid CUE
- **WHEN** `LoadReleaseFile()` is called with a `.cue` file containing syntax errors
- **THEN** the loader SHALL return an error describing the CUE parse/evaluation failure

#### Scenario: Release file with unrecognised kind
- **WHEN** `GetReleaseFile()` is called for a release file whose kind is absent or unrecognised
- **THEN** it SHALL return an error describing the unsupported release kind
