## MODIFIED Requirements

### Requirement: internal/build/component/ package is removed

The `internal/build/component/` package SHALL be deleted. All import sites (10 files) SHALL be updated to use `core.Component` from `internal/core/`.

#### Scenario: No import of internal/build/component remains
- **WHEN** the codebase is compiled after this change
- **THEN** no Go file SHALL import `github.com/open-platform-model/cli/internal/build/component`
- **AND** all references to `component.Component` SHALL be replaced with `core.Component`

### Requirement: Component types live in a dedicated subpackage

`Component`, `ComponentMetadata`, and `ExtractComponents` SHALL be defined in `internal/core/component` (package `component`), mirroring `component.cue` in the CUE catalog. The package SHALL only import `internal/core` (for base types), CUE SDK, and stdlib.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/open-platform-model/cli/internal/core/component`
- **THEN** all three exported symbols (`Component`, `ComponentMetadata`, `ExtractComponents`) are accessible and the package compiles without referencing `internal/core` for these types

#### Scenario: No circular imports
- **WHEN** `internal/core/component` is loaded
- **THEN** it SHALL NOT import any package that is higher in the chain (`module`, `modulerelease`, `transformer`, `provider`)
