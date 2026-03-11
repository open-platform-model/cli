## MODIFIED Requirements

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
| `createdBy` | `CreatedBy` | Original release creator: `cli` or `controller`; omitted only for legacy inventories |

#### Scenario: ReleaseMetadata name field holds release name

- **WHEN** serializing a `ReleaseMetadata` for release name `mc` and module `minecraft`
- **THEN** the JSON `"name"` field SHALL be `"mc"`
- **AND** there SHALL be no `"releaseName"` field in the JSON
- **AND** the JSON `"uuid"` field SHALL hold the release identity UUID

#### Scenario: ReleaseMetadata createdBy records creator

- **WHEN** serializing release metadata for a newly created CLI-managed release
- **THEN** the JSON `"createdBy"` field SHALL be `"cli"`

#### Scenario: Legacy release metadata omits createdBy

- **WHEN** deserializing a pre-existing inventory Secret with no `createdBy` field
- **THEN** deserialization SHALL succeed without migration

### Requirement: Inventory metadata is write-once

The `ReleaseMetadata` and `ModuleMetadata` fields of an `InventorySecret` SHALL be set only at create time (when no previous inventory exists). On subsequent updates, the metadata fields SHALL be preserved verbatim from the previously unmarshaled Secret. `WriteInventory` SHALL accept the values needed to create new metadata, but on update it SHALL preserve the existing release and module metadata exactly, including `createdBy`.

#### Scenario: Metadata preserved on re-apply

- **WHEN** `WriteInventory` is called for a release that already has an inventory Secret
- **THEN** the `ReleaseMetadata` and `ModuleMetadata` in the written Secret SHALL be identical to those in the existing Secret
- **AND** only the change history (index and change entries) SHALL be updated

#### Scenario: Metadata set from parameters on first apply

- **WHEN** `WriteInventory` is called for a release with no existing inventory Secret
- **THEN** `ReleaseMetadata.ReleaseName` SHALL be set from the release name
- **AND** `ReleaseMetadata.CreatedBy` SHALL be set from the creator parameter
- **AND** `ModuleMetadata.Name` SHALL be set from the `moduleName` parameter
- **AND** `ModuleMetadata.UUID` SHALL be set from the `moduleUUID` parameter (may be empty)

## ADDED Requirements

### Requirement: Inventory labels remain unchanged when provenance is added

Adding provenance support SHALL NOT change the inventory Secret label set. Inventory Secrets SHALL continue to use the existing five-label selector contract.

#### Scenario: Provenance does not add new label
- **WHEN** an inventory Secret includes `createdBy`
- **THEN** the Secret SHALL still have exactly the existing five labels used for inventory discovery
