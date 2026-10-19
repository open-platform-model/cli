## REMOVED Requirements

### Requirement: Instance delete of an instance that deploys the operator is guarded
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

### Requirement: Instance delete keeps the ModuleInstance when a tracked resource could not be read
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## MODIFIED Requirements

### Requirement: Instance delete re-checks live ownership before each delete

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again and SHALL take the decision to delete it from the delete verdict the CLI shares with the controller and with its own prune, asked with the instance identity stored in the record. It SHALL delete the resource only when the live `app.kubernetes.io/managed-by` label carries an OPM value, its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID, and its `opmodel.dev/adopt` annotation does not name another instance. An object with no UUID label SHALL be judged without the UUID comparison, and so SHALL every object when the instance has no recorded UUID. A resource that fails the check SHALL be left behind and listed with the reason, and SHALL NOT count as a failure; the closing line SHALL give the number of resources left behind. Each delete SHALL carry a precondition on the UID of the object that was read; a delete the API server refuses on that precondition SHALL count as a failure for that resource and SHALL NOT be reported as deleted. A resource that is already gone, whether its re-read or its delete call returns NotFound, SHALL count neither as deleted nor as a failure. Any other read error SHALL count as a failure for that resource, so the `ModuleInstance` record is kept and a re-run retries. Source: enhancement 0012.

#### Scenario: Resource no longer managed by OPM is left behind

- **WHEN** a tracked resource's live `app.kubernetes.io/managed-by` label is missing or not an OPM value
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that OPM does not manage it

#### Scenario: Resource owned by another instance is left behind

- **WHEN** a tracked resource's live `module-instance.opmodel.dev/uuid` label differs from the instance's recorded UUID
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that it belongs to another instance

#### Scenario: Resource without a UUID label is deleted on managed-by alone

- **WHEN** a tracked resource carries an OPM managed-by value and no `module-instance.opmodel.dev/uuid` label
- **THEN** the resource SHALL be deleted

#### Scenario: Resource gone before its delete

- **WHEN** a tracked resource no longer exists when it is read again
- **THEN** the command SHALL NOT report an error for it
- **AND** it SHALL NOT be counted as deleted

#### Scenario: Resource gone between its re-read and its delete

- **WHEN** a tracked resource passes the ownership check and its delete call returns NotFound
- **THEN** the command SHALL NOT report an error for it
- **AND** it SHALL NOT be counted as deleted

#### Scenario: Instance without a recorded UUID deletes on managed-by alone

- **WHEN** the instance has no recorded UUID
- **AND** a tracked resource carries an OPM managed-by value and any `module-instance.opmodel.dev/uuid` label
- **THEN** the resource SHALL be deleted

#### Scenario: Read error keeps the ModuleInstance

- **WHEN** re-reading a tracked resource fails with an error other than NotFound, such as Forbidden
- **THEN** the resource SHALL be reported as failed
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the command SHALL exit non-zero

#### Scenario: Resource adopted by another instance is left behind

- **WHEN** a tracked resource carries an OPM managed-by value and the instance's UUID
- **AND** its live `opmodel.dev/adopt` annotation names another instance
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that another instance is adopting it
- **AND** the closing line SHALL say that one resource was left behind

#### Scenario: Delete carries a UID precondition

- **WHEN** a tracked resource passes the ownership check
- **THEN** the delete request SHALL carry a precondition on the UID of the object that was read

#### Scenario: Object replaced between its re-read and its delete

- **WHEN** the API server refuses a delete because the UID precondition does not match
- **THEN** the resource SHALL be reported as failed and SHALL NOT be counted as deleted
- **AND** the `ModuleInstance` record SHALL NOT be deleted

### Requirement: Instance delete keeps PersistentVolumeClaims unless delete-data is set

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. Each kept claim SHALL be listed on its own line, naming its kind, namespace and name with the status `kept`, in the dry run and in the real run, at the informational log level and never as a warning or an error. A kept claim SHALL NOT count as a failure: the other tracked resources SHALL be deleted, the `ModuleInstance` record SHALL be deleted last, and the command SHALL exit 0. After the delete the claim is in no inventory, so no later `opm` command deletes it. The closing output SHALL state how many claims were kept, SHALL print for each kept claim the `kubectl delete pvc` command that deletes it, and SHALL name `--delete-data`. With `--delete-data`, a tracked claim SHALL be deleted under the same live-ownership check as every other tracked resource. The kept-claim rule SHALL apply to the core group only: a kind of the same name in another API group is deleted like any other resource. The command SHALL NOT handle claims that a StatefulSet created from `volumeClaimTemplates`: they are not in the inventory, with or without the flag.

