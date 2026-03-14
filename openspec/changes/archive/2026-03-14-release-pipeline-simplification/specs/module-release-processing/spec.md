## MODIFIED Requirements

### Requirement: ProcessModuleRelease is the public rendering entrypoint
The `pkg/render` package SHALL export `ProcessModuleRelease` that accepts a prepared `*module.Release` and a `*provider.Provider`, and returns `*ModuleResult`. It SHALL own the full rendering pipeline: component finalization, matching, and execution.

```go
func ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider) (*ModuleResult, error)
```

#### Scenario: Successful rendering
- **WHEN** `ProcessModuleRelease` is called with a prepared `*module.Release` and a valid provider
- **THEN** it SHALL read schema-preserving components via `rel.MatchComponents()`
- **AND** it SHALL derive finalized, constraint-free components via `finalizeValue` as a local variable
- **AND** it SHALL compute a `*MatchPlan` by calling `Match(schemaComponents, p)`
- **AND** it SHALL execute matched transforms via the module renderer
- **AND** it SHALL return a `*ModuleResult` containing resources, match plan, component summaries, and warnings

#### Scenario: No components in release spec
- **WHEN** `ProcessModuleRelease` is called and `rel.MatchComponents()` does not exist
- **THEN** it SHALL return an error indicating the release has no components

#### Scenario: Component finalization failure
- **WHEN** `finalizeValue` fails to strip CUE constraints from the schema components
- **THEN** it SHALL return an error

#### Scenario: Matching failure
- **WHEN** `Match` returns an error (e.g., provider has no `#transformers`)
- **THEN** it SHALL return the matching error

#### Scenario: Unmatched components produce error
- **WHEN** the match plan contains unmatched components
- **THEN** it SHALL return an `*UnmatchedComponentsError`

### Requirement: ProcessModuleRelease does not validate config
`ProcessModuleRelease` SHALL NOT call `ValidateConfig`. Config validation is the responsibility of `module.ParseModuleRelease`, which runs before `ProcessModuleRelease`.

#### Scenario: No config validation in ProcessModuleRelease
- **WHEN** `ProcessModuleRelease` is called
- **THEN** it SHALL NOT validate values against any config schema
- **AND** it SHALL assume `rel.Spec` is already concrete and `rel.Values` is already validated

### Requirement: Finalized components are transient
`ProcessModuleRelease` SHALL derive finalized components as local variables. It SHALL NOT store them on `*module.Release` or return them as a separate intermediate type.

#### Scenario: Finalized components not stored
- **WHEN** `ProcessModuleRelease` derives finalized components via `finalizeValue`
- **THEN** the finalized components SHALL exist only as local variables within the function
- **AND** they SHALL be passed to the module renderer for execution
- **AND** they SHALL NOT be written to any field on `*module.Release`
