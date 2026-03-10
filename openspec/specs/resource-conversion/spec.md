# Resource Conversion

## Purpose

Defines the `pkg/core.Resource` type — a CUE-value-based resource with provenance metadata and lazy conversion methods. Replaces the previous `Resource` wrapping `*unstructured.Unstructured`.

## Requirements

### Requirement: Resource wraps cue.Value with provenance
The `pkg/core.Resource` struct SHALL hold a `cue.Value` field (concrete, fully evaluated CUE value of a rendered resource) and provenance strings: `Release`, `Component`, `Transformer`.

#### Scenario: Resource stores CUE value
- **WHEN** the engine produces a resource from a transformer output
- **THEN** the `Resource.Value` field contains the concrete CUE value and `Release`, `Component`, `Transformer` are populated with provenance strings

### Requirement: Resource provides accessor methods
The `Resource` type SHALL provide accessor methods that read from the underlying `cue.Value` without triggering full conversion.

#### Scenario: Kind accessor
- **WHEN** `resource.Kind()` is called
- **THEN** it returns the string value at CUE path `kind` (e.g., "Deployment", "Service")

#### Scenario: Name accessor
- **WHEN** `resource.Name()` is called
- **THEN** it returns the string value at CUE path `metadata.name`

#### Scenario: Namespace accessor
- **WHEN** `resource.Namespace()` is called on a namespaced resource
- **THEN** it returns the string value at CUE path `metadata.namespace`
- **WHEN** `resource.Namespace()` is called on a cluster-scoped resource
- **THEN** it returns an empty string

#### Scenario: GVK accessor
- **WHEN** `resource.GVK()` is called
- **THEN** it returns a `schema.GroupVersionKind` parsed from the `apiVersion` and `kind` CUE fields

#### Scenario: Labels accessor
- **WHEN** `resource.Labels()` is called
- **THEN** it returns `map[string]string` decoded from CUE path `metadata.labels`

#### Scenario: Annotations accessor
- **WHEN** `resource.Annotations()` is called
- **THEN** it returns `map[string]string` decoded from CUE path `metadata.annotations`

### Requirement: Resource provides conversion methods
The `Resource` type SHALL provide lazy conversion methods that transform the `cue.Value` into other formats on demand.

#### Scenario: MarshalJSON conversion
- **WHEN** `resource.MarshalJSON()` is called
- **THEN** it returns the JSON byte representation of the CUE value

#### Scenario: MarshalYAML conversion
- **WHEN** `resource.MarshalYAML()` is called
- **THEN** it returns the YAML byte representation of the CUE value

#### Scenario: ToUnstructured conversion
- **WHEN** `resource.ToUnstructured()` is called
- **THEN** it returns a `*unstructured.Unstructured` populated from the CUE value's JSON representation

#### Scenario: ToMap conversion
- **WHEN** `resource.ToMap()` is called
- **THEN** it returns a `map[string]any` representation of the CUE value

#### Scenario: Conversion errors are returned
- **WHEN** a conversion method is called on a Resource with a non-concrete or errored CUE value
- **THEN** the method returns a descriptive error (not a panic)

### Requirement: Label constants and GVK weights in pkg/core
The `pkg/core` package SHALL export all label constants (`LabelManagedBy`, `LabelReleaseName`, `LabelReleaseNamespace`, `LabelReleaseUUID`, `LabelComponent`, etc.) and the `GetWeight(gvk)` function for resource ordering. These MUST have identical values to the current `internal/core` constants.

#### Scenario: Label constants are accessible from pkg/core
- **WHEN** code imports `pkg/core`
- **THEN** all label constants (e.g., `core.LabelManagedBy`, `core.LabelReleaseName`) are accessible with the same string values as before

#### Scenario: GetWeight returns ordering weights
- **WHEN** `core.GetWeight(gvk)` is called with a known GVK (e.g., CRD, Deployment)
- **THEN** it returns the same integer weight as the current `internal/core.GetWeight()`
