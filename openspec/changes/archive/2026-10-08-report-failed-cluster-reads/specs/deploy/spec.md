## ADDED Requirements

### Requirement: Instance delete fails when the ModuleInstance record cannot be deleted

When `opm instance delete` deletes a CLI-owned instance, every tracked resource was deleted, and the delete of the `ModuleInstance` record then fails with an error other than NotFound, the command SHALL NOT print a success line and SHALL exit non-zero. The output SHALL name the record and the error, SHALL say that the tracked resources were deleted and that the record remains, and SHALL say that re-running is safe. The exit code SHALL be 4 when the API server denied the delete (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer SHALL count as deleted. Re-running the command after the cause is fixed SHALL delete the record and exit 0. A dry run deletes no record and SHALL NOT be affected.

#### Scenario: Forbidden record delete

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a ConfigMap
- **AND** the delete of the `ModuleInstance` record fails with Forbidden
- **THEN** the ConfigMap SHALL be deleted
- **AND** the output SHALL NOT contain `Instance deleted`
- **AND** the output SHALL say that the record remains and that re-running is safe
- **AND** the command SHALL exit 4

#### Scenario: Record delete fails with a server error

- **WHEN** the delete of the `ModuleInstance` record fails with an internal server error
- **THEN** the command SHALL exit 1 without a success line

#### Scenario: Re-run after the cause is fixed

- **WHEN** `opm instance delete` runs again after an earlier run deleted the resources but failed to delete the record
- **THEN** it SHALL delete the record and exit 0
