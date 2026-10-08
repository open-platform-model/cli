## MODIFIED Requirements

### Requirement: Instance delete keeps PersistentVolumeClaims unless delete-data is set

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. Each kept claim SHALL be listed on its own line, naming its kind, namespace and name with the status `kept`, in the dry run and in the real run, at the informational log level and never as a warning or an error. A kept claim SHALL NOT count as a failure: the other tracked resources SHALL be deleted, the `ModuleInstance` record SHALL be deleted last, and the command SHALL exit 0. After the delete the claim is in no inventory, so no later `opm` command deletes it. The closing output SHALL state how many claims were kept, SHALL print for each kept claim the `kubectl delete pvc` command that deletes it, and SHALL name `--delete-data`. With `--delete-data`, a tracked claim SHALL be deleted under the same live-ownership check as every other tracked resource. The kept-claim rule SHALL apply to the core group only: a kind of the same name in another API group is deleted like any other resource. The command SHALL NOT handle claims that a StatefulSet created from `volumeClaimTemplates`: they are not in the inventory, with or without the flag.

On an operator-managed instance `--delete-data` SHALL NOT change what the operator does and SHALL NOT fail the command: the command SHALL print a warning, before it asks for confirmation, that says so and names `spec.dataPolicy` of the `ModuleInstance` as the setting that decides whether the operator deletes PersistentVolumeClaims. The kept-claim rule of the CLI SHALL NOT be claimed for an operator-managed instance: the operator decides what it removes, and the command SHALL report what the instance's `spec.prune` and `spec.dataPolicy` make the operator do. When the delete of an operator-managed instance with `spec.prune` set completes and its inventory tracked at least one PersistentVolumeClaim that the data policy keeps, the closing output SHALL NOT say that every tracked resource was pruned: it SHALL name each such claim, SHALL say that an operator released before `spec.dataPolicy` deleted them, and SHALL print the `kubectl delete pvc` command for each.

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

#### Scenario: Operator-managed instance

- **WHEN** running `opm instance delete --delete-data` for an operator-managed instance
- **THEN** the command SHALL warn that `--delete-data` does not change what the operator does and SHALL name `spec.dataPolicy`
- **AND** the delete SHALL go on as without the flag

#### Scenario: Operator-managed delete that keeps claims closes without claiming a full prune

- **WHEN** `opm instance delete --yes` completes for an operator-managed instance whose `spec.prune` is true, whose `spec.dataPolicy` is absent and whose inventory tracks the PersistentVolumeClaim `data` in namespace `apps`
- **THEN** the closing output SHALL NOT say that the operator pruned every tracked resource
- **AND** it SHALL name the claim and carry `kubectl delete pvc data -n apps`
- **AND** it SHALL say that an operator released before `spec.dataPolicy` deleted the claim

#### Scenario: Operator-managed delete with the Delete policy closes as a full prune

- **WHEN** the same delete completes for an instance whose `spec.dataPolicy` is `Delete`
- **THEN** the closing output SHALL say that the operator pruned the tracked resources
- **AND** it SHALL print no `kubectl delete pvc` command

### Requirement: Instance delete confirmation names the claims that delete-data deletes

`opm instance delete` SHALL read the instance record before it asks for confirmation. When `--delete-data` is set, the instance is CLI-owned and its inventory tracks at least one live PersistentVolumeClaim, the confirmation prompt SHALL name every such claim by namespace and name and SHALL say that the data on them is deleted. Without `--delete-data` the prompt for a CLI-owned instance SHALL say that PersistentVolumeClaims are kept. For an operator-managed instance the prompt SHALL be the same with or without `--delete-data`, and SHALL say that the instance is operator-managed and what the operator does, read from the `spec.prune` and `spec.dataPolicy` of the instance. When `spec.prune` is not set, it SHALL say that the operator leaves the tracked resources running. When `spec.prune` is set and `spec.dataPolicy` is `Delete`, it SHALL say that the operator deletes the tracked resources, PersistentVolumeClaims and their data included. When `spec.prune` is set and `spec.dataPolicy` is `Keep`, absent, or any other value, it SHALL say that the operator deletes the tracked resources and keeps PersistentVolumeClaims and their data, and it SHALL add one sentence which says that an operator released before `spec.dataPolicy` deletes the claims whatever the field says and that `opm` cannot tell which operator runs in the cluster. The prompt SHALL show the value of `spec.dataPolicy` as it is, or say that it is not set; a value other than `Keep` and `Delete` SHALL be shown quoted and named as read as `Keep`. The dry run, and the run with `--yes`, SHALL state the same outcome for PersistentVolumeClaims in their output. An instance that has no record SHALL be reported as not found before any prompt.

#### Scenario: Prompt lists the claims

- **WHEN** running `opm instance delete jellyfin -n media --delete-data` without `--yes`
- **AND** the inventory tracks the claims `config` and `media` in namespace `media`
- **THEN** the prompt SHALL name `media/config` and `media/media`
- **AND** the prompt SHALL say that their data is deleted

#### Scenario: Prompt without delete-data

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes`
- **THEN** the prompt SHALL say that PersistentVolumeClaims are kept

#### Scenario: Prompt for an operator-managed instance with prune set

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for an operator-managed instance whose `spec.prune` is true
- **AND** whose `spec.dataPolicy` is `Delete`
- **THEN** the prompt SHALL say that the instance is operator-managed and that the operator deletes its tracked resources, PersistentVolumeClaims and their data included
- **AND** the prompt SHALL name `spec.dataPolicy` and its value `Delete`
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are kept

#### Scenario: Prompt for an operator-managed instance with prune set and no data policy

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for an operator-managed instance whose `spec.prune` is true and whose `spec.dataPolicy` is absent or `Keep`
- **THEN** the prompt SHALL say that the operator deletes its tracked resources and keeps PersistentVolumeClaims and the data on them
- **AND** the prompt SHALL say that `spec.dataPolicy` is not set, or that it is `Keep`
- **AND** the prompt SHALL say that an operator released before `spec.dataPolicy` deletes the claims whatever the field says

#### Scenario: Prompt for an unknown data policy

- **WHEN** the `spec.dataPolicy` of an operator-managed instance with `spec.prune` set is `Retain`
- **THEN** the prompt SHALL show `"Retain"` and say that it is read as `Keep`
- **AND** the prompt SHALL say that the operator keeps PersistentVolumeClaims

#### Scenario: Prompt for an operator-managed instance without prune

- **WHEN** the `spec.prune` of an operator-managed instance is not set and its `spec.dataPolicy` is `Delete`
- **THEN** the prompt SHALL say that the operator leaves its tracked resources running
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are deleted

#### Scenario: Delete-data note comes before the question

- **WHEN** running `opm instance delete jellyfin -n media --delete-data` without `--yes` for an operator-managed instance
- **THEN** the warning that `--delete-data` does not change what the operator does SHALL be printed before the confirmation question

#### Scenario: Missing instance is reported before the prompt

- **WHEN** running `opm instance delete nosuch -n media` without `--yes`
- **THEN** the command SHALL exit 5 without printing a confirmation prompt
