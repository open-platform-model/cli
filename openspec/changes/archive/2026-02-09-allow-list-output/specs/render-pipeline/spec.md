## ADDED Requirements

### Requirement: LoadedComponent carries annotations
The `LoadedComponent` struct SHALL include an `Annotations map[string]string` field that stores component-level annotations extracted during release building.

#### Scenario: Annotations field initialized for all components

- **WHEN** the release builder creates a `LoadedComponent`
- **THEN** the `Annotations` field SHALL be initialized to an empty map (not nil), regardless of whether the component has annotations

### Requirement: TransformerComponentMetadata carries annotations

The `TransformerComponentMetadata` struct SHALL include an `Annotations map[string]string` field that propagates component annotations into the transformer execution context.

#### Scenario: Annotations propagated from LoadedComponent to TransformerComponentMetadata

- **WHEN** `NewTransformerContext()` is called with a `LoadedComponent` that has annotations
- **THEN** the resulting `TransformerComponentMetadata.Annotations` SHALL contain the same key-value pairs

#### Scenario: Annotations included in CUE context map when present

- **WHEN** `ToMap()` is called on a `TransformerContext` whose component has annotations
- **THEN** the `#componentMetadata` map SHALL include an `annotations` key with the annotation map

#### Scenario: Annotations omitted from CUE context map when empty

- **WHEN** `ToMap()` is called on a `TransformerContext` whose component has no annotations
- **THEN** the `#componentMetadata` map SHALL NOT include an `annotations` key
