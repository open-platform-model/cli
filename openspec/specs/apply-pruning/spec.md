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

On first-time apply (no previous inventory), the system SHALL check each rendered resource against the cluster. If a resource exists with a `deletionTimestamp` (terminating), or exists without OPM labels (untracked), the apply SHALL fail with a clear error message. If a resource cannot be read (the read fails with any error other than NotFound), the apply SHALL fail too: the error SHALL name the resource and carry the read error. Every refusal of this check by `opm instance apply` and `opm module apply` SHALL say that the apply stopped before any change; that holds with `--create-namespace` too, since the check runs before the namespace is created and skips the objects in a namespace that is still missing. `opm operator install` SHALL NOT say so when the check refuses after the install applied its CustomResourceDefinitions. For `opm instance apply` and `opm module apply` the exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure; When this check refuses inside `opm operator install`, as that command's apply guard, the command SHALL exit with the code of its other apply-guard refusals (2), also when the refusal is a failed read. That code SHALL apply only to a read this check itself fails: `opm operator install` reads every object it applies in two earlier checks, so an object that is unreadable from the start never reaches this check and is refused there with the exit code of the read error (capability `operator-lifecycle`, "Every check that can refuse install runs before its first write"). A NotFound answer SHALL mean the resource does not exist, and passes. This check SHALL be skipped entirely when a previous inventory exists. A caller MAY pass an explicit admission set of resources; a resource in that set SHALL pass the untracked-resource test, and SHALL still fail the terminating-resource test and the unreadable-resource test. Only `opm operator install` SHALL pass a non-empty set, holding exactly the existing resources it proved came from an earlier opm-operator release manifest, or that already carry the operator instance's identity; every other caller SHALL pass none. No flag SHALL fill the set. The untracked-resource error SHALL NOT name a flag that would bypass it, since no flag does; it SHALL tell the user to remove or rename the existing resource, or to change the module so it renders a different name. Source: 0012:D8:R6.

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

- **WHEN** `opm instance apply --create-namespace` performs a first-time apply into a namespace that already exists, an earlier run having created it
- **AND** the read of a rendered resource fails with Forbidden
- **THEN** the namespace SHALL stay as it is
- **AND** no rendered resource SHALL be applied

#### Scenario: Refusal with create-namespace changes nothing

- **WHEN** `opm instance apply --create-namespace` performs a first-time apply into a missing namespace
- **AND** the read of a rendered cluster-scoped resource fails with Forbidden
- **THEN** the namespace SHALL NOT have been created
- **AND** no rendered resource SHALL be applied
- **AND** the error SHALL say that the apply stopped before any change

#### Scenario: Operator install refuses an unreadable resource

- **WHEN** `opm operator install` plans a first install and every read of an object it would apply fails with Forbidden
- **THEN** the command SHALL refuse with exit code 4 and an error naming the resource and the read error
- **AND** the refusal SHALL come from the first of install's reads of that object, before this check runs

#### Scenario: Operator install apply guard fails its own read

- **WHEN** `opm operator install` plans a first install, its two earlier reads of an object it would apply were answered, and this check's read of that object fails with Forbidden
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

After all rendered resources have been successfully applied, stale resources SHALL be deleted in reverse weight order (highest weight first: custom resources before CRDs). Deletion SHALL treat 404 as success (resource already gone). A `Namespace` in the core API group and a `CustomResourceDefinition` in the `apiextensions.k8s.io` group SHALL never be pruned, with no flag to override this: deleting a Namespace deletes everything inside it, and deleting a CRD deletes every custom resource of its kind in the cluster. Every stale resource of a protected kind SHALL be reported as left behind, one line per resource naming its kind, namespace and name with the status `left behind`. A stale `PersistentVolumeClaim` in the core API group SHALL be pruned only with `--delete-data` (see "Prune keeps stale PersistentVolumeClaims unless delete-data is set").

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

