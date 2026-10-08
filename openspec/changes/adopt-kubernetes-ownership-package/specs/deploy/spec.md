## MODIFIED Requirements

### Requirement: Instance delete re-checks live ownership before each delete

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again and SHALL take the decision to delete it from the delete verdict the CLI shares with the operator and with its own prune. It SHALL delete the resource only when the live `app.kubernetes.io/managed-by` label carries an OPM value, its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID, and its `opmodel.dev/adopt` annotation does not name another instance. An object with no UUID label SHALL be judged without the UUID comparison, and so SHALL every object when the instance has no recorded UUID. A resource that fails the check SHALL be left behind and listed with the reason, and SHALL NOT count as a failure. Each delete SHALL carry a precondition on the UID of the object that was read; a delete the API server refuses on that precondition SHALL count as a failure for that resource and SHALL NOT be reported as deleted. A resource that is already gone, whether its re-read or its delete call returns NotFound, SHALL count neither as deleted nor as a failure. Any other read error SHALL count as a failure for that resource, so the `ModuleInstance` record is kept and a re-run retries. Source: 0012:D4:R1, 0012:D8:R8.

#### Scenario: Resource no longer managed by OPM is left behind

- **WHEN** a tracked resource's live `app.kubernetes.io/managed-by` label is missing or not an OPM value
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that OPM does not manage it

#### Scenario: Resource owned by another instance is left behind

- **WHEN** a tracked resource's live `module-instance.opmodel.dev/uuid` label differs from the instance's recorded UUID
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that it belongs to another instance

#### Scenario: Resource without a UUID label is deleted on managed-by alone

- **WHEN** a tracked resource carries an OPM managed-by value and no `module-instance.opmodel.dev/uuid` label
- **THEN** the resource SHALL be deleted

#### Scenario: Resource gone before its delete

- **WHEN** a tracked resource no longer exists when it is read again
- **THEN** the command SHALL NOT report an error for it
- **AND** it SHALL NOT be counted as deleted

#### Scenario: Resource gone between its re-read and its delete

- **WHEN** a tracked resource passes the ownership check and its delete call returns NotFound
- **THEN** the command SHALL NOT report an error for it
- **AND** it SHALL NOT be counted as deleted

#### Scenario: Instance without a recorded UUID deletes on managed-by alone

- **WHEN** the instance has no recorded UUID
- **AND** a tracked resource carries an OPM managed-by value and any `module-instance.opmodel.dev/uuid` label
- **THEN** the resource SHALL be deleted

#### Scenario: Read error keeps the ModuleInstance

- **WHEN** re-reading a tracked resource fails with an error other than NotFound, such as Forbidden
- **THEN** the resource SHALL be reported as failed
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the command SHALL exit non-zero

#### Scenario: Resource adopted by another instance is left behind

- **WHEN** a tracked resource carries an OPM managed-by value and the instance's UUID
- **AND** its live `opmodel.dev/adopt` annotation names another instance
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that another instance is adopting it

#### Scenario: Delete carries a UID precondition

- **WHEN** a tracked resource passes the ownership check
- **THEN** the delete request SHALL carry a precondition on the UID of the object that was read

#### Scenario: Object replaced between its re-read and its delete

- **WHEN** the API server refuses a delete because the UID precondition does not match
- **THEN** the resource SHALL be reported as failed and SHALL NOT be counted as deleted
- **AND** the `ModuleInstance` record SHALL NOT be deleted