On a controller-managed instance `--delete-data` SHALL NOT change what the controller does and SHALL NOT fail the command: the command SHALL print a warning, before it asks for confirmation, that says so, names `spec.dataPolicy` of the `ModuleInstance` as the setting that a controller with that field obeys for PersistentVolumeClaims, and says that an older controller deletes them. The kept-claim rule of the CLI SHALL NOT be claimed for a controller-managed instance: the controller decides what it removes. When the delete of a controller-managed instance with `spec.prune` set completes, its inventory tracked at least one PersistentVolumeClaim, its `spec.dataPolicy` is not `Delete` and the installed `ModuleInstance` CRD has `spec.dataPolicy` or has no schema that can be read, the closing output SHALL NOT say that every tracked resource was pruned and SHALL NOT say that the claims are still in the cluster: it SHALL name each such claim, SHALL say that a controller with `spec.dataPolicy` keeps them and that a controller older than its CRDs deleted them, and SHALL print the `kubectl delete pvc` command for each. In every other case the closing output SHALL report the prune of the tracked resources as before.

#### Scenario: Claim is kept by default

- **WHEN** running `opm instance delete --yes` for a CLI-owned instance whose inventory tracks a Deployment and a PersistentVolumeClaim
- **THEN** the Deployment SHALL be deleted
- **AND** the PersistentVolumeClaim SHALL NOT be deleted
- **AND** the output SHALL list the claim with the status `kept`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0
- **AND** the closing output SHALL carry `kubectl delete pvc` with the name and namespace of the claim

#### Scenario: Delete-data deletes the claim

- **WHEN** running `opm instance delete --yes --delete-data` for the same instance
- **THEN** the Deployment and the PersistentVolumeClaim SHALL be deleted
- **AND** the output SHALL list no resource with the status `kept`

#### Scenario: Dry run lists the claim as kept

- **WHEN** running `opm instance delete --dry-run` for an instance whose inventory tracks a PersistentVolumeClaim
- **THEN** the output SHALL list the claim with the status `kept`
- **AND** the dry-run output SHALL say how many claims would be kept and name `--delete-data`
- **AND** nothing SHALL be deleted

#### Scenario: Unreadable claim is kept

- **WHEN** discovery could not read a tracked PersistentVolumeClaim
- **AND** `--delete-data` is not set
- **THEN** the claim SHALL be listed as `kept` and SHALL NOT count as a failure

#### Scenario: Controller-managed instance

- **WHEN** running `opm instance delete --delete-data` for a controller-managed instance
- **THEN** the command SHALL warn that `--delete-data` does not change what the controller does and SHALL name `spec.dataPolicy`
- **AND** the warning SHALL say that an older controller deletes PersistentVolumeClaims
- **AND** the delete SHALL go on as without the flag

#### Scenario: Controller-managed delete that may have kept claims closes without claiming either outcome

- **WHEN** `opm instance delete --yes` completes for a controller-managed instance whose `spec.prune` is true, whose `spec.dataPolicy` is absent and whose inventory tracks the PersistentVolumeClaim `data` in namespace `apps`
- **AND** the installed `ModuleInstance` CRD has `spec.dataPolicy`
- **THEN** the closing output SHALL NOT say that the controller pruned every tracked resource
- **AND** it SHALL NOT say that the claim is still in the cluster
- **AND** it SHALL name the claim and carry `kubectl delete pvc data -n apps`
- **AND** it SHALL say that a controller older than its CRDs deleted the claim

#### Scenario: Controller-managed delete with the Delete policy closes as a full prune

- **WHEN** the same delete completes for an instance whose `spec.dataPolicy` is `Delete`
- **THEN** the closing output SHALL say that the controller pruned the tracked resources
- **AND** it SHALL print no `kubectl delete pvc` command

