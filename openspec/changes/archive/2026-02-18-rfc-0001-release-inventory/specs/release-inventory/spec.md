## ADDED Requirements

### Requirement: Inventory entry identity

An `InventoryEntry` SHALL represent a single tracked Kubernetes resource. Two entries SHALL be considered identity-equal when their Group, Kind, Namespace, Name, and Component fields all match. The Version field SHALL be excluded from identity comparison to prevent false orphans during Kubernetes API version migrations.

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

The system SHALL construct an `InventoryEntry` from a `*build.Resource` by extracting Group and Kind from the resource's GVK, Version from the GVK's Version field, Namespace and Name from the resource's metadata, and Component from the `build.Resource.Component` field.

#### Scenario: Build entry from a namespaced Deployment

- **WHEN** constructing an entry from a `*build.Resource` with GVK `apps/v1/Deployment`, name `my-app`, namespace `production`, component `app`
- **THEN** the entry SHALL have Group=`apps`, Kind=`Deployment`, Namespace=`production`, Name=`my-app`, Version=`v1`, Component=`app`

#### Scenario: Build entry from a cluster-scoped ClusterRole

- **WHEN** constructing an entry from a `*build.Resource` with GVK `rbac.authorization.k8s.io/v1/ClusterRole`, name `my-role`, empty namespace, component `rbac`
- **THEN** the entry SHALL have Group=`rbac.authorization.k8s.io`, Kind=`ClusterRole`, Namespace=`""`, Name=`my-role`, Version=`v1`, Component=`rbac`

### Requirement: Inventory Secret name convention

The inventory Secret name SHALL follow the pattern `opm.<release-name>.<release-id>` where release-name is the module release name and release-id is the deterministic UUID v5 release identity.

#### Scenario: Secret name for a module release

- **WHEN** computing the Secret name for release name `jellyfin` with release ID `a3b8f2e1-1234-5678-9abc-def012345678`
- **THEN** the Secret name SHALL be `opm.jellyfin.a3b8f2e1-1234-5678-9abc-def012345678`

### Requirement: Inventory Secret labels

The inventory Secret SHALL carry the following six labels:

| Label key | Value | Description |
|---|---|---|
| `app.kubernetes.io/managed-by` | `open-platform-model` | Standard managed-by label |
| `module.opmodel.dev/name` | canonical module name (e.g. `minecraft`) | The module definition name, not the release name |
| `module-release.opmodel.dev/name` | release name (e.g. `mc`) | The user-supplied release name (`--release-name`) |
| `module.opmodel.dev/namespace` | release namespace | Namespace the release is deployed into |
| `module-release.opmodel.dev/uuid` | release UUID | Deterministic UUID v5 release identity |
| `opmodel.dev/component` | `inventory` | Distinguishes the inventory Secret from application resources |

The `module.opmodel.dev/name` and `module-release.opmodel.dev/name` labels are distinct: the former is the canonical module name from the module definition, the latter is the user-supplied name used to identify a specific deployment of that module. Both are required to enable discovery by either identifier. The `opmodel.dev/component` label is a new key distinct from `component.opmodel.dev/name`.

#### Scenario: Inventory Secret has correct labels

- **WHEN** creating an inventory Secret for module `minecraft` deployed as release name `mc` in namespace `games` with release ID `abc123`
- **THEN** the Secret SHALL have all six labels:
  - `app.kubernetes.io/managed-by: open-platform-model`
  - `module.opmodel.dev/name: minecraft`
  - `module-release.opmodel.dev/name: mc`
  - `module.opmodel.dev/namespace: games`
  - `module-release.opmodel.dev/uuid: abc123`
  - `opmodel.dev/component: inventory`
- **AND** the Secret type SHALL be `opmodel.dev/release`

### Requirement: Inventory Secret serialization roundtrip

`MarshalToSecret` SHALL serialize an `InventorySecret` to a typed `corev1.Secret` using `stringData` keys: `metadata` (JSON-encoded `InventoryMetadata`), `index` (JSON-encoded `[]string` of change IDs), and one key per change entry (`change-sha1-<8hex>` with JSON-encoded `ChangeEntry`). `UnmarshalFromSecret` SHALL deserialize a `corev1.Secret` back into an `InventorySecret`, handling both `stringData` and `data` (base64-encoded) fields since Kubernetes returns `data` on GET. The `resourceVersion` from the Secret SHALL be preserved as an unexported field for optimistic concurrency.

#### Scenario: Marshal and unmarshal roundtrip

- **WHEN** marshaling an `InventorySecret` with metadata, 2 change entries, and an index
- **AND** unmarshaling the resulting `corev1.Secret`
- **THEN** the resulting `InventorySecret` SHALL be identical to the original

#### Scenario: Unmarshal from Kubernetes GET response

- **WHEN** unmarshaling a Secret returned by the Kubernetes API (base64-encoded `data` field)
- **THEN** the resulting `InventorySecret` SHALL contain correct values
- **AND** the `resourceVersion` SHALL be preserved from the Secret's metadata

#### Scenario: Empty inventory with no changes

- **WHEN** marshaling an `InventorySecret` with empty index and no change entries
- **THEN** the resulting Secret SHALL be valid with an empty JSON array for index

