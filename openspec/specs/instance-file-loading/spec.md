## Purpose

Defines the contract for loading standalone `.cue` instance files and turning them into raw parse data for later module instance preparation.

## Requirements

### Requirement: Instance file loader lives in `pkg/loader/`

A `LoadInstanceFile()` function in `pkg/loader/` (file: `pkg/loader/instance_file.go`) SHALL load a `.cue` file, evaluate it with CUE import resolution, and return the evaluated CUE value plus the resolve directory.

```go
func LoadInstanceFile(ctx *cue.Context, filePath string, opts LoadOptions) (cue.Value, string, error)
```

#### Scenario: Load instance file with registry import
- **WHEN** `LoadInstanceFile()` is called with a `.cue` file that imports a module from a registry
- **AND** the file defines `core.#ModuleInstance & { #module: importedModule, ... }`
- **THEN** the loader SHALL resolve the import, evaluate the CUE, and return the instance value with `#module` filled

#### Scenario: Load instance file with invalid CUE
- **WHEN** `LoadInstanceFile()` is called with a `.cue` file containing syntax errors
- **THEN** the loader SHALL return an error describing the CUE parse/evaluation failure

### Requirement: Internal instance-file inspection returns raw parse data

An internal `GetInstanceFile()` function in `internal/instancefile/` SHALL load and inspect an absolute `instance.cue` path, detect whether it represents a `ModuleInstance` or `BundleRelease`, and return raw parse data suitable for input to `module.ParseModuleInstance`. It SHALL NOT construct a fully prepared `*module.Instance`.

The `FileRelease` struct SHALL carry the raw instance spec `cue.Value`, best-effort module metadata, and the detected kind. It SHALL NOT carry a `*module.Instance` — instance construction is the responsibility of `ParseModuleInstance` after the caller has resolved module information and values.

```go
func GetInstanceFile(ctx *cue.Context, filePath string) (*FileRelease, error)
```

#### Scenario: Load ModuleInstance file returns raw parse data
- **WHEN** `GetInstanceFile()` is called with a `.cue` file containing `kind: "ModuleInstance"`
- **THEN** `FileRelease.Kind` SHALL be `KindModuleInstance`
- **AND** `FileRelease` SHALL carry the raw instance spec `cue.Value`
- **AND** `FileRelease` SHALL carry best-effort module info (metadata, config) extracted from the spec
- **AND** `FileRelease` SHALL NOT carry a `*module.Instance`

#### Scenario: Load BundleRelease file
- **WHEN** `GetInstanceFile()` is called with a `.cue` file containing `kind: "BundleRelease"`
- **THEN** `FileRelease.Kind` SHALL be `KindBundleRelease`
- **AND** `FileRelease.Bundle` SHALL be a `*bundle.Release`

#### Scenario: Load instance file without `#module` filled
- **WHEN** `LoadInstanceFile()` is called with a `.cue` file where `#module` is not filled (left open)
- **THEN** the loader SHALL return the partially evaluated instance value without error
- **AND** `GetInstanceFile()` SHALL still return raw parse data

#### Scenario: #module not filled produces error in render pipeline
- **WHEN** the instance file does not import or fill `#module`
- **THEN** the render pipeline SHALL exit with an error indicating `#module` is not filled and the user must import a module

#### Scenario: Instance file with unrecognised kind
- **WHEN** `GetInstanceFile()` is called for an instance file whose kind is absent or unrecognised
- **THEN** it SHALL return an error describing the unsupported kind

### Requirement: Instance metadata must be concrete during parse-only extraction

The instance file's computed fields such as `metadata.uuid` and merged metadata labels SHALL be concrete and decodable during `GetInstanceFile()` so parse-time extraction can populate the authoritative Go metadata before later processing begins.

#### Scenario: UUID is available during parse-only extraction
- **WHEN** a module instance is parsed from an instance file whose metadata is fully concrete
- **THEN** `GetInstanceFile()` SHALL decode the concrete `metadata` into the returned raw parse data

#### Scenario: Parse-only extraction fails when instance metadata is not concrete
- **WHEN** `GetInstanceFile()` parses an instance file whose computed metadata depends on unresolved inputs and is therefore not concrete
- **THEN** it SHALL return an error describing that instance metadata must be concrete

### Requirement: Workflow orchestration calls ParseModuleInstance then ProcessModuleInstance

The `internal/workflow/render.FromInstanceFile` function SHALL orchestrate the pipeline as:
1. Load the instance file via `GetInstanceFile`
2. Resolve values from instance file and `--values` flags
3. Build a `module.Module` from available module data
4. Call `module.ParseModuleInstance(ctx, spec, mod, values)` to get `*module.Instance`
5. Apply namespace override if needed (on `Instance.Metadata.Namespace`)
6. Load the provider
7. Call `render.ProcessModuleInstance(ctx, instance, provider)` to get `*render.ModuleResult`
8. Convert to workflow result

#### Scenario: Full pipeline from instance file
- **WHEN** `FromInstanceFile` is called with valid options
- **THEN** it SHALL call `ParseModuleInstance` before `ProcessModuleInstance`
- **AND** `ParseModuleInstance` SHALL receive the raw spec (with `#module` filled), the module, and resolved values
- **AND** `ProcessModuleInstance` SHALL receive the prepared `*module.Instance` and provider
