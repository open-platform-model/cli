## ADDED Requirements

### Requirement: LoadedComponent carries annotations

The `LoadedComponent` struct SHALL include an `Annotations` field that stores component-level annotations extracted during release building.

#### Scenario: Component with list-output annotation is loaded

- **WHEN** the release builder extracts a component that has `"transformer.opmodel.dev/list-output": true`
- **THEN** `LoadedComponent.Annotations` SHALL contain the key `"transformer.opmodel.dev/list-output"` with value `"true"`

#### Scenario: Component without annotations is loaded

- **WHEN** the release builder extracts a component that has no annotations
- **THEN** `LoadedComponent.Annotations` SHALL be an empty map (not nil)

### Requirement: TransformerComponentMetadata carries annotations

The `TransformerComponentMetadata` struct SHALL include an `Annotations` field that propagates component annotations into the transformer execution context.

#### Scenario: Annotations propagated from component to transformer context

- **WHEN** `NewTransformerContext()` is called with a component that has annotations
- **THEN** the resulting `TransformerComponentMetadata.Annotations` SHALL contain the same annotation key-value pairs

#### Scenario: Annotations included in CUE context injection

- **WHEN** the executor fills `#context.#componentMetadata` via `FillPath`
- **THEN** the encoded map SHALL include the `annotations` field when annotations are present

#### Scenario: No annotations in CUE context when component has none

- **WHEN** the executor fills `#context.#componentMetadata` for a component without annotations
- **THEN** the encoded map SHALL omit the `annotations` field

## MODIFIED Requirements

### Requirement: Transformer output type accepts list or struct

The executor's output decoding SHALL handle three output shapes from transformer evaluation: single resource (struct with `apiVersion`), map of resources (struct without `apiVersion`), and list of resources (array).

This formalizes existing behavior that already handles all three cases in the Go executor (`internal/build/executor.go`), but was previously not exercised for lists due to CUE schema restrictions.

#### Scenario: Executor decodes single resource output

- **WHEN** a transformer produces `output` as a struct with `apiVersion` at the top level
- **THEN** the executor SHALL decode it as a single `*unstructured.Unstructured` resource

#### Scenario: Executor decodes map output

- **WHEN** a transformer produces `output` as a struct without `apiVersion` at the top level
- **THEN** the executor SHALL iterate struct fields and decode each value as a separate `*unstructured.Unstructured` resource

#### Scenario: Executor decodes list output

- **WHEN** a transformer produces `output` as a CUE list
- **THEN** the executor SHALL iterate list elements and decode each as a separate `*unstructured.Unstructured` resource

#### Scenario: All decoded resources attributed to source component and transformer

- **WHEN** the executor decodes multiple resources from a single transformer output (map or list)
- **THEN** each `Resource` in the result SHALL have its `Component` and `Transformer` fields set to the source component name and transformer FQN
