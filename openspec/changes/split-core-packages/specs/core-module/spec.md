## ADDED Requirements

### Requirement: Module types live in a dedicated subpackage
`Module` and `ModuleMetadata` SHALL be defined in `internal/core/module` (package `module`), mirroring `module.cue` in the CUE catalog. The package SHALL only import `internal/core`, `internal/core/component`, CUE SDK, and stdlib.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/module`
- **THEN** `Module` and `ModuleMetadata` are accessible and the package compiles without referencing `internal/core` for these types

#### Scenario: No circular imports
- **WHEN** `internal/core/module` is loaded
- **THEN** it SHALL NOT import any package higher in the chain (`modulerelease`, `transformer`, `provider`)

### Requirement: Module receiver methods are preserved
All receiver methods on `Module` (`Validate`, `ResolvePath`, `SetPkgName`, `PkgName`) SHALL be defined in the same package and produce identical behavior.

#### Scenario: Module validation is unchanged
- **WHEN** `module.Validate()` is called on a fully populated `Module`
- **THEN** the same validation rules apply and the same errors are returned as before the move

#### Scenario: Module path resolution is unchanged
- **WHEN** `module.ResolvePath()` is called
- **THEN** the path resolution and cue.mod validation behave identically to the previous implementation