#### Scenario: Controller-managed delete on a controller without the field closes as a full prune

- **WHEN** the same delete completes in a cluster whose `ModuleInstance` CRD has no `spec.dataPolicy`
- **THEN** the closing output SHALL say that the controller pruned the tracked resources
- **AND** no output of the command SHALL say that a PersistentVolumeClaim is kept

### Requirement: Instance delete confirmation names the claims that delete-data deletes

`opm instance delete` SHALL read the instance record before it asks for confirmation. When `--delete-data` is set, the instance is CLI-owned and its inventory tracks at least one live PersistentVolumeClaim, the confirmation prompt SHALL name every such claim by namespace and name and SHALL say that the data on them is deleted. Without `--delete-data` the prompt for a CLI-owned instance SHALL say that PersistentVolumeClaims are kept. For a controller-managed instance the prompt SHALL be the same with or without `--delete-data`, and SHALL say that the instance is controller-managed and what the controller does. When `spec.prune` is not set, it SHALL say that the controller leaves the tracked resources running. When `spec.prune` is set and the inventory tracks no PersistentVolumeClaim, it SHALL say that the controller deletes the tracked resources and SHALL say nothing about PersistentVolumeClaims. When `spec.prune` is set and the inventory tracks at least one PersistentVolumeClaim, the prompt SHALL say one of four things, decided by whether the installed `ModuleInstance` CRD has `spec.dataPolicy`. The command SHALL take that from the CRD its controller-readiness check reads and SHALL NOT read the CRD a second time for it. When the CRD has no such field, the controller does not know it: the prompt SHALL say so and SHALL say that the controller deletes the tracked resources, PersistentVolumeClaims and their data included, whatever the instance carries. When `spec.dataPolicy` is `Delete`, it SHALL say the same and name the value. When the CRD has the field and `spec.dataPolicy` is `Keep`, absent, or any other value, it SHALL say that the controller deletes the tracked resources and keeps PersistentVolumeClaims and their data, and SHALL add one sentence which says that a controller older than its CRDs deletes the claims whatever the field says. When the CRD has no schema that can be read and `spec.dataPolicy` is not `Delete`, it SHALL NOT say that claims are kept or deleted: it SHALL say that a controller with the field keeps them, that an older controller deletes them, and that `opm` could not read the schema of the CRD to tell. The prompt SHALL show the value of `spec.dataPolicy` as it is, or say that it is not set; a value other than `Keep` and `Delete` SHALL be shown quoted and named as read as `Keep`. The dry run, and the run with `--yes`, SHALL state the same outcome for PersistentVolumeClaims in their output. An instance that has no record SHALL be reported as not found before any prompt. Every check that can refuse the delete SHALL run before the prompt: the guard for an instance that deploys the controller, and, for a controller-managed instance, the controller-readiness check, whose failure includes a CRD that cannot be read. A refused delete SHALL NOT ask for confirmation.

#### Scenario: Prompt lists the claims

- **WHEN** running `opm instance delete jellyfin -n media --delete-data` without `--yes`
- **AND** the inventory tracks the claims `config` and `media` in namespace `media`
- **THEN** the prompt SHALL name `media/config` and `media/media`
- **AND** the prompt SHALL say that their data is deleted

