## MODIFIED Requirements

### Requirement: Unreadable instance record stops the apply

When the read of the instance's `ModuleInstance` record fails with any error other than NotFound, `opm instance apply` and `opm module apply` SHALL stop with an error. The system SHALL NOT treat the failed read as a first install: it SHALL NOT apply any resource, SHALL NOT prune, and SHALL NOT write the record. This SHALL hold for a dry run too, which SHALL NOT print a preview built without the record. The error SHALL name the instance and its namespace, carry the cause of the failed read, and tell the user to check access to the record and run the command again. The exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer SHALL still mean that no record exists. The error of `opm instance apply` and `opm module apply` SHALL say that the apply stopped before any change, and the command SHALL have changed nothing. With `--create-namespace` and a missing namespace the record SHALL NOT be read at all: none can exist there. `opm controller install` SHALL NOT say that nothing was changed when the read fails after the install applied its CustomResourceDefinitions.

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

#### Scenario: Read denied inside controller install after its CRD step

- **WHEN** `opm controller install` has applied its CustomResourceDefinitions and the `ModuleInstance` read of its instance apply fails with Forbidden
- **THEN** the error SHALL name the record and the read error
- **AND** it SHALL NOT say that the apply stopped before any change

### Requirement: First install over OPM-managed resources warns

On a first-time apply (no `ModuleInstance` record), when one or more rendered resources already exist in the cluster with an OPM managed-by label, `opm instance apply` and `opm module apply` SHALL print one warning on the log stream before any rendered resource is applied. The warning SHALL give the number of such resources and the number of rendered resources, and SHALL say that the instance has no `ModuleInstance` record, that an apply updates those resources in place and records them, and that it prunes nothing. The warning SHALL NOT change the exit code and SHALL NOT stop the apply.

A dry run SHALL print the warning too, when its ownership guard refuses nothing ("Ownership guard judges every apply and dry run"): a dry run that previews a refusal stops there, as the real run does, and prints no such warning. The dry-run warning SHALL name the release to apply with first (`v1.0.0-beta.10`) when an older release recorded the instance in a Secret.

The warning of a real run SHALL NOT send the user to that release, because the run writes the record and that release then deletes the Secret without reading it. It SHALL name the Secret (`opm.<name>.<id>`), SHALL say to keep it and not to apply with an older release, and SHALL name the docs page that describes the cleanup.

The apply SHALL print no such warning when no rendered resource exists yet, or when the instance has a record. `opm controller install` SHALL NOT print it: install applies the render's CustomResourceDefinitions itself before the instance apply, so a fresh install always finds them.

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

#### Scenario: Controller install prints no warning

- **WHEN** `opm controller install` runs on a cluster with no controller
- **THEN** the command SHALL print no such warning

#### Scenario: Clean first install prints no warning

- **WHEN** `opm instance apply` runs for an instance with no `ModuleInstance` record
- **AND** none of its rendered resources exists
- **THEN** the command SHALL print no such warning

#### Scenario: Instance with a record prints no warning

- **WHEN** `opm instance apply` runs for an instance that has a `ModuleInstance` record
- **THEN** the command SHALL print no such warning

#### Scenario: Dry run that previews a refusal prints no warning

- **WHEN** `opm instance apply --dry-run` runs for an instance with no `ModuleInstance` record
- **AND** a rendered resource exists without OPM labels
- **THEN** the dry run SHALL print the `would refuse` line of that resource and SHALL print no such warning

### Requirement: Prune keeps stale PersistentVolumeClaims unless delete-data is set

