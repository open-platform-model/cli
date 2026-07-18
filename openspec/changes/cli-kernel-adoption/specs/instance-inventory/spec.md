# Delta: instance-inventory (cli-kernel-adoption)

The `lastApplied*` digests become operator-parity. Investigation during implementation corrected this delta's original premise: on the CUE-native resolution path the operator's own `lastAppliedSourceDigest` IS the module-reference identity digest (`ModuleSourceDigest(path@version)`) — there is no Flux artifact content digest to mirror — so the CLI keeps the identical computation. The render digest adopts the operator's exact algorithm and serialization. Everything else in this capability is unchanged.

## MODIFIED Requirements

### Requirement: CLI writes a strict status subset via the status subresource

After resources are applied and pruned, the CLI SHALL write, via the status subresource with field manager `opm-cli`: `status.inventory` (revision, digest, count, entries), `status.instanceUUID`, `status.lastAppliedRenderDigest`, `status.lastAppliedSourceDigest`, `status.lastAppliedConfigDigest`, and `status.lastAppliedAt`. The CLI MUST NOT write `status.conditions`, `status.observedGeneration`, `status.lastAttempted*`, `status.failureCounters`, `status.history`, or `status.nextRetryAt`.

The digest fields SHALL be operator-parity: `lastAppliedRenderDigest` computed over the kernel-compiled resources with the operator's exact algorithm and serialization (sort by Group, Kind, Namespace, Name; concatenated CUE-value JSON; SHA-256 — see `kernel-render`), `lastAppliedSourceDigest` computed as the operator's `ModuleSourceDigest` (SHA-256 of the canonical `path@version` reference — identical on both actors' CUE-native paths), and `lastAppliedConfigDigest` matching the operator's `ConfigDigest` canonical-JSON semantics including the empty case (SHA-256 of no bytes).

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
