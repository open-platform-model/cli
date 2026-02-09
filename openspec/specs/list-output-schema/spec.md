# List Output Schema

## Requirements

### Requirement: CLI extracts component annotations during release building

The release builder SHALL extract `metadata.annotations` from each component's CUE value and store them in `LoadedComponent.Annotations`.

#### Scenario: Component with list-output annotation is loaded

- **WHEN** the release builder extracts a component whose `metadata.annotations` contains `"transformer.opmodel.dev/list-output": true`
- **THEN** `LoadedComponent.Annotations` SHALL contain the key `"transformer.opmodel.dev/list-output"` with value `"true"`

#### Scenario: Component without annotations is loaded

- **WHEN** the release builder extracts a component that has no `metadata.annotations`
- **THEN** `LoadedComponent.Annotations` SHALL be an empty map (not nil)

#### Scenario: Component with multiple annotations

- **WHEN** the release builder extracts a component with multiple annotations
- **THEN** `LoadedComponent.Annotations` SHALL contain all annotation key-value pairs

### Requirement: Annotations propagated to transformer context

The `TransformerComponentMetadata` SHALL include an `Annotations` field that carries component annotations into the transformer execution context, making them accessible to CUE transformers via `#context.#componentMetadata.annotations`.

#### Scenario: Annotations available in transformer context

- **WHEN** a component has `"transformer.opmodel.dev/list-output": true` in its propagated `metadata.annotations`
- **THEN** the transformer SHALL be able to access the annotation via `#context.#componentMetadata.annotations`

#### Scenario: No annotations in transformer context when component has none

- **WHEN** a component has no annotations
- **THEN** the `annotations` field SHALL be omitted from the transformer context (not present as empty)

### Requirement: Executor handles map output from transformers

The executor SHALL correctly decode map output (struct without `apiVersion` at top level) where each field value is a separate Kubernetes resource. This formalizes existing behavior.

#### Scenario: Executor decodes single resource output

- **WHEN** a transformer produces `output` as a struct with `apiVersion` at the top level
- **THEN** the executor SHALL decode it as a single `*unstructured.Unstructured` resource

#### Scenario: Executor decodes map output

- **WHEN** a transformer produces `output` as a struct without `apiVersion` at the top level (e.g., PVC, ConfigMap, or Secret transformer)
- **THEN** the executor SHALL iterate struct fields and decode each value as a separate `*unstructured.Unstructured` resource

#### Scenario: Executor decodes list output

- **WHEN** a transformer produces `output` as a CUE list
- **THEN** the executor SHALL iterate list elements and decode each as a separate `*unstructured.Unstructured` resource

#### Scenario: All decoded resources attributed to source component and transformer

- **WHEN** the executor decodes multiple resources from a single transformer output (map or list)
- **THEN** each `Resource` in the result SHALL have its `Component` and `Transformer` fields set to the source component name and transformer FQN
