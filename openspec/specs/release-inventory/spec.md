## Purpose

Defines the ownership-focused release inventory data model, persisted release record envelope, serialization format, and CRUD semantics used by the CLI for pruning, discovery, and release metadata.

## Requirements

### Requirement: Inventory entry identity

An `InventoryEntry` SHALL represent a single currently owned Kubernetes resource. Two entries SHALL be considered identity-equal when their Group, Kind, Namespace, Name, and Component fields all match. The Version field SHALL be excluded from identity comparison to prevent false orphans during Kubernetes API version migrations.

#### Scenario: Same resource with different API version

- **WHEN** comparing two entries with identical Group, Kind, Namespace, Name, Component but different Version
- **THEN** the entries SHALL be identity-equal

#### Scenario: Same resource with different component

- **WHEN** comparing two entries with identical Group, Kind, Namespace, Name, Version but different Component
- **THEN** the entries SHALL NOT be identity-equal

### Requirement: Kubernetes identity equality

A separate K8s identity comparison SHALL compare entries by Group, Kind, Namespace, and Name only (excluding both Version and Component). This SHALL be used by the component-rename safety check to detect when the same Kubernetes resource appears under a different component name.

#### Scenario: Same K8s resource under different components

- **WHEN** comparing two entries with identical Group, Kind, Namespace, Name but different Component
- **THEN** the entries SHALL be K8s-identity-equal

#### Scenario: Different K8s resources

- **WHEN** comparing two entries with different Name
- **THEN** the entries SHALL NOT be K8s-identity-equal

### Requirement: Entry construction from rendered resource

The system SHALL construct an `InventoryEntry` from a rendered Kubernetes resource by extracting Group and Kind from the resource's GVK, Version from the GVK's Version field, Namespace and Name from the resource's metadata, and Component from the OPM component label when present.

#### Scenario: Build entry from a namespaced Deployment

- **WHEN** constructing an entry from a resource with GVK `apps/v1/Deployment`, name `my-app`, namespace `production`, component `app`
- **THEN** the entry SHALL have Group=`apps`, Kind=`Deployment`, Namespace=`production`, Name=`my-app`, Version=`v1`, Component=`app`

#### Scenario: Build entry from a cluster-scoped ClusterRole

- **WHEN** constructing an entry from a resource with GVK `rbac.authorization.k8s.io/v1/ClusterRole`, name `my-role`, empty namespace, component `rbac`
- **THEN** the entry SHALL have Group=`rbac.authorization.k8s.io`, Kind=`ClusterRole`, Namespace=`""`, Name=`my-role`, Version=`v1`, Component=`rbac`

### Requirement: Inventory Secret name convention

The inventory Secret name SHALL follow the pattern `opm.<release-name>.<release-id>` where release-name is the module release name and release-id is the deterministic UUID v5 release identity.

#### Scenario: Secret name for a module release

- **WHEN** computing the Secret name for release name `jellyfin` with release ID `a3b8f2e1-1234-5678-9abc-def012345678`
- **THEN** the Secret name SHALL be `opm.jellyfin.a3b8f2e1-1234-5678-9abc-def012345678`

### Requirement: Inventory Secret labels

The inventory Secret SHALL carry the following five labels:

| Label key | Value | Description |
|---|---|---|
| `app.kubernetes.io/managed-by` | `open-platform-model` | Standard managed-by label |
| `module-release.opmodel.dev/name` | release name (e.g. `mc`) | The user-supplied release name (`--release-name`) |
| `module-release.opmodel.dev/namespace` | release namespace | Namespace the release is deployed into |
| `module-release.opmodel.dev/uuid` | release UUID | Deterministic UUID v5 release identity |
| `opmodel.dev/component` | `inventory` | Distinguishes the inventory Secret from application resources |

The `module.opmodel.dev/name` and `module.opmodel.dev/namespace` labels SHALL NOT be present on the inventory Secret. Module identity is carried in the persisted release inventory record's `moduleMetadata` instead.

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

### Requirement: Inventory represents current ownership only

The public inventory contract SHALL represent the current set of resources owned by a release. It SHALL contain the current `entries` list and MAY include ownership summary fields such as `revision`, `digest`, and `count`.

The public inventory contract MUST NOT require or embed:

- raw values
- source path or source version metadata
- per-change timestamps
- history index
- change map
- remediation counters

#### Scenario: Ownership-only inventory contains current resource refs

