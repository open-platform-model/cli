# secret-inventory-migration

## Purpose

The one-time, delete-after-success migration of legacy Secret inventories onto the `ModuleInstance` CR, performed on apply. Enhancement 0006 D8 as amended by D14 — no deprecation window; apply is the only command that reads Secrets, and only to migrate them.

## ADDED Requirements

### Requirement: Migration triggers on apply when a Secret exists and no CR does

On `opm instance apply` (and `opm module apply`), when no `ModuleInstance` CR exists for the instance but a legacy inventory Secret does (found by the legacy direct-name and UUID-label lookup), the CLI SHALL migrate: treat the Secret's record as the previous inventory for stale-set computation, proceed with the normal apply, write the CR (spec + status subset, `spec.owner: cli`), and then delete the Secret.

#### Scenario: Secret-tracked instance is migrated on apply

- **WHEN** an instance has a legacy inventory Secret and no `ModuleInstance` CR
- **AND** `opm instance apply` succeeds
- **THEN** a `ModuleInstance` CR SHALL exist with the ported inventory
- **AND** the legacy Secret SHALL be deleted

#### Scenario: Stale-set continuity across migration

- **WHEN** the Secret's record tracks entries [A, B, C] and the current render produces [A, B]
- **THEN** entry C SHALL be pruned during the migrating apply

### Requirement: Record port mapping

The migration SHALL map the Secret record onto the CR as follows: `inventory` (entries, revision, digest, count) into `status.inventory` with the revision continued (next write increments it, not resets it); the record's instance UUID into `status.instanceUUID`; `spec.owner: cli`; `spec.module` and `spec.values` from the current apply's resolved module and unified values.

#### Scenario: Revision continues

- **WHEN** the Secret record's inventory revision is 4 and the migrating apply succeeds
- **THEN** the CR's `status.inventory.revision` SHALL be 5

### Requirement: Secret is deleted only after the CR status write succeeds

The migration SHALL delete the legacy Secret only after the CR's status subresource write has succeeded. Any failure before that point SHALL leave the Secret intact. A re-run of apply after a partial migration SHALL be idempotent (CR exists ⇒ normal apply; leftover Secret with an existing CR ⇒ Secret is deleted without being read).

#### Scenario: Failure preserves the Secret

- **WHEN** the migrating apply fails before the CR status write succeeds
- **THEN** the legacy Secret SHALL still exist
- **AND** a subsequent apply SHALL retry the migration

#### Scenario: Idempotent re-run after partial migration

- **WHEN** a CR exists and a leftover legacy Secret also exists
- **AND** apply runs
- **THEN** the apply SHALL use the CR as the previous inventory and delete the leftover Secret

### Requirement: No Secret reads outside migration

`opm instance status`, `delete`, `list`, and `diff` MUST NOT read legacy inventory Secrets. An instance tracked only by a Secret is invisible to these commands until its next apply migrates it.

#### Scenario: Status does not fall back to Secrets

- **WHEN** an instance has only a legacy Secret inventory (no CR)
- **AND** `opm instance status --name <name>` runs
- **THEN** the command SHALL report the instance as not found
