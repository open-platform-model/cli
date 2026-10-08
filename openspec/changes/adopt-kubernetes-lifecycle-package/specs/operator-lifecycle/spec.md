## ADDED Requirements

### Requirement: Uninstall follows the shared deletion plan

`opm operator uninstall` SHALL take every action on a recorded object from the deletion plan `opm instance delete` uses (capability `deploy`, "Instance delete follows the shared deletion plan"), built from the entries of the operator's instance record and the identity stored in it. It SHALL read a recorded object, delete it or leave it in place only when the plan names that action as the next one, and SHALL delete the instance record only when the plan's release verdict allows it. The finalizer guard and the no-record refusal SHALL run before the plan is built. The output lines and the exit codes of the command SHALL stay as the other requirements of this capability state them. Source: 0012:D4:R1.

#### Scenario: Recorded objects are deleted in the plan's order

- **WHEN** `opm operator uninstall` runs on a cluster with no armed ModuleInstance
- **THEN** the delete request for the controller Deployment SHALL be sent before the ones for its ServiceAccount and ClusterRole
- **AND** each delete request SHALL carry Foreground propagation and a precondition on the UID of the object that was read

#### Scenario: Failed object holds the record

- **WHEN** the delete of the recorded ClusterRole fails with Forbidden
- **THEN** the other recorded objects SHALL still be deleted
- **AND** the instance record SHALL NOT be deleted
- **AND** the command SHALL exit 1 without reporting the operator uninstalled

#### Scenario: Guard refuses before any plan

- **WHEN** a ModuleInstance still carries the operator's cleanup finalizer and `--remove-finalizers` is not set
- **THEN** no recorded object SHALL be read for a delete verdict and none SHALL be deleted
