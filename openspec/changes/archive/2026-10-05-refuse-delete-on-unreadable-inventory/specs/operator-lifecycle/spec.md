## ADDED Requirements

### Requirement: Uninstall keeps the record when a recorded object could not be read

When `opm operator uninstall` reads the objects its instance record lists and the read of one fails with an error other than NotFound, that object SHALL count as a failure for that object: it SHALL be listed with its read error and not deleted, the other recorded objects SHALL still be deleted, the instance record SHALL NOT be deleted, and the command SHALL exit 1 and SHALL NOT report the operator uninstalled. A recorded `Namespace` or `CustomResourceDefinition` that could not be read SHALL instead be left behind, as it would be if read, and SHALL NOT count as a failure. Re-running uninstall after the cause is fixed SHALL delete the remaining objects and the record.

#### Scenario: Unreadable recorded object keeps the record

- **WHEN** `opm operator uninstall` runs and reading the recorded ClusterRole fails with Forbidden
- **THEN** the recorded ServiceAccount and Deployment SHALL be deleted
- **AND** the ClusterRole SHALL be listed with the Forbidden error and SHALL NOT be deleted
- **AND** the instance record SHALL NOT be deleted
- **AND** the command SHALL exit 1 without reporting the operator uninstalled
