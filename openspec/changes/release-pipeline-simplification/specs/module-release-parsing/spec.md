## ADDED Requirements

### Requirement: ParseModuleRelease constructs a fully prepared Release
The `pkg/module` package SHALL export a `ParseModuleRelease` function that takes a raw release spec, a module, and one or more values, and returns a fully prepared `*Release`.

```go
func ParseModuleRelease(ctx context.Context, spec cue.Value, mod Module, values []cue.Value) (*Release, error)
```

#### Scenario: Successful preparation with valid values
- **WHEN** `ParseModuleRelease` is called with a raw release spec, a module with a valid `Config`, and values that satisfy the config schema
- **THEN** it SHALL validate the values against `mod.Config` using `validate.Config`
- **AND** it SHALL merge all values into a single concrete `cue.Value`
- **AND** it SHALL fill the merged values into `spec` at the `values` path
- **AND** it SHALL validate that the filled spec is fully concrete (`cue.Concrete(true)`)
- **AND** it SHALL decode `*ReleaseMetadata` from the concrete spec's `metadata` field
- **AND** it SHALL return a `*Release` with `Metadata`, `Module`, `Spec`, and `Values` all populated

#### Scenario: Config validation failure
- **WHEN** `ParseModuleRelease` is called with values that do not satisfy `mod.Config`
- **THEN** it SHALL return a `*errors.ConfigError`
- **AND** it SHALL NOT return a `*Release`

#### Scenario: Concreteness failure after values filling
- **WHEN** `ParseModuleRelease` is called with valid values but the resulting filled spec is not fully concrete
- **THEN** it SHALL return an error indicating the release is not fully concrete
- **AND** it SHALL NOT return a `*Release`

#### Scenario: Metadata decode failure
- **WHEN** the filled spec's `metadata` field cannot be decoded into `*ReleaseMetadata`
- **THEN** it SHALL return an error
- **AND** it SHALL NOT return a `*Release`

### Requirement: ParseModuleRelease does not mutate inputs
`ParseModuleRelease` SHALL NOT mutate the `spec` or `mod` arguments. Values filling SHALL produce a new `cue.Value` via `FillPath`, not overwrite the input `spec`.

#### Scenario: Input spec is unchanged after call
- **WHEN** `ParseModuleRelease` is called
- **THEN** the original `spec cue.Value` passed by the caller SHALL remain unchanged
- **AND** the returned `Release.Spec` SHALL be the filled version, not the original
