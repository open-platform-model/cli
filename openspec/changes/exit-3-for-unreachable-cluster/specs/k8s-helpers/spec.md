## MODIFIED Requirements

### Requirement: An apply stops at a failed discovery request and exits with code 4, 3 or 1

An apply SHALL stop at a failed discovery request, in its first-install check, in its apply of objects and in its prune, and SHALL exit with code 4 when access is denied, 3 when the server is unavailable, timed out or sent no response, and 1 otherwise.

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
- **AND** it SHALL exit with code 4 (3 when the discovery request timed out, the server is unavailable or the cluster sent no response, 1 for any other discovery failure)

#### Scenario: Discovery is forbidden during diff

- **WHEN** `opm instance diff` runs and the discovery request for the group and version of a rendered object answers Forbidden
- **THEN** the command SHALL report that object as a read failure, not as added
- **AND** it SHALL exit with code 4
