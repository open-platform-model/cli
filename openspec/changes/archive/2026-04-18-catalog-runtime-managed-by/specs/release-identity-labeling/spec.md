## ADDED Requirements

### Requirement: Runtime identity injected via catalog mandatory field
The CLI's `mod apply` (and any other render entrypoint that produces Kubernetes resources) MUST fill the catalog's `#TransformerContext.#runtimeName` field with `core.LabelManagedByValue` (`"opm-cli"`). The catalog declares `#runtimeName` as a mandatory field; CUE evaluation MUST fail if the CLI omits it. The `#runtimeName` value drives the `app.kubernetes.io/managed-by` label on every rendered resource.

#### Scenario: CLI-applied resources carry runtime identity
- **WHEN** `opm mod apply` renders a `#ModuleRelease` and applies the resulting resources
- **THEN** every applied resource has `metadata.labels["app.kubernetes.io/managed-by"]` set to `"opm-cli"`
- **AND** no applied resource carries the legacy literal `"open-platform-model"` for that label key

#### Scenario: Render fails fast when runtime identity is omitted
- **WHEN** a code path inside the CLI render pipeline constructs a CUE evaluation that includes `#TransformerContext` without filling `#runtimeName`
- **THEN** CUE evaluation returns an error mentioning the missing required field
- **AND** no resources are produced

#### Scenario: Runtime identity stays in sync with Go constant
- **GIVEN** the CLI render pipeline executed against a minimal valid `#ModuleRelease`
- **WHEN** the rendered resources are inspected
- **THEN** the value of `metadata.labels["app.kubernetes.io/managed-by"]` exactly equals `core.LabelManagedByValue`
- **AND** the value of `metadata.labels["module-release.opmodel.dev/uuid"]` is non-empty (sanity check that the catalog ownership labels continue to flow)