#### Scenario: Prompt without delete-data

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes`
- **THEN** the prompt SHALL say that PersistentVolumeClaims are kept

#### Scenario: Prompt for a controller-managed instance with prune set

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for a controller-managed instance whose `spec.prune` is true and whose inventory tracks a PersistentVolumeClaim
- **AND** whose `spec.dataPolicy` is `Delete`
- **THEN** the prompt SHALL say that the instance is controller-managed and that the controller deletes its tracked resources, PersistentVolumeClaims and their data included
- **AND** the prompt SHALL name `spec.dataPolicy` and its value `Delete`
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are kept

#### Scenario: Prompt on a controller without the field

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for a controller-managed instance whose `spec.prune` is true and whose inventory tracks a PersistentVolumeClaim
- **AND** the installed `ModuleInstance` CRD has no `spec.dataPolicy`
- **THEN** the prompt SHALL say that the controller in the cluster has no `spec.dataPolicy` and deletes its tracked resources, PersistentVolumeClaims and their data included
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are kept

#### Scenario: Prompt for a controller-managed instance with prune set and no data policy

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for a controller-managed instance whose `spec.prune` is true, whose inventory tracks a PersistentVolumeClaim and whose `spec.dataPolicy` is absent or `Keep`
- **AND** the installed `ModuleInstance` CRD has `spec.dataPolicy`
- **THEN** the prompt SHALL say that the controller deletes its tracked resources and keeps PersistentVolumeClaims and the data on them
- **AND** the prompt SHALL say that `spec.dataPolicy` is not set, or that it is `Keep`
- **AND** the prompt SHALL say that a controller older than its CRDs deletes the claims whatever the field says

#### Scenario: Prompt when the CRD cannot be read

- **WHEN** the same prompt is built and the `ModuleInstance` CRD has no schema that can be read
- **THEN** the prompt SHALL NOT say that PersistentVolumeClaims are kept
- **AND** the prompt SHALL say that `opm` could not read the schema of the CRD to tell which controller runs

#### Scenario: Refusal comes before the question

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for a controller-managed instance
- **AND** the controller is not ready, or one of its CRDs or its controller Deployment cannot be read
- **THEN** the command SHALL exit 2 with the controller-readiness refusal
- **AND** it SHALL NOT print a confirmation prompt

#### Scenario: Prompt for an unknown data policy

- **WHEN** the `spec.dataPolicy` of a controller-managed instance with `spec.prune` set and a tracked PersistentVolumeClaim is `Retain`, and the CRD has the field
- **THEN** the prompt SHALL show `"Retain"` and say that it is read as `Keep`
- **AND** the prompt SHALL say that the controller keeps PersistentVolumeClaims

#### Scenario: Prompt for a controller-managed instance without prune

- **WHEN** the `spec.prune` of a controller-managed instance is not set and its `spec.dataPolicy` is `Delete`
- **THEN** the prompt SHALL say that the controller leaves its tracked resources running
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are deleted

#### Scenario: Prompt for an instance that tracks no claim

- **WHEN** the `spec.prune` of a controller-managed instance is true and its inventory tracks no PersistentVolumeClaim
- **THEN** the prompt SHALL say that the controller deletes its tracked resources
- **AND** the prompt SHALL NOT mention PersistentVolumeClaims

#### Scenario: Delete-data note comes before the question

- **WHEN** running `opm instance delete jellyfin -n media --delete-data` without `--yes` for a controller-managed instance
- **THEN** the warning that `--delete-data` does not change what the controller does SHALL be printed before the confirmation question

#### Scenario: Missing instance is reported before the prompt

- **WHEN** running `opm instance delete nosuch -n media` without `--yes`
- **THEN** the command SHALL exit 5 without printing a confirmation prompt

### Requirement: Instance delete follows the shared deletion plan

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the controller, built from the resources of the instance's `ModuleInstance` record that the command found in the cluster or could not read, and from the instance identity stored in the record. A recorded resource that the command did not find when it read the instance SHALL NOT be in the plan: it is not read again and not deleted. It SHALL read a tracked resource, delete it or leave it in place only when the plan names that action as the next one, and it SHALL send each delete with the propagation policy and the precondition the plan names. The plan SHALL NOT hold a `PersistentVolumeClaim` that the command keeps ("Instance delete keeps PersistentVolumeClaims unless delete-data is set"). The command SHALL delete the `ModuleInstance` record only when the plan's release verdict allows it, which is when every planned resource was deleted or left in place and none failed. A dry run SHALL follow the same plan, SHALL send no delete, and SHALL NOT delete the record. The output lines, their wording and the exit codes of the command SHALL stay as the other requirements of this capability state them. The controller-owned delete path SHALL NOT be affected. Source: enhancement 0012.

#### Scenario: Resources are deleted in the plan's order

- **WHEN** running `opm instance delete` for a CLI-owned instance whose record lists a ConfigMap, a Service and a Deployment
- **THEN** the delete requests SHALL be sent in descending kind-weight order: the Deployment, then the Service, then the ConfigMap

#### Scenario: Delete carries what the plan names

- **WHEN** the plan names the delete of a tracked Deployment
- **THEN** the delete request SHALL carry Foreground propagation and a precondition on the UID of the object that was read

#### Scenario: Failed resource holds the record

- **WHEN** the delete of one tracked resource fails with Forbidden and every other resource is deleted
- **THEN** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the command SHALL exit 1

#### Scenario: Left-behind resources do not hold the record

- **WHEN** the plan leaves a tracked Namespace and a resource owned by another instance in place, and deletes every other resource
- **THEN** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: Kept claim is outside the plan

- **WHEN** the record lists a PersistentVolumeClaim and `--delete-data` is not set
- **THEN** the claim SHALL NOT be read for a delete verdict and SHALL NOT be deleted
- **AND** it SHALL be listed as `kept`
- **AND** the `ModuleInstance` record SHALL be deleted when no other resource failed

#### Scenario: Resource that appears after the instance was read

- **WHEN** a recorded ConfigMap does not exist when the command reads the instance
- **AND** it exists again before the deletes are sent
- **THEN** the ConfigMap SHALL NOT be deleted
- **AND** the `ModuleInstance` record SHALL be deleted when no planned resource failed

#### Scenario: Dry run follows the plan without deleting

- **WHEN** running `opm instance delete --dry-run`
- **THEN** each tracked resource SHALL be read and judged as on a real run
- **AND** no delete request SHALL be sent
- **AND** the `ModuleInstance` record SHALL still exist

### Requirement: Instance delete waits for the deleted resources when wait is set

`opm instance delete` SHALL accept `--wait`. With `--wait`, on a real run of a CLI-owned instance in which no tracked resource failed, the command SHALL, after every planned action is done and before it deletes the `ModuleInstance` record, poll each resource whose delete the API server accepted until the resource is gone or `--timeout` (default `5m0s`) has passed since the wait started. A resource is gone when its read returns NotFound, or when the live object under its name has a UID other than the one of the object that was deleted. A read that fails with another error SHALL NOT count as gone.

While it waits, the command SHALL print a line that says how many of the deleted resources are still terminating each time that number drops, and at least every 30 seconds otherwise.

The command SHALL NOT wait for a resource it did not delete: a kept `PersistentVolumeClaim`, a resource left behind, or a resource that was already absent.

When every such resource is gone, the command SHALL delete the `ModuleInstance` record and report as it does without `--wait`. When the timeout passes first, the command SHALL list each resource that is still terminating, with its finalizers when it has any, SHALL NOT delete the `ModuleInstance` record, SHALL NOT print a success line, and SHALL exit 1.

With `--dry-run`, and in a run in which a tracked resource failed, `--wait` SHALL have no effect. For a controller-managed instance `--wait` SHALL have no effect: the command waits for the `ModuleInstance` to be gone, bounded by `--timeout`, with or without it.

Without `--wait`, the output and the exit code of the command SHALL be the same as before this requirement.

#### Scenario: Deleted resources go away within the timeout

- **WHEN** running `opm instance delete --wait` for a CLI-owned instance that tracks a Deployment
- **AND** the Deployment still exists with a `deletionTimestamp` after its delete is accepted, and is gone some seconds later
- **THEN** the command SHALL return only after a read of the Deployment returns NotFound
- **AND** the `ModuleInstance` record SHALL be deleted and the command SHALL exit 0

#### Scenario: Resource held by a finalizer times out

- **WHEN** running `opm instance delete --wait --timeout 30s` for a CLI-owned instance that tracks a ConfigMap carrying a finalizer that no controller removes
- **THEN** after 30 seconds the command SHALL list the ConfigMap as terminating and name its finalizer
- **AND** a resource that no read answered for before the timeout SHALL be listed with that as its error, not as gone
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the command SHALL exit 1

#### Scenario: Name taken by a new object

- **WHEN** a deleted ConfigMap is gone and another object with the same name and a different UID exists
- **THEN** the wait SHALL count the ConfigMap as gone

#### Scenario: Kept claim is not waited for

- **WHEN** running `opm instance delete --wait` without `--delete-data` for an instance that tracks a PersistentVolumeClaim and a ConfigMap
- **THEN** the command SHALL wait for the ConfigMap only
- **AND** it SHALL exit 0 while the claim still exists

#### Scenario: Resource left behind is not waited for

- **WHEN** running `opm instance delete --wait` for an instance whose record lists a ConfigMap that now belongs to another instance
- **THEN** that ConfigMap SHALL be listed as `left behind` and SHALL NOT be waited for

#### Scenario: Re-run after a timeout

- **WHEN** a run with `--wait` timed out, the cause was removed, and the same command runs again
- **THEN** the command SHALL find the instance, wait until the resources are gone, delete the `ModuleInstance` record and exit 0

#### Scenario: Dry run does not wait

- **WHEN** running `opm instance delete --wait --dry-run`
- **THEN** no delete request SHALL be sent and the command SHALL NOT wait

#### Scenario: Controller-managed instance

- **WHEN** running `opm instance delete --wait` for a controller-managed instance
- **THEN** the command SHALL behave as without `--wait`: it deletes the `ModuleInstance` and waits, bounded by `--timeout`, for it to be gone

#### Scenario: Without the flag nothing changes

- **WHEN** running `opm instance delete` without `--wait` for a CLI-owned instance whose resources are still terminating after their deletes are accepted
- **THEN** the command SHALL NOT read the resources again
- **AND** the `ModuleInstance` record SHALL be deleted and the command SHALL exit 0

## ADDED Requirements

### Requirement: Instance delete of an instance that deploys the controller is guarded

`opm instance delete` SHALL NOT delete an instance that deploys the controller while that instance is controller-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer.

An instance deploys the controller when any one of these holds, read from its record without rendering: it is named `opm-controller` in the namespace `opm-system`; its module path, with any `@<major>` suffix removed, is `opmodel.dev/modules/opm_controller`; or its recorded inventory holds a `CustomResourceDefinition` (group `apiextensions.k8s.io`) whose name, after its first `.`, is exactly `opmodel.dev`. These are the same signals by which the controller recognises the instance that deploys it, so the CLI and the controller agree on which instance that is.

The guard SHALL run after the target instance is resolved and before any object is deleted, on the CLI-owned and the controller-owned branch alike, and on a dry run with the same outcome as the real run.

- When the target is controller-owned, the command SHALL refuse with exit code 2 and delete and patch nothing, naming the signal that matched and the remedy of setting `spec.owner` to `cli`. The controller applies and prunes nothing for the instance that deploys it; when that instance is deleted, the controller at most removes its own cleanup finalizer, without pruning. The controller-owned delete would therefore remove the instance without the cleanup the user expects.
- Otherwise the command SHALL list `ModuleInstance` resources cluster-wide. When any carries `opmodel.dev/cleanup`, the command SHALL refuse with exit code 2, delete and patch nothing, and name each such instance as `namespace/name`, marking the target itself when it is one of them, and name `opm controller uninstall` with its `--remove-finalizers` choice and that choice's consequence (the named instances' workloads are orphaned). Deleting the controller while it still owes cleanup leaves every armed instance unable to finish deletion until a controller runs again. When the cluster-wide list fails, the command SHALL fail closed with the exit code `opm` maps that Kubernetes error to (4 for a permission or authentication denial) and delete nothing.

`opm instance delete` SHALL offer no flag that removes finalizers and SHALL NOT write `spec.owner`. `--yes` SHALL keep its one meaning, skipping the confirmation prompt, and SHALL NOT bypass either refusal; neither SHALL `--force`, its deprecated alias on this command (capability `flag-conventions`). Every other instance's delete SHALL be unchanged by this requirement.

#### Scenario: Refused while an instance waits on the controller's cleanup

- **WHEN** `opm instance delete opm-controller -n opm-system --yes` runs for a CLI-owned record while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object, and the record and every object its inventory lists SHALL still exist
- **AND** the error SHALL name `default/hello` and `opm controller uninstall --remove-finalizers` and state that this choice orphans its workloads

#### Scenario: Controller-owned instance of the controller is refused

- **WHEN** the record `opm-controller` in `opm-system` is controller-owned, with `spec.prune: true`, and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **AND** `opm instance delete opm-controller -n opm-system --yes` runs
- **THEN** the command SHALL exit 2 and the `ModuleInstance` `opm-controller` SHALL still exist
- **AND** the error SHALL name the matched signal and say to set `spec.owner` to `cli`, and SHALL NOT report that any resource was pruned

#### Scenario: Dry run reports the same refusal

- **WHEN** `opm instance delete opm-controller -n opm-system --dry-run` runs for a CLI-owned record while an instance carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 with the same refusal as the real run, and nothing SHALL be deleted

#### Scenario: The target itself carries the finalizer

- **WHEN** `opm instance delete opm-controller -n opm-system --yes` runs for a CLI-owned record while the `ModuleInstance` `opm-controller` itself carries `opmodel.dev/cleanup` and no other instance does
- **THEN** the command SHALL exit 2, naming `opm-system/opm-controller` as the instance being deleted

#### Scenario: The controller module under another name is guarded

- **WHEN** `opm instance delete ops -n platform --yes` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_controller@v0` while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: A record holding the controller's CRDs is guarded

