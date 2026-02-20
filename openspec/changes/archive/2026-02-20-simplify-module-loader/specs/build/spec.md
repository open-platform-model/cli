## MODIFIED Requirements

### Requirement: ModuleLoader extracts metadata from module
The pipeline SHALL extract all module metadata (`name`, `defaultNamespace`, `fqn`, `version`, `identity`, `labels`) from the fully evaluated `cue.Value` produced by `BuildInstance()`. All metadata fields SHALL be populated via `LookupPath` + `.String()` on the evaluated value. No AST inspection of `inst.Files` SHALL be used for metadata extraction.

`inst.PkgName` SHALL still be read from the `*build.Instance` returned by `load.Instances()`, as it is not available from the evaluated value.

#### Scenario: All metadata extracted from CUE evaluation
- **WHEN** the pipeline loads a module with static string literals for all metadata fields
- **THEN** `name`, `defaultNamespace`, `fqn`, `version`, and `identity` SHALL each be populated from `LookupPath` on the evaluated `cue.Value`
- **AND** no AST walk of `inst.Files` SHALL occur

#### Scenario: Computed metadata name resolves correctly
- **WHEN** a module defines `metadata.name` as a computed CUE expression that evaluates to a concrete string
- **THEN** `mod.Metadata.Name` SHALL be populated with the evaluated concrete string
- **AND** the pipeline SHALL not treat computed names differently from literal names

#### Scenario: Package name extracted from build instance
- **WHEN** a module directory is loaded via `load.Instances()`
- **THEN** `mod.PkgName()` SHALL be populated from `inst.PkgName`

### Requirement: Build() gates on IsConcrete() per component
`Build()` SHALL return a non-nil error if any component extracted from `#components` after `FillPath("#config", values)` is not concrete. The check SHALL be performed immediately after `core.ExtractComponents()` returns, before constructing the `ModuleRelease`. The error SHALL identify the component name.

#### Scenario: All components concrete — Build() succeeds
- **WHEN** `Build()` is called with values that satisfy all `#config` constraints
- **AND** all components in `#components` are concrete after `FillPath`
- **THEN** `Build()` SHALL return a non-nil `*core.ModuleRelease` and a nil error

#### Scenario: Non-concrete component — Build() returns error
- **WHEN** after `FillPath("#config", values)` a component's `Value` is not concrete
- **THEN** `Build()` SHALL return a non-nil error containing the component name
- **AND** the error message SHALL indicate the component is not concrete after value injection
- **AND** no `*core.ModuleRelease` SHALL be returned

## REMOVED Requirements

### Requirement: ReleaseBuilder provides module inspection without CUE evaluation
**Reason**: Superseded by the unified CUE-evaluation-based metadata extraction. AST-only inspection is no longer used or needed; `module.Load()` always performs full evaluation, making a separate inspection path redundant.
**Migration**: No action required for callers. `module.Load()` continues to return fully populated `*core.Module` with all metadata fields set.
