## MODIFIED Requirements

### Requirement: Uninstall deletes the operator instance's recorded inventory

`opm operator uninstall` SHALL read the operator's instance record `opm-operator` in `opm-operator-system`. When none exists, it SHALL delete nothing and exit 2 naming `opm operator install` as the step that records the running operator. Otherwise, after the finalizer guard, it SHALL delete every object the record's inventory lists except CRDs and the Namespace, in descending resource-weight order, leaving behind any object that the delete verdict of `opm instance delete` leaves behind (capability `deploy`, "Instance delete re-checks live ownership before each delete"): one that OPM no longer manages, one that carries another instance's identity, or one whose `opmodel.dev/adopt` annotation names another instance, then, when no object failed, delete the record. It SHALL delete no object the inventory does not list, SHALL NOT wait for deletion to complete, and SHALL treat an already absent object as deleted. Source: 0006:D34, 0021:D11:R11.

#### Scenario: Uninstall after a module install

- **WHEN** `opm operator uninstall` runs on a cluster with no armed ModuleInstance
- **THEN** every recorded object except the four CRDs and the Namespace is deleted, then the record, and the CRDs and the Namespace still exist

#### Scenario: Object an older release installed is removed

- **WHEN** the record lists a ClusterRole the CLI's pinned module no longer renders
- **THEN** uninstall deletes it

#### Scenario: No record

- **WHEN** `opm operator uninstall` runs on a cluster whose operator was applied with kubectl and has no instance record
- **THEN** it exits 2 naming `opm operator install`, and no object is deleted

#### Scenario: Re-running uninstall

- **WHEN** uninstall runs again after an earlier run deleted the objects but failed to delete the record
- **THEN** it treats the absent objects as deleted, deletes the record and exits zero

#### Scenario: Recorded object adopted by another instance is left behind

- **WHEN** a recorded ClusterRole carries `opmodel.dev/adopt` with the UUID of another instance
- **THEN** uninstall SHALL NOT delete it, SHALL list it as `left behind`, and SHALL count it in the closing line
- **AND** the record SHALL still be deleted when no object failed
