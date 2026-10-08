## Purpose

Defines the stale resource pruning behavior for `opm mod apply`. When a module is updated, resources that existed in the previous inventory but are absent from the current render are automatically pruned. Pruning is gated behind safety checks and can be disabled with `--no-prune`.

## Requirements

### Requirement: Stale resource detection

After rendering and before apply, the system SHALL compute the stale set with the library's `opm/k8s/inventory.StaleSet`: every previous inventory entry that no current entry is the same object as, comparing group, kind, namespace and name only. The component and the API version SHALL NOT count, so an object that moved to another component or another API version of its group is never stale. Resources in the stale set are candidates for pruning. Source: 0012:D7:R1.

#### Scenario: Resource removed from module

- **WHEN** the previous inventory contains entries [A, B, C] and the current render produces [A, B]
- **THEN** entry C SHALL appear in the stale set

#### Scenario: Resource renamed in module

- **WHEN** the previous inventory contains `Service/old-name` and the current render produces `Service/new-name`
- **THEN** `Service/old-name` SHALL appear in the stale set
- **AND** `Service/new-name` SHALL NOT appear in the stale set

#### Scenario: First-time apply has empty stale set

- **WHEN** there is no previous inventory (first-time apply)
- **THEN** the stale set SHALL be empty

#### Scenario: Idempotent re-apply has empty stale set

- **WHEN** the previous inventory entries are identical to the current render entries
- **THEN** the stale set SHALL be empty

#### Scenario: Component renamed without resource change is not stale

- **WHEN** the previous inventory has `Deployment/my-app` under component `web`
- **AND** the current render has `Deployment/my-app` under component `frontend`
- **THEN** `Deployment/my-app` SHALL NOT appear in the stale set
- **AND** the resource SHALL NOT be deleted

#### Scenario: Removal under a renamed component stays stale

- **WHEN** the previous inventory has `Deployment/old-app` under component `web`
- **AND** the current render does not contain `Deployment/old-app` under any component
- **THEN** `Deployment/old-app` SHALL appear in the stale set

### Requirement: Pre-apply existence check on first install

On first-time apply (no previous inventory), the system SHALL check each rendered resource against the cluster. If a resource exists with a `deletionTimestamp` (terminating), or exists without OPM labels (untracked), the apply SHALL fail with a clear error message. If a resource cannot be read (the read fails with any error other than NotFound), the apply SHALL fail too: the error SHALL name the resource and carry the read error, and SHALL NOT claim more than that no rendered resource was applied, since `--create-namespace` creates the namespace before the check. For `opm instance apply` and `opm module apply` the exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure; `opm operator install` SHALL refuse with the exit code of its other apply-guard refusals (2). A NotFound answer SHALL mean the resource does not exist, and passes. This check SHALL be skipped entirely when a previous inventory exists. A caller MAY pass an explicit admission set of resources; a resource in that set SHALL pass the untracked-resource test, and SHALL still fail the terminating-resource test and the unreadable-resource test. Only `opm operator install` SHALL pass a non-empty set, holding exactly the existing resources it proved came from an earlier opm-operator release manifest, or that already carry the operator instance's identity; every other caller SHALL pass none. No flag SHALL fill the set. The untracked-resource error SHALL NOT name a flag that would bypass it, since no flag does; it SHALL tell the user to remove or rename the existing resource, or to change the module so it renders a different name. Source: 0012:D8:R6.

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
- **AND** `opm instance apply` SHALL exit 4
- **AND** no rendered resource SHALL be applied

#### Scenario: Refusal after the namespace was created

- **WHEN** `opm instance apply --create-namespace` performs a first-time apply into a missing namespace
- **AND** the read of a rendered resource fails with Forbidden
- **THEN** the namespace SHALL have been created
- **AND** no rendered resource SHALL be applied
- **AND** the error SHALL NOT say that nothing changed

#### Scenario: Operator install refuses an unreadable resource

- **WHEN** `opm operator install` plans a first install and the read of an object it would apply fails with Forbidden
- **THEN** the command SHALL refuse with exit code 2 and an error naming the resource and the read error

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

### Requirement: Stale resources pruned after successful apply

