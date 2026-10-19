## REMOVED Requirements

### Requirement: A kind the cluster does not serve is an error that names it
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version", "`opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone".

### Requirement: A failed discovery request is an error
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A failed discovery request is not read as an unserved kind or a missing object and the API error stays in the error chain", "An apply stops at a failed discovery request and exits with code 4, 3 or 1", "`opm instance delete` exits with the same codes and any other command that reads recorded objects reports the failure through the same path as a failed read".

## ADDED Requirements

### Requirement: A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version

When discovery answers and the group and version, or the kind in it, is not served, a command that changes or deletes the object, or that reads a recorded object, SHALL treat the object as unreadable with an error that names the kind and its API version. The error SHALL NOT be a NotFound error.

#### Scenario: Apply of a kind the cluster does not serve

- **WHEN** an apply sends an object whose kind the cluster does not serve and no CustomResourceDefinition of the same apply defines
- **THEN** the apply SHALL report an error for that object that names the kind and the API version

#### Scenario: An existence check before an apply

- **WHEN** the first-install check or `opm instance diff` reads a rendered object whose kind the cluster does not serve
- **THEN** the object SHALL count as not existing in the cluster

### Requirement: `opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone

`opm instance delete` and the stale-resource prune SHALL NOT count such an object as already gone: the delete SHALL report a failure for that resource and keep the `ModuleInstance` record, and the prune SHALL keep the entry in the record it writes.
(Requirement: A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version)

#### Scenario: Delete with a recorded kind the cluster does not serve

- **WHEN** `opm instance delete` runs and the record lists an object whose kind the cluster does not serve at the recorded version
- **THEN** the command SHALL report a failure for that object, with a message that names the kind and the API version
- **AND** it SHALL NOT delete the `ModuleInstance` record
- **AND** it SHALL exit with code 1
- **AND** it SHALL print a hint that names the ways out: install the definition of the kind again, or apply the instance again when only the API version was removed

#### Scenario: Prune with a stale kind the cluster does not serve

- **WHEN** an apply prunes a stale entry whose kind the cluster does not serve at the recorded version
- **THEN** the prune SHALL report a failure for that entry, with a message that names the kind and the API version
- **AND** the record written after the prune SHALL still list the entry

### Requirement: A failed discovery request is not read as an unserved kind or a missing object and the API error stays in the error chain

When a discovery request fails (the API server denies it, is unavailable, or the request fails in any other way), the cli SHALL NOT read the failure as "the kind is not served" or as "the object does not exist". The API error SHALL stay in the error chain.

#### Scenario: The error chain carries the ServiceUnavailable error

- **WHEN** the first-install check runs and the discovery request for a rendered object answers ServiceUnavailable
- **THEN** the apply SHALL stop before it applies anything
- **AND** the error chain SHALL carry the ServiceUnavailable error

### Requirement: An apply stops at a failed discovery request and exits with code 4, 3 or 1

An apply SHALL stop at a failed discovery request, in its first-install check, in its apply of objects and in its prune, and SHALL exit with code 4 when access is denied, 3 when the server is unavailable and 1 otherwise.

#### Scenario: Discovery is unavailable during the first-install check

- **WHEN** the first-install check runs and the discovery request for a rendered object answers ServiceUnavailable
- **THEN** the apply SHALL stop before it applies anything
- **AND** the error chain SHALL carry the ServiceUnavailable error

#### Scenario: Discovery fails during the apply of objects

- **WHEN** an apply sends its objects and the discovery request for one of them answers ServiceUnavailable
- **THEN** the apply SHALL send no further object, SHALL NOT prune and SHALL NOT write the record
- **AND** it SHALL exit with code 3
- **AND** a dry run SHALL stop in the same way

#### Scenario: Discovery fails during prune

- **WHEN** an apply prunes stale entries and the discovery request for the group and version of one of them answers Forbidden
- **THEN** the prune SHALL delete nothing more
- **AND** the record written after it SHALL still list that entry and every stale entry not yet pruned
- **AND** the command SHALL exit with code 4

### Requirement: `opm instance delete` exits with the same codes and any other command that reads recorded objects reports the failure through the same path as a failed read

`opm instance delete` SHALL exit with the same codes when a discovery request for a recorded object failed. Any other command that reads recorded objects SHALL report the failure through the same path as a failed read of the object.
(Requirements: An apply stops at a failed discovery request and exits with code 4, 3 or 1; A failed discovery request is not read as an unserved kind or a missing object and the API error stays in the error chain)

#### Scenario: Discovery is forbidden during delete

- **WHEN** `opm instance delete` runs and the discovery request for the group and version of a recorded object answers Forbidden
- **THEN** the command SHALL report a failure for that object, with the Forbidden error in its message, and keep the `ModuleInstance` record
- **AND** it SHALL exit with code 4 (3 when the discovery request timed out or the server is unavailable, 1 for any other discovery failure)

#### Scenario: Discovery is forbidden during diff

- **WHEN** `opm instance diff` runs and the discovery request for the group and version of a rendered object answers Forbidden
- **THEN** the command SHALL report that object as a read failure, not as added
- **AND** it SHALL exit with code 4
