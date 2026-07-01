## MODIFIED Requirements

### Requirement: Resource type location and structure
The `Resource` type SHALL be defined in `pkg/core/` (moved from `internal/core/`). It SHALL wrap a `cue.Value` instead of `*unstructured.Unstructured`. It SHALL provide accessor methods (`Kind()`, `Name()`, `Namespace()`, `GVK()`, `Labels()`, `Annotations()`, `APIVersion()`) and conversion methods (`MarshalJSON()`, `MarshalYAML()`, `ToUnstructured()`, `ToMap()`).

#### Scenario: Resource is importable from pkg/core
- **WHEN** code imports `github.com/open-platform-model/cli/pkg/core`
- **THEN** `core.Resource` is accessible and contains a `Value cue.Value` field

#### Scenario: Resource no longer wraps Unstructured
- **WHEN** code accesses `resource.Object`
- **THEN** compilation fails — the `Object` field no longer exists; use `resource.ToUnstructured()` instead
