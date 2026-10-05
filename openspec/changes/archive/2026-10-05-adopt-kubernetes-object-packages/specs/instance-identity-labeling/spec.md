## MODIFIED Requirements

### Requirement: Runtime identity injected via catalog mandatory field

The CLI's `mod apply` (and any other render entrypoint that produces Kubernetes resources) MUST fill the catalog's `#TransformerContext.#runtimeName` field with the library's `opm/k8s/labels.ManagedByCLI` (`"opm-cli"`); the CLI's `render.RuntimeName` SHALL be that constant, not a copy of its value. The catalog declares `#runtimeName` as a mandatory field; CUE evaluation MUST fail if the CLI omits it. The `#runtimeName` value drives the `app.kubernetes.io/managed-by` label on every rendered resource.

#### Scenario: CLI-applied resources carry runtime identity

- **WHEN** `opm mod apply` renders a `#ModuleInstance` and applies the resulting resources <!-- Was: #ModuleRelease -->
- **THEN** every applied resource has `metadata.labels["app.kubernetes.io/managed-by"]` set to `"opm-cli"`
- **AND** no applied resource carries the legacy literal `"open-platform-model"` for that label key

#### Scenario: Runtime identity stays in sync with Go constant

- **GIVEN** the CLI render pipeline executed against a minimal valid `#ModuleInstance`
- **WHEN** the rendered resources are inspected
- **THEN** the value of `metadata.labels["app.kubernetes.io/managed-by"]` exactly equals `opm/k8s/labels.ManagedByCLI`
- **AND** the value of `metadata.labels["module-instance.opmodel.dev/uuid"]` is non-empty <!-- Was: module-release.opmodel.dev/uuid -->
