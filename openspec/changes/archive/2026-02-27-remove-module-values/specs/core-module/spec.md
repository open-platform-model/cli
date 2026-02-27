## MODIFIED Requirements

### Requirement: Module types live in a dedicated subpackage

`Module` and `ModuleMetadata` SHALL be defined in `internal/core/module` (package `module`), mirroring `module.cue` in the CUE catalog. The `Module` struct SHALL contain: `Metadata`, `Components`, `Config`, `ModulePath`, `Raw`, and `pkgName`. The struct SHALL NOT contain `Values`, `HasValuesCue`, or `SkippedValuesFiles` fields.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/module`
- **THEN** `Module` and `ModuleMetadata` are accessible and the package compiles without referencing `internal/core` for these types

#### Scenario: No circular imports
- **WHEN** `internal/core/module` is loaded
- **THEN** it SHALL NOT import any package higher in the chain (`modulerelease`, `transformer`, `provider`)

#### Scenario: Module struct does not carry values
- **WHEN** a `Module` is constructed by the loader
- **THEN** the struct SHALL NOT have a `Values` field, a `HasValuesCue` field, or a `SkippedValuesFiles` field

## REMOVED Requirements

### Requirement: Module carries default values
**Reason**: In v1alpha1, `#Module` no longer has a `values` field. Default values resolution is now a build-time concern owned by the builder, not a module-level property.
**Migration**: The builder discovers `values.cue` from `Module.ModulePath` at build time. No values state is passed through the `Module` struct.
