## ADDED Requirements

### Requirement: Instance delete waits for the deleted resources when wait is set

`opm instance delete` SHALL accept `--wait`. With `--wait`, on a real run of a CLI-owned instance in which no tracked resource failed, the command SHALL, after every planned action is done and before it deletes the `ModuleInstance` record, poll each resource whose delete the API server accepted until the resource is gone or `--timeout` (default `5m0s`) has passed since the wait started. A resource is gone when its read returns NotFound, or when the live object under its name has a UID other than the one of the object that was deleted. A read that fails with another error SHALL NOT count as gone.

The command SHALL NOT wait for a resource it did not delete: a kept `PersistentVolumeClaim`, a resource left behind, or a resource that was already absent.

When every such resource is gone, the command SHALL delete the `ModuleInstance` record and report as it does without `--wait`. When the timeout passes first, the command SHALL list each resource that is still terminating, with its finalizers when it has any, SHALL NOT delete the `ModuleInstance` record, SHALL NOT print a success line, and SHALL exit 1.

With `--dry-run`, and in a run in which a tracked resource failed, `--wait` SHALL have no effect. For an operator-managed instance `--wait` SHALL have no effect: the command waits for the `ModuleInstance` to be gone, bounded by `--timeout`, with or without it.

Without `--wait`, the output and the exit code of the command SHALL be the same as before this requirement.

#### Scenario: Deleted resources go away within the timeout

- **WHEN** running `opm instance delete --wait` for a CLI-owned instance that tracks a Deployment
- **AND** the Deployment still exists with a `deletionTimestamp` after its delete is accepted, and is gone some seconds later
- **THEN** the command SHALL return only after a read of the Deployment returns NotFound
- **AND** the `ModuleInstance` record SHALL be deleted and the command SHALL exit 0

#### Scenario: Resource held by a finalizer times out

- **WHEN** running `opm instance delete --wait --timeout 30s` for a CLI-owned instance that tracks a ConfigMap carrying a finalizer that no controller removes
- **THEN** after 30 seconds the command SHALL list the ConfigMap as terminating and name its finalizer
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

#### Scenario: Operator-managed instance

- **WHEN** running `opm instance delete --wait` for an operator-managed instance
- **THEN** the command SHALL behave as without `--wait`: it deletes the `ModuleInstance` and waits, bounded by `--timeout`, for it to be gone

#### Scenario: Without the flag nothing changes

- **WHEN** running `opm instance delete` without `--wait` for a CLI-owned instance whose resources are still terminating after their deletes are accepted
- **THEN** the command SHALL NOT read the resources again
- **AND** the `ModuleInstance` record SHALL be deleted and the command SHALL exit 0

## MODIFIED Requirements

### Requirement: Instance delete reports a resource as deleted when its delete is accepted

`opm instance delete` of a CLI-owned instance SHALL report a tracked resource as deleted when the API server accepts its delete request. Each delete uses Foreground propagation, so a deleted resource can still exist, with a `deletionTimestamp`, when the command exits: without `--wait` the command returns once every planned action is done. Without `--wait`, a resource whose delete was accepted SHALL NOT count as a failure because it still exists, also when a finalizer keeps it; with no other failure the `ModuleInstance` record is deleted and the exit code is 0. This requirement describes when a resource counts as deleted. With `--wait` the command also waits, bounded, for the deleted resources to disappear ("Instance delete waits for the deleted resources when wait is set").

#### Scenario: Resource held by a finalizer

- **WHEN** running `opm instance delete` without `--wait` for a CLI-owned instance that tracks a ConfigMap carrying a finalizer that no controller removes
- **THEN** the ConfigMap SHALL be listed as `deleted`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: Accepted delete of a resource with dependents

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a Deployment with running Pods
- **AND** the API server accepts every delete request
- **THEN** the Deployment SHALL be listed as `deleted` and the command SHALL exit 0
