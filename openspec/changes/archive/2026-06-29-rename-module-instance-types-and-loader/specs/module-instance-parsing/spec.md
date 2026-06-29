## ADDED Requirements

<!-- Renamed from `module-release-parsing` (enhancement 0002 D10). Spec dir is git mv'd at archive. -->

### Requirement: ParseModuleInstance constructs a fully prepared Instance
The `pkg/module` package SHALL export a `ParseModuleInstance` function that takes a raw instance spec, a module, and one or more values, and returns a fully prepared `*Instance`.

```go
func ParseModuleInstance(ctx context.Context, spec cue.Value, mod Module, values []cue.Value) (*Instance, error)
```

#### Scenario: Successful preparation with valid values
- **WHEN** `ParseModuleInstance` is called with a raw instance spec, a module with a valid `Config`, and values that satisfy the config schema
- **THEN** it SHALL validate the values against `mod.Config` using `validate.Config`
- **AND** it SHALL merge all values into a single concrete `cue.Value`
- **AND** it SHALL fill the merged values into `spec` at the `values` path
- **AND** it SHALL validate that the filled spec is fully concrete (`cue.Concrete(true)`)
- **AND** it SHALL decode `*InstanceMetadata` from the concrete spec's `metadata` field
- **AND** it SHALL return a `*Instance` with `Metadata`, `Module`, `Spec`, and `Values` all populated

#### Scenario: Config validation failure
- **WHEN** `ParseModuleInstance` is called with values that do not satisfy `mod.Config`
- **THEN** it SHALL return a `*errors.ConfigError`
- **AND** it SHALL NOT return a `*Instance`

#### Scenario: Concreteness failure after values filling
- **WHEN** `ParseModuleInstance` is called with valid values but the resulting filled spec is not fully concrete
- **THEN** it SHALL return an error indicating the instance is not fully concrete
- **AND** it SHALL NOT return a `*Instance`

#### Scenario: Metadata decode failure
- **WHEN** the filled spec's `metadata` field cannot be decoded into `*InstanceMetadata`
- **THEN** it SHALL return an error
- **AND** it SHALL NOT return a `*Instance`

### Requirement: ParseModuleInstance does not mutate inputs
`ParseModuleInstance` SHALL NOT mutate the `spec` or `mod` arguments. Values filling SHALL produce a new `cue.Value` via `FillPath`, not overwrite the input `spec`.

#### Scenario: Input spec is unchanged after call
- **WHEN** `ParseModuleInstance` is called
- **THEN** the original `spec cue.Value` passed by the caller SHALL remain unchanged
- **AND** the returned `Instance.Spec` SHALL be the filled version, not the original
