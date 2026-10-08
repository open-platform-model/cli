## ADDED Requirements

### Requirement: Unreadable instance record stops the apply

When the read of the instance's `ModuleInstance` record fails with any error other than NotFound, `opm instance apply` and `opm module apply` SHALL stop with an error. The system SHALL NOT treat the failed read as a first install: it SHALL NOT apply any resource, SHALL NOT prune, and SHALL NOT write the record. This SHALL hold for a dry run too, which SHALL NOT print a preview built without the record. The error SHALL name the instance and its namespace, carry the cause of the failed read, and tell the user to check access to the record and run the command again. The exit code SHALL be 4 when the read was denied (Forbidden or Unauthorized), 3 when the server timed out or was unavailable, and 1 otherwise. A NotFound answer SHALL still mean that no record exists.

#### Scenario: Read error on a real apply

- **WHEN** `opm instance apply` runs and the `ModuleInstance` read fails with an internal server error
- **THEN** the command SHALL exit 1 with an error naming the instance and the read error
- **AND** no rendered resource SHALL be applied
- **AND** no `ModuleInstance` spec or status SHALL be written

#### Scenario: Read denied

- **WHEN** `opm instance apply` runs and the `ModuleInstance` read fails with Forbidden
- **THEN** the command SHALL exit 4
- **AND** no rendered resource SHALL be applied

#### Scenario: Read error on a dry run

- **WHEN** `opm instance apply --dry-run` runs and the `ModuleInstance` read fails with an internal server error
- **THEN** the command SHALL exit with an error
- **AND** it SHALL NOT print a dry-run summary

#### Scenario: Missing record is still a first install

- **WHEN** `opm instance apply` runs and the `ModuleInstance` read answers NotFound
- **THEN** the apply SHALL proceed as a first-time apply

### Requirement: Failed prune keeps the entry and fails the command

When the delete of a stale resource fails with any error other than NotFound, the system SHALL go on to the remaining stale resources, and SHALL then write the inventory record with the current entries plus every stale entry it failed to delete, so the record still tracks each object that is still in the cluster. The output SHALL name each such resource by kind, namespace and name with its delete error, and SHALL say that the next apply retries the prune. The command SHALL exit 1 and SHALL NOT print the success line. A stale resource that was deleted, or that was already gone, SHALL NOT be kept in the record. On the next apply a kept entry SHALL be stale again and SHALL be pruned again.

#### Scenario: Delete denied for one stale resource

- **WHEN** the stale set holds ConfigMaps `a` and `b`
- **AND** the delete of `a` fails with Forbidden and the delete of `b` succeeds
- **THEN** the written inventory SHALL hold the current entries and the entry of `a`
- **AND** it SHALL NOT hold the entry of `b`
- **AND** the output SHALL name ConfigMap `a` with the delete error
- **AND** the command SHALL exit 1 without the success line

#### Scenario: Next apply retries the prune

- **WHEN** a previous apply kept the entry of a stale resource it failed to delete
- **AND** the next apply renders the same resources
- **THEN** that entry SHALL appear in the stale set again

## MODIFIED Requirements

### Requirement: Pre-apply existence check on first install

On first-time apply (no previous inventory), the system SHALL check each rendered resource against the cluster. If a resource exists with a `deletionTimestamp` (terminating), or exists without OPM labels (untracked), the apply SHALL fail with a clear error message. If a resource cannot be read (the read fails with any error other than NotFound), the apply SHALL fail too: the error SHALL name the resource and carry the read error, and the exit code SHALL be 4 when the read was denied (Forbidden or Unauthorized), 3 when the server timed out or was unavailable, and 1 otherwise. A NotFound answer SHALL mean the resource does not exist, and passes. This check SHALL be skipped entirely when a previous inventory exists. A caller MAY pass an explicit admission set of resources; a resource in that set SHALL pass the untracked-resource test, and SHALL still fail the terminating-resource test and the unreadable-resource test. Only `opm operator install` SHALL pass a non-empty set, holding exactly the existing resources it proved came from an earlier opm-operator release manifest, or that already carry the operator instance's identity; every other caller SHALL pass none. No flag SHALL fill the set. The untracked-resource error SHALL NOT name a flag that would bypass it, since no flag does; it SHALL tell the user to remove or rename the existing resource, or to change the module so it renders a different name. Source: 0012:D8:R6.

#### Scenario: Untracked resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked

#### Scenario: Terminating resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource exists on the cluster with a `deletionTimestamp`
- **THEN** the command SHALL fail with an error indicating the resource is terminating

#### Scenario: Unreadable resource refused on first install

- **WHEN** performing a first-time apply
- **AND** the read of a rendered resource fails with Forbidden
- **THEN** the command SHALL fail with an error naming the resource and the read error
- **AND** the command SHALL exit 4
- **AND** no rendered resource SHALL be applied

#### Scenario: Absent resource passes

- **WHEN** performing a first-time apply
- **AND** the read of a rendered resource answers NotFound
- **THEN** the check SHALL NOT fail for that resource

#### Scenario: Check skipped when inventory exists

- **WHEN** performing a subsequent apply (previous inventory exists)
- **THEN** the pre-apply existence check SHALL be skipped entirely

#### Scenario: Untracked-resource error names no bypass flag

- **WHEN** performing a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the error SHALL NOT mention `--force`
- **AND** the error SHALL tell the user to remove or rename the existing resource, or to change the module to render a different name

#### Scenario: Admitted resource passes the untracked test

- **WHEN** performing a first-time apply of the operator instance from `opm operator install`
- **AND** a rendered resource exists without OPM labels, and install proved it came from an earlier operator release manifest
- **THEN** the check SHALL NOT fail for that resource

#### Scenario: Admitted terminating resource still fails

- **WHEN** performing a first-time apply with an admission set
- **AND** an admitted resource exists with a `deletionTimestamp`
- **THEN** the command SHALL fail with an error indicating the resource is terminating

#### Scenario: Other commands admit nothing

- **WHEN** `opm instance apply` performs a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked

### Requirement: Apply flow orchestration

The apply flow SHALL follow this sequence: (1) render resources, (2) compute manifest digest, (3) compute change ID, (4) read previous inventory, and stop if the record cannot be read, (5a) compute stale set, (5b) run pre-apply existence check if first install, (6) apply all rendered resources via SSA, (7a) prune stale resources if all applied successfully, (7b) skip prune and inventory write if any apply failed, (8) write the inventory record, with every stale entry that step 7a failed to delete kept in it, (9) exit with an error if step 7a failed to delete a stale resource. No step SHALL filter the stale set after it is computed: the stale set is already component-blind.

#### Scenario: Normal apply with pruning

- **WHEN** a module is applied with changes from a previous apply
- **THEN** the system SHALL render, compute digest and change ID, read inventory, compute stale, apply resources, prune stale, and write inventory in order

#### Scenario: First-time apply

- **WHEN** a module is applied for the first time
- **THEN** the system SHALL render, compute digest and change ID, find no inventory, run pre-apply check, apply resources, skip pruning (empty stale set), and write a new inventory record

#### Scenario: Record write follows a failed prune

- **WHEN** a module is applied and a stale resource fails to delete
- **THEN** the system SHALL still write the inventory record, with the failed entry kept
- **AND** the command SHALL exit with an error after the write
