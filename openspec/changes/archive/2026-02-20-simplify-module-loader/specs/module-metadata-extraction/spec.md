## ADDED Requirements

### Requirement: All scalar metadata fields extracted from CUE evaluation
`module.Load` SHALL populate all scalar `ModuleMetadata` fields (`Name`, `DefaultNamespace`, `FQN`, `Version`, `UUID`, `Labels`) by looking up the corresponding paths in the fully evaluated `cue.Value`. No AST inspection SHALL be used for metadata extraction.

#### Scenario: Static string literal name
- **WHEN** `metadata.name` is a string literal in the module CUE source
- **THEN** `mod.Metadata.Name` is populated with that string value after `Load` returns

#### Scenario: Computed name resolves correctly
- **WHEN** `metadata.name` is a computed CUE expression that evaluates to a concrete string (e.g., `_base + "-suffix"`)
- **THEN** `mod.Metadata.Name` is populated with the evaluated concrete string value after `Load` returns

#### Scenario: Non-concrete name left empty
- **WHEN** `metadata.name` is not concrete after CUE evaluation (e.g., an open type or unresolved reference)
- **THEN** `mod.Metadata.Name` is empty and `mod.Validate()` returns an error

### Requirement: Config and Components remain as CUE values
`module.Load` SHALL populate `mod.Config` and `mod.Components` as non-concrete `cue.Value` references. These fields SHALL NOT be decoded into concrete Go types.

#### Scenario: Config is a CUE definition
- **WHEN** `#config` is a CUE struct definition with constraints (e.g., `image: string`)
- **THEN** `mod.Config` is a `cue.Value` that preserves the schema definition, not a decoded Go struct

#### Scenario: Components contain resource references
- **WHEN** `#components` references other CUE definitions like `#config`
- **THEN** `mod.Components` is extracted via `core.ExtractComponents` and preserves CUE value semantics

### Requirement: PkgName extracted from build instance
`module.Load` SHALL populate `mod.PkgName()` from `build.Instance.PkgName`, which is set by `load.Instances()` and is not available from the evaluated `cue.Value`.

#### Scenario: Package name available after load
- **WHEN** the module CUE files declare a package (e.g., `package testmodule`)
- **THEN** `mod.PkgName()` returns that package name after `Load` returns

## REMOVED Requirements

### Requirement: AST-based name extraction
**Reason**: Superseded by CUE-evaluation-based extraction. AST inspection only worked for string literals, silently failing for computed expressions. The evaluated value provides the same information more reliably.
**Migration**: No action required. `module.Load` continues to populate `Name` and `DefaultNamespace`; the extraction mechanism is internal.
