## ADDED Requirements

### Requirement: ModuleRelease types live in a dedicated subpackage
`ModuleRelease` and `ReleaseMetadata` SHALL be defined in `internal/core/modulerelease` (package `modulerelease`), mirroring `module_release.cue` in the CUE catalog. The package SHALL only import `internal/core`, `internal/core/component`, `internal/core/module`, CUE SDK, and stdlib.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/modulerelease`
- **THEN** `ModuleRelease` and `ReleaseMetadata` are accessible and the package compiles

#### Scenario: No circular imports
- **WHEN** `internal/core/modulerelease` is loaded
- **THEN** it SHALL NOT import `internal/core/transformer` or `internal/core/provider`

### Requirement: CUE validation helpers are unexported
`validateFieldsRecursive` and `pathRewrittenError` SHALL be unexported symbols within the `modulerelease` package, accessible only to `ModuleRelease.ValidateValues()`.

#### Scenario: Helpers are not accessible to external packages
- **WHEN** an external package imports `internal/core/modulerelease`
- **THEN** `validateFieldsRecursive` and `pathRewrittenError` SHALL NOT be accessible

### Requirement: ModuleRelease validation methods are preserved
`ValidateValues` and `Validate` on `ModuleRelease` SHALL produce identical behavior after the move.

#### Scenario: Value validation is unchanged
- **WHEN** `ModuleRelease.ValidateValues()` is called with values that violate the `#config` schema
- **THEN** a `ValidationError` with the same message and CUE error details is returned

#### Scenario: Concreteness check is unchanged
- **WHEN** `ModuleRelease.Validate()` is called with non-concrete component values
- **THEN** the same validation error describing non-concrete components is returned