- **WHEN** a release currently owns a Deployment, Service, and Ingress
- **THEN** the inventory SHALL contain exactly three entries representing those resources
- **AND** no history entries SHALL be required to determine current ownership

#### Scenario: Inventory exposes summary metadata without history

- **WHEN** an inventory includes `revision`, `digest`, and `count`
- **THEN** those fields SHALL describe the current inventory set only
- **AND** they SHALL NOT imply a retained change history

### Requirement: Persisted release inventory record preserves release and module metadata

The CLI persisted release inventory record SHALL preserve `releaseMetadata` and `moduleMetadata` alongside the ownership-only inventory so the CLI can identify the release, identify the module, and report deployed module version without retaining inventory change history.

#### Scenario: Persisted record includes module version without change history

- **WHEN** a release is persisted using the v2 record shape
- **THEN** the record SHALL contain `moduleMetadata.version` for the deployed module version
- **AND** that version SHALL NOT require a latest history entry to be read

### Requirement: Persisted release inventory record stores creator provenance at the top level

The CLI persisted release inventory record SHALL store `createdBy` as a top-level field rather than nesting it inside `releaseMetadata`.

#### Scenario: Top-level creator provenance

- **WHEN** a persisted release inventory record is read
- **THEN** the CLI SHALL be able to determine whether it is CLI-managed or controller-managed from the top-level `createdBy` field
- **AND** that determination SHALL NOT depend on inventory history

### Requirement: Inventory Secret serialization roundtrip

`MarshalToSecret` SHALL serialize a `ReleaseInventoryRecord` to a typed `corev1.Secret` using a single JSON-encoded document stored under `data.inventory`. `UnmarshalFromSecret` SHALL deserialize a `corev1.Secret` back into a `ReleaseInventoryRecord`, handling both `stringData` and `data` (base64-encoded) forms. If the decoded record omits `inventory.entries`, deserialization SHALL normalize that field to an empty list. The `resourceVersion` from the Secret SHALL be preserved as an unexported field for optimistic concurrency.

#### Scenario: Marshal and unmarshal roundtrip

- **WHEN** marshaling a `ReleaseInventoryRecord` with top-level `createdBy`, release metadata, module metadata, and an ownership inventory
- **AND** unmarshaling the resulting `corev1.Secret`
- **THEN** the resulting `ReleaseInventoryRecord` SHALL be identical to the original

#### Scenario: Unmarshal from Kubernetes GET response

- **WHEN** unmarshaling a Secret returned by the Kubernetes API (base64-encoded `data` field)
- **THEN** the resulting `ReleaseInventoryRecord` SHALL contain correct values
- **AND** the `resourceVersion` SHALL be preserved from the Secret's metadata

#### Scenario: Missing inventory key is an error

- **WHEN** unmarshaling a Secret that has no `inventory` key in its data
- **THEN** deserialization SHALL fail with a clear error

### Requirement: Inventory metadata enables future CRD migration

The `ReleaseMetadata` SHALL include `kind: "ModuleRelease"` and `apiVersion: "core.opmodel.dev/v1alpha1"` fields. The `ModuleMetadata` SHALL include `kind: "Module"` and `apiVersion: "core.opmodel.dev/v1alpha1"` fields. Both enable future migration from Secrets to CRDs without changing the persisted release inventory record shape.

#### Scenario: ReleaseMetadata kind and apiVersion

- **WHEN** creating release metadata
- **THEN** the `kind` field SHALL be `"ModuleRelease"`
- **AND** the `apiVersion` field SHALL be `"core.opmodel.dev/v1alpha1"`

#### Scenario: ModuleMetadata kind and apiVersion

- **WHEN** creating module metadata
- **THEN** the `kind` field SHALL be `"Module"`
- **AND** the `apiVersion` field SHALL be `"core.opmodel.dev/v1alpha1"`

### Requirement: Release metadata data key structure

The `releaseMetadata` object inside the persisted release inventory record SHALL use the following JSON field names:

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

#### Scenario: ReleaseMetadata omits createdBy

- **WHEN** serializing a persisted release inventory record
- **THEN** the `releaseMetadata` object SHALL NOT contain a `"createdBy"` field

### Requirement: Module metadata data key structure

The `moduleMetadata` object inside the persisted release inventory record SHALL use the following JSON field names:

