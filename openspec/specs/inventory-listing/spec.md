## ADDED Requirements

### Requirement: ListInventories discovers all inventory Secrets in a namespace

The inventory package SHALL provide a `ListInventories` function that discovers all inventory Secrets in a given namespace by listing Secrets matching the label selector `app.kubernetes.io/managed-by=open-platform-model` AND `opmodel.dev/component=inventory`. Each matching Secret SHALL be deserialized via `UnmarshalFromSecret`. The results SHALL be sorted alphabetically by `ReleaseMetadata.ReleaseName`.

#### Scenario: List in a specific namespace

- **WHEN** `ListInventories` is called with namespace `"production"`
- **AND** 3 inventory Secrets exist in `production`
- **THEN** it SHALL return 3 `*InventorySecret` values sorted by release name

#### Scenario: List across all namespaces

- **WHEN** `ListInventories` is called with namespace `""` (empty string)
- **THEN** it SHALL list inventory Secrets across all namespaces
- **AND** the results SHALL be sorted alphabetically by release name

#### Scenario: Empty namespace returns empty slice

- **WHEN** `ListInventories` is called with a namespace containing no inventory Secrets
- **THEN** it SHALL return an empty slice and nil error

#### Scenario: Unmarshal failure skips corrupt Secret

- **WHEN** one inventory Secret in the namespace has corrupt data
- **THEN** `ListInventories` SHALL log a warning for the corrupt Secret
- **AND** SHALL continue processing remaining Secrets
- **AND** SHALL return the successfully deserialized results

### Requirement: ListInventories uses existing label constants

The function SHALL use the label constants already defined in `internal/core/` (`LabelManagedBy`, `LabelManagedByValue`, `LabelComponent`) to build the selector. It MUST NOT hardcode label strings.

#### Scenario: Label selector matches inventory convention

- **WHEN** `ListInventories` constructs its label selector
- **THEN** it SHALL use `core.LabelManagedBy=core.LabelManagedByValue` and `core.LabelComponent=inventory`