After all rendered resources have been successfully applied, stale resources SHALL be deleted in reverse weight order (highest weight first — custom resources before CRDs). Deletion SHALL treat 404 as success (resource already gone). A `Namespace` in the core API group and a `CustomResourceDefinition` in the `apiextensions.k8s.io` group SHALL never be pruned, with no flag to override this: deleting a Namespace deletes everything inside it, and deleting a CRD deletes every custom resource of its kind in the cluster. Every stale resource of a protected kind SHALL be reported as left behind, one line per resource naming its kind, namespace and name with the status `left behind`.

#### Scenario: Stale resources deleted in reverse weight order

- **WHEN** the stale set contains a Deployment (weight 100) and a ConfigMap (weight 15)
- **THEN** the Deployment SHALL be deleted before the ConfigMap

#### Scenario: Already-deleted stale resource

- **WHEN** a stale resource no longer exists on the cluster
- **THEN** the prune operation SHALL treat the 404 as success

#### Scenario: Namespace excluded from pruning

- **WHEN** the stale set contains a Namespace resource
- **THEN** the Namespace SHALL NOT be pruned

#### Scenario: CRD excluded from pruning

- **WHEN** the stale set contains a `CustomResourceDefinition` in the `apiextensions.k8s.io` group
- **THEN** the CRD SHALL NOT be deleted
- **AND** the other stale resources SHALL still be pruned

#### Scenario: Protected stale resources are reported as left behind

- **WHEN** the stale set contains a Namespace and a CRD
- **THEN** the apply output SHALL list both with the status `left behind`
- **AND** the apply SHALL NOT fail because of them

#### Scenario: Same kind name in another group is not protected

- **WHEN** the stale set contains a resource of kind `Namespace` whose API group is not the core group
- **THEN** that resource SHALL be pruned like any other stale resource

### Requirement: No prune and no inventory write on apply failure

If any resource fails to apply, the system SHALL NOT prune stale resources and SHALL NOT write the inventory record. The inventory SHALL remain at the previous state, allowing a retry to converge naturally.

#### Scenario: Partial apply failure skips prune

- **WHEN** 3 of 5 resources apply successfully and 2 fail
- **THEN** no stale resources SHALL be pruned
- **AND** the inventory record SHALL NOT be written
- **AND** the command SHALL exit with an error

### Requirement: Unreadable instance record stops the apply

When the read of the instance's `ModuleInstance` record fails with any error other than NotFound, `opm instance apply` and `opm module apply` SHALL stop with an error. The system SHALL NOT treat the failed read as a first install: it SHALL NOT apply any resource, SHALL NOT prune, and SHALL NOT write the record. This SHALL hold for a dry run too, which SHALL NOT print a preview built without the record. The error SHALL name the instance and its namespace, carry the cause of the failed read, and tell the user to check access to the record and run the command again. The exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer SHALL still mean that no record exists. A namespace that `--create-namespace` created before the read SHALL stay.

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

### Requirement: Empty render safety gate

If the current render produces zero resources and the previous inventory is non-empty, the apply SHALL fail with an error unless `--force` is provided. This prevents accidental deletion of all resources due to a misconfigured module.

#### Scenario: Empty render without --force

- **WHEN** the render produces 0 resources
- **AND** the previous inventory has 5 entries
- **AND** `--force` is not set
- **THEN** the command SHALL fail with an error message indicating that all resources would be pruned

#### Scenario: Empty render with --force

- **WHEN** the render produces 0 resources
- **AND** the previous inventory has 5 entries
- **AND** `--force` is set
- **THEN** all 5 previous resources SHALL be pruned after apply

### Requirement: --no-prune flag skips pruning

The `--no-prune` flag SHALL skip the pruning step entirely. Stale resources SHALL remain on the cluster. The inventory record SHALL still be written with the current resource set.

#### Scenario: No-prune leaves stale resources

- **WHEN** the stale set contains 2 resources
- **AND** `--no-prune` is set
- **THEN** no stale resources SHALL be deleted
- **AND** the inventory SHALL be written with the current entries

### Requirement: --max-history flag caps change history

The `--max-history` flag SHALL control the maximum number of change entries retained in the inventory. The default SHALL be 10. After writing a new change, entries exceeding the limit SHALL be pruned from the tail of the index.

#### Scenario: Default max-history

- **WHEN** `--max-history` is not specified
- **THEN** the maximum history SHALL be 10

