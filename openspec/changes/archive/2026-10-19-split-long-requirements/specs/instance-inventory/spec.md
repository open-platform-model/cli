## REMOVED Requirements

### Requirement: CLI writes a strict status subset via the status subresource
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI writes only its own fields through the status subresource", "The last applied render digest is the library shared render digest", "The inventory digest in the status is the library inventory digest", "The last applied source digest is the operator module source digest", "The last applied config digest follows the operator canonical-JSON semantics".

### Requirement: Spec write contents
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI sets the owner of the ModuleInstance spec to cli", "The CLI writes the module identity verbatim from core-v2 metadata", "The spec values are the single unified values blob", "The module identity pair applies to local resolution and the CR holds no filesystem path", "The declared module identity pair is verified where it meets a registry".

### Requirement: Render provenance annotation
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution", "The source local annotation is a fail-closed provenance signal that no gate reads as an authority".

### Requirement: Deterministic inventory digest
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI computes the inventory digest with the library and no digest of its own", "The inventory digest depends only on the field values of the entries", "The release that first records the library digest carries a migration note".

### Requirement: Skipped contracts annotation
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A skipping apply records the skipped demands in the skipped-contracts annotation", "A later apply that skips nothing omits the skipped-contracts annotation", "The skipped-contracts annotation is information and no gate reads it as an authority".

### Requirement: Inventory entries are the library's component-blind entries
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI represents owned resources as the library's inventory entries", "Two entries are the same object when group, kind, namespace and name agree".

## ADDED Requirements

### Requirement: The CLI writes only its own fields through the status subresource

After resources are applied and pruned, the CLI SHALL write, via the status subresource with field manager `opm-cli`: `status.inventory` (revision, digest, count, entries), `status.instanceUUID`, `status.lastAppliedRenderDigest`, `status.lastAppliedSourceDigest`, `status.lastAppliedConfigDigest`, and `status.lastAppliedAt`. The CLI MUST NOT write `status.conditions`, `status.observedGeneration`, `status.lastAttempted*`, `status.failureCounters`, `status.history`, or `status.nextRetryAt`.

#### Scenario: Status subset after successful apply

- **WHEN** an apply deploys 3 resources successfully
- **THEN** `status.inventory.count` SHALL be 3 and `status.lastAppliedAt` SHALL be set
- **AND** `status.conditions` SHALL NOT be present in the CLI's applied status document

#### Scenario: Revision increments across applies

- **WHEN** a second apply succeeds for an instance whose `status.inventory.revision` was 1
- **THEN** the written `status.inventory.revision` SHALL be 2

### Requirement: The last applied render digest is the library shared render digest

The digest fields SHALL be operator-parity: `lastAppliedRenderDigest` SHALL be the library's shared render digest, `opm/k8s/inventory.RenderDigest`, over the render's single export (see `kernel-render`), which leaves the managed-by label value out so the CLI and the operator digest one render to the same value (enhancement 0012, "Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value").

#### Scenario: Render digest matches the operator's algorithm

- **WHEN** the CLI and the operator compile the same instance against the same Platform spec
- **THEN** the two render digests SHALL be byte-identical
- **AND** the CLI's value SHALL equal `inventory.RenderDigest` of the render's export, which the operator also computes

### Requirement: The inventory digest in the status is the library inventory digest

The digest fields SHALL be operator-parity: `status.inventory.digest` SHALL be the library's `inventory.Digest` of the written entries (see "The CLI computes the inventory digest with the library and no digest of its own").

#### Scenario: Stored digests change once on upgrade

- **WHEN** an instance last applied by a CLI release that predates the library digests is applied again with no change to its objects
- **THEN** `status.inventory.digest` and `status.lastAppliedRenderDigest` SHALL change to the library values
- **AND** the entries, revision increment and count SHALL be written as for any other apply

### Requirement: The last applied source digest is the operator module source digest

The digest fields SHALL be operator-parity: `lastAppliedSourceDigest` SHALL be computed as the operator's `ModuleSourceDigest` (SHA-256 of the canonical `path@version` reference: identical on both actors' CUE-native paths).

#### Scenario: Source digest matches the operator's computation

- **WHEN** the CLI applies a module with canonical reference `opmodel.dev/modules/test/podinfo@v0` at version `0.1.2`
- **THEN** `status.lastAppliedSourceDigest` SHALL equal the operator's `ModuleSourceDigest` for the same path and version

