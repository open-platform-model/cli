## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Return structured LoadedProvider
**Reason**: `LoadedProvider` was a thin intermediate type used only to bridge the gap between `provider.Load()` (returning a slice) and `core.Provider` (needing a map). With `loader.LoadProvider()` returning `*core.Provider` directly, the intermediate type and its `Requirements()` method are no longer needed.
**Migration**: Replace `provider.Load()` calls with `loader.LoadProvider()`. The returned `*core.Provider` already has `Transformers`, `Metadata.Name`, and `CueCtx` set. Use `coreProvider.Requirements()` for transformer FQN lists.
