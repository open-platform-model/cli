## REMOVED Requirements

### Requirement: Every check that can refuse install runs before its first write
**Reason**: It ran the migration proof of the retired `operator-migration` capability before the apply guard, let the guard admit what the proof admitted, and ordered the migration's writes (0012:D8:R6 and 0012:D8:R7, withdrawn). Restated without them as "Install runs every refusing check before its first write". Two scenarios go with it: "Migration proof runs before the apply guard" and "A read the migration proof fails exits by the read error".
**Migration**: An operator installed from an earlier release manifest is no longer taken over by install. Annotate each object the refusal names with `opmodel.dev/adopt=<instance UUID>`, or delete the earlier install, then run `opm operator install` again.

### Requirement: Another module version is installed from the registry as served
**Reason**: Its old-style operator tag message existed only for users of the release-manifest install. Restated without it as "Install resolves another module version as the registry serves it"; the scenario "Old-style operator tag" goes with it.
**Migration**: None. A `--version` that names an opm-operator release tag is refused as any other selector the registry cannot satisfy.

### Requirement: The running-operator check locates the operator by its fixed names
**Reason**: It promised to find an operator applied from a release manifest, which nobody runs. Restated without that case as "The running-operator check finds the operator by its fixed names alone"; the scenario "Manifest-installed operator is found by its fixed names" goes with it. The check itself does not change.
**Migration**: None.

## ADDED Requirements

### Requirement: Install runs every refusing check before its first write

Install SHALL run, before it writes any object, every check that can refuse it: resolving the module and reading the operator version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the operator version against the CLI's, the rendered image against the module's stated operator version, and the rendered ModuleInstance CRD against the CLI's floor), the status-subresource permission check, and the apply guard over every object the render names (capability `apply-pruning`, "Ownership guard judges every apply and dry run"). Install SHALL pass the guard no admission set and SHALL run no proof of where an existing object came from: an object that an earlier opm-operator release manifest created is judged as any other existing object. The guard SHALL run on every install, with or without a record of the operator's instance. An object the record lists is judged as recorded and SHALL pass whatever its UUID label says. An existing object the record does not list SHALL pass when its `opmodel.dev/adopt` annotation holds the UUID of the operator's instance, or when it carries an OPM managed-by label and no UUID label of another instance; it SHALL refuse the install when OPM does not manage it or when it carries another instance's identity. Install needs every object its render names, so, unlike any other apply, it SHALL also refuse when the `opmodel.dev/adopt` annotation of an object names another instance, whether or not the record lists the object: it SHALL NOT leave the object unapplied and go on (owner decision of 2026-10-08 on `opm operator install`). Every refusal of the guard in this phase SHALL exit 2, SHALL name the object and the instance that owns or adopts it, SHALL name the `opmodel.dev/adopt` annotation and the UUID to set where the annotation lifts the refusal, and SHALL leave the cluster unchanged; that includes a failed read by the guard, with or without a record. No refusal of install SHALL tell the user to delete or rename an object. Source: 0012:D8:R1, 0012:D8:R2, 0012:D8:R3, 0012:D8:R8. Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied. Only after every check passed SHALL install write, in this order: the CRD step, then the instance apply, then the Platform seed. Outside the instance apply and its prune, install SHALL send no delete and no patch of managed fields. An install that refuses SHALL change no object in the cluster. Two of these checks read every object install applies, in this order: the wait for terminating objects and the apply guard. A read that fails with an error other than NotFound SHALL refuse the install in the check that made it. When the wait for terminating objects fails a read, the command SHALL exit with the code of the read error: 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. When the apply guard fails a read, the command SHALL exit 2, the code of the guard's other refusals. An object that install cannot read from the start is therefore refused by the wait for terminating objects, with exit code 4 when the read is denied.

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm operator uninstall` has returned and `opm operator install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies only after it is gone, and completes

#### Scenario: Terminating object outlives the budget

- **WHEN** an object of the render stays terminating for longer than `--timeout`
- **THEN** the command exits non-zero naming that object and the elapsed timeout, and nothing has been applied

#### Scenario: An object unreadable from the start exits by the read error

- **WHEN** install runs on a cluster with no record of the operator's instance and every read of a ClusterRole the render names fails with Forbidden
- **THEN** the command SHALL exit 4 naming that ClusterRole and the read error, and no object in the cluster has changed

#### Scenario: A read the apply guard fails exits with the guard code

- **WHEN** the earlier read of a rendered object was answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed

#### Scenario: Reinstall refuses a foreign object new to the inventory

- **WHEN** install runs on a cluster with a record of the operator's instance
- **AND** the render names a ClusterRole the record does not list, which exists without an OPM managed-by label
- **THEN** the command SHALL exit 2 naming that ClusterRole and the adopt annotation, and no object in the cluster has changed

#### Scenario: First install refuses an object of another instance

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** a CustomResourceDefinition the render names exists with an OPM managed-by label and the UUID of another instance
- **THEN** the command SHALL exit 2 naming that CustomResourceDefinition and the other instance, and no object in the cluster has changed

#### Scenario: Install refuses an object adopted by another instance

- **WHEN** the controller Deployment, a rendered CustomResourceDefinition or the operator Namespace carries `opmodel.dev/adopt` with the UUID of another instance
- **THEN** the command SHALL exit 2 naming that object and that instance
- **AND** no object in the cluster has changed, the managed fields of that object included

#### Scenario: Reinstall with a failed guard read exits with the guard code

- **WHEN** install runs on a cluster with a record of the operator's instance, the earlier read of a rendered object was answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed

#### Scenario: Reinstall over its own recorded objects passes the guard

