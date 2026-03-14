## REMOVED Requirements

### Requirement: Release file with `--module` override scenario

**Reason**: The `--module` flag is removed. Module resolution is exclusively via CUE imports.

**Migration**: Use CUE imports in the release file to fill `#module`.

## MODIFIED Requirements

### Requirement: `LoadModulePackage()` loads a local module CUE package

The `pkg/loader` package SHALL export `LoadModulePackage()` in `pkg/loader/release_file.go`. This function loads a module CUE package from a local directory and returns the raw `cue.Value`. It is used by `opm module vet` to load a module from a directory path.

```go
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error)
```

#### Scenario: Local module loaded for module vet

- **WHEN** `LoadModulePackage()` is called with a valid module directory
- **THEN** it returns the evaluated `cue.Value` of the module package
- **AND** the caller can use it for module-level validation
