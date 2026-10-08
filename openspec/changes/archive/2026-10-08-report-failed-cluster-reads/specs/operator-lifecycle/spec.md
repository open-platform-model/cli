## ADDED Requirements

### Requirement: Uninstall fails when the instance record cannot be deleted

When `opm operator uninstall` deleted every recorded object and the delete of the operator's instance record then fails with an error other than NotFound, the command SHALL NOT report the operator uninstalled and SHALL exit non-zero. The error SHALL name the record and the cause, and SHALL say that the recorded objects were deleted, that the record remains and that re-running is safe. The exit code SHALL be 4 when the API server denied the delete (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure.

#### Scenario: Forbidden record delete

- **WHEN** `opm operator uninstall` runs and the delete of the instance record fails with Forbidden
- **THEN** the recorded ClusterRole, ServiceAccount and Deployment SHALL be deleted
- **AND** the instance record SHALL still exist
- **AND** the command SHALL exit 4 without reporting the operator uninstalled
