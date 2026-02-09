## ADDED Requirements

### Requirement: Component list-output annotation

The `#Component` CUE definition SHALL support an optional `"transformer.opmodel.dev/list-output"?: bool` annotation that declares the component's transformers may produce list or map outputs.

#### Scenario: Component with list-output annotation set to true

- **WHEN** a module author defines a component with `"transformer.opmodel.dev/list-output": true`
- **THEN** the CUE evaluator SHALL accept the annotation without validation errors

#### Scenario: Component without list-output annotation

- **WHEN** a module author defines a component without the `"transformer.opmodel.dev/list-output"` annotation
- **THEN** the component SHALL behave identically to current behavior with no changes to validation or output handling

#### Scenario: Component with list-output annotation set to false

- **WHEN** a module author defines a component with `"transformer.opmodel.dev/list-output": false`
- **THEN** the CUE evaluator SHALL accept the annotation and the component SHALL behave identically to one without the annotation

### Requirement: Transformer output type accepts list or struct

The `#Transformer.#transform.output` CUE constraint SHALL accept both struct (`{...}`) and list (`[...{...}]`) output shapes.

#### Scenario: Transformer produces a single resource struct

- **WHEN** a transformer's `#transform` function returns `output: { apiVersion: "apps/v1", kind: "Deployment", ... }`
- **THEN** the CUE evaluator SHALL accept the output as valid

#### Scenario: Transformer produces a map of resources

- **WHEN** a transformer's `#transform` function returns `output: { "vol-a": { apiVersion: "v1", kind: "PersistentVolumeClaim", ... }, "vol-b": { ... } }`
- **THEN** the CUE evaluator SHALL accept the output as valid

#### Scenario: Transformer produces a list of resources

- **WHEN** a transformer's `#transform` function returns `output: [{ apiVersion: "v1", kind: "PersistentVolumeClaim", ... }, { ... }]`
- **THEN** the CUE evaluator SHALL accept the output as valid

### Requirement: Annotation propagated to transformer context

The `#TransformerContext.#componentMetadata` CUE definition SHALL include an optional `annotations` field that carries component annotations to transformers.

#### Scenario: Component annotation available in transformer context

- **WHEN** a component has `"transformer.opmodel.dev/list-output": true`
- **THEN** the transformer SHALL be able to access the annotation value via `#context.#componentMetadata.annotations`

#### Scenario: Component without annotations

- **WHEN** a component has no annotations
- **THEN** `#context.#componentMetadata.annotations` SHALL be absent or empty and SHALL NOT cause validation errors
