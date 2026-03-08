# Core Provider

## Purpose

Defines provider types in `pkg/provider/`. `Provider` is a thin CUE value wrapper — matching is performed via CUE-native `#MatchPlan` in `pkg/engine`, not Go-side.

---

## Requirements

### Requirement: Provider type location and interface
The `Provider` and `ProviderMetadata` types SHALL be defined in `pkg/provider/` (moved from `internal/core/provider/`). `Provider` SHALL be a thin CUE value wrapper with NO Go-side `Match()` method.

#### Scenario: Provider is importable from pkg/provider
- **WHEN** code imports `github.com/opmodel/cli/pkg/provider`
- **THEN** `provider.Provider` and `provider.ProviderMetadata` are accessible

#### Scenario: No Match method on Provider
- **WHEN** code calls `provider.Match(components)`
- **THEN** compilation fails — matching is done via CUE `#MatchPlan` in the engine
