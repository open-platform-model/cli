## MODIFIED Requirements

### Requirement: Inventory Secret labels

The inventory Secret SHALL carry the following five labels:

| Label key | Value | Description |
|---|---|---|
| `app.kubernetes.io/managed-by` | `open-platform-model` | Standard managed-by label |
| `module-release.opmodel.dev/name` | release name (e.g. `mc`) | The user-supplied release name (`--release-name`) |
| `module-release.opmodel.dev/namespace` | release namespace | Namespace the release is deployed into |
| `module-release.opmodel.dev/uuid` | release UUID | Deterministic UUID v5 release identity |
| `opmodel.dev/component` | `inventory` | Distinguishes the inventory Secret from application resources |

The `module.opmodel.dev/name` and `module.opmodel.dev/namespace` labels SHALL NOT be present on the inventory Secret. Module identity is carried in `data.moduleMetadata` instead.

#### Scenario: Inventory Secret has correct labels

- **WHEN** creating an inventory Secret for module `minecraft` deployed as release name `mc` in namespace `games` with release ID `abc123`
- **THEN** the Secret SHALL have exactly five labels:
  - `app.kubernetes.io/managed-by: open-platform-model`
  - `module-release.opmodel.dev/name: mc`
  - `module-release.opmodel.dev/namespace: games`
  - `module-release.opmodel.dev/uuid: abc123`
  - `opmodel.dev/component: inventory`
- **AND** the Secret type SHALL be `opmodel.dev/release`
- **AND** the Secret SHALL NOT have label `module.opmodel.dev/name`
- **AND** the Secret SHALL NOT have label `module.opmodel.dev/namespace`

### Requirement: Inventory Secret serialization roundtrip

`MarshalToSecret` SHALL serialize an `InventorySecret` to a typed `corev1.Secret` using `stringData` keys: `releaseMetadata` (JSON-encoded `ReleaseMetadata`), `moduleMetadata` (JSON-encoded `ModuleMetadata`), `index` (JSON-encoded `[]string` of change IDs), and one key per change entry (`change-sha1-<8hex>` with JSON-encoded `ChangeEntry`). `UnmarshalFromSecret` SHALL deserialize a `corev1.Secret` back into an `InventorySecret`, handling both `stringData` and `data` (base64-encoded) fields. The `moduleMetadata` key is optional — if absent, the resulting `ModuleMetadata` SHALL be a zero value (no error). The `resourceVersion` from the Secret SHALL be preserved as an unexported field for optimistic concurrency.

#### Scenario: Marshal and unmarshal roundtrip

- **WHEN** marshaling an `InventorySecret` with release metadata, module metadata, 2 change entries, and an index
- **AND** unmarshaling the resulting `corev1.Secret`
- **THEN** the resulting `InventorySecret` SHALL be identical to the original

#### Scenario: Unmarshal from Kubernetes GET response

- **WHEN** unmarshaling a Secret returned by the Kubernetes API (base64-encoded `data` field)
- **THEN** the resulting `InventorySecret` SHALL contain correct values
- **AND** the `resourceVersion` SHALL be preserved from the Secret's metadata

#### Scenario: Empty inventory with no changes

- **WHEN** marshaling an `InventorySecret` with empty index and no change entries
- **THEN** the resulting Secret SHALL be valid with an empty JSON array for index

#### Scenario: Missing moduleMetadata key is not an error

- **WHEN** unmarshaling a Secret that has no `moduleMetadata` key in its data
- **THEN** the resulting `InventorySecret.ModuleMetadata` SHALL be a zero-value struct
- **AND** no error SHALL be returned

### Requirement: Inventory metadata enables future CRD migration

The `ReleaseMetadata` SHALL include `kind: "ModuleRelease"` and `apiVersion: "core.opmodel.dev/v1alpha1"` fields. The `ModuleMetadata` SHALL include `kind: "Module"` and `apiVersion: "core.opmodel.dev/v1alpha1"` fields. Both enable future migration from Secrets to CRDs without changing the data model.

#### Scenario: ReleaseMetadata kind and apiVersion

- **WHEN** creating release metadata
- **THEN** the `kind` field SHALL be `"ModuleRelease"`
- **AND** the `apiVersion` field SHALL be `"core.opmodel.dev/v1alpha1"`

