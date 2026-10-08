## Purpose

What the CLI does with the inventory Secret that opm releases up to v1.0.0-alpha.1 wrote: nothing. The migration onto the `ModuleInstance` CR (0006:D8/D14) was removed before v1.0.0; v1.0.0-beta.10 is the last release that performs it. The capability keeps its name for history.

## Requirements

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
