# Delta: instance-inventory (cli-cr-inventory-backend)

The persisted envelope and CRUD move from an inventory Secret to the `ModuleInstance` CR. Entry identity, K8s identity, entry construction, ownership-only inventory semantics, and deterministic digest requirements are unchanged.

## ADDED Requirements

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

## REMOVED Requirements

### Requirement: Inventory Secret name convention

**Reason**: The inventory Secret is replaced by the `ModuleInstance` CR (enhancement 0006 D1); there is no Secret to name.
**Migration**: The CR is named after the instance in the instance namespace; the one-time migration (see `secret-inventory-migration`) ports existing Secrets.

### Requirement: Inventory Secret labels

**Reason**: No inventory Secret exists; instance identity labels live on the CR's rendered resources and metadata.
**Migration**: None required — CR lookup is by name/namespace, not label selector.

### Requirement: Inventory Secret serialization roundtrip

**Reason**: The JSON-in-Secret envelope (`MarshalToSecret`/`UnmarshalFromSecret`) is deleted with the Secret backend.
**Migration**: The CRD's OpenAPI schema anchors the persisted shape; explicit mapping functions replace envelope marshaling. Unmarshaling survives only inside the one-time migration path.

### Requirement: Inventory Secret CRUD operations

**Reason**: `GetInventory`/`WriteInventory`/`DeleteInventory` over Secrets are replaced by CR CRUD (see ADDED "CR CRUD semantics").
**Migration**: One-time Secret→CR migration on apply; no Secret reads elsewhere.

### Requirement: Inventory labels remain unchanged when provenance is added

**Reason**: Obsolete with the Secret label contract removed.
**Migration**: None.

### Requirement: Inventory metadata enables future CRD migration

**Reason**: Fulfilled — this change *is* the CRD migration the `kind`/`apiVersion` fields anticipated.
**Migration**: The migration path consumes those fields one final time when porting Secrets.

### Requirement: Persisted instance inventory record preserves instance and module metadata

**Reason**: The record envelope is gone; the CR carries instance identity in `metadata`, module identity in `spec.module`, and the UUID in `status.instanceUUID`.
**Migration**: The one-time migration maps `instanceMetadata.uuid` → `status.instanceUUID`; module identity comes from the current apply's resolved module.

### Requirement: Persisted instance inventory record stores creator provenance at the top level

**Reason**: Creator provenance (`createdBy`) is replaced by the CR's `spec.owner` marker (see `inventory-ownership` delta).
**Migration**: Migrated CRs are written with `spec.owner: cli`.

### Requirement: Instance metadata data key structure

**Reason**: The `instanceMetadata` envelope object is deleted with the record shape.
**Migration**: Instance name/namespace live on the CR's `metadata`; the UUID in `status.instanceUUID`.

### Requirement: Module metadata data key structure

**Reason**: The `moduleMetadata` envelope object is deleted with the record shape.
**Migration**: Module path/version live in `spec.module`.