#### Scenario: ModuleMetadata kind and apiVersion

- **WHEN** creating module metadata
- **THEN** the `kind` field SHALL be `"Module"`
- **AND** the `apiVersion` field SHALL be `"core.opmodel.dev/v1alpha1"`

## ADDED Requirements

### Requirement: Release metadata data key structure

The `ReleaseMetadata` JSON stored under `data.releaseMetadata` SHALL use the following field names:

| JSON key | Go field | Description |
|---|---|---|
| `kind` | `Kind` | Always `"ModuleRelease"` |
| `apiVersion` | `APIVersion` | Always `"core.opmodel.dev/v1alpha1"` |
| `name` | `ReleaseName` | The release name (user-supplied `--release-name`, e.g. `"mc"`) |
| `namespace` | `ReleaseNamespace` | The Kubernetes namespace of the release |
| `uuid` | `ReleaseID` | Deterministic UUID v5 release identity |
| `lastTransitionTime` | `LastTransitionTime` | RFC 3339 timestamp of last change |

#### Scenario: ReleaseMetadata name field holds release name

- **WHEN** serializing a `ReleaseMetadata` for release name `mc` and module `minecraft`
- **THEN** the JSON `"name"` field SHALL be `"mc"`
- **AND** there SHALL be no `"releaseName"` field in the JSON
- **AND** the JSON `"uuid"` field SHALL hold the release identity UUID

### Requirement: Module metadata data key structure

The `ModuleMetadata` JSON stored under `data.moduleMetadata` SHALL use the following field names:

| JSON key | Go field | Description |
|---|---|---|
| `kind` | `Kind` | Always `"Module"` |
| `apiVersion` | `APIVersion` | Always `"core.opmodel.dev/v1alpha1"` |
| `name` | `Name` | The canonical module name (e.g. `"minecraft"`) |
| `uuid` | `UUID` | Module identity UUID (omitted if empty) |

#### Scenario: ModuleMetadata name field holds module name

- **WHEN** serializing a `ModuleMetadata` for module `minecraft` with UUID `a1b2c3d4-...`
- **THEN** the JSON `"name"` field SHALL be `"minecraft"`
- **AND** the JSON `"uuid"` field SHALL be `"a1b2c3d4-..."`

#### Scenario: ModuleMetadata uuid omitted when empty

- **WHEN** serializing a `ModuleMetadata` with an empty UUID
- **THEN** the JSON SHALL NOT contain a `"uuid"` field

### Requirement: Inventory metadata is write-once

The `ReleaseMetadata` and `ModuleMetadata` fields of an `InventorySecret` SHALL be set only at create time (when no previous inventory exists). On subsequent updates, the metadata fields SHALL be preserved verbatim from the previously unmarshaled Secret. `WriteInventory` SHALL accept `moduleName` and `moduleUUID` parameters that are used only when constructing a new inventory; on update they SHALL be ignored.

#### Scenario: Metadata preserved on re-apply

- **WHEN** `WriteInventory` is called for a release that already has an inventory Secret
- **THEN** the `ReleaseMetadata` and `ModuleMetadata` in the written Secret SHALL be identical to those in the existing Secret
- **AND** only the change history (index and change entries) SHALL be updated

#### Scenario: Metadata set from parameters on first apply

- **WHEN** `WriteInventory` is called for a release with no existing inventory Secret
- **THEN** `ReleaseMetadata.ReleaseName` SHALL be set from the release name
- **AND** `ModuleMetadata.Name` SHALL be set from the `moduleName` parameter
- **AND** `ModuleMetadata.UUID` SHALL be set from the `moduleUUID` parameter (may be empty)

## REMOVED Requirements

### Requirement: Inventory metadata enables future CRD migration (old — single metadata key)

**Reason**: Replaced by two typed metadata keys (`releaseMetadata`, `moduleMetadata`) with cleaner per-kind fields. The old `InventoryMetadata` struct with `json:"name"` = module name is removed.
**Migration**: Re-apply all releases with `opm mod apply`. Existing Secrets with `data.metadata` key must be deleted and re-created.
