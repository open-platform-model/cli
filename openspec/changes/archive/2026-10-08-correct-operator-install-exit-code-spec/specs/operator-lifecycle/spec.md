## MODIFIED Requirements

### Requirement: Every check that can refuse install runs before its first write

Install SHALL run, before it writes any object, every check that can refuse it: resolving the module and reading the operator version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the operator version against the CLI's, the rendered image against the module's stated operator version, and the rendered ModuleInstance CRD against the CLI's floor), the status-subresource permission check, the proof of the migration of an operator installed from an earlier release manifest (capability `operator-migration`), and, on a cluster with no record of the operator's instance, the apply guard over every object the render names, which admits the objects that proof admits. With a record, install runs no apply guard: the instance apply takes over any object the render names, including one new to the inventory that already exists and OPM does not manage. Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the migration proof and the apply guard run, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied. Only after every check passed SHALL install write, in this order: the CRD step, then the migration's writes (the field-ownership moves, then the delete of the earlier Deployment, then the deletes of the superseded role bindings), then the instance apply, then the Platform seed. On a cluster with nothing to migrate, the migration writes nothing. An install that refuses SHALL change no object in the cluster. Three of these checks read every object install applies, in this order: the wait for terminating objects, the migration proof and the apply guard. A read that fails with an error other than NotFound SHALL refuse the install in the check that made it. When the wait for terminating objects or the migration proof fails a read, the command SHALL exit with the code of the read error: 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. When the apply guard fails a read, the command SHALL exit 2, the code of the guard's other refusals. An object that install cannot read from the start is therefore refused by the wait for terminating objects, with exit code 4 when the read is denied.

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm operator uninstall` has returned and `opm operator install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies only after it is gone, and completes

#### Scenario: Terminating object outlives the budget

- **WHEN** an object of the render stays terminating for longer than `--timeout`
- **THEN** the command exits non-zero naming that object and the elapsed timeout, and nothing has been applied

#### Scenario: Migration proof runs before the apply guard

- **WHEN** install runs on a cluster whose operator was installed from the v1.0.0-beta.5 release manifest, and every object of that manifest is proven
- **THEN** the apply guard does not refuse `Namespace opm-operator-system` or any other proven object, and install proceeds to the CRD step

#### Scenario: An object unreadable from the start exits by the read error

- **WHEN** install runs on a cluster with no record of the operator's instance and every read of a ClusterRole the render names fails with Forbidden
- **THEN** the command SHALL exit 4 naming that ClusterRole and the read error, and no object in the cluster has changed

#### Scenario: A read the migration proof fails exits by the read error

- **WHEN** the wait for terminating objects read a rendered object and the migration proof's read of the same object fails with Forbidden
- **THEN** the command SHALL exit 4 naming that object and the read error, and no object in the cluster has changed

#### Scenario: A read the apply guard fails exits with the guard code

- **WHEN** the two earlier reads of a rendered object were answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed
