## ADDED Requirements

### Requirement: Component types live in a dedicated subpackage
`Component`, `ComponentMetadata`, and `ExtractComponents` SHALL be defined in `internal/core/component` (package `component`), mirroring `component.cue` in the CUE catalog. The package SHALL only import `internal/core` (for base types), CUE SDK, and stdlib.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/component`
- **THEN** all three exported symbols (`Component`, `ComponentMetadata`, `ExtractComponents`) are accessible and the package compiles without referencing `internal/core` for these types

#### Scenario: No circular imports
- **WHEN** `internal/core/component` is loaded
- **THEN** it SHALL NOT import any package that is higher in the chain (`module`, `modulerelease`, `transformer`, `provider`)

### Requirement: Behavioral equivalence after move
`ExtractComponents` SHALL produce identical output to the implementation it replaces in `internal/core/component.go`.

#### Scenario: Component extraction returns same results
- **WHEN** `ExtractComponents` is called with a valid CUE value
- **THEN** the returned `map[string]*Component` is identical in structure and content to what the previous implementation produced

#### Scenario: Validation errors are unchanged
- **WHEN** `ExtractComponents` is called with a CUE value containing a component that fails `Validate()`
- **THEN** the same error message and type are returned as before
