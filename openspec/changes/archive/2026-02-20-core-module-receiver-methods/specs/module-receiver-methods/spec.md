## ADDED Requirements

### Requirement: Module resolves and validates its own path
`core.Module` SHALL expose a `ResolvePath() error` receiver method that validates and resolves the `ModulePath` field in-place. The method SHALL:
- Convert `ModulePath` to an absolute path using `filepath.Abs`
- Verify the directory exists on the filesystem
- Verify a `cue.mod/` subdirectory is present (confirming it is a CUE module)
- Update `Module.ModulePath` to the resolved absolute path on success

This method SHALL NOT perform CUE evaluation. It is a filesystem-only check.

#### Scenario: Valid module path resolves to absolute path
- **WHEN** `ResolvePath()` is called on a `Module` with a relative `ModulePath` pointing to a directory containing `cue.mod/`
- **THEN** `ResolvePath()` SHALL return `nil`
- **AND** `Module.ModulePath` SHALL be updated to the absolute path

#### Scenario: Non-existent directory is rejected
- **WHEN** `ResolvePath()` is called on a `Module` with a `ModulePath` that does not exist on the filesystem
- **THEN** `ResolvePath()` SHALL return a non-nil error describing that the module directory was not found

#### Scenario: Directory without cue.mod is rejected
- **WHEN** `ResolvePath()` is called on a `Module` with a `ModulePath` pointing to a directory that exists but has no `cue.mod/` subdirectory
- **THEN** `ResolvePath()` SHALL return a non-nil error indicating the path is not a CUE module

### Requirement: Module validates its own structural integrity
`core.Module` SHALL expose a `Validate() error` receiver method that checks the module is structurally complete enough to proceed with release building. The method SHALL verify:
- `Module.ModulePath` is non-empty
- `Module.Metadata` is non-nil
- `Module.Metadata.Name` is non-empty

`Validate()` SHALL NOT check `Metadata.FQN`. FQN is computed by CUE evaluation during Phase 2 (`release.Build`) and is not available after AST inspection in Phase 1.

`Validate()` SHALL NOT enforce CUE concreteness on `Config` or `Values` fields. It is a structural guard, not a CUE evaluation step.

#### Scenario: Fully populated Module passes validation
- **WHEN** `Validate()` is called on a `Module` with non-empty `ModulePath`, non-nil `Metadata`, and non-empty `Metadata.Name`
- **THEN** `Validate()` SHALL return `nil`

#### Scenario: Missing Metadata is rejected
- **WHEN** `Validate()` is called on a `Module` with a `nil` `Metadata` field
- **THEN** `Validate()` SHALL return a non-nil error

#### Scenario: Empty Name is rejected
- **WHEN** `Validate()` is called on a `Module` where `Metadata.Name` is an empty string
- **THEN** `Validate()` SHALL return a non-nil error

#### Scenario: FQN is not checked by Validate
- **WHEN** `Validate()` is called on a `Module` where `Metadata.FQN` is an empty string
- **THEN** `Validate()` SHALL return `nil` — FQN is populated in Phase 2, not Phase 1

#### Scenario: Non-concrete CUE values do not cause validation failure
- **WHEN** `Validate()` is called on a `Module` whose `Config` or `Values` CUE fields are not yet concrete
- **THEN** `Validate()` SHALL return `nil` (structural fields are sufficient)

### Requirement: Module loader returns core.Module via AST inspection only
The `internal/build/module` package SHALL expose a `Load(cueCtx *cue.Context, modulePath, registry string) (*core.Module, error)` function that constructs and returns a fully populated `*core.Module`. The function SHALL:
- Construct a `core.Module{ModulePath: modulePath}`
- Call `mod.ResolvePath()` and return its error if non-nil
- Perform AST inspection to populate `Metadata.Name`, `Metadata.DefaultNamespace`, and the internal package name field
- Return the populated `*core.Module`

The function SHALL NOT fall back to CUE evaluation if AST inspection returns an empty `Metadata.Name`. Because `metadata.name!` is a mandatory required field in the CUE `#Module` schema, a module with a non-literal (computed) name is an unsupported authoring pattern. An empty name after AST inspection will be caught by `Validate()` as a fatal error.

The following SHALL be removed from `internal/build/module/`:
- The standalone `ResolvePath(modulePath string) (string, error)` function — superseded by `Module.ResolvePath()`
- The `ExtractMetadata(cueCtx, modulePath, registry)` function — no longer needed
- The `MetadataPreview` type — no longer needed

#### Scenario: Load returns populated Module with resolved path
- **WHEN** `module.Load(cueCtx, "./my-module", registry)` is called with a valid module directory whose `metadata.name` is a string literal
- **THEN** the returned `*core.Module` SHALL have `ModulePath` set to the absolute path of the module directory
- **AND** `Metadata.Name` SHALL be populated from the module's `metadata.name` field
- **AND** `Metadata.DefaultNamespace` SHALL be populated from `metadata.defaultNamespace` if present, or empty string if absent

#### Scenario: Load propagates ResolvePath error
- **WHEN** `module.Load(cueCtx, "/nonexistent", registry)` is called with an invalid path
- **THEN** `Load()` SHALL return a non-nil error wrapping the resolution failure
- **AND** the returned `*core.Module` SHALL be `nil`

#### Scenario: Module with computed name fails Validate after Load
- **WHEN** `module.Load()` is called on a module where `metadata.name` is a computed CUE expression (not a string literal)
- **THEN** `Load()` SHALL return a `*core.Module` with an empty `Metadata.Name`
- **AND** the subsequent call to `mod.Validate()` SHALL return a non-nil error
