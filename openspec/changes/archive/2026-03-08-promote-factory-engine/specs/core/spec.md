## MODIFIED Requirements

### Requirement: Resource type location and structure
The `Resource` type SHALL be defined in `pkg/core/` (moved from `internal/core/`). It SHALL wrap a `cue.Value` instead of `*unstructured.Unstructured`. It SHALL provide accessor methods (`Kind()`, `Name()`, `Namespace()`, `GVK()`, `Labels()`, `Annotations()`, `APIVersion()`) and conversion methods (`MarshalJSON()`, `MarshalYAML()`, `ToUnstructured()`, `ToMap()`).

#### Scenario: Resource is importable from pkg/core
- **WHEN** code imports `github.com/opmodel/cli/pkg/core`
- **THEN** `core.Resource` is accessible and contains a `Value cue.Value` field

#### Scenario: Resource no longer wraps Unstructured
- **WHEN** code accesses `resource.Object`
- **THEN** compilation fails — the `Object` field no longer exists; use `resource.ToUnstructured()` instead

### Requirement: Label constants location
All label constants SHALL be defined in `pkg/core/` (moved from `internal/core/`). Values MUST be identical.

#### Scenario: Label constants accessible from new location
- **WHEN** code imports `pkg/core` and references `core.LabelManagedBy`
- **THEN** it resolves to the same string value as the previous `internal/core.LabelManagedBy`

### Requirement: GetWeight function location
The `GetWeight(gvk schema.GroupVersionKind) int` function SHALL be defined in `pkg/core/` (moved from `internal/core/`). Weight values MUST be identical.

#### Scenario: Weight function accessible from new location
- **WHEN** code calls `core.GetWeight(gvk)` from `pkg/core`
- **THEN** it returns the same weight values as the previous `internal/core.GetWeight(gvk)`
