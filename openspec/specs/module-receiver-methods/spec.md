# Module Receiver Methods

## Purpose

This spec defines the behavior of receiver methods on `core.Module` and the `module.Load()` constructor in `internal/build/module/`. These make `core.Module` self-describing: it owns its own path resolution and structural validation, replacing the previous pattern of scattered standalone functions.

---

## Requirements

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

`core.Module` SHALL expose a `Validate() error` receiver method that checks the module is structurally complete enough to proceed with release building. After this change, `Validate()` SHALL verify:

- `Module.ModulePath` is non-empty
- `Module.Metadata` is non-nil
- `Module.Metadata.Name` is non-empty
- `Module.Metadata.FQN` is non-empty (now available after CUE evaluation in `Load()`)
- `mod.CUEValue().Exists()` returns `true`

`Validate()` SHALL NOT enforce CUE concreteness on `Config`, `Values`, or `Components`. It is a structural guard, not a CUE evaluation step.

> **Change from prior spec**: The prior requirement excluded FQN from `Validate()` because FQN was not available after Phase 1 (AST-only Load). With `Load()` now performing CUE evaluation, FQN is populated before `Validate()` is ever called, and the check is both safe and useful.

#### Scenario: Fully populated Module passes validation
- **WHEN** `Validate()` is called on a `*core.Module` returned by `module.Load()` with a valid module
- **THEN** `Validate()` SHALL return `nil`

#### Scenario: Missing FQN is rejected
- **WHEN** `Validate()` is called on a `Module` where `Metadata.FQN` is an empty string
- **THEN** `Validate()` SHALL return a non-nil error

#### Scenario: Zero CUEValue is rejected
- **WHEN** `Validate()` is called on a `Module` where `mod.CUEValue().Exists()` returns `false`
- **THEN** `Validate()` SHALL return a non-nil error indicating that `Load()` was not completed

#### Scenario: Missing Metadata is rejected (unchanged)
- **WHEN** `Validate()` is called on a `Module` with a nil `Metadata` field
- **THEN** `Validate()` SHALL return a non-nil error

#### Scenario: Empty Name is rejected (unchanged)
- **WHEN** `Validate()` is called on a `Module` where `Metadata.Name` is an empty string
- **THEN** `Validate()` SHALL return a non-nil error

#### Scenario: Non-concrete CUE values do not cause validation failure (unchanged)
- **WHEN** `Validate()` is called on a `Module` whose `Config` or `Values` CUE fields are not concrete
- **THEN** `Validate()` SHALL return `nil`

### Requirement: Module loader performs AST inspection followed by CUE evaluation

`internal/build/module.Load()` SHALL perform two sequential phases and return a fully populated `*core.Module`:

1. **AST phase** (unchanged): call `mod.ResolvePath()`, run `load.Instances()`, walk `inst.Files` to extract `Metadata.Name`, `Metadata.DefaultNamespace`, and the internal package name field.
2. **CUE evaluation phase** (new): call `cueCtx.BuildInstance(inst)` on the same instance from Phase 1 to obtain the base `cue.Value`; extract `Module.Config`, `Module.Values`, `Module.Components`, and the remaining `Module.Metadata` fields (`FQN`, `UUID`, `Version`, `Labels`) from the evaluated value.

> **Change from prior spec**: The prior requirement was titled "Module loader returns core.Module via AST inspection only" and explicitly excluded CUE evaluation. That constraint is removed. `load.Instances()` is still called exactly once; its result is reused for both phases.

#### Scenario: Load returns Module with all Metadata fields populated
- **WHEN** `module.Load()` is called with a valid module directory
- **THEN** the returned `*core.Module` SHALL have `Metadata.Name` populated (from AST)
- **AND** `Metadata.FQN` SHALL be populated (from CUE evaluation)
- **AND** `Metadata.UUID` SHALL be populated (from CUE evaluation)
- **AND** `Metadata.Version` SHALL be populated (from CUE evaluation)

#### Scenario: Load propagates CUE evaluation error
- **WHEN** `module.Load()` is called on a module with a CUE evaluation error
- **THEN** `Load()` SHALL return a non-nil error
- **AND** the returned `*core.Module` SHALL be `nil`

#### Scenario: Load propagates ResolvePath error (unchanged)
- **WHEN** `module.Load()` is called with a non-existent or invalid module path
- **THEN** `Load()` SHALL return a non-nil error wrapping the resolution failure
- **AND** the returned `*core.Module` SHALL be `nil`
