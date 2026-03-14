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

### Requirement: Module injection via `--module` flag uses `LoadModulePackage()` + FillPath

When the `--module` flag is provided, the CLI SHALL load the module from the specified directory using `pkg/loader.LoadModulePackage()` and inject it into the raw release spec via `FillPath` at the `#module` path before calling `module.ParseModuleRelease`.

#### Scenario: FillPath injection with `--module`
- **WHEN** a release file has `#module` unfilled
- **AND** `--module ./jellyfin` is provided
- **THEN** the CLI SHALL call `loader.LoadModulePackage("./jellyfin")`
- **AND** fill `#module` on the raw release spec before calling `ParseModuleRelease`

#### Scenario: Module already imported, `--module` flag provided
- **WHEN** a release file imports and fills `#module` from a registry
- **AND** `--module ./jellyfin` is also provided
- **THEN** the `--module` flag SHALL take precedence by overwriting the raw release spec at the `#module` path before calling `ParseModuleRelease`

### Requirement: Workflow orchestration calls ParseModuleRelease then ProcessModuleRelease

The `internal/workflow/render.FromReleaseFile` function SHALL orchestrate the pipeline as:
1. Load the release file via `GetReleaseFile`
2. Apply `--module` injection if needed (on the raw spec)
3. Resolve values from release file and `--values` flags
4. Build a `module.Module` from available module data
5. Call `module.ParseModuleRelease(ctx, spec, mod, values)` to get `*module.Release`
6. Apply namespace override if needed (on `Release.Metadata.Namespace`)
7. Load the provider
8. Call `render.ProcessModuleRelease(ctx, release, provider)` to get `*render.ModuleResult`
9. Convert to workflow result

#### Scenario: Full pipeline from release file
- **WHEN** `FromReleaseFile` is called with valid options
- **THEN** it SHALL call `ParseModuleRelease` before `ProcessModuleRelease`
- **AND** `ParseModuleRelease` SHALL receive the raw spec (with `#module` filled), the module, and resolved values
- **AND** `ProcessModuleRelease` SHALL receive the prepared `*module.Release` and provider
