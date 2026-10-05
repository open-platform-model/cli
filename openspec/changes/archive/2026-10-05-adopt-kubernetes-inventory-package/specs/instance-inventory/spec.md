## ADDED Requirements

### Requirement: Inventory entries are the library's component-blind entries

The CLI SHALL represent each currently owned Kubernetes resource as the library's `opm/k8s/inventory.Entry` (group, kind, namespace, name, API version, component), and SHALL NOT declare an entry type of its own. Two entries SHALL be the same object when their group, kind, namespace and name agree, as `inventory.SameObject` compares them: neither the API version nor the component SHALL count, so an API version migration or a component rename never makes an object look like a different one. Source: 0012:D7:R1.

#### Scenario: Same object at a different API version

- **WHEN** comparing two entries with identical group, kind, namespace, name and component but a different API version
- **THEN** the entries SHALL be the same object

#### Scenario: Same object under a different component

- **WHEN** comparing two entries with identical group, kind, namespace, name and API version but a different component
- **THEN** the entries SHALL be the same object

#### Scenario: Different objects

- **WHEN** comparing two entries that differ in name
- **THEN** the entries SHALL NOT be the same object

## MODIFIED Requirements

### Requirement: Entry construction from rendered resource

The system SHALL construct an inventory entry from a rendered Kubernetes resource with the library's `inventory.NewEntry`, which takes Group and Kind from the resource's GVK, Version from the GVK's Version field, Namespace and Name from the resource's metadata, and Component from the OPM component label when present.

#### Scenario: Build entry from a namespaced Deployment

- **WHEN** constructing an entry from a resource with GVK `apps/v1/Deployment`, name `my-app`, namespace `production`, component `app`
- **THEN** the entry SHALL have Group=`apps`, Kind=`Deployment`, Namespace=`production`, Name=`my-app`, Version=`v1`, Component=`app`

#### Scenario: Build entry from a cluster-scoped ClusterRole

- **WHEN** constructing an entry from a resource with GVK `rbac.authorization.k8s.io/v1/ClusterRole`, name `my-role`, empty namespace, component `rbac`
- **THEN** the entry SHALL have Group=`rbac.authorization.k8s.io`, Kind=`ClusterRole`, Namespace=`""`, Name=`my-role`, Version=`v1`, Component=`rbac`

### Requirement: CLI writes a strict status subset via the status subresource

After resources are applied and pruned, the CLI SHALL write, via the status subresource with field manager `opm-cli`: `status.inventory` (revision, digest, count, entries), `status.instanceUUID`, `status.lastAppliedRenderDigest`, `status.lastAppliedSourceDigest`, `status.lastAppliedConfigDigest`, and `status.lastAppliedAt`. The CLI MUST NOT write `status.conditions`, `status.observedGeneration`, `status.lastAttempted*`, `status.failureCounters`, `status.history`, or `status.nextRetryAt`.