### Requirement: The last applied config digest follows the operator canonical-JSON semantics

The digest fields SHALL be operator-parity: `lastAppliedConfigDigest` SHALL match the operator's `ConfigDigest` canonical-JSON semantics including the empty case (SHA-256 of no bytes).

#### Scenario: Config digest of an empty config
- **WHEN** the CLI computes `lastAppliedConfigDigest` for the empty case
- **THEN** it SHALL match the operator's `ConfigDigest` canonical-JSON semantics (SHA-256 of no bytes)

### Requirement: The CLI sets the owner of the ModuleInstance spec to cli

On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field manager `opm-cli`, containing: `spec.owner: cli`.

#### Scenario: CLI-executor apply sets the owner
- **WHEN** the CLI applies the CR spec on apply in CLI-executor mode
- **THEN** the spec SHALL contain `spec.owner: cli`

### Requirement: The CLI writes the module identity verbatim from core-v2 metadata

On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field manager `opm-cli`, containing: `spec.module.path` and `spec.module.version` set to the module's declared identity **read verbatim from core-v2 metadata**: `spec.module.path` is `metadata.modulePath` as-is (the complete major-suffixed registry address; no composition from a parent prefix and a name), and `spec.module.version` is `metadata.version` normalized to the `v`-prefixed registry-tag form.

#### Scenario: No address arithmetic

- **WHEN** the written path is compared to the module's declared `metadata.modulePath`
- **THEN** they SHALL be byte-identical — no major tag, leaf, or prefix is computed by the CLI

### Requirement: The spec values are the single unified values blob

On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field manager `opm-cli`, containing: `spec.values` set to the single unified values blob that the render consumed.

#### Scenario: Values are the unified blob

- **WHEN** applying with multiple `--values` files
- **THEN** `spec.values` SHALL contain the single unified result the render consumed, not the individual layers

### Requirement: The module identity pair applies to local resolution and the CR holds no filesystem path

The pair (`spec.module.path` and `spec.module.version`) applies for local-directory and locally-replaced module resolution as well; the CR MUST NOT contain a filesystem path.

#### Scenario: Local-path apply writes the declared reference

- **WHEN** applying from a local module directory whose `module.cue` declares `modulePath: "opmodel.dev/modules/podinfo@v0"` and `version: "0.1.4"`
- **THEN** `spec.module.path` SHALL be `opmodel.dev/modules/podinfo@v0` and `spec.module.version` SHALL be `v0.1.4`

### Requirement: The declared module identity pair is verified where it meets a registry

The declared pair (`spec.module.path` and `spec.module.version`) SHALL be verified against fetched coordinates wherever it later meets a registry (operator acquire; any future ownership transfer), not at write time.

#### Scenario: Declared pair verified at the registry
- **WHEN** the declared pair later meets a registry (operator acquire; any future ownership transfer)
- **THEN** it SHALL be verified against fetched coordinates, not at write time

### Requirement: The CLI stamps the source local annotation when render bytes did not come from pure registry resolution

When the applied render's module bytes did not come from pure registry resolution: the main module is a local directory, or the main module's `cue.mod/local-module.cue` contains any local-path `replaceWith`, the CLI SHALL include the annotation `module-instance.opmodel.dev/source: local` in its spec apply. When a later apply resolves fully from registries, the CLI SHALL omit the annotation so server-side apply removes it.

#### Scenario: Local render stamps the annotation

- **WHEN** an apply renders from a local module directory
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Replacement in effect stamps the annotation

- **WHEN** an apply's main module has a `cue.mod/local-module.cue` with a local-path `replaceWith`
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Registry apply clears the annotation

- **WHEN** an instance carrying the annotation is re-applied with fully registry-resolved modules
- **THEN** the annotation SHALL no longer be present on the CR

### Requirement: The source local annotation is a fail-closed provenance signal that no gate reads as an authority

The annotation `module-instance.opmodel.dev/source: local` is a fail-closed provenance signal: the thin-editor apply path refuses a module that resolves from local bytes, and any future transfer of an instance to operator ownership reads it as a refusal; no CLI-executor-mode gate SHALL read it as an authority.

#### Scenario: No gate reads the annotation as an authority
- **WHEN** a CLI-executor-mode gate runs on an instance that carries the annotation
- **THEN** it SHALL NOT read the annotation as an authority