When the read of the instance's `ModuleInstance` record fails with any error other than NotFound, `opm instance apply` and `opm module apply` SHALL stop with an error. The system SHALL NOT treat the failed read as a first install: it SHALL NOT apply any resource, SHALL NOT prune, and SHALL NOT write the record. This SHALL hold for a dry run too, which SHALL NOT print a preview built without the record. The error SHALL name the instance and its namespace, carry the cause of the failed read, and tell the user to check access to the record and run the command again. The exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer SHALL still mean that no record exists. The error of `opm instance apply` and `opm module apply` SHALL say that the apply stopped before any change, and the command SHALL have changed nothing. With `--create-namespace` and a missing namespace the record SHALL NOT be read at all: none can exist there. `opm operator install` SHALL NOT say that nothing was changed when the read fails after the install applied its CustomResourceDefinitions.

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

#### Scenario: A missing namespace is not read for a record

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **THEN** the command SHALL NOT read the `ModuleInstance` record
- **AND** the apply SHALL proceed as a first-time apply

#### Scenario: Read denied inside operator install after its CRD step

- **WHEN** `opm operator install` has applied its CustomResourceDefinitions and the `ModuleInstance` read of its instance apply fails with Forbidden
- **THEN** the error SHALL name the record and the read error
- **AND** it SHALL NOT say that the apply stopped before any change

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

The apply flow SHALL follow this sequence: (1) render resources, (2) compute manifest digest, (3) compute change ID, (4) read previous inventory, and stop if the record cannot be read, (5a) compute stale set, (5b) run pre-apply existence check if first install, (6) apply all rendered resources via SSA, (7a) if all applied successfully, read each stale resource and prune those the delete verdict allows, judged with the instance identity the record held when step 4 read it, (7b) skip prune and inventory write if any apply failed, (8) write the inventory record, with every stale entry that step 7a failed to delete kept in it and with the instance identity of this render, (9) exit with an error if step 7a failed to delete a stale resource. No step SHALL filter the stale set after it is computed: the stale set is already component-blind.

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

#### Scenario: The record takes the new identity after the prune

- **WHEN** a module is applied whose rendered instance identity differs from the identity in the record
- **THEN** the system SHALL prune the stale resources judged with the identity the record held
- **AND** only then SHALL it write the record with the rendered identity

### Requirement: Dry-run prune preview lists left-behind resources

When `--dry-run` is set and pruning is enabled, the apply SHALL list the stale resources a real apply would prune, each with the status `would prune`, SHALL list separately each stale resource of a protected kind (core `Namespace`, `apiextensions.k8s.io` `CustomResourceDefinition`) with the status `left behind`, and SHALL list separately each stale core `PersistentVolumeClaim` that a real apply with the same flags would keep, with the status `kept`. The preview SHALL use the same protected-kind rule and the same `--delete-data` rule as the real prune, so the three lists together cover the whole stale set. With `--no-prune`, none of the lists SHALL be printed.

#### Scenario: Preview separates pruned and left-behind resources

- **WHEN** `opm instance apply --dry-run` runs with a stale set holding a ConfigMap, a Namespace and a CRD
- **THEN** the output SHALL list the ConfigMap as `would prune`
- **AND** the output SHALL list the Namespace and the CRD as `left behind`
- **AND** nothing SHALL be deleted

#### Scenario: No-prune dry run lists nothing

- **WHEN** `opm instance apply --dry-run --no-prune` runs with a non-empty stale set
- **THEN** the output SHALL list no `would prune` and no `left behind` lines

#### Scenario: Preview lists a claim as kept

- **WHEN** `opm instance apply --dry-run` runs with a stale set holding a ConfigMap and a PersistentVolumeClaim
- **THEN** the output SHALL list the ConfigMap as `would prune` and the claim as `kept`

#### Scenario: Preview with delete-data lists a claim as pruned

- **WHEN** `opm instance apply --dry-run --delete-data` runs with a stale set holding a PersistentVolumeClaim
- **THEN** the output SHALL list the claim as `would prune`

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

### Requirement: Prune keeps stale PersistentVolumeClaims unless delete-data is set

