## REMOVED Requirements

### Requirement: CLI writes a strict status subset via the status subresource
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-controller-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## MODIFIED Requirements

### Requirement: ModuleInstance CR is the inventory store

The CLI SHALL persist instance inventory in a `ModuleInstance` custom resource (`opmodel.dev/v1alpha1`, resource `moduleinstances`) named after the instance, in the instance's namespace, handled as `unstructured` via the dynamic client. The CLI MUST NOT import `opm-controller` Go packages. The `ModuleInstance` GVR and related constants SHALL be defined once in `internal/inventory` and consumed by all CLI packages that reference the CR (including `internal/controller`).

#### Scenario: Apply creates the CR

- **WHEN** `opm instance apply` succeeds for instance `podinfo` in namespace `demo` and no `ModuleInstance` exists
- **THEN** a `ModuleInstance` named `podinfo` SHALL exist in `demo` with the CLI's spec and status subset

#### Scenario: No inventory Secret is written

- **WHEN** any apply completes
- **THEN** no `opm.<name>.<id>` inventory Secret SHALL be created or updated

### Requirement: Spec write contents

On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field manager `opm-cli`, containing: `spec.owner: cli`, `spec.module.path` and `spec.module.version` set to the module's declared identity **read verbatim from core-v2 metadata** — `spec.module.path` is `metadata.modulePath` as-is (the complete major-suffixed registry address; no composition from a parent prefix and a name), and `spec.module.version` is `metadata.version` normalized to the `v`-prefixed registry-tag form — and `spec.values` set to the single unified values blob that the render consumed. The pair applies for local-directory and locally-replaced module resolution as well; the CR MUST NOT contain a filesystem path. The declared pair is verified against fetched coordinates wherever it later meets a registry (controller acquire; any future ownership transfer), not at write time.

#### Scenario: Local-path apply writes the declared reference

- **WHEN** applying from a local module directory whose `module.cue` declares `modulePath: "opmodel.dev/modules/podinfo@v0"` and `version: "0.1.4"`
- **THEN** `spec.module.path` SHALL be `opmodel.dev/modules/podinfo@v0` and `spec.module.version` SHALL be `v0.1.4`

#### Scenario: No address arithmetic

- **WHEN** the written path is compared to the module's declared `metadata.modulePath`
- **THEN** they SHALL be byte-identical — no major tag, leaf, or prefix is computed by the CLI

#### Scenario: Values are the unified blob

- **WHEN** applying with multiple `--values` files
- **THEN** `spec.values` SHALL contain the single unified result the render consumed, not the individual layers

### Requirement: Render provenance annotation

When the applied render's module bytes did not come from pure registry resolution — the main module is a local directory, or the main module's `cue.mod/local-module.cue` contains any local-path `replaceWith` — the CLI SHALL include the annotation `module-instance.opmodel.dev/source: local` in its spec apply. When a later apply resolves fully from registries, the CLI SHALL omit the annotation so server-side apply removes it. The annotation is a fail-closed provenance signal: the thin-editor apply path refuses a module that resolves from local bytes, and any future transfer of an instance to controller ownership reads it as a refusal; no CLI-executor-mode gate SHALL read it as an authority.

#### Scenario: Local render stamps the annotation

- **WHEN** an apply renders from a local module directory
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Replacement in effect stamps the annotation

- **WHEN** an apply's main module has a `cue.mod/local-module.cue` with a local-path `replaceWith`
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Registry apply clears the annotation

- **WHEN** an instance carrying the annotation is re-applied with fully registry-resolved modules
- **THEN** the annotation SHALL no longer be present on the CR

## ADDED Requirements

### Requirement: The CLI writes only its status subset through the status subresource

After resources are applied and pruned, the CLI SHALL write, via the status subresource with field manager `opm-cli`: `status.inventory` (revision, digest, count, entries), `status.instanceUUID`, `status.lastAppliedRenderDigest`, `status.lastAppliedSourceDigest`, `status.lastAppliedConfigDigest`, and `status.lastAppliedAt`. The CLI MUST NOT write `status.conditions`, `status.observedGeneration`, `status.lastAttempted*`, `status.failureCounters`, `status.history`, or `status.nextRetryAt`.

The digest fields SHALL be controller-parity: `lastAppliedRenderDigest` SHALL be the library's shared render digest, `opm/k8s/inventory.RenderDigest`, over the render's single export (see `kernel-render`), which leaves the managed-by label value out so the CLI and the controller digest one render to the same value (0012:D6:R2); `status.inventory.digest` SHALL be the library's `inventory.Digest` of the written entries (see "Deterministic inventory digest"); `lastAppliedSourceDigest` SHALL be computed as the controller's `ModuleSourceDigest` (SHA-256 of the canonical `path@version` reference — identical on both actors' CUE-native paths), and `lastAppliedConfigDigest` SHALL match the controller's `ConfigDigest` canonical-JSON semantics including the empty case (SHA-256 of no bytes).

#### Scenario: Status subset after successful apply

- **WHEN** an apply deploys 3 resources successfully
- **THEN** `status.inventory.count` SHALL be 3 and `status.lastAppliedAt` SHALL be set
- **AND** `status.conditions` SHALL NOT be present in the CLI's applied status document

#### Scenario: Revision increments across applies

- **WHEN** a second apply succeeds for an instance whose `status.inventory.revision` was 1
- **THEN** the written `status.inventory.revision` SHALL be 2

#### Scenario: Source digest matches the controller's computation

- **WHEN** the CLI applies a module with canonical reference `opmodel.dev/modules/test/podinfo@v0` at version `0.1.2`
- **THEN** `status.lastAppliedSourceDigest` SHALL equal the controller's `ModuleSourceDigest` for the same path and version

#### Scenario: Render digest matches the controller's algorithm

- **WHEN** the CLI and the controller compile the same instance against the same Platform spec
- **THEN** the two render digests SHALL be byte-identical
- **AND** the CLI's value SHALL equal `inventory.RenderDigest` of the render's export, which the controller also computes

#### Scenario: Stored digests change once on upgrade

- **WHEN** an instance last applied by a CLI release that predates the library digests is applied again with no change to its objects
- **THEN** `status.inventory.digest` and `status.lastAppliedRenderDigest` SHALL change to the library values
- **AND** the entries, revision increment and count SHALL be written as for any other apply
