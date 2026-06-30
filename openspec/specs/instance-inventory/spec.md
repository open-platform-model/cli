## Purpose

Defines the ownership-focused instance inventory data model, persisted instance record envelope, serialization format, and CRUD semantics used by the CLI for pruning, discovery, and instance metadata.

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

The inventory Secret name SHALL follow the pattern `opm.<instance-name>.<instance-id>` where instance-name is the module instance name and instance-id is the deterministic UUID v5 instance identity.

#### Scenario: Secret name for a module instance

- **WHEN** computing the Secret name for instance name `jellyfin` with instance ID `a3b8f2e1-1234-5678-9abc-def012345678`
- **THEN** the Secret name SHALL be `opm.jellyfin.a3b8f2e1-1234-5678-9abc-def012345678`

### Requirement: Inventory Secret labels

The inventory Secret SHALL carry these OPM identity labels, derived from the instance metadata: <!-- Was: module-release.opmodel.dev/* (0002 D4) -->

| Label | Value | Source |
| --- | --- | --- |
| `module-instance.opmodel.dev/name` | instance name (e.g. `mc`) | The user-supplied instance name (`--instance-name`) |
| `module-instance.opmodel.dev/namespace` | instance namespace | Namespace the instance is deployed into |
| `module-instance.opmodel.dev/uuid` | instance UUID | Deterministic UUID v5 instance identity |

#### Scenario: Inventory Secret label values

- **WHEN** an inventory Secret is written for instance name `mc` in namespace `games`
- **THEN** the Secret SHALL carry `module-instance.opmodel.dev/name: mc`
- **AND** SHALL carry `module-instance.opmodel.dev/namespace: games`
- **AND** SHALL carry `module-instance.opmodel.dev/uuid` set to the deterministic instance UUID

### Requirement: Inventory represents current ownership only

The public inventory contract SHALL represent the current set of resources owned by a instance. It SHALL contain the current `entries` list and MAY include ownership summary fields such as `revision`, `digest`, and `count`.

The public inventory contract MUST NOT require or embed:

- raw values
- source path or source version metadata
- per-change timestamps
- history index
- change map
- remediation counters

#### Scenario: Ownership-only inventory contains current resource refs

- **WHEN** a instance currently owns a Deployment, Service, and Ingress
- **THEN** the inventory SHALL contain exactly three entries representing those resources
- **AND** no history entries SHALL be required to determine current ownership

#### Scenario: Inventory exposes summary metadata without history

- **WHEN** an inventory includes `revision`, `digest`, and `count`
- **THEN** those fields SHALL describe the current inventory set only
- **AND** they SHALL NOT imply a retained change history

### Requirement: Persisted instance inventory record preserves instance and module metadata

The CLI persisted instance inventory record SHALL preserve `instanceMetadata` and `moduleMetadata` alongside the ownership-only inventory so the CLI can identify the instance, identify the module, and report deployed module version without retaining inventory change history.

#### Scenario: Persisted record includes module version without change history

- **WHEN** a instance is persisted using the v2 record shape
- **THEN** the record SHALL contain `moduleMetadata.version` for the deployed module version
- **AND** that version SHALL NOT require a latest history entry to be read

### Requirement: Persisted instance inventory record stores creator provenance at the top level

The CLI persisted instance inventory record SHALL store `createdBy` as a top-level field rather than nesting it inside `instanceMetadata`.

#### Scenario: Top-level creator provenance

- **WHEN** a persisted instance inventory record is read
- **THEN** the CLI SHALL be able to determine whether it is CLI-managed or controller-managed from the top-level `createdBy` field
- **AND** that determination SHALL NOT depend on inventory history

### Requirement: Inventory Secret serialization roundtrip

`MarshalToSecret` SHALL serialize an `InstanceInventoryRecord` to a typed `corev1.Secret` using a single JSON-encoded document stored under `data.inventory`. `UnmarshalFromSecret` SHALL deserialize a `corev1.Secret` back into an `InstanceInventoryRecord`, handling both `stringData` and `data` (base64-encoded) forms. If the decoded record omits `inventory.entries`, deserialization SHALL normalize that field to an empty list. The `resourceVersion` from the Secret SHALL be preserved as an unexported field for optimistic concurrency. <!-- Was: ReleaseInventoryRecord (0002 D8) -->

#### Scenario: Roundtrip preserves record

- **WHEN** marshaling an `InstanceInventoryRecord` with top-level `createdBy`, instance metadata, module metadata, and an ownership inventory
- **AND** then unmarshaling the resulting Secret
- **THEN** the resulting `InstanceInventoryRecord` SHALL be identical to the original

### Requirement: Inventory metadata enables future CRD migration

The `InstanceMetadata` SHALL include `kind: "ModuleInstance"` and `apiVersion: "core.opmodel.dev/v1alpha1"` fields. The `ModuleMetadata` SHALL include `kind: "Module"` and `apiVersion: "core.opmodel.dev/v1alpha1"` fields. Both enable future migration from Secrets to CRDs without changing the persisted instance inventory record shape. <!-- Was: ReleaseMetadata, kind: "ModuleRelease" (0002 D8) -->

#### Scenario: InstanceMetadata kind and apiVersion

- **WHEN** an `InstanceMetadata` is serialized
- **THEN** the `kind` field SHALL be `"ModuleInstance"`
- **AND** the `apiVersion` field SHALL be `"core.opmodel.dev/v1alpha1"`

### Requirement: Instance metadata data key structure

The persisted instance metadata SHALL use these data keys, mapped from the `InstanceMetadata` Go struct: <!-- Was: Release metadata / ReleaseMetadata (0002 D8) -->

| Data key | Go field | Value |
| --- | --- | --- |
| `kind` | `Kind` | Always `"ModuleInstance"` |
| `name` | `InstanceName` | The instance name (user-supplied `--instance-name`, e.g. `"mc"`) |
| `namespace` | `InstanceNamespace` | The Kubernetes namespace of the instance |
| `uuid` | `InstanceID` | Deterministic UUID v5 instance identity |

#### Scenario: InstanceMetadata name field holds instance name

- **WHEN** serializing an `InstanceMetadata` for instance name `mc` and module `minecraft`
- **THEN** the `name` data key SHALL hold `"mc"`

### Requirement: Module metadata data key structure

The `moduleMetadata` object inside the persisted instance inventory record SHALL use the following JSON field names:

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

`GetInventory` SHALL first attempt a direct GET by constructed Secret name (`opm.<instanceName>.<instanceID>`). If the Secret is not found, it SHALL fall back to listing Secrets with label `module-instance.opmodel.dev/uuid=<instanceID>`. If no inventory is found (first-time apply), it SHALL return `nil, nil`. `WriteInventory` SHALL use full PUT semantics (create or replace) with optimistic concurrency via `resourceVersion`, preserving instance identity metadata and creator provenance from previously read records while updating the ownership inventory and deployed module version. `DeleteInventory` SHALL delete the inventory Secret and treat 404 as success (idempotent). <!-- Was: releaseName/releaseID, module-release.opmodel.dev/uuid (0002 D4/D8) -->

#### Scenario: GetInventory falls back to UUID label

- **WHEN** `GetInventory` cannot find the Secret by constructed name
- **AND** a Secret with label `module-instance.opmodel.dev/uuid=<instanceID>` exists
- **THEN** `GetInventory` SHALL return the record from that Secret

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