The prune of `opm instance apply` and `opm module apply` SHALL NOT delete a stale `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. A kept claim SHALL stay in the written inventory, so that it is stale again on the next apply and an apply with `--delete-data` removes it. Each kept claim SHALL be listed on its own line, naming its kind, namespace and name with the status `kept`, at the informational log level and never as a warning or an error; one more line SHALL say how many claims were kept and name `--delete-data`. A stale claim that is no longer in the cluster (its read answers NotFound) SHALL NOT be kept: it SHALL NOT be listed, and it SHALL leave the written inventory without `--delete-data`, as an already-deleted stale resource does. A stale claim whose read fails in any other way SHALL be kept. A kept claim SHALL NOT fail the apply: the exit code SHALL be 0 and the success line SHALL print. With `--delete-data`, a stale claim SHALL be pruned like any other stale resource and SHALL leave the inventory. `--delete-data` and `--no-prune` SHALL exclude each other: both together SHALL be a usage error. On an operator-managed instance `--delete-data` SHALL NOT change what the operator does and SHALL NOT fail the command: the command SHALL print a warning that says so, names `spec.dataPolicy` of the `ModuleInstance` as the setting that an operator with that field obeys for PersistentVolumeClaims, and says that an older operator deletes them.

#### Scenario: Stale claim is kept and stays in the inventory

- **WHEN** the stale set contains a ConfigMap and a PersistentVolumeClaim
- **AND** `--delete-data` is not set
- **THEN** the ConfigMap SHALL be deleted and the PersistentVolumeClaim SHALL NOT be deleted
- **AND** the written inventory SHALL hold the current entries and the PersistentVolumeClaim
- **AND** the output SHALL list the claim with the status `kept`
- **AND** the command SHALL exit 0

#### Scenario: Stale claim that is already gone

- **WHEN** the stale set contains a PersistentVolumeClaim that is no longer in the cluster
- **AND** `--delete-data` is not set
- **THEN** the output SHALL NOT list that claim as `kept`
- **AND** the written inventory SHALL NOT hold it

#### Scenario: Stale claim that cannot be read

- **WHEN** the read of a stale PersistentVolumeClaim fails with an error other than NotFound
- **AND** `--delete-data` is not set
- **THEN** the claim SHALL be listed as `kept` and SHALL stay in the written inventory

#### Scenario: Delete-data prunes the claim

- **WHEN** the stale set contains a PersistentVolumeClaim
- **AND** `--delete-data` is set
- **THEN** the PersistentVolumeClaim SHALL be deleted
- **AND** the written inventory SHALL NOT hold it

#### Scenario: A later apply with the flag removes a claim kept earlier

- **WHEN** an apply kept a stale PersistentVolumeClaim
- **AND** the next apply of the same render runs with `--delete-data`
- **THEN** the claim SHALL be in the stale set of that apply and SHALL be deleted

#### Scenario: Empty render with force keeps claims

- **WHEN** the render produces 0 resources, `--force` is set and `--delete-data` is not
- **AND** the previous inventory holds a ConfigMap and a PersistentVolumeClaim
- **THEN** the ConfigMap SHALL be pruned and the PersistentVolumeClaim SHALL be kept

#### Scenario: Delete-data with no-prune

- **WHEN** `opm instance apply --no-prune --delete-data` is run
- **THEN** the process SHALL exit 1 and standard error SHALL name both flags

#### Scenario: Delete-data on an operator-managed instance

- **WHEN** `opm instance apply --delete-data` or `opm module apply --delete-data` runs for an operator-managed instance
- **THEN** the command SHALL print a warning that `--delete-data` does not change what the operator does
- **AND** the warning SHALL name `spec.dataPolicy` and say that an older operator deletes PersistentVolumeClaims
- **AND** the apply SHALL go on as without the flag

### Requirement: Prune asks the ownership verdict for each stale resource

Before the prune deletes a stale resource, it SHALL read the live object and SHALL take the decision to delete it from the delete verdict the CLI shares with the operator. The verdict SHALL be asked with the instance identity stored in the instance's `ModuleInstance` record (`status.instanceUUID`), which is the identity that applied the stale resources, also when the identity of the current render differs from it. The prune SHALL delete the resource only when the live object carries an OPM managed-by label, its `module-instance.opmodel.dev/uuid` label does not name an instance other than the recorded one, and its `opmodel.dev/adopt` annotation does not name an instance other than the recorded one. An object with no UUID label SHALL be judged without the UUID comparison, and so SHALL every object when the record holds no identity. A resource that fails the verdict SHALL NOT be deleted: it SHALL be listed with the status `left behind` and the reason, SHALL NOT count as a failure, and SHALL NOT be kept in the written inventory. The stale set itself SHALL NOT change: it stays every recorded resource the render no longer names. Source: 0012:D4:R1, 0012:D7:R1, 0012:D8:R8.

An instance's identity derives from its module's path without the major version, the instance name and the namespace. The record is found by name and namespace, so within one record the identity changes only when the module's path changes. On the first apply after such a change the prune SHALL still delete the stale resources the instance owned under the recorded identity, and the record SHALL take the rendered identity only in the write that follows the prune.

Each delete SHALL carry a precondition on the UID of the object that was read, so that an object deleted and created again under the same name since the read is not deleted. A delete the API server refuses on that precondition, and a live read that fails with any error other than NotFound, SHALL each count as a failed delete of that resource ("Failed prune keeps the entry and fails the command"). A stale `PersistentVolumeClaim` that is kept SHALL NOT be judged; with `--delete-data` it SHALL be judged like any other stale resource. The dry-run prune preview SHALL stay as it is: it reads no stale resource for a verdict, so it MAY list as `would prune` a resource the real run leaves behind.

#### Scenario: Stale name taken by a user's object

- **WHEN** the stale set holds ConfigMap `old`
- **AND** the live ConfigMap `old` carries no OPM managed-by label
- **THEN** the ConfigMap SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that OPM does not manage it
- **AND** the written inventory SHALL NOT hold it
- **AND** the command SHALL exit 0

#### Scenario: Stale resource now owned by another instance

- **WHEN** the live object of a stale resource carries a `module-instance.opmodel.dev/uuid` that is not the recorded identity
- **THEN** the resource SHALL NOT be deleted and SHALL be listed as `left behind`

#### Scenario: Stale resource adopted by another instance

- **WHEN** the live object of a stale resource carries `opmodel.dev/adopt` with another instance's UUID
- **THEN** the resource SHALL NOT be deleted and SHALL be listed as `left behind`

#### Scenario: Stale object after the instance identity changed

- **WHEN** the module of an instance moved to a new path, so the rendered identity differs from the identity in the record
- **AND** the stale set holds a ConfigMap whose live `module-instance.opmodel.dev/uuid` label is the recorded identity
- **THEN** the ConfigMap SHALL be deleted
- **AND** the written record SHALL hold the rendered identity and SHALL NOT hold the ConfigMap

#### Scenario: Stale object that carries neither identity

- **WHEN** the rendered identity differs from the identity in the record
- **AND** the live object of a stale resource carries a UUID label equal to neither
- **THEN** the resource SHALL NOT be deleted and SHALL be listed as `left behind`

#### Scenario: Owned stale resource is deleted with a UID precondition

- **WHEN** the live object of a stale resource carries an OPM managed-by label and the recorded identity
- **THEN** the delete request SHALL carry a precondition on that object's UID

#### Scenario: Object replaced between the read and the delete

- **WHEN** the API server refuses the delete of a stale resource because the UID precondition does not match
- **THEN** the resource SHALL NOT be reported as pruned
- **AND** its entry SHALL stay in the written inventory
- **AND** the command SHALL exit 1 without the success line

#### Scenario: Live read denied

- **WHEN** the live read of a stale resource fails with Forbidden
- **THEN** the resource SHALL NOT be deleted
- **AND** its entry SHALL stay in the written inventory
- **AND** the command SHALL exit 1 without the success line

#### Scenario: Kept claim is not judged

- **WHEN** the stale set holds a PersistentVolumeClaim and `--delete-data` is not set
- **THEN** the claim SHALL be listed as `kept` and SHALL stay in the written inventory