| JSON key | Go field | Description |
|---|---|---|
| `kind` | `Kind` | Always `"Module"` |
| `apiVersion` | `APIVersion` | Always `"core.opmodel.dev/v1alpha1"` |
| `name` | `Name` | The canonical module name (e.g. `"minecraft"`) |
| `uuid` | `UUID` | Module identity UUID (omitted if empty) |
| `version` | `Version` | Deployed module version (omitted if empty) |

#### Scenario: ModuleMetadata name field holds module name

- **WHEN** serializing a `ModuleMetadata` for module `minecraft` with UUID `a1b2c3d4-...`
- **THEN** the JSON `"name"` field SHALL be `"minecraft"`
- **AND** the JSON `"uuid"` field SHALL be `"a1b2c3d4-..."`

#### Scenario: ModuleMetadata uuid omitted when empty

- **WHEN** serializing a `ModuleMetadata` with an empty UUID
- **THEN** the JSON SHALL NOT contain a `"uuid"` field

#### Scenario: ModuleMetadata version recorded when present

- **WHEN** serializing a `ModuleMetadata` with version `1.2.3`
- **THEN** the JSON SHALL contain `"version": "1.2.3"`

### Requirement: Inventory Secret CRUD operations

`GetInventory` SHALL first attempt a direct GET by constructed Secret name (`opm.<releaseName>.<releaseID>`). If the Secret is not found, it SHALL fall back to listing Secrets with label `module-release.opmodel.dev/uuid=<releaseID>`. If no inventory is found (first-time apply), it SHALL return `nil, nil`. `WriteInventory` SHALL use full PUT semantics (create or replace) with optimistic concurrency via `resourceVersion`, preserving release identity metadata and creator provenance from previously read records while updating the ownership inventory and deployed module version. `DeleteInventory` SHALL delete the inventory Secret and treat 404 as success (idempotent).

#### Scenario: First-time apply returns nil inventory

- **WHEN** calling `GetInventory` for a release that has never been applied
- **THEN** the result SHALL be `nil, nil` (no error)

#### Scenario: Get by name succeeds

- **WHEN** calling `GetInventory` and the Secret exists with the expected name
- **THEN** the inventory SHALL be returned from the direct GET

#### Scenario: Get falls back to label lookup

- **WHEN** calling `GetInventory` and the named Secret does not exist
- **AND** a Secret with label `module-release.opmodel.dev/uuid=<releaseID>` exists
- **THEN** the inventory SHALL be returned from the label-based list

#### Scenario: Write creates new Secret on first write

- **WHEN** calling `WriteInventory` with no existing Secret
- **THEN** the Secret SHALL be created

#### Scenario: Write replaces existing Secret with optimistic concurrency

- **WHEN** calling `WriteInventory` with a `resourceVersion` from a previous read
- **THEN** the Secret SHALL be replaced using the `resourceVersion` for conflict detection
- **AND** a concurrent modification SHALL cause a conflict error

#### Scenario: Existing creator provenance is preserved on update

- **WHEN** `WriteInventory` updates an existing persisted release inventory record
- **THEN** `createdBy` SHALL remain the normalized value from the existing record rather than being overwritten by a new creator hint

#### Scenario: Deployed module version is refreshed on update

- **WHEN** `WriteInventory` is called with a non-empty deployed module version for an existing record
- **THEN** `moduleMetadata.version` SHALL be updated to that deployed version

#### Scenario: Delete is idempotent

- **WHEN** calling `DeleteInventory` and the Secret does not exist
- **THEN** the operation SHALL succeed without error

### Requirement: Inventory labels remain unchanged when provenance is added

Adding provenance support SHALL NOT change the inventory Secret label set. Inventory Secrets SHALL continue to use the existing five-label selector contract.

#### Scenario: Provenance does not add new label

- **WHEN** an inventory Secret includes top-level `createdBy`
- **THEN** the Secret SHALL still have exactly the existing five labels used for inventory discovery

### Requirement: Deterministic inventory digest

When the system computes an inventory digest, it SHALL do so deterministically from the current owned resource set regardless of input order.

#### Scenario: Same entries in different order produce same digest

- **WHEN** computing the digest for the same ownership set in two different slice orders
- **THEN** the digest SHALL be identical

#### Scenario: Added or removed resource changes digest

- **WHEN** computing the digest of an ownership set with 3 resources
- **AND** computing the digest with one resource removed
- **THEN** the digests SHALL differ

#### Scenario: Component rename changes digest

- **WHEN** two ownership sets differ only by the `component` field of an entry
- **THEN** the digests SHALL differ because inventory identity includes component ownership
