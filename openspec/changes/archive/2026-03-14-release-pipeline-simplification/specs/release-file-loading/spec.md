## MODIFIED Requirements

### Requirement: Release file loader lives in `pkg/loader/`

A `LoadReleaseFile()` function in `pkg/loader/` (file: `pkg/loader/release_file.go`) SHALL load a `.cue` file, evaluate it with CUE import resolution, and return the evaluated CUE value plus the resolve directory.

```go
func LoadReleaseFile(ctx *cue.Context, filePath string, opts LoadOptions) (cue.Value, string, error)
```

### Requirement: Internal release-file inspection returns raw parse data

An internal `GetReleaseFile()` function in `internal/releasefile/` SHALL load and inspect an absolute `release.cue` path, detect whether it represents a `ModuleRelease` or `BundleRelease`, and return raw parse data suitable for input to `module.ParseModuleRelease`. It SHALL NOT construct a fully prepared `*module.Release`.

The `FileRelease` struct SHALL carry the raw release spec `cue.Value`, best-effort module metadata, and the detected kind. It SHALL NOT carry a `*module.Release` — release construction is the responsibility of `ParseModuleRelease` after the caller has resolved module injection and values.

```go
func GetReleaseFile(ctx *cue.Context, filePath string) (*FileRelease, error)
```

#### Scenario: Load ModuleRelease file returns raw parse data
- **WHEN** `GetReleaseFile()` is called with a `.cue` file containing `kind: "ModuleRelease"`
- **THEN** `FileRelease.Kind` SHALL be `KindModuleRelease`
- **AND** `FileRelease` SHALL carry the raw release spec `cue.Value`
- **AND** `FileRelease` SHALL carry best-effort module info (metadata, config) extracted from the spec
- **AND** `FileRelease` SHALL NOT carry a `*module.Release`

#### Scenario: Load BundleRelease file
- **WHEN** `GetReleaseFile()` is called with a `.cue` file containing `kind: "BundleRelease"`
- **THEN** `FileRelease.Kind` SHALL be `KindBundleRelease`
- **AND** `FileRelease.Bundle` SHALL be a `*bundle.Release`

### Requirement: Workflow orchestration calls ParseModuleRelease then ProcessModuleRelease

The `internal/workflow/render.FromReleaseFile` function SHALL orchestrate the pipeline as:
1. Load the release file via `GetReleaseFile`
2. Resolve values from release file and `--values` flags
3. Build a `module.Module` from available module data
4. Call `module.ParseModuleRelease(ctx, spec, mod, values)` to get `*module.Release`
5. Apply namespace override if needed (on `Release.Metadata.Namespace`)
6. Load the provider
7. Call `render.ProcessModuleRelease(ctx, release, provider)` to get `*render.ModuleResult`
8. Convert to workflow result

#### Scenario: Full pipeline from release file
- **WHEN** `FromReleaseFile` is called with valid options
- **THEN** it SHALL call `ParseModuleRelease` before `ProcessModuleRelease`
- **AND** `ParseModuleRelease` SHALL receive the raw spec (with `#module` filled), the module, and resolved values
- **AND** `ProcessModuleRelease` SHALL receive the prepared `*module.Release` and provider
