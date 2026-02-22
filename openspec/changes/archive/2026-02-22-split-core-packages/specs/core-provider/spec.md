## ADDED Requirements

### Requirement: Provider types live in a dedicated subpackage
`Provider`, `ProviderMetadata`, and `Requirements()` SHALL be defined in `internal/core/provider` (package `provider`), mirroring `provider.cue` in the CUE catalog. The package SHALL only import `internal/core`, `internal/core/component`, `internal/core/transformer`, CUE SDK, and stdlib.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/provider`
- **THEN** `Provider`, `ProviderMetadata`, and `Match()` are accessible

#### Scenario: No circular imports
- **WHEN** `internal/core/provider` is loaded
- **THEN** it imports nothing from `internal/loader`, `internal/builder`, or `internal/pipeline`

### Requirement: Provider matching behavior is preserved
`Provider.Match()` SHALL produce an identical `*TransformerMatchPlan` for identical inputs after the move.

#### Scenario: Matched components are unchanged
- **WHEN** `Match()` is called with a provider and component map
- **THEN** the resulting match plan contains the same matches and unmatched components as before

#### Scenario: Match output is deterministic
- **WHEN** `Match()` is called multiple times with the same inputs
- **THEN** the match plan is identical each time (sorted component and transformer names)
