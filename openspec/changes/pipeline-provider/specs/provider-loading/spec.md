## ADDED Requirements

### Requirement: Load a named provider from config
The system SHALL load a provider by name from a map of provider CUE values sourced from GlobalConfig, returning a fully-parsed `*LoadedProvider` containing all transformer definitions.

#### Scenario: Named provider found
- **WHEN** `Load` is called with a provider name that exists in the providers map
- **THEN** the system returns a `*LoadedProvider` with all transformer definitions parsed from the provider's CUE value

#### Scenario: Provider name not found
- **WHEN** `Load` is called with a provider name that does not exist in the providers map
- **THEN** the system returns an error identifying the missing provider name and listing available provider names

#### Scenario: Auto-select when exactly one provider is configured
- **WHEN** `Load` is called with an empty provider name and the providers map contains exactly one entry
- **THEN** the system selects that provider automatically without error

#### Scenario: Empty name with multiple providers
- **WHEN** `Load` is called with an empty provider name and the providers map contains more than one entry
- **THEN** the system returns an error indicating a provider name must be specified

### Requirement: Parse transformer definitions from provider CUE value
The system SHALL iterate over the `transformers` field of the provider CUE value and parse each transformer into a `LoadedTransformer` with a fully-qualified name (FQN), required matcher criteria, and optional matcher criteria.

#### Scenario: Transformer with required and optional criteria
- **WHEN** a transformer definition contains `requiredLabels`, `requiredResources`, `requiredTraits`, `optionalLabels`, `optionalResources`, and `optionalTraits` fields
- **THEN** all fields are extracted and stored on the `LoadedTransformer`

#### Scenario: Transformer FQN construction
- **WHEN** a provider named `kubernetes` has a transformer named `deployment`
- **THEN** the transformer's FQN SHALL be `kubernetes#deployment`

#### Scenario: Transformer with no optional criteria
- **WHEN** a transformer definition omits optional fields
- **THEN** the `LoadedTransformer` is returned with empty optional criteria and no error

#### Scenario: Provider with no transformers
- **WHEN** the provider CUE value has an empty or absent `transformers` field
- **THEN** the system returns an error indicating the provider has no transformer definitions

### Requirement: Return structured LoadedProvider
The system SHALL return a `*LoadedProvider` that exposes the provider name, a slice of `*LoadedTransformer`, and a `Requirements()` method returning the FQNs of all transformers for use in error reporting.

#### Scenario: Requirements list matches loaded transformers
- **WHEN** a provider with three transformers is loaded successfully
- **THEN** `LoadedProvider.Requirements()` returns a slice containing the FQN of each transformer

#### Scenario: Invalid transformer definition
- **WHEN** a transformer CUE value has a structural error (missing required fields, evaluation error)
- **THEN** the system returns an error identifying which transformer failed to parse
