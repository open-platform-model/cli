# Delta: instance-inventory (cli-kernel-adoption)

The `lastApplied*` digests become kernel-derived; `lastAppliedSourceDigest` upgrades from C1's module-reference identity stopgap to the kernel content digest. Everything else in this capability is unchanged.

## MODIFIED Requirements

### Requirement: CLI writes a strict status subset via the status subresource

After resources are applied and pruned, the CLI SHALL write, via the status subresource with field manager `opm-cli`: `status.inventory` (revision, digest, count, entries), `status.instanceUUID`, `status.lastAppliedRenderDigest`, `status.lastAppliedSourceDigest`, `status.lastAppliedConfigDigest`, and `status.lastAppliedAt`. The CLI MUST NOT write `status.conditions`, `status.observedGeneration`, `status.lastAttempted*`, `status.failureCounters`, `status.history`, or `status.nextRetryAt`.

The digest fields SHALL be kernel-derived: `lastAppliedRenderDigest` computed over the kernel-finalized manifests with the operator's canonical serialization (see `kernel-render`), and `lastAppliedSourceDigest` computed as the module content digest matching the operator's field semantics — not a module-reference identity digest.

#### Scenario: Status subset after successful apply

- **WHEN** an apply deploys 3 resources successfully
- **THEN** `status.inventory.count` SHALL be 3 and `status.lastAppliedAt` SHALL be set
- **AND** `status.conditions` SHALL NOT be present in the CLI's applied status document

#### Scenario: Revision increments across applies

- **WHEN** a second apply succeeds for an instance whose `status.inventory.revision` was 1
- **THEN** the written `status.inventory.revision` SHALL be 2

#### Scenario: Source digest is content-based

- **WHEN** the same module content is applied from two different reference spellings resolving to identical content
- **THEN** `status.lastAppliedSourceDigest` SHALL be identical
- **AND** a content change in the module SHALL change the digest even when the reference is unchanged
