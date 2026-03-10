## ADDED Requirements

### Requirement: Go matcher reproduces the current CUE match-plan semantics
The system SHALL provide a public Go `match.Match(components, provider)` API that reproduces the matching behavior currently defined in `catalog/v1alpha1/core/matcher/matcher.cue`.

#### Scenario: Match result captures missing labels, resources, and traits
- **WHEN** a transformer is evaluated against a component
- **THEN** the match result SHALL include `MissingLabels`, `MissingResources`, and `MissingTraits`
- **AND** the pair SHALL be marked as matched only when all three lists are empty

#### Scenario: Unmatched components are detected
- **WHEN** a component has zero matching transformers
- **THEN** the returned match plan SHALL include that component name in `Unmatched`

#### Scenario: Unhandled traits are detected from matched transformers only
- **WHEN** a component includes traits that are not covered by any matched transformer's `requiredTraits` or `optionalTraits`
- **THEN** the returned match plan SHALL include those trait FQNs in `UnhandledTraits`

#### Scenario: Match output is deterministic
- **WHEN** `MatchedPairs()`, `NonMatchedPairs()`, or `Warnings()` is called on the match plan
- **THEN** the returned output SHALL be deterministic across runs
- **AND** component names, transformer FQNs, and trait diagnostics SHALL be sorted consistently

## MODIFIED Requirements

### Requirement: MatchPlan provides structured diagnostics
The render pipeline SHALL provide a match plan structure with `Matches`, `Unmatched`, and `UnhandledTraits` that can be consumed by the engine and command output. Match-plan diagnostics SHALL be produced by Go matching logic rather than by evaluating a CUE `#MatchPlan` definition.

#### Scenario: Go match plan evaluation
- **WHEN** matching is performed for a provider and a concrete component map
- **THEN** Go code SHALL build the match plan directly from provider transformers and component metadata
- **AND** the resulting match plan SHALL contain `Matches`, `Unmatched`, and `UnhandledTraits`

#### Scenario: MatchPlan provides structured diagnostics
- **WHEN** a transformer does not match a component
- **THEN** the `MatchResult` for that pair SHALL contain `Matched: false` and non-empty missing-label, missing-resource, or missing-trait lists identifying exactly what was missing

#### Scenario: MatchedPairs are deterministically sorted
- **WHEN** `MatchPlan.MatchedPairs()` is called
- **THEN** the returned pairs SHALL be sorted by component name ascending, then transformer FQN ascending

#### Scenario: Warnings are deterministically sorted
- **WHEN** `MatchPlan.Warnings()` is called
- **THEN** the returned warning strings SHALL be sorted by component name then trait FQN

## REMOVED Requirements

### Requirement: CUE-native matching via #MatchPlan
**Reason**: Match-plan construction moves from CUE evaluation into a dedicated Go matcher to simplify the render pipeline and make matching behavior explicit and testable in Go.

**Migration**: Replace CUE `#MatchPlan` loading and decoding with calls to the public Go `match.Match()` API. Preserve the existing `MatchPlan` diagnostics shape and sorting behavior.