- **WHEN** install runs on a cluster with a record of the operator's instance and every existing object the render names is in that record
- **THEN** the apply guard refuses none of them

#### Scenario: Recorded objects that carry another identity are applied

- **WHEN** install runs on a cluster with a record of the operator's instance that lists every object the render names
- **AND** every one of those objects, the four CRDs and the Namespace included, carries an OPM managed-by label and the UUID of another instance
- **THEN** the apply guard refuses none of them and install applies them

#### Scenario: Annotated objects that carry another identity are applied

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** every object the render names exists with the UUID of another instance and `opmodel.dev/adopt` holding the UUID of the operator's instance
- **THEN** the apply guard refuses none of them, and install applies and records them

#### Scenario: Objects that carry another identity, with no record and no annotation, are refused

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** every object the render names exists with an OPM managed-by label and the UUID of another instance, and none carries an adopt annotation
- **THEN** the command SHALL exit 2 naming each object, the other instance and `opmodel.dev/adopt` with the UUID to set
- **AND** the output SHALL NOT contain `operator migration refused` or `remove or rename`
- **AND** no object in the cluster has changed

#### Scenario: Unlabelled CRDs of an earlier manifest install are refused

- **WHEN** install runs on a cluster whose four `opmodel.dev` CRDs, Namespace and Deployment were applied from an opm-operator release manifest and carry no OPM managed-by label and no adopt annotation
- **THEN** the command SHALL exit 2 with `refusing to install:` and one line for each such object that says it exists and is not managed by OPM, and names `opmodel.dev/adopt` with the UUID to set
- **AND** install SHALL adopt none of them by itself, and no object in the cluster has changed

#### Scenario: CRDs-only refuses unlabelled CRDs

- **WHEN** `opm operator install --crds-only` runs and a CRD the render names exists without an OPM managed-by label and without an adopt annotation
- **THEN** the command SHALL exit 2 naming that CRD, and no CRD has changed

#### Scenario: The instance's own objects without a record pass the guard

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** every existing object the render names carries an OPM managed-by label and the UUID of the operator's instance (a run that stopped before it wrote the record, a CRDs-only install, or a `kubectl apply` of the module's render)
- **THEN** the apply guard refuses none of them

#### Scenario: An unrecorded object with the instance's UUID and no OPM managed-by label is refused

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** an object the render names carries the UUID of the operator's instance, and its managed-by label is absent or names another tool
- **THEN** the command SHALL exit 2 naming that object as not managed by OPM, and `opmodel.dev/adopt` with the UUID to set

### Requirement: Install resolves another module version as the registry serves it

`opm operator install --version <selector>` SHALL resolve the operator module version through the CLI's configured registry, a release version pinning it and a major (`v0`) floating to that major's newest release, and SHALL install it through the same render, checks and steps as the default. Install SHALL report the module version and the operator version it deploys. A selector the registry cannot satisfy, or that is not a version selector, SHALL refuse before any cluster call; a selector that has the shape of an opm-operator release tag SHALL get no message of its own. With the CLI's registry mapping pointed at a mirror holding the module and its dependencies, install SHALL need no other registry. Source: 0021:D11:R1, 0021:D11:R2.

#### Scenario: Selecting another version

- **WHEN** `opm operator install --version 0.2.0` runs and the registry serves that version
- **THEN** install applies it and its output names module `0.2.0` and the operator version it deploys

#### Scenario: Unserved version

- **WHEN** `opm operator install --version 1.0.0-beta.5` runs and no module version matches
- **THEN** the command exits 2 before contacting the cluster

#### Scenario: Mirror only

- **WHEN** the registry mapping routes `opmodel.dev` to a mirror holding the module and its dependencies, and the public registry is unreachable
- **THEN** install succeeds

### Requirement: The running-operator check finds the operator by its fixed names alone

Every CLI command that checks for a running operator before it acts SHALL locate the operator by its fixed names, reading no instance record.

The operator's objects SHALL be its fixed names, which every install of the operator module keeps: the `Deployment` `opm-operator-controller-manager` in the namespace `opm-operator-system`, and the `CustomResourceDefinition`s `moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev` and `transformerregistrations.opmodel.dev`. The `Namespace` is checked only as the `Deployment`'s namespace; the check reads no `Namespace` object and no instance record. An operator that lacks one of these CRDs SHALL be reported as not ready.

The operator SHALL count as running only when every one of these `CustomResourceDefinition`s reports `Established=True` and the `Deployment` has completed its rollout. Otherwise the check SHALL report the operator as not ready, name each object that failed, and point at `opm operator install`. A read of any of these objects that fails SHALL count that object as not ready, so the command proceeds only on a positive finding.

#### Scenario: Module-installed operator is found by the same names

- **WHEN** the cluster holds the `ModuleInstance` `opm-operator` in `opm-operator-system` whose render created the four CRDs and the `Deployment` `opm-operator-controller-manager` in `opm-operator-system`
- **AND** every CRD is `Established` and the `Deployment` has rolled out
- **THEN** the running-operator check SHALL report the operator as running

#### Scenario: Operator older than the fourth CRD is not ready

- **WHEN** the cluster serves `moduleinstances`, `modulepackages` and `platforms` in `opmodel.dev` but not `transformerregistrations.opmodel.dev`, and `opm-operator-controller-manager` has rolled out
- **THEN** the running-operator check SHALL report the operator as not ready and name `transformerregistrations.opmodel.dev`

#### Scenario: No operator at all

- **WHEN** the cluster holds none of the four CRDs and no `opm-operator-controller-manager` Deployment
- **THEN** the running-operator check SHALL report the operator as not ready, naming the missing objects, and point at `opm operator install`
