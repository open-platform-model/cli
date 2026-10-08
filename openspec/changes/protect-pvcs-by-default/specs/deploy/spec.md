## ADDED Requirements

### Requirement: Instance delete keeps PersistentVolumeClaims unless delete-data is set

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. Each kept claim SHALL be listed on its own line, naming its kind, namespace and name with the status `kept`, in the dry run and in the real run, at the informational log level and never as a warning or an error. A kept claim SHALL NOT count as a failure: the other tracked resources SHALL be deleted, the `ModuleInstance` record SHALL be deleted last, and the command SHALL exit 0. After the delete the claim is in no inventory, so no later `opm` command deletes it. The closing output SHALL state how many claims were kept, SHALL print for each kept claim the `kubectl delete pvc` command that deletes it, and SHALL name `--delete-data`. With `--delete-data`, a tracked claim SHALL be deleted under the same live-ownership check as every other tracked resource. The kept-claim rule SHALL apply to the core group only: a kind of the same name in another API group is deleted like any other resource. The command SHALL NOT handle claims that a StatefulSet created from `volumeClaimTemplates`: they are not in the inventory, with or without the flag.

On an operator-managed instance `--delete-data` SHALL have no effect, and the command SHALL print a warning that says so before it asks for confirmation. The kept-claim rule SHALL NOT be claimed for an operator-managed instance: the operator decides what it removes.

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
- **THEN** the command SHALL warn that `--delete-data` has no effect on an operator-managed instance
- **AND** the delete SHALL go on as without the flag

### Requirement: Instance delete confirmation names the claims that delete-data deletes

`opm instance delete` SHALL read the instance record before it asks for confirmation. When `--delete-data` is set, the instance is CLI-owned and its inventory tracks at least one live PersistentVolumeClaim, the confirmation prompt SHALL name every such claim by namespace and name and SHALL say that the data on them is deleted. Without `--delete-data` the prompt for a CLI-owned instance SHALL say that PersistentVolumeClaims are kept. For an operator-managed instance the prompt SHALL NOT say that PersistentVolumeClaims are kept, with or without `--delete-data`: it SHALL say that the instance is operator-managed and what `spec.prune` makes the operator do, and when `spec.prune` is set it SHALL say that the operator deletes the tracked resources, PersistentVolumeClaims and their data included. An instance that has no record SHALL be reported as not found before any prompt.

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
- **THEN** the prompt SHALL say that the instance is operator-managed and that the operator deletes its tracked resources, PersistentVolumeClaims and their data included
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are kept

#### Scenario: Delete-data note comes before the question

- **WHEN** running `opm instance delete jellyfin -n media --delete-data` without `--yes` for an operator-managed instance
- **THEN** the warning that `--delete-data` has no effect SHALL be printed before the confirmation question

#### Scenario: Missing instance is reported before the prompt

- **WHEN** running `opm instance delete nosuch -n media` without `--yes`
- **THEN** the command SHALL exit 5 without printing a confirmation prompt