The digest fields SHALL be operator-parity: `lastAppliedRenderDigest` SHALL be the library's shared render digest, `opm/k8s/inventory.RenderDigest`, over the render's single export (see `kernel-render`), which leaves the managed-by label value out so the CLI and the operator digest one render to the same value (0012:D6:R2); `status.inventory.digest` SHALL be the library's `inventory.Digest` of the written entries (see "Deterministic inventory digest"); `lastAppliedSourceDigest` SHALL be computed as the operator's `ModuleSourceDigest` (SHA-256 of the canonical `path@version` reference — identical on both actors' CUE-native paths), and `lastAppliedConfigDigest` SHALL match the operator's `ConfigDigest` canonical-JSON semantics including the empty case (SHA-256 of no bytes).

#### Scenario: Status subset after successful apply

- **WHEN** an apply deploys 3 resources successfully
- **THEN** `status.inventory.count` SHALL be 3 and `status.lastAppliedAt` SHALL be set
- **AND** `status.conditions` SHALL NOT be present in the CLI's applied status document

#### Scenario: Revision increments across applies

- **WHEN** a second apply succeeds for an instance whose `status.inventory.revision` was 1
- **THEN** the written `status.inventory.revision` SHALL be 2

#### Scenario: Source digest matches the operator's computation

- **WHEN** the CLI applies a module with canonical reference `opmodel.dev/modules/test/podinfo@v0` at version `0.1.2`
- **THEN** `status.lastAppliedSourceDigest` SHALL equal the operator's `ModuleSourceDigest` for the same path and version

#### Scenario: Render digest matches the operator's algorithm

- **WHEN** the CLI and the operator compile the same instance against the same Platform spec
- **THEN** the two render digests SHALL be byte-identical
- **AND** the CLI's value SHALL equal `inventory.RenderDigest` of the render's export, which the operator also computes

#### Scenario: Stored digests change once on upgrade

- **WHEN** an instance last applied by a CLI release that predates the library digests is applied again with no change to its objects
- **THEN** `status.inventory.digest` and `status.lastAppliedRenderDigest` SHALL change to the library values
- **AND** the entries, revision increment and count SHALL be written as for any other apply

### Requirement: Entry wire shape targets the CRD schema

Conversion between the library's inventory entry and the CR's `status.inventory.entries[]` SHALL be performed by explicit mapping functions in the CLI that produce and consume the CRD's field names (`group`, `kind`, `namespace`, `name`, `v`, `component`). The library entry carries no struct tags, so no Go struct tag SHALL decide the wire shape. The mapping SHALL round-trip losslessly. The reader of a legacy inventory Secret SHALL decode the Secret's JSON with the same field names, `v` for the API version, into library entries.

#### Scenario: Version serializes as `v`

- **WHEN** an entry with Version `v1` is written to the CR
- **THEN** the entry object in `status.inventory.entries[]` SHALL carry the key `v` with value `v1`

#### Scenario: Round-trip preserves the entry set

- **WHEN** an entry list is written to a CR and read back
- **THEN** the resulting entries SHALL equal the originals

#### Scenario: Legacy Secret entries keep their API version

- **WHEN** a legacy inventory Secret whose entry carries `"v": "v1"` is read for migration
- **THEN** the migrated entry SHALL have Version `v1`

### Requirement: Inventory represents current ownership only

The inventory the CLI records SHALL represent the current set of resources owned by a instance. It SHALL contain the current `entries` list and MAY include ownership summary fields such as `revision`, `digest`, and `count`.

The recorded inventory MUST NOT require or embed:

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

### Requirement: Deterministic inventory digest

The CLI SHALL compute the inventory digest with the library's `opm/k8s/inventory.Digest`, which hashes a canonical field-by-field encoding of the entries. The digest SHALL depend only on the entries' field values, never on their order or on how the CLI serialises an entry, and two inventories that differ in their set of entries or in any field of an entry SHALL produce different digests. The CLI SHALL NOT compute an inventory digest of its own. Source: 0012:D7:R2/R3. The release that first records this digest SHALL carry a migration note naming the one-time change of the stored value. Source: 0012:D7:R4.

#### Scenario: Same entries in different order produce same digest

- **WHEN** computing the digest for the same ownership set in two different slice orders
- **THEN** the digest SHALL be identical

#### Scenario: Added or removed resource changes digest

- **WHEN** computing the digest of an ownership set with 3 resources
- **AND** computing the digest with one resource removed
- **THEN** the digests SHALL differ

#### Scenario: Component rename changes digest

- **WHEN** two ownership sets differ only by the `component` field of an entry
- **THEN** the digests SHALL differ, because every field of an entry is part of the encoding the digest hashes

#### Scenario: The stored digest is the library's

- **WHEN** an apply writes `status.inventory.digest`
- **THEN** the value SHALL equal `inventory.Digest` of the written entries

## REMOVED Requirements

### Requirement: Inventory entry identity
**Reason**: The component-aware identity (group, kind, namespace, name and component) is retired. The stale set is component-blind (0012:D7), so no CLI comparison counts the component.
**Migration**: Entry identity is the ADDED requirement "Inventory entries are the library's component-blind entries", which keeps the API version out of identity as this requirement did.

### Requirement: Kubernetes identity equality
**Reason**: It existed for the component-rename safety check, which is removed (0012:D7). The comparison it described is now the only one, the library's `inventory.SameObject`.
**Migration**: See the ADDED requirement "Inventory entries are the library's component-blind entries".
