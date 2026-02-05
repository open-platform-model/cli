# Delta Spec: CUE-based Component Matcher

## Overview

This spec defines the pluggable component-transformer matching system with CUE-based predicate evaluation. It enables providers to define custom matching logic declaratively in CUE rather than requiring Go code modifications.

## ADDED Requirements

### Requirement: ComponentMatcher interface

The system SHALL provide a `ComponentMatcher` interface that abstracts matching behavior, enabling pluggable implementations.

```go
type ComponentMatcher interface {
    Match(components []*LoadedComponent, transformers []*LoadedTransformer) *MatchResult
}
```

#### Scenario: Interface allows multiple implementations

- **WHEN** the Pipeline needs a matcher
- **THEN** it SHALL accept any type implementing `ComponentMatcher`

#### Scenario: Default implementation provided

- **WHEN** no custom matcher is injected
- **THEN** the Pipeline SHALL use a default `ComponentMatcher` implementation

---

### Requirement: Pipeline matcher injection

The Pipeline SHALL support injecting a custom `ComponentMatcher` via functional options, enabling testing and customization.

#### Scenario: Inject custom matcher via option

- **WHEN** `NewPipeline(cfg, WithMatcher(customMatcher))` is called
- **THEN** the Pipeline SHALL use `customMatcher` for all Match operations

#### Scenario: Default matcher when no option provided

- **WHEN** `NewPipeline(cfg)` is called without matcher option
- **THEN** the Pipeline SHALL create and use the default CUE-based matcher

---

### Requirement: CUE #Matches predicate

Transformers MAY define a `#Matches` field containing a CUE expression that evaluates to a boolean. The matcher SHALL evaluate this predicate to determine if a transformer matches a component.

#### Scenario: Transformer with #Matches predicate

- **WHEN** a transformer defines `#Matches: component.labels["workload-type"] == "stateless"`
- **AND** a component has label `workload-type: stateless`
- **THEN** the matcher SHALL evaluate `#Matches` to `true` and match the transformer

#### Scenario: Transformer with complex #Matches predicate

- **WHEN** a transformer defines:

  ```cue
  #Matches: {
      hasContainer: len(component.#resources["opm.dev/Container"]) > 0
      isStateless: component.labels["workload-type"] == "stateless"
      result: hasContainer && isStateless
  }.result
  ```

- **AND** a component satisfies both conditions
- **THEN** the matcher SHALL evaluate `#Matches` to `true`

#### Scenario: #Matches evaluation failure

- **WHEN** `#Matches` evaluation produces a CUE error (incomplete, conflict)
- **THEN** the matcher SHALL treat the match as `false`
- **AND** SHALL record the error in `MatchDetail` for verbose output

---

### Requirement: #Matches context injection

The CUE evaluation context for `#Matches` SHALL include the component being matched, allowing predicates to access component properties.

#### Scenario: Access component labels

- **WHEN** `#Matches` references `component.labels`
- **THEN** the expression SHALL receive the component's effective labels map

#### Scenario: Access component resources

- **WHEN** `#Matches` references `component.#resources`
- **THEN** the expression SHALL receive the component's resources keyed by FQN

#### Scenario: Access component traits

- **WHEN** `#Matches` references `component.#traits`
- **THEN** the expression SHALL receive the component's traits keyed by FQN

---

### Requirement: Fallback to Go-based matching

When a transformer does NOT define a `#Matches` predicate, the matcher SHALL fall back to the existing Go-based matching logic using `requiredLabels`, `requiredResources`, and `requiredTraits`.

#### Scenario: Transformer without #Matches uses Go logic

- **WHEN** a transformer has no `#Matches` field
- **AND** defines `requiredLabels: {"workload-type": "stateless"}`
- **THEN** the matcher SHALL use Go-based label matching

#### Scenario: Mixed transformers (some with #Matches, some without)

- **WHEN** provider has transformer A with `#Matches` and transformer B with only `requiredLabels`
- **THEN** the matcher SHALL evaluate A using CUE and B using Go logic

---

### Requirement: MatchResult structure

The `MatchResult` returned by `ComponentMatcher.Match()` SHALL contain:

- `ByTransformer`: Map of transformer FQN to matched components
- `Unmatched`: Components with no matching transformers
- `Details`: Per-component per-transformer matching decisions for verbose output

#### Scenario: MatchResult contains all decisions

- **WHEN** matching completes
- **THEN** `MatchResult.Details` SHALL contain one entry per (component, transformer) pair
- **AND** each detail SHALL include `Matched` boolean and `Reason` string

#### Scenario: MatchResult identifies unmatched components

- **WHEN** a component matches no transformers
- **THEN** `MatchResult.Unmatched` SHALL include that component

---

### Requirement: Multiple transformer matches

A single component MAY match multiple transformers. Each match SHALL produce separate resources.

#### Scenario: Component matches two transformers

- **WHEN** component C matches both transformer T1 and T2
- **THEN** `MatchResult.ByTransformer[T1.FQN]` SHALL contain C
- **AND** `MatchResult.ByTransformer[T2.FQN]` SHALL contain C

---

### Requirement: Verbose matching output

The matcher SHALL provide detailed matching decisions for verbose/debug output, including why each transformer did or did not match each component.

#### Scenario: Verbose output shows match reason

- **WHEN** `--verbose` flag is set
- **AND** transformer matches via `#Matches`
- **THEN** output SHALL show "Matched: #Matches predicate evaluated true"

#### Scenario: Verbose output shows non-match reason

- **WHEN** `--verbose` flag is set
- **AND** transformer does not match due to missing label
- **THEN** output SHALL show "Not matched: missing labels: workload-type"

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-CM-001 | CUE predicate evaluation SHALL complete within 100ms per transformer-component pair |
| NFR-CM-002 | Matcher interface SHALL be stable for at least one minor version after introduction |

---

## Edge Cases

| Case | Handling |
|------|----------|
| Empty `#Matches` field | Treat as undefined, use Go fallback |
| `#Matches: true` (always match) | Valid - transformer matches all components |
| `#Matches: false` (never match) | Valid - transformer matches no components |
| Circular reference in `#Matches` | CUE error, treat as non-match |
| `#Matches` references undefined field | CUE error, treat as non-match |