#### Scenario: Custom max-history

- **WHEN** the user specifies `--max-history=5`
- **THEN** the inventory SHALL retain at most 5 change entries

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

### Requirement: Dry-run prune preview lists left-behind resources

When `--dry-run` is set and pruning is enabled, the apply SHALL list the stale resources a real apply would prune, each with the status `would prune`, and SHALL list separately each stale resource of a protected kind (core `Namespace`, `apiextensions.k8s.io` `CustomResourceDefinition`) with the status `left behind`. The preview SHALL use the same protected-kind rule as the real prune, so the two lists together cover the whole stale set. With `--no-prune`, neither list SHALL be printed.

#### Scenario: Preview separates pruned and left-behind resources

- **WHEN** `opm instance apply --dry-run` runs with a stale set holding a ConfigMap, a Namespace and a CRD
- **THEN** the output SHALL list the ConfigMap as `would prune`
- **AND** the output SHALL list the Namespace and the CRD as `left behind`
- **AND** nothing SHALL be deleted

#### Scenario: No-prune dry run lists nothing

- **WHEN** `opm instance apply --dry-run --no-prune` runs with a non-empty stale set
- **THEN** the output SHALL list no `would prune` and no `left behind` lines

### Requirement: First install over resources OPM already manages warns

On a first-time apply (no `ModuleInstance` record), when one or more rendered resources already exist in the cluster with an OPM managed-by label, `opm instance apply` and `opm module apply` SHALL print one warning on the log stream before any rendered resource is applied. The warning SHALL give the number of such resources and the number of rendered resources, and SHALL say that the instance has no `ModuleInstance` record, that an apply updates those resources in place and records them, and that it prunes nothing. The warning SHALL NOT change the exit code and SHALL NOT stop the apply.

A dry run SHALL print the warning too. It SHALL read the rendered resources for this purpose only and SHALL refuse nothing because of that read: a resource the real run would refuse SHALL end the look without a warning and without an error. The dry-run warning SHALL name the release to apply with first (`v1.0.0-beta.10`) when an older release recorded the instance in a Secret.

The warning of a real run SHALL NOT send the user to that release, because the run writes the record and that release then deletes the Secret without reading it. It SHALL name the Secret (`opm.<name>.<id>`), SHALL say to keep it and not to apply with an older release, and SHALL name the docs page that describes the cleanup.

The apply SHALL print no such warning when no rendered resource exists yet, or when the instance has a record. `opm operator install` SHALL NOT print it: install applies the render's CustomResourceDefinitions itself before the instance apply, so a fresh install always finds them.

#### Scenario: Existing managed resources on a first install

- **WHEN** `opm instance apply` runs for an instance with no `ModuleInstance` record
- **AND** two of its three rendered resources already exist with the OPM managed-by label
- **THEN** the command SHALL print one warning that names 2 of 3 resources, the missing record and the Secret `opm.<name>.<id>`
- **AND** the warning SHALL NOT name `v1.0.0-beta.10`
- **AND** the apply SHALL go on and exit 0 when it succeeds

#### Scenario: Dry run warns before anything is written

- **WHEN** `opm instance apply --dry-run` runs for an instance with no `ModuleInstance` record
- **AND** two of its three rendered resources already exist with the OPM managed-by label
- **THEN** the command SHALL print one warning that names 2 of 3 resources and the release `v1.0.0-beta.10`
- **AND** no record SHALL be written

#### Scenario: Dry-run look refuses nothing

- **WHEN** `opm instance apply --dry-run` runs for an instance with no `ModuleInstance` record
- **AND** a rendered resource exists without OPM labels
- **THEN** the dry run SHALL NOT fail because of that resource and SHALL print no such warning

#### Scenario: Operator install prints no warning

- **WHEN** `opm operator install` runs on a cluster with no operator
- **THEN** the command SHALL print no such warning

#### Scenario: Clean first install prints no warning

- **WHEN** `opm instance apply` runs for an instance with no `ModuleInstance` record
- **AND** none of its rendered resources exists
- **THEN** the command SHALL print no such warning

#### Scenario: Instance with a record prints no warning

- **WHEN** `opm instance apply` runs for an instance that has a `ModuleInstance` record
- **THEN** the command SHALL print no such warning
