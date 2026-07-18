## Purpose

Defines the ownership-focused instance inventory data model and the `ModuleInstance` custom resource that persists it, including entry identity, the CR spec/status write contract, entry wire mapping, CR CRUD semantics, and deterministic digest requirements used by the CLI for pruning, discovery, and instance metadata.

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

### Requirement: ModuleInstance CR is the inventory store

The CLI SHALL persist instance inventory in a `ModuleInstance` custom resource (`opmodel.dev/v1alpha1`, resource `moduleinstances`) named after the instance, in the instance's namespace, handled as `unstructured` via the dynamic client. The CLI MUST NOT import `opm-operator` Go packages. The `ModuleInstance` GVR and related constants SHALL be defined once in `internal/inventory` and consumed by all CLI packages that reference the CR (including `internal/operator`).

#### Scenario: Apply creates the CR

- **WHEN** `opm instance apply` succeeds for instance `podinfo` in namespace `demo` and no `ModuleInstance` exists
- **THEN** a `ModuleInstance` named `podinfo` SHALL exist in `demo` with the CLI's spec and status subset

#### Scenario: No inventory Secret is written

- **WHEN** any apply completes
- **THEN** no `opm.<name>.<id>` inventory Secret SHALL be created or updated

### Requirement: CLI writes a strict status subset via the status subresource

After resources are applied and pruned, the CLI SHALL write, via the status subresource with field manager `opm-cli`: `status.inventory` (revision, digest, count, entries), `status.instanceUUID`, `status.lastAppliedRenderDigest`, `status.lastAppliedSourceDigest`, `status.lastAppliedConfigDigest`, and `status.lastAppliedAt`. The CLI MUST NOT write `status.conditions`, `status.observedGeneration`, `status.lastAttempted*`, `status.failureCounters`, `status.history`, or `status.nextRetryAt`.

#### Scenario: Status subset after successful apply

- **WHEN** an apply deploys 3 resources successfully
- **THEN** `status.inventory.count` SHALL be 3 and `status.lastAppliedAt` SHALL be set
- **AND** `status.conditions` SHALL NOT be present in the CLI's applied status document

#### Scenario: Revision increments across applies

- **WHEN** a second apply succeeds for an instance whose `status.inventory.revision` was 1
- **THEN** the written `status.inventory.revision` SHALL be 2

### Requirement: Spec write contents

On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field manager `opm-cli`, containing: `spec.owner: cli`, `spec.module.path` and `spec.module.version` set to the module's canonical declared path and version (for local-directory and locally-replaced module resolution as well — the CR MUST NOT contain a filesystem path), and `spec.values` set to the single unified values blob that the render consumed.

#### Scenario: Local-path apply writes the declared reference

- **WHEN** applying from a local module directory whose `module.cue` declares path `opmodel.dev/modules/podinfo@v0` and version `v0.1.0`
- **THEN** `spec.module.path` SHALL be `opmodel.dev/modules/podinfo@v0` and `spec.module.version` SHALL be `v0.1.0`

#### Scenario: Values are the unified blob

- **WHEN** applying with multiple `--values` files
- **THEN** `spec.values` SHALL contain the single unified result the render consumed, not the individual layers

### Requirement: Entry wire shape targets the CRD schema

Conversion between the CLI's `InventoryEntry` type and the CR's `status.inventory.entries[]` SHALL be performed by explicit mapping functions that produce/consume the CRD's field names (`group`, `kind`, `namespace`, `name`, `v`, `component`), independent of the Go struct's own JSON tags. The mapping SHALL round-trip losslessly.

#### Scenario: Version serializes as `v`

- **WHEN** an entry with Version `v1` is written to the CR
- **THEN** the entry object in `status.inventory.entries[]` SHALL carry the key `v` with value `v1`

#### Scenario: Round-trip preserves the entry set

- **WHEN** an entry list is written to a CR and read back
- **THEN** the resulting entries SHALL equal the originals

### Requirement: instanceUUID is extracted from the render

The CLI SHALL populate `status.instanceUUID` from the rendered resources' `module-instance.opmodel.dev/uuid` label (first non-empty value). If no rendered resource carries the label, the field SHALL be omitted. The CLI MUST NOT generate the UUID itself.

#### Scenario: UUID extracted from rendered labels

- **WHEN** rendered resources carry `module-instance.opmodel.dev/uuid: 7c9e6679-7425-40de-944b-e07fc1f90ae7`
- **THEN** `status.instanceUUID` SHALL be `7c9e6679-7425-40de-944b-e07fc1f90ae7`

### Requirement: Render provenance annotation

When the applied render's module bytes did not come from pure registry resolution — the main module is a local directory, or the main module's `cue.mod/local-module.cue` contains any local-path `replaceWith` — the CLI SHALL include the annotation `module-instance.opmodel.dev/source: local` in its spec apply. When a later apply resolves fully from registries, the CLI SHALL omit the annotation so server-side apply removes it. The annotation is a fail-closed signal for the handoff pre-gate (slice C3); no CLI gate in this slice SHALL read it as an authority.

#### Scenario: Local render stamps the annotation

- **WHEN** an apply renders from a local module directory
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Replacement in effect stamps the annotation

- **WHEN** an apply's main module has a `cue.mod/local-module.cue` with a local-path `replaceWith`
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Registry apply clears the annotation

- **WHEN** an instance carrying the annotation is re-applied with fully registry-resolved modules
- **THEN** the annotation SHALL no longer be present on the CR

### Requirement: CR CRUD semantics

Reading inventory SHALL be a direct GET of the `ModuleInstance` by name and namespace, with NotFound returned as "no inventory" (first-apply). `--instance-id` selectors SHALL resolve by listing `ModuleInstance` CRs and matching `status.instanceUUID`. On `instance delete`, the CLI SHALL delete owned resources first (existing reverse-weight prune semantics) and delete the CR last; CR deletion SHALL treat NotFound as success.

#### Scenario: First apply finds no inventory

- **WHEN** `opm instance apply` runs and no `ModuleInstance` exists for the name
- **THEN** the apply SHALL proceed as a first-time apply with an empty previous inventory

#### Scenario: Delete removes the CR last

- **WHEN** `opm instance delete` succeeds
- **THEN** every tracked resource SHALL be deleted before the `ModuleInstance` CR itself

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