The prune of `opm instance apply` and `opm module apply` SHALL NOT delete a stale `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. A kept claim SHALL stay in the written inventory, so that it is stale again on the next apply and an apply with `--delete-data` removes it. Each kept claim SHALL be listed on its own line, naming its kind, namespace and name with the status `kept`, at the informational log level and never as a warning or an error; one more line SHALL say how many claims were kept and name `--delete-data`. A stale claim that is no longer in the cluster (its read answers NotFound) SHALL NOT be kept: it SHALL NOT be listed, and it SHALL leave the written inventory without `--delete-data`, as an already-deleted stale resource does. A stale claim whose read fails in any other way SHALL be kept. A kept claim SHALL NOT fail the apply: the exit code SHALL be 0 and the success line SHALL print. With `--delete-data`, a stale claim SHALL be pruned like any other stale resource and SHALL leave the inventory. `--delete-data` and `--no-prune` SHALL exclude each other: both together SHALL be a usage error. On a controller-managed instance `--delete-data` SHALL NOT change what the controller does and SHALL NOT fail the command: the command SHALL print a warning that says so, names `spec.dataPolicy` of the `ModuleInstance` as the setting that a controller with that field obeys for PersistentVolumeClaims, and says that an older controller deletes them.

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

#### Scenario: Delete-data on a controller-managed instance

- **WHEN** `opm instance apply --delete-data` or `opm module apply --delete-data` runs for a controller-managed instance
- **THEN** the command SHALL print a warning that `--delete-data` does not change what the controller does
- **AND** the warning SHALL name `spec.dataPolicy` and say that an older controller deletes PersistentVolumeClaims
- **AND** the apply SHALL go on as without the flag

### Requirement: Prune asks the ownership verdict for each stale resource

Before the prune deletes a stale resource, it SHALL read the live object and SHALL take the decision to delete it from the delete verdict the CLI shares with the controller. The verdict SHALL be asked with the instance identity stored in the instance's `ModuleInstance` record (`status.instanceUUID`), which is the identity that applied the stale resources, also when the identity of the current render differs from it. The prune SHALL delete the resource only when the live object carries an OPM managed-by label, its `module-instance.opmodel.dev/uuid` label does not name an instance other than the recorded one, and its `opmodel.dev/adopt` annotation does not name an instance other than the recorded one. An object with no UUID label SHALL be judged without the UUID comparison, and so SHALL every object when the record holds no identity. A resource that fails the verdict SHALL NOT be deleted: it SHALL be listed with the status `left behind` and the reason, SHALL NOT count as a failure, and SHALL NOT be kept in the written inventory. The stale set itself SHALL NOT change: it stays every recorded resource the render no longer names. Source: enhancement 0012.

An instance's identity derives from its module's path without the major version, the instance name and the namespace. The record is found by name and namespace, so within one record the identity changes only when the module's path changes. On the first apply after such a change the prune SHALL still delete the stale resources the instance owned under the recorded identity, and the record SHALL take the rendered identity only in the write that follows the prune.

Each delete SHALL carry a precondition on the UID of the object that was read, so that an object deleted and created again under the same name since the read is not deleted. A delete the API server refuses on that precondition, and a live read that fails with any error other than NotFound, SHALL each count as a failed delete of that resource ("Failed prune keeps the entry and fails the command"). A stale `PersistentVolumeClaim` that is kept SHALL NOT be judged; with `--delete-data` it SHALL be judged like any other stale resource. The dry-run prune preview SHALL ask the same verdict with the same identity and SHALL send no delete ("Dry-run prune preview lists left-behind resources").

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

### Requirement: Resource adopted by another instance is let go

When the instance's UUID is known and the live `opmodel.dev/adopt` annotation of a rendered resource names another instance, the apply SHALL NOT apply that resource and SHALL NOT fail because of it. This SHALL hold for a resource the instance's record lists, and for an OPM-managed resource the record does not list that carries no UUID label, this instance's UUID or the UUID its annotation names. The apply SHALL print one warning that names the resource and the adopting instance and says how to take the resource back, SHALL apply the other rendered resources, and SHALL write the inventory without that resource. The apply SHALL NOT prune or otherwise delete it. A dry run SHALL NOT send such a resource to the server either: in place of the warning it SHALL print one line that names the resource with the status `would skip` and the same text, and it SHALL NOT fail because of it.

When every rendered resource is let go, so that the apply applies nothing, the real apply SHALL print, after the warnings and the inventory write, one closing line that says that nothing was applied because all rendered resources are adopted by another instance, with their number. It SHALL print no success line that says resources were applied or are up to date, and SHALL keep exit 0. A dry run in the same state SHALL print one closing line that says that nothing would be applied, for the same reason.

`opm controller install` is the exception: it refuses on such a resource (capability `controller-lifecycle`). Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override" and "The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: Recorded resource handed to another instance

- **WHEN** `opm instance apply` runs for an instance whose record lists ConfigMap `settings`
- **AND** the live ConfigMap carries `opmodel.dev/adopt` with another instance's UUID
- **THEN** the ConfigMap SHALL NOT be applied and SHALL NOT be deleted
- **AND** the output SHALL warn that the instance drops it from its inventory
- **AND** the written inventory SHALL NOT hold the ConfigMap
- **AND** the command SHALL exit 0 when the other resources apply

#### Scenario: The instance does not take the resource back

- **WHEN** a later apply of the same instance renders ConfigMap `settings` again
- **AND** the live ConfigMap carries the adopting instance's UUID in both its adopt annotation and its UUID label
- **THEN** the ConfigMap SHALL NOT be applied, and the command SHALL NOT fail because of it

#### Scenario: Annotating the resource back reverses the hand-over

- **WHEN** the adopt annotation of ConfigMap `settings` is set to the first instance's UUID
- **AND** the first instance is applied
- **THEN** the ConfigMap SHALL be applied and recorded in its inventory

#### Scenario: Every rendered resource is adopted elsewhere

- **WHEN** `opm instance apply` runs and the live adopt annotation of each rendered resource names another instance
- **THEN** no rendered resource SHALL be applied or deleted
- **AND** the output SHALL hold one line that says that nothing was applied because all rendered resources are adopted by another instance
- **AND** the inventory SHALL be written without those resources
- **AND** the command SHALL exit 0

#### Scenario: Dry run skips an adopted resource

- **WHEN** `opm instance apply --dry-run` runs and the live adopt annotation of rendered ConfigMap `settings` names another instance
- **THEN** the output SHALL list the ConfigMap as `would skip` with a reason that names the adopting instance
- **AND** the ConfigMap SHALL NOT be sent to the server
- **AND** the command SHALL exit 0 when the other resources pass

#### Scenario: Dry run where every rendered resource is adopted elsewhere

- **WHEN** `opm instance apply --dry-run` runs and the live adopt annotation of each rendered resource names another instance
- **THEN** the output SHALL hold one line that says that nothing would be applied
- **AND** the command SHALL exit 0

### Requirement: Prune follows the shared deletion plan

The prune of `opm instance apply` and `opm module apply` SHALL take every action on a stale resource from the deletion plan the CLI shares with the controller, built from the stale entries the prune may delete and the instance identity stored in the instance's record. That set SHALL NOT hold a `Namespace`, a `CustomResourceDefinition`, or a `PersistentVolumeClaim` that is kept. The prune SHALL read a stale resource, delete it or leave it in place only when the plan names that action as the next one, in descending kind-weight order, and it SHALL send each delete with the propagation policy and the precondition the plan names. A resource whose read or delete fails SHALL NOT stop the plan, except that after a failed API discovery request the prune SHALL send no further request and SHALL report that resource and every resource not yet tried as failed with the discovery error. A stale resource SHALL count as pruned when the API server accepts its delete request. Which entries stay in the written inventory, the output lines and the exit codes SHALL stay as the other requirements of this capability state them. The dry-run prune preview SHALL NOT be affected. Source: enhancement 0012.

#### Scenario: Stale resources are deleted in the plan's order

- **WHEN** the stale set holds a ConfigMap and a Deployment
- **THEN** the delete request for the Deployment SHALL be sent before the one for the ConfigMap

#### Scenario: Delete carries what the plan names

- **WHEN** the plan names the delete of a stale ConfigMap the instance owns
- **THEN** the delete request SHALL carry Foreground propagation and a precondition on the UID of the object that was read

#### Scenario: One failed delete does not stop the others

- **WHEN** the stale set holds ConfigMaps `a` and `b` and the delete of `a` fails with Forbidden
- **THEN** ConfigMap `b` SHALL still be read and deleted
- **AND** the written inventory SHALL hold the entry of `a` and SHALL NOT hold the entry of `b`

#### Scenario: Failed discovery request stops the requests

- **WHEN** the stale set holds two resources and the API discovery request for the first one's group and version fails with Forbidden
- **THEN** no read and no delete SHALL be sent for the second resource
- **AND** both entries SHALL stay in the written inventory
- **AND** the command SHALL exit 4

#### Scenario: Stale resource held by a finalizer

- **WHEN** the delete of a stale ConfigMap is accepted and a finalizer keeps the ConfigMap from disappearing
- **THEN** the resource SHALL count as pruned
- **AND** the written inventory SHALL NOT hold its entry
- **AND** the command SHALL exit 0

### Requirement: Ownership guard judges every apply and dry run

On every apply in CLI-executor mode, the first as well as every later one, `opm instance apply` and `opm module apply` SHALL read each rendered resource from the cluster and SHALL take the decision to apply it from the ownership verdict the CLI shares with the controller, asked with the instance identity of the render. A resource that does not exist SHALL pass. An existing resource that the instance's `ModuleInstance` record does not list SHALL be refused when it carries no OPM managed-by label, or when its `module-instance.opmodel.dev/uuid` label names another instance. An existing resource with a `deletionTimestamp` SHALL be refused whether or not the record lists it. When any resource is refused, the apply SHALL fail before its first write, the namespace of `--create-namespace` included, SHALL report every refused resource by kind, namespace and name with the reason, SHALL say that the apply stopped before any change, and SHALL exit 1. Source: enhancement 0012.

When the instance has no record and a resource is refused because it carries another instance's UUID, the refusal SHALL add one line that says what to do when the resources are the instance's own under an earlier identity (its module path, its name or its namespace changed, its record was deleted, or opm v1.0.0-alpha.1 or older recorded it in a Secret): annotate each resource as the refusal shows, with nothing to remove first.

The one override SHALL be the adopt annotation: an existing resource whose `opmodel.dev/adopt` annotation holds this instance's UUID SHALL pass the ownership refusals, SHALL be applied and SHALL be recorded in the instance's inventory. The annotation SHALL NOT override the refusal of a terminating resource. The CLI SHALL NOT set the annotation and SHALL offer no flag that sets it or that bypasses the guard. Every ownership refusal SHALL name the annotation and the UUID to set, and SHALL NOT name a flag. Source: enhancement 0012.

If a rendered resource cannot be read (the read fails with any error other than NotFound), the apply SHALL fail before its first write: the error SHALL name the resource and carry the read error, and the exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer, or a kind the cluster does not serve yet, SHALL mean the resource does not exist. With `--create-namespace` and a missing namespace, the guard SHALL skip the rendered resources in that namespace.

A dry run SHALL run the same guard, with the same reads, and the same verdict as the real run, and SHALL write nothing. For each resource the real run would refuse, the dry run SHALL print one line that names the resource by kind, namespace and name with the status `would refuse` and the reason of the verdict, which names the owning or adopting instance where there is one and the adopt annotation to set where one lifts the refusal. The dry run SHALL then stop where the real run stops: it SHALL send no rendered resource to the server, SHALL print no prune preview, and SHALL exit 1, the code of the real refusal, with an error that gives the number of refused resources, says that a real apply would be refused, and says that the dry run changed nothing. On a first apply it SHALL add the same earlier-identity line as the real refusal. When a rendered resource cannot be read, the dry run SHALL fail with the same error and the same exit code (4, 3 or 1) as the real run. A dry run of a controller-managed instance SHALL NOT run the guard: the controller applies that instance.

The guard SHALL take no admission set: no caller, `opm controller install` included, SHALL lift a refusal for a resource by a proof of where the resource came from. An existing resource SHALL pass only when the record of the instance lists it, when its adopt annotation holds this instance's UUID, or when it carries an OPM managed-by label and no UUID label of another instance. When the guard refuses in the check phase of `opm controller install`, the exit code SHALL be that command's apply-guard code (capability `controller-lifecycle`). When it refuses inside that command's instance apply, after the install's writes, the refusal SHALL NOT say that nothing was changed.

#### Scenario: Foreign resource refused on a first apply

- **WHEN** `opm instance apply` runs for an instance with no record
- **AND** a rendered resource already exists on the cluster without an OPM managed-by label
- **THEN** the command SHALL exit 1 before any change, naming the resource
- **AND** the error SHALL name the annotation `opmodel.dev/adopt` and the instance's UUID
- **AND** the error SHALL NOT mention `--force`

#### Scenario: Foreign resource refused on a later apply

- **WHEN** `opm instance apply` runs for an instance that has a record
- **AND** the render names a resource the record does not list, which exists on the cluster without an OPM managed-by label
- **THEN** the command SHALL exit 1 and SHALL apply no rendered resource

#### Scenario: Resource of another instance refused

- **WHEN** `opm instance apply` runs
- **AND** a rendered resource the record does not list exists with an OPM managed-by label and the `module-instance.opmodel.dev/uuid` of another instance
- **THEN** the command SHALL exit 1 before any change, naming the resource and the other instance's UUID

#### Scenario: Recorded resource is applied whatever its UUID label

- **WHEN** `opm instance apply` runs for an instance whose record lists a rendered resource
- **AND** the live resource carries a UUID label that differs from the instance's and no adopt annotation
- **THEN** the guard SHALL NOT refuse it and the resource SHALL be applied

#### Scenario: Terminating resource refused on every apply

- **WHEN** `opm instance apply` runs for an instance whose record lists a rendered resource
- **AND** that resource exists with a `deletionTimestamp`
- **THEN** the command SHALL exit 1 before any change with an error saying the resource is being deleted

#### Scenario: Adopt annotation lets the instance take a resource

- **WHEN** `opm instance apply` runs and a rendered resource exists without an OPM managed-by label
- **AND** its `opmodel.dev/adopt` annotation holds the instance's UUID
- **THEN** the resource SHALL be applied and recorded in the instance's inventory

#### Scenario: Adopt annotation does not lift the terminating refusal

- **WHEN** a rendered resource has a `deletionTimestamp` and an `opmodel.dev/adopt` annotation holding the instance's UUID
- **THEN** the command SHALL exit 1 before any change

#### Scenario: First install over the instance's own resources under an earlier identity

- **WHEN** `opm instance apply` runs for an instance with no record
- **AND** every rendered resource exists with an OPM managed-by label and one other UUID
- **THEN** the command SHALL exit 1 before any change, naming each resource and the annotation to set
- **AND** the output SHALL say that nothing has to be removed first when the resources are the instance's own

#### Scenario: Every refused resource is reported

- **WHEN** two rendered resources exist without an OPM managed-by label
- **THEN** the output SHALL name both before the command exits 1

#### Scenario: Unreadable resource refused on a later apply

- **WHEN** `opm instance apply` runs for an instance that has a record
- **AND** the read of a rendered resource fails with Forbidden
- **THEN** the command SHALL exit 4 with an error naming the resource and the read error
- **AND** no rendered resource SHALL be applied

#### Scenario: Refusal with create-namespace changes nothing

- **WHEN** `opm instance apply --create-namespace` performs a first-time apply into a missing namespace
- **AND** a rendered cluster-scoped resource exists without an OPM managed-by label
- **THEN** the namespace SHALL NOT have been created
- **AND** the error SHALL say that the apply stopped before any change

#### Scenario: Absent resource passes

- **WHEN** the read of a rendered resource answers NotFound
- **THEN** the guard SHALL NOT refuse that resource

#### Scenario: Dry run previews a refusal and exits 1

- **WHEN** `opm instance apply --dry-run` runs and a rendered resource exists without an OPM managed-by label
- **THEN** the output SHALL list that resource as `would refuse` with the reason and the adopt annotation to set
- **AND** the command SHALL exit 1
- **AND** no rendered resource SHALL be sent to the server and no record SHALL be written

#### Scenario: Dry run names the owner of a refused resource

- **WHEN** `opm module apply --dry-run` runs and a rendered resource the record does not list carries another instance's UUID label
- **THEN** the `would refuse` line SHALL name that instance's UUID
- **AND** the command SHALL exit 1

#### Scenario: Dry run lists every refused resource

- **WHEN** `opm instance apply --dry-run` runs and two rendered resources exist without an OPM managed-by label
- **THEN** the output SHALL hold one `would refuse` line for each

#### Scenario: Dry run with an unreadable resource

- **WHEN** `opm instance apply --dry-run` runs and the read of a rendered resource fails with Forbidden
- **THEN** the command SHALL exit 4 with an error naming the resource and the read error

#### Scenario: Dry run with nothing to refuse

- **WHEN** `opm instance apply --dry-run` runs and every existing rendered resource is the instance's own
- **THEN** the output SHALL hold no `would refuse` line and the command SHALL exit 0 when nothing else fails

#### Scenario: Controller install lifts no refusal

- **WHEN** `opm controller install` runs on a cluster with no record of the controller's instance
- **AND** a rendered resource exists without an OPM managed-by label and without an adopt annotation
- **THEN** the guard SHALL refuse that resource as it does for `opm instance apply`
