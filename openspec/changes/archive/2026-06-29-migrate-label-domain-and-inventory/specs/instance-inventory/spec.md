## ADDED Requirements

<!-- Renamed from `release-inventory` (enhancement 0002 D4/D8/D10). Spec dir is git mv'd at archive. Only the requirements whose normative text changes under the rename (label-domain keys, ReleaseInventoryRecord/ReleaseMetadata type names + fields, the kind: "ModuleRelease" literal, the --release-name flag reference) are restated; unchanged requirements (entry identity, Secret-name convention, digest determinism, etc.) ride the archive spec-sync. -->

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

### Requirement: Inventory Secret CRUD operations

`GetInventory` SHALL first attempt a direct GET by constructed Secret name (`opm.<instanceName>.<instanceID>`). If the Secret is not found, it SHALL fall back to listing Secrets with label `module-instance.opmodel.dev/uuid=<instanceID>`. If no inventory is found (first-time apply), it SHALL return `nil, nil`. `WriteInventory` SHALL use full PUT semantics (create or replace) with optimistic concurrency via `resourceVersion`, preserving instance identity metadata and creator provenance from previously read records while updating the ownership inventory and deployed module version. `DeleteInventory` SHALL delete the inventory Secret and treat 404 as success (idempotent). <!-- Was: releaseName/releaseID, module-release.opmodel.dev/uuid (0002 D4/D8) -->

#### Scenario: GetInventory falls back to UUID label

- **WHEN** `GetInventory` cannot find the Secret by constructed name
- **AND** a Secret with label `module-instance.opmodel.dev/uuid=<instanceID>` exists
- **THEN** `GetInventory` SHALL return the record from that Secret
