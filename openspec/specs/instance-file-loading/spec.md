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
