## Purpose

Defines the contract for `loader.LoadProvider()` — the single call that converts a provider CUE value into a fully-populated `*core.Provider` with `CueCtx`, `Transformers` (as `map[string]*core.Transformer`), and all `ProviderMetadata` fields set.

## Requirements

### Requirement: LoadProvider returns a fully-populated core.Provider
The system SHALL provide `loader.LoadProvider(cueCtx, name, providers)` that returns a `*core.Provider` with `CueCtx`, `Transformers` (as `map[string]*core.Transformer` keyed by transformer name), and fully-populated `Metadata` all set in a single call.

#### Scenario: Returned provider has CueCtx set
- **WHEN** `LoadProvider` returns successfully
- **THEN** the returned `*core.Provider` has `CueCtx` set to the provided `*cue.Context`

#### Scenario: Returned provider has Transformers as map
- **WHEN** a provider with transformer named `deployment` is loaded
- **THEN** the transformer is accessible via `provider.Transformers["deployment"]`

#### Scenario: Returned provider is ready for matching
- **WHEN** `LoadProvider` returns successfully
- **THEN** `provider.Match(components)` can be called directly without further setup

### Requirement: Extract all provider metadata fields from CUE value
The system SHALL extract `name`, `description`, `version`, `minVersion`, and `labels` from the `metadata` field of the provider CUE value, and `apiVersion` and `kind` from the root of the provider CUE value. The config map key SHALL be used as `Metadata.Name` fallback when `metadata.name` is absent.

#### Scenario: Provider CUE value has full metadata
- **WHEN** the provider CUE value contains `metadata.name`, `metadata.version`, `metadata.description`, `metadata.minVersion`, `metadata.labels`, `apiVersion`, and `kind`
- **THEN** all fields are populated on the returned `*core.Provider`

#### Scenario: Provider CUE value has no metadata block
- **WHEN** the provider CUE value has no `metadata` field
- **THEN** `Metadata.Name` is set to the config map key and all other metadata fields are zero values; no error is returned
