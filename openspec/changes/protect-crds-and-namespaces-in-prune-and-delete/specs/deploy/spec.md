## ADDED Requirements

### Requirement: Instance delete never deletes CRDs or Namespaces

`opm instance delete` of a CLI-owned instance SHALL NOT delete a `Namespace` in the core API group or a `CustomResourceDefinition` in the `apiextensions.k8s.io` group, even when the instance's inventory tracks it. There SHALL be no flag that overrides this. Each such resource SHALL be listed as left behind, naming its kind, namespace and name with the status `left behind` and the reason, in the dry run and in the real run. A left-behind resource SHALL NOT count as a failure: the remaining tracked resources SHALL be deleted, the `ModuleInstance` record SHALL be deleted last, and the command SHALL exit 0. The closing line SHALL state how many resources were left behind and how to remove them.

#### Scenario: Namespace tracked by the instance is left behind

- **WHEN** running `opm instance delete` for a CLI-owned instance whose inventory tracks a Deployment and the Namespace it runs in
- **THEN** the Deployment SHALL be deleted
- **AND** the Namespace SHALL NOT be deleted
- **AND** the output SHALL list the Namespace as `left behind`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: CRD tracked by the instance is left behind

- **WHEN** running `opm instance delete` for a CLI-owned instance whose inventory tracks a `CustomResourceDefinition`
- **THEN** the CRD SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind`

#### Scenario: Dry run reports left-behind resources

- **WHEN** running `opm instance delete --dry-run` for an instance whose inventory tracks a Namespace
- **THEN** the output SHALL list the Namespace as `left behind`
- **AND** the dry-run summary SHALL count it separately from the resources that would be deleted

### Requirement: Instance delete re-checks live ownership before each delete

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again and SHALL delete it only when its `app.kubernetes.io/managed-by` label carries an OPM value and its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID. As in the operator's prune, an object with no UUID label SHALL be judged on the managed-by label alone, and so SHALL every object when the instance has no recorded UUID. A resource that fails the check SHALL be left behind and listed with the reason, and SHALL NOT count as a failure. A resource that is already gone, whether its re-read or its delete call returns NotFound, SHALL count neither as deleted nor as a failure. Any other read error SHALL count as a failure for that resource, so the `ModuleInstance` record is kept and a re-run retries.

#### Scenario: Resource no longer managed by OPM is left behind

- **WHEN** a tracked resource's live `app.kubernetes.io/managed-by` label is missing or not an OPM value
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that OPM no longer manages it

#### Scenario: Resource owned by another instance is left behind

- **WHEN** a tracked resource's live `module-instance.opmodel.dev/uuid` label differs from the instance's recorded UUID
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that another instance owns it

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