### Requirement: The CLI computes the inventory digest with the library and no digest of its own

The CLI SHALL compute the inventory digest with the library's `opm/k8s/inventory.Digest`, which hashes a canonical field-by-field encoding of the entries. The CLI SHALL NOT compute an inventory digest of its own. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: The stored digest is the library's

- **WHEN** an apply writes `status.inventory.digest`
- **THEN** the value SHALL equal `inventory.Digest` of the written entries

### Requirement: The inventory digest depends only on the field values of the entries

The digest SHALL depend only on the entries' field values, never on their order or on how the CLI serialises an entry, and two inventories that differ in their set of entries or in any field of an entry SHALL produce different digests. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

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

### Requirement: The release that first records the library digest carries a migration note

The release that first records this inventory digest computed with the library's `opm/k8s/inventory.Digest` SHALL carry a migration note naming the one-time change of the stored value. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: Migration note for the first release
- **WHEN** a release first records the inventory digest computed with the library's `opm/k8s/inventory.Digest`
- **THEN** it SHALL carry a migration note naming the one-time change of the stored value

### Requirement: A skipping apply records the skipped demands in the skipped-contracts annotation

When an apply rendered with `--skip-unprovided` and the kernel skipped at least one demand, the CLI SHALL include the annotation `module-instance.opmodel.dev/skipped-contracts` in its spec apply. Its value SHALL be the skipped demands as `<component>=<contract fqn>` pairs, sorted, deduplicated and joined with commas.

#### Scenario: A skipping apply records its skips

- **WHEN** `opm instance apply --skip-unprovided` renders an instance whose component `db` skips `opmodel.dev/catalogs/opm/traits/backup@v1alpha1`
- **THEN** the ModuleInstance SHALL carry `module-instance.opmodel.dev/skipped-contracts: db=opmodel.dev/catalogs/opm/traits/backup@v1alpha1`

#### Scenario: A flag that skipped nothing writes nothing

- **WHEN** `opm instance apply --skip-unprovided` renders an instance with no unprovided demand
- **THEN** the ModuleInstance SHALL NOT carry the annotation

### Requirement: A later apply that skips nothing omits the skipped-contracts annotation

When a later apply skips nothing, with or without the flag `--skip-unprovided`, the CLI SHALL omit the annotation `module-instance.opmodel.dev/skipped-contracts` so server-side apply removes it.

#### Scenario: A complete apply clears the record

- **WHEN** the same instance is applied again and its render skips nothing, for example because the component `db` no longer attaches the backup trait
- **THEN** the annotation SHALL no longer be present on the ModuleInstance

### Requirement: The skipped-contracts annotation is information and no gate reads it as an authority

The annotation `module-instance.opmodel.dev/skipped-contracts` is information for whoever inspects the instance: no CLI gate SHALL read it as an authority.

#### Scenario: No gate reads the skipped-contracts annotation as an authority
- **WHEN** a CLI gate runs on an instance that carries the annotation
- **THEN** no CLI gate SHALL read it as an authority

### Requirement: The CLI represents owned resources as the library's inventory entries

The CLI SHALL represent each currently owned Kubernetes resource as the library's `opm/k8s/inventory.Entry` (group, kind, namespace, name, API version, component), and SHALL NOT declare an entry type of its own. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: The CLI declares no entry type of its own
- **WHEN** the CLI represents a currently owned Kubernetes resource
- **THEN** it SHALL use the library's `opm/k8s/inventory.Entry` and SHALL NOT declare an entry type of its own

### Requirement: Two entries are the same object when group, kind, namespace and name agree

Two entries SHALL be the same object when their group, kind, namespace and name agree, as `inventory.SameObject` compares them: neither the API version nor the component SHALL count, so an API version migration or a component rename never makes an object look like a different one. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: Same object at a different API version

- **WHEN** comparing two entries with identical group, kind, namespace, name and component but a different API version
- **THEN** the entries SHALL be the same object

#### Scenario: Same object under a different component

- **WHEN** comparing two entries with identical group, kind, namespace, name and API version but a different component
- **THEN** the entries SHALL be the same object

#### Scenario: Different objects

- **WHEN** comparing two entries that differ in name
- **THEN** the entries SHALL NOT be the same object
