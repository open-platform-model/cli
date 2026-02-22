## Purpose

Defines how the system loads and parses transformer definitions from a provider's CUE value sourced from GlobalConfig, producing a fully-populated `*core.Provider` (with `CueCtx`, `Transformers`, and `Metadata` set) ready for use in component matching.

## Requirements

### Requirement: Load a named provider from config
The system SHALL load a provider by name from a map of provider CUE values sourced from GlobalConfig, returning a fully-parsed `*core.Provider` with all transformer definitions, metadata, and `CueCtx` set.

#### Scenario: Named provider found
- **WHEN** `LoadProvider` is called with a provider name that exists in the providers map
- **THEN** the system returns a `*core.Provider` with all transformer definitions parsed from the provider's CUE value

#### Scenario: Provider name not found
- **WHEN** `LoadProvider` is called with a provider name that does not exist in the providers map
- **THEN** the system returns an error identifying the missing provider name and listing available provider names

#### Scenario: Auto-select when exactly one provider is configured
- **WHEN** `LoadProvider` is called with an empty provider name and the providers map contains exactly one entry
- **THEN** the system selects that provider automatically without error

#### Scenario: Empty name with multiple providers
- **WHEN** `LoadProvider` is called with an empty provider name and the providers map contains more than one entry
- **THEN** the system returns an error indicating a provider name must be specified

### Requirement: Parse transformer definitions from provider CUE value
The system SHALL iterate over the `transformers` field of the provider CUE value and parse each transformer into a `*core.Transformer` with a fully-qualified name (FQN), required matcher criteria, and optional matcher criteria.

#### Scenario: Transformer with required and optional criteria
- **WHEN** a transformer definition contains `requiredLabels`, `requiredResources`, `requiredTraits`, `optionalLabels`, `optionalResources`, and `optionalTraits` fields
- **THEN** all fields are extracted and stored on the `*core.Transformer`

#### Scenario: Transformer FQN construction
- **WHEN** a provider named `kubernetes` has a transformer named `deployment`
- **THEN** the transformer's FQN SHALL be `kubernetes#deployment`

#### Scenario: Transformer with no optional criteria
- **WHEN** a transformer definition omits optional fields
- **THEN** the `*core.Transformer` is returned with empty optional criteria and no error

#### Scenario: Provider with no transformers
- **WHEN** the provider CUE value has an empty or absent `transformers` field
- **THEN** the system returns an error indicating the provider has no transformer definitions

#### Scenario: Invalid transformer definition
- **WHEN** a transformer CUE value has a structural error (missing required fields, evaluation error)
- **THEN** the system returns an error identifying which transformer failed to parse


