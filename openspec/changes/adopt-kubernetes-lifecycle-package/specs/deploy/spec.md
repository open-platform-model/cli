## ADDED Requirements

### Requirement: Instance delete follows the shared deletion plan

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator, built from the entries of the instance's `ModuleInstance` record and the instance identity stored in it. It SHALL read a tracked resource, delete it or leave it in place only when the plan names that action as the next one, and it SHALL send each delete with the propagation policy and the precondition the plan names. The plan SHALL NOT hold a `PersistentVolumeClaim` that the command keeps ("Instance delete keeps PersistentVolumeClaims unless delete-data is set"). The command SHALL delete the `ModuleInstance` record only when the plan's release verdict allows it, which is when every planned resource was deleted or left in place and none failed. A dry run SHALL follow the same plan, SHALL send no delete, and SHALL NOT delete the record. The output lines, their wording and the exit codes of the command SHALL stay as the other requirements of this capability state them. The operator-owned delete path SHALL NOT be affected. Source: 0012:D4:R1, 0012:D4:R6.

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

#### Scenario: Dry run follows the plan without deleting

- **WHEN** running `opm instance delete --dry-run`
- **THEN** each tracked resource SHALL be read and judged as on a real run
- **AND** no delete request SHALL be sent
- **AND** the `ModuleInstance` record SHALL still exist

### Requirement: Instance delete reports a resource as deleted when its delete is accepted

`opm instance delete` of a CLI-owned instance SHALL report a tracked resource as deleted when the API server accepts its delete request. Each delete uses Foreground propagation, so a deleted resource can still exist, with a `deletionTimestamp`, when the command exits: the command returns once every planned action is done. A resource whose delete was accepted SHALL NOT count as a failure because it still exists, also when a finalizer keeps it; with no other failure the `ModuleInstance` record is deleted and the exit code is 0. This requirement describes when a resource counts as deleted. It does not rule out a later, bounded wait for the resources to disappear.

#### Scenario: Resource held by a finalizer

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a ConfigMap carrying a finalizer that no controller removes
- **THEN** the ConfigMap SHALL be listed as `deleted`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: Accepted delete of a resource with dependents

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a Deployment with running Pods
- **AND** the API server accepts every delete request
- **THEN** the Deployment SHALL be listed as `deleted` and the command SHALL exit 0
