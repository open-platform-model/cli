## REMOVED Requirements

### Requirement: Every check that can refuse install runs before its first write
**Reason**: It ran the migration proof of the retired `operator-migration` capability and ordered the migration's writes. Restated without them as "Install runs every refusing check before its first write".
**Migration**: None.

### Requirement: Another module version is installed from the registry as served
**Reason**: Its old-style operator tag message existed only for users of the release-manifest install. Restated without it as "Install resolves another module version as the registry serves it".
**Migration**: None. A `--version` that names an opm-operator release tag is refused as any other selector the registry cannot satisfy.

### Requirement: The running-operator check locates the operator by its fixed names
**Reason**: It promised to find an operator applied from a release manifest, which nobody runs. Restated without that case as "The running-operator check finds the operator by its fixed names alone".
**Migration**: None.

## ADDED Requirements

### Requirement: Install runs every refusing check before its first write

Install SHALL run, before it writes any object, every check that can refuse it: resolving the module and reading the operator version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the operator version against the CLI's, the rendered image against the module's stated operator version, and the rendered ModuleInstance CRD against the CLI's floor), the status-subresource permission check, and, on a cluster with no record of the operator's instance, the apply guard over every object the render names. The apply guard SHALL admit no object it would refuse for any other instance. With a record, install runs no apply guard: the instance apply takes over any object the render names, including one new to the inventory that already exists and OPM does not manage. Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied. Only after every check passed SHALL install write, in this order: the CRD step, then the instance apply, then the Platform seed. An install that refuses SHALL change no object in the cluster.

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm operator uninstall` has returned and `opm operator install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies only after it is gone, and completes

#### Scenario: Terminating object outlives the budget

- **WHEN** an object of the render stays terminating for longer than `--timeout`
- **THEN** the command exits non-zero naming that object and the elapsed timeout, and nothing has been applied

#### Scenario: An existing Namespace OPM does not manage is refused

- **WHEN** install runs on a cluster with no record of the operator's instance, and `Namespace opm-operator-system` exists without OPM's managed-by label
- **THEN** the apply guard refuses naming that Namespace, and nothing has been applied

### Requirement: Install resolves another module version as the registry serves it

`opm operator install --version <selector>` SHALL resolve the operator module version through the CLI's configured registry, a release version pinning it and a major (`v0`) floating to that major's newest release, and SHALL install it through the same render, checks and steps as the default. Install SHALL report the module version and the operator version it deploys. A selector the registry cannot satisfy, or that is not a version selector, SHALL refuse before any cluster call. With the CLI's registry mapping pointed at a mirror holding the module and its dependencies, install SHALL need no other registry. Source: 0021:D11:R1, 0021:D11:R2.

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

#### Scenario: Module-installed operator is found by its fixed names

- **WHEN** the cluster holds the `ModuleInstance` `opm-operator` in `opm-operator-system` whose render created the four CRDs and the `Deployment` `opm-operator-controller-manager` in `opm-operator-system`
- **AND** every CRD is `Established` and the `Deployment` has rolled out
- **THEN** the running-operator check SHALL report the operator as running

#### Scenario: Operator without the fourth CRD is not ready

- **WHEN** the cluster serves `moduleinstances`, `modulepackages` and `platforms` in `opmodel.dev` but not `transformerregistrations.opmodel.dev`, and `opm-operator-controller-manager` has rolled out
- **THEN** the running-operator check SHALL report the operator as not ready and name `transformerregistrations.opmodel.dev`

#### Scenario: No operator at all

- **WHEN** the cluster holds none of the four CRDs and no `opm-operator-controller-manager` Deployment
- **THEN** the running-operator check SHALL report the operator as not ready, naming the missing objects, and point at `opm operator install`
