## MODIFIED Requirements

### Requirement: Pipeline output is identical before and after AST refactor

The render pipeline SHALL produce byte-identical `RenderResult` output after the namespace precedence change, **except** when `module.metadata.defaultNamespace` is set and neither `--namespace` nor `OPM_NAMESPACE` is provided — in which case the target namespace MAY differ from the previous behavior (where `config.kubernetes.namespace` or the hardcoded `"default"` was used).

#### Scenario: Existing module renders identically when namespace is explicitly set

- **WHEN** a module is rendered with `--namespace` or `OPM_NAMESPACE` set to the same value as before
- **THEN** `RenderResult.Release.Namespace` SHALL be the same value as before
- **AND** `RenderResult.Release.UUID` SHALL be the same UUID as before
- **AND** all `module-release.opmodel.dev/*` labels SHALL have the same values

#### Scenario: Module with defaultNamespace may produce different release identity

- **WHEN** a module with `metadata.defaultNamespace: "staging"` was previously rendered without `--namespace` (falling through to `config.kubernetes.namespace: "default"`) and is now rendered under the new precedence
- **THEN** `RenderResult.Release.Namespace` SHALL be `"staging"` (not `"default"`)
- **AND** `RenderResult.Release.UUID` SHALL differ from the previous UUID (because namespace is an input to UUID5 computation)

#### Scenario: PREPARATION phase resolves namespace after module load

- **WHEN** the pipeline has loaded a `core.Module` with `Metadata.DefaultNamespace` set
- **AND** the namespace was not explicitly set by `--namespace` or `OPM_NAMESPACE`
- **THEN** the pipeline SHALL use `Metadata.DefaultNamespace` as the target namespace before proceeding to the BUILD phase