### Requirement: Inventory metadata enables future CRD migration

The `InventoryMetadata` SHALL include `kind: "ModuleRelease"` and `apiVersion: "core.opmodel.dev/v1alpha1"` fields to enable future migration from a Secret to a CRD without changing the data model.

#### Scenario: Metadata kind and apiVersion

- **WHEN** creating inventory metadata
- **THEN** the `kind` field SHALL be `"ModuleRelease"`
- **AND** the `apiVersion` field SHALL be `"core.opmodel.dev/v1alpha1"`

### Requirement: Deterministic manifest digest

The system SHALL compute a SHA256 digest of rendered resources that is deterministic regardless of input order. Resources SHALL be sorted with a 5-key total ordering: weight (ascending via `pkg/weights`), API group (alphabetical), Kind (alphabetical), Namespace (alphabetical), Name (alphabetical). Each resource SHALL be serialized independently via `json.Marshal` (which sorts map keys alphabetically). Serialized bytes SHALL be concatenated with newline separators and hashed with SHA256. The result SHALL be formatted as `sha256:<hex>`.

#### Scenario: Same resources in different input order produce same digest

- **WHEN** computing the digest of resources [Deployment/app, Service/app, ConfigMap/config]
- **AND** computing the digest of resources [ConfigMap/config, Deployment/app, Service/app]
- **THEN** both digests SHALL be identical

#### Scenario: Content change produces different digest

- **WHEN** computing the digest of a resource set
- **AND** modifying any field in any resource
- **THEN** the new digest SHALL differ from the original

#### Scenario: Added or removed resource changes digest

- **WHEN** computing the digest of a resource set with 3 resources
- **AND** computing the digest with one resource removed
- **THEN** the digests SHALL differ

### Requirement: Change ID computation

The system SHALL compute a change ID as `change-sha1-<8hex>` where the hash input is the concatenation of module path, module version, resolved values string, and manifest digest. This ensures that module upgrades, value changes, and content changes all produce distinct change IDs.

#### Scenario: Same inputs produce same change ID

- **WHEN** computing the change ID with path=`opmodel.dev/modules/jellyfin@v1`, version=`1.0.0`, values=`{port: 8096}`, digest=`sha256:abc123`
- **AND** computing again with identical inputs
- **THEN** both change IDs SHALL be identical

#### Scenario: Version bump with identical output produces different change ID

- **WHEN** computing the change ID with version=`1.0.0` and a given digest
- **AND** computing with version=`1.1.0` and the same digest
- **THEN** the change IDs SHALL differ

#### Scenario: Local module uses empty version string

- **WHEN** computing the change ID for a local module (no version)
- **THEN** the version input SHALL be empty string
- **AND** the `ModuleRef.Local` field SHALL be `true`

### Requirement: Change history management

The index SHALL be an ordered list of change IDs with newest first. When a change ID already exists in the index (idempotent re-apply), it SHALL be moved to the front and its entry overwritten with an updated timestamp. The index SHALL NOT grow when the same inputs are re-applied.

#### Scenario: New change appended to front

- **WHEN** applying a new change with ID `change-sha1-aaa11111`
- **AND** the current index is `[change-sha1-bbb22222]`
- **THEN** the index SHALL become `[change-sha1-aaa11111, change-sha1-bbb22222]`

#### Scenario: Idempotent re-apply moves to front

- **WHEN** re-applying with the same inputs producing `change-sha1-bbb22222`
- **AND** the current index is `[change-sha1-aaa11111, change-sha1-bbb22222]`
- **THEN** the index SHALL become `[change-sha1-bbb22222, change-sha1-aaa11111]`

#### Scenario: Identical re-apply at head skips inventory write and preserves original timestamp

- **WHEN** re-applying with inputs that produce `change-sha1-bbb22222`
- **AND** the current index is already `[change-sha1-bbb22222, ...]` (the computed change ID is already at the front)
- **THEN** the inventory Secret write SHALL be skipped entirely
- **AND** the original timestamp of the first apply SHALL be preserved
- **AND** the index SHALL remain unchanged

### Requirement: History pruning

When the index exceeds the maximum history size, the oldest entries (at the tail) SHALL be removed from both the index and the changes map. The default maximum history SHALL be 10.

#### Scenario: Pruning removes oldest entry

- **WHEN** the index has 10 entries and a new change is added
- **AND** `maxHistory` is 10
- **THEN** the oldest entry SHALL be removed from both index and changes map
- **AND** the index length SHALL remain 10

### Requirement: Inventory Secret CRUD operations

`GetInventory` SHALL first attempt a direct GET by constructed Secret name (`opm.<name>.<releaseID>`). If the Secret is not found, it SHALL fall back to listing Secrets with label `module-release.opmodel.dev/uuid=<releaseID>`. If no inventory is found (first-time apply), it SHALL return `nil, nil`. `WriteInventory` SHALL use full PUT semantics (create or replace) with optimistic concurrency via `resourceVersion`. `DeleteInventory` SHALL delete the inventory Secret and treat 404 as success (idempotent).

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

#### Scenario: Delete is idempotent

- **WHEN** calling `DeleteInventory` and the Secret does not exist
- **THEN** the operation SHALL succeed without error
