## MODIFIED Requirements

### Requirement: Every check that can refuse install runs before its first write

Install SHALL run, before it writes any object, every check that can refuse it: resolving and rendering the module, the content check of the default module version, the check of the values against the target version's `#config`, the version rules of install, the check of the rendered ModuleInstance CRD against the CLI's floor, the status-subresource permission check, the proof of the migration of an operator installed from an earlier release manifest (capability `operator-migration`), and then the apply guard over every object the render names, which admits the objects that proof admits. Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied. Only after every check passed SHALL install write, in this order: the CRD step, then the migration's writes (the field-ownership moves, then the delete of the earlier Deployment, then the deletes of the superseded role bindings), then the instance apply, then the Platform seed. On a cluster with nothing to migrate, the migration writes nothing. An install that refuses SHALL change no object in the cluster.

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
