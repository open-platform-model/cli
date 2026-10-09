## REMOVED Requirements

### Requirement: Migration triggers on apply when a Secret exists and no CR does

**Reason**: The legacy inventory Secret path is removed before v1.0.0. Apply no longer reads a Secret.
**Migration**: Apply the instance once with opm v1.0.0-beta.10, the last release that migrates, then upgrade.

### Requirement: Record port mapping

**Reason**: Nothing is ported any more; the path is removed.
**Migration**: Apply the instance once with opm v1.0.0-beta.10, which ports the record, then upgrade.

### Requirement: Secret is deleted only after the CR status write succeeds

**Reason**: Apply no longer deletes a legacy inventory Secret.
**Migration**: A Secret that is still in the cluster after the upgrade is deleted by hand with `kubectl delete secret`.

### Requirement: No Secret reads outside migration

**Reason**: Replaced by "No command reads a legacy inventory Secret", which covers apply too.
**Migration**: None.

## ADDED Requirements

### Requirement: No command reads a legacy inventory Secret

The CLI MUST NOT get, list, decode, migrate or delete a legacy inventory Secret (`opm.<name>.<id>`) in any command, and an apply SHALL NOT need any permission on Secrets for inventory purposes. An instance whose only inventory is a legacy Secret has no `ModuleInstance` record: `opm instance status`, `list`, `diff` and `delete` SHALL NOT see it, and `opm instance apply` and `opm module apply` SHALL treat it as a first install. That apply SHALL NOT delete or prune any resource, SHALL leave the Secret in place, and SHALL record the resources it rendered at revision 1.

#### Scenario: Apply makes no request on Secrets

- **WHEN** `opm instance apply` succeeds for an instance, with or without a `ModuleInstance` record
- **THEN** the CLI SHALL have made no get, list or delete request on Secrets

#### Scenario: Instance recorded only in a legacy Secret is a first install

- **WHEN** an instance has a legacy inventory Secret that records resources A, B and C, no `ModuleInstance` record, and A, B and C exist with the OPM managed-by label
- **AND** `opm instance apply` renders A and B
- **THEN** A and B SHALL be applied and recorded in a new `ModuleInstance` at revision 1
- **AND** C SHALL NOT be deleted
- **AND** the legacy Secret SHALL still exist

#### Scenario: Status does not fall back to Secrets

- **WHEN** an instance has only a legacy Secret inventory and no `ModuleInstance` record
- **AND** `opm instance status --name <name>` runs
- **THEN** the command SHALL report the instance as not found