- **WHEN** `opm instance delete crds -n platform --yes` runs for a CLI-owned record of another name and module path whose inventory holds the `CustomResourceDefinition` `moduleinstances.opmodel.dev`, while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: Look-alike names are not the controller

- **WHEN** `opm instance delete dash -n default --yes` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_controller_dashboard@v0` and whose inventory holds only the `CustomResourceDefinition` `widgets.example.opmodel.dev.io`, while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

#### Scenario: No armed instance, no refusal

- **WHEN** `opm instance delete opm-controller -n opm-system --yes` runs for a CLI-owned record and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **THEN** the guard SHALL pass and the delete SHALL proceed as for any CLI-owned instance

#### Scenario: List failure fails closed

- **WHEN** the user may not list `moduleinstances` cluster-wide and runs `opm instance delete opm-controller -n opm-system --force` for a CLI-owned record
- **THEN** the command SHALL exit 4 without deleting anything

#### Scenario: Other instances are not guarded

- **WHEN** `opm instance delete hello -n default --yes` runs for a CLI-owned instance of another module while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

#### Scenario: The deprecated alias does not bypass the guard

- **WHEN** `opm instance delete opm-controller -n opm-system --force` runs for a CLI-owned record while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object, exactly as with `--yes`

### Requirement: Instance delete keeps the ModuleInstance record when a tracked resource is unreadable

When `opm instance delete` deletes a CLI-owned instance and a tracked resource could not be read during discovery (the read failed with an error other than NotFound), that resource SHALL count as a failure for that resource, with the same outcome as a failed re-read: it SHALL be listed with its read error, the other tracked resources SHALL still be processed, the `ModuleInstance` record SHALL NOT be deleted, and the command SHALL exit 1. On a real run the output SHALL say that the `ModuleInstance` was kept and that re-running is safe. A dry run SHALL count the resource among those it could not check. A tracked `Namespace` in the core group or `CustomResourceDefinition` in `apiextensions.k8s.io` that could not be read SHALL instead be listed as `left behind`, as it would be if read, and SHALL NOT count as a failure. The controller-owned delete path SHALL NOT be affected.

#### Scenario: Discovery read error keeps the ModuleInstance

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a Deployment and a ConfigMap
- **AND** reading the ConfigMap during discovery fails with Forbidden
- **THEN** the Deployment SHALL be deleted
- **AND** the ConfigMap SHALL be listed with the Forbidden error
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the output SHALL say that the ModuleInstance was kept and that re-running is safe
- **AND** the command SHALL exit 1

#### Scenario: Dry run counts an unreadable resource as not checked

- **WHEN** running `opm instance delete --dry-run` and a tracked resource cannot be read during discovery
- **THEN** nothing SHALL be deleted
- **AND** the command SHALL report that 1 resource could not be checked
- **AND** SHALL exit 1

#### Scenario: Unreadable Namespace is left behind, not a failure

- **WHEN** running `opm instance delete` and reading a tracked Namespace during discovery fails with Forbidden
- **AND** every other tracked resource is read and deleted
- **THEN** the Namespace SHALL be listed as `left behind`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: Controller-owned delete ignores discovery read errors

- **WHEN** running `opm instance delete` for a controller-owned instance and a tracked resource cannot be read
- **THEN** the command SHALL delete the `ModuleInstance` and wait for the controller as before
