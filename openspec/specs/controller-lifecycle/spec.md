# controller-lifecycle Specification

## Purpose
Controller lifecycle surface: `opm controller install` / `opm controller uninstall`, the pinned controller module the CLI installs as its own instance, readiness waits, uninstall safety, and the CLI's server-side-apply field-manager identity. Implements enhancement 0006 (0006:D5/D23/D32/D33/D34/D35) under the product name set by 0032:D1.

## Requirements

### Requirement: Opt-in RBAC emission via `--rbac`

`opm controller install --rbac` SHALL additionally create a ClusterRole `opm-cli-user` granting full verbs on `moduleinstances`, `get/patch/update` on `moduleinstances/status`, and `get/list` on `platforms`. When `--user <U>` or `--group <G>` is supplied alongside `--rbac`, the command SHALL also create a ClusterRoleBinding binding that subject to the role. Without `--rbac`, no RBAC objects beyond those of the controller module's render are created. `--user`/`--group` without `--rbac` SHALL be rejected as a flag-validation error before any cluster interaction.

#### Scenario: RBAC off by default

- **WHEN** `opm controller install --crds-only` is run
- **THEN** no `opm-cli-user` ClusterRole or ClusterRoleBinding is created

#### Scenario: Role plus binding for a user

- **WHEN** `opm controller install --crds-only --rbac --user alice` is run
- **THEN** the `opm-cli-user` ClusterRole and a ClusterRoleBinding for user `alice` are applied with field manager `opm-cli`

#### Scenario: Subject flags require --rbac

- **WHEN** `opm controller install --user alice` is run without `--rbac`
- **THEN** the command fails flag validation with an error stating `--user`/`--group` require `--rbac`, before contacting the cluster

### Requirement: Uninstall refuses while controller cleanup finalizers are armed

Before deleting anything, `opm controller uninstall` SHALL list `ModuleInstance` resources cluster-wide; if any carries the `opmodel.dev/cleanup` finalizer, the command SHALL refuse (exit non-zero) and name each such instance as `namespace/name`. With `--remove-finalizers`, the command SHALL attempt to remove exactly the `opmodel.dev/cleanup` finalizer (leaving all other finalizers intact) from every such instance, state that the instances and their workloads are now orphaned, and then proceed to delete only if every armed instance was successfully stripped. If the cluster-wide list fails (including RBAC denial), the command SHALL fail closed without deleting anything.

#### Scenario: Refusal names the armed instances

- **WHEN** `opm controller uninstall` is run while `default/jellyfin` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command exits non-zero without deleting anything
- **AND** the error names `default/jellyfin` and points at `--remove-finalizers` and its orphaning consequence

#### Scenario: Override strips only the controller's finalizer

- **WHEN** `opm controller uninstall --remove-finalizers` is run while an instance carries both `opmodel.dev/cleanup` and a foreign finalizer
- **THEN** `opmodel.dev/cleanup` is removed from the instance, the foreign finalizer remains
- **AND** the command states the orphaning consequence and proceeds with the uninstall

#### Scenario: List failure fails closed

- **WHEN** the uninstalling user lacks permission to list `moduleinstances` cluster-wide
- **THEN** the command exits non-zero without deleting anything, mapping the RBAC error to the standard permission-denied exit code

#### Scenario: Partial finalizer-removal failure blocks the delete step

- **WHEN** `--remove-finalizers` is given and stripping the finalizer fails for one of several armed instances
- **THEN** the command still attempts every armed instance rather than stopping at the first failure
- **AND** the returned error names every instance that failed
- **AND** the controller's own resources are not deleted unless every armed instance was successfully stripped

### Requirement: All CLI server-side-apply writes use field manager `opm-cli`

Every server-side-apply write the CLI performs — instance/module applies and controller installs alike — SHALL use the field manager `opm-cli`. The former manager name `opm` is retired; ownership of resources previously applied under `opm` transfers on their next apply via the existing forced-conflicts behavior, with no user-visible change.

#### Scenario: One manager identity across the CLI

- **WHEN** any resource is applied by any CLI command after this change
- **THEN** its `managedFields` attribute the CLI's fields to manager `opm-cli`
- **AND** no CLI code path applies with manager `opm`

### Requirement: Catalog resolution precedes every cluster write

When `opm controller install` will seed a Platform, it SHALL resolve the catalog version from the registry before it resolves the kubeconfig, contacts the cluster, or applies any document. A resolution that cannot produce a version SHALL abort the command with nothing applied, so a registry problem can never leave a partially installed cluster.

Resolution failures SHALL map to the CLI's standard exit codes: a catalog major with no selectable version is a validation refusal (exit 2) that names the flag which would select a prerelease, and an unreachable or unreadable registry is a connectivity failure (exit 3).

#### Scenario: No selectable release aborts before install

- **WHEN** `opm controller install` is run and the subscribed catalog major has published no stable release
- **THEN** the command SHALL exit 2 naming `--catalog-prerelease` as the flag that selects a prerelease
- **AND** no controller resource, CRD included, SHALL have been created

#### Scenario: Unreachable registry aborts before install

- **WHEN** the catalog registry cannot be reached
- **THEN** the command SHALL exit 3 naming the lookup and the registry
- **AND** no controller resource SHALL have been created

### Requirement: Platform flags are validated before any lookup or cluster contact

`--catalog-prerelease` SHALL be rejected as a flag-validation error when combined with `--crds-only` or `--skip-platform`, because neither path seeds a Platform and the flag would have no effect. The rejection SHALL occur before any registry lookup, kubeconfig resolution, or cluster call, mirroring the existing rule that `--user`/`--group` require `--rbac`.

#### Scenario: Prerelease flag with CRDs-only

- **WHEN** `opm controller install --crds-only --catalog-prerelease` is run
- **THEN** the command SHALL fail flag validation with an error stating the flag has no effect without Platform seeding
- **AND** no registry lookup and no cluster call SHALL be made

#### Scenario: Prerelease flag with skipped platform

- **WHEN** `opm controller install --skip-platform --catalog-prerelease` is run
- **THEN** the command SHALL fail flag validation before contacting the registry or the cluster

### Requirement: Install deploys the controller module as a CLI-owned instance in two steps

`opm controller install` SHALL obtain the controller module `opmodel.dev/modules/opm_controller` from the CLI's configured module registry, render it as an instance named `opm-controller` in the namespace `opm-controller-system`, apply the rendered `CustomResourceDefinition` objects with server-side apply as field manager `opm-cli` and wait until each reports `Established=True`, and then apply the whole render as a CLI-owned ModuleInstance (`spec.owner: cli`) through the same apply the CLI uses for every instance. The instance record SHALL be written only after the CRDs are served. Every object the module renders, the CRDs and the Namespace included, SHALL be recorded in the instance's inventory, and no step SHALL refuse an object the same install applied. Install SHALL NOT create the Namespace outside the render. Apart from the render, install SHALL create only the cluster Platform and the opt-in `opm-cli-user` role. Re-running install with an unchanged module version and unchanged values SHALL change no live object, and installing another module version over the instance SHALL apply what changed and remove every object the previous version rendered and the new one does not, except CRDs and the Namespace. Source: 0021:D11:R1, 0021:D11:R2, 0032:D1:R5.

#### Scenario: Fresh cluster

- **WHEN** `opm controller install` runs against a cluster with none of OPM's CRDs
- **THEN** the four CRDs are applied and `Established` before any other object is applied
- **AND** the ModuleInstance `opm-controller` in `opm-controller-system` exists with `spec.owner: cli`, and its inventory lists every rendered object, the four CRDs and the Namespace included
- **AND** the controller Deployment is running

#### Scenario: The module's Namespace is not refused

- **WHEN** install runs on a cluster where `opm-controller-system` does not exist
- **THEN** the Namespace is created by the instance apply as one of its objects, and the apply does not refuse it as a foreign object

#### Scenario: Unchanged reinstall

- **WHEN** install runs a second time with the same module version and no new values
- **THEN** every object keeps its uid and resourceVersion, and the command reports every resource unchanged

#### Scenario: Upgrade drops an object the new version no longer renders

- **WHEN** install of module version B runs over an instance of version A, and A rendered a ClusterRole that B does not
- **THEN** that ClusterRole is deleted, the CRDs and the Namespace stay, and the record lists B's objects

### Requirement: Install refuses a controller instance record that is not CLI-owned

Before any object changes, install SHALL refuse when the controller's instance record `opm-controller` in `opm-controller-system` exists and its `spec.owner` is not `cli`, an absent owner included, since the CLI treats a record without an owner as controller-owned. The refusal SHALL exit 2, name the record, state that its `spec.owner` is not `cli`, and give the command that sets `spec.owner: cli`. Install SHALL NOT change the owner itself: the instance apply would otherwise only edit the record's spec, and the controller never reconciles the instance that deploys it, so nothing would install.

#### Scenario: Controller-owned record

- **WHEN** install runs on a cluster whose record `opm-controller` in `opm-controller-system` has `spec.owner: controller`
- **THEN** install exits 2 naming the record, saying its `spec.owner` is not `cli` and how to set it, and nothing changed

#### Scenario: Record without an owner

- **WHEN** install runs on a cluster whose record `opm-controller` in `opm-controller-system` has no `spec.owner`
- **THEN** install refuses in the same way, and nothing changed

### Requirement: The CLI pins a default controller module version and the controller it deploys

Each CLI release SHALL name one default controller module version and SHALL record the controller version that module deploys, both readable from the CLI's source without a registry. The recorded controller version SHALL be the version the module's own source states it deploys. Install without `--version` SHALL install exactly that module version. The pin SHALL be produced by `task controller:pin VERSION=<v>`, which reads the controller version from the module's own statement of the controller it deploys, without a render. The same task SHALL render that module version as install renders the controller's instance (the instance `opm-controller` in `opm-controller-system`, no values, against the module's own dependency pins) and write the render's `CustomResourceDefinition` objects as the manifests the CLI embeds for `--crds-only`, one file per CRD, replacing the previous set. A render that fails SHALL leave the pin and the manifests unchanged. No other task SHALL write the pin or the manifests, except that the pin tool MAY write the manifests alone from a local checkout of the module before that version is published. The release pin check (G1) SHALL fail when the embedded manifests differ from a fresh render of the pinned version. The module is trusted as the configured registry serves it, as every other module the CLI installs is. Source: 0021:D11:R6.

#### Scenario: Default install uses the pin

- **WHEN** `opm controller install` runs without `--version`
- **THEN** install renders the pinned module version and its output names that module version and the recorded controller version

#### Scenario: Pin refresh is one reviewable diff

- **WHEN** `task controller:pin VERSION=0.2.0` runs and module `0.2.0` states it deploys controller `1.0.0-beta.6`
- **THEN** only the pin file and the embedded CRD manifests change: the pin carries module version `0.2.0` and controller version `v1.0.0-beta.6`, and the manifests are the CRDs that version renders

#### Scenario: A module without a readable controller version is refused by the pin task

- **WHEN** `task controller:pin VERSION=<v>` runs against a module version whose source states no controller version, or a non-semver one
- **THEN** the task exits non-zero naming the module version, and the pin file and the manifests are unchanged

#### Scenario: The pin check finds drifted CRD manifests

- **WHEN** an embedded CRD manifest differs from the render of the pinned module version, or one is missing or extra
- **THEN** the release pin check fails naming each such manifest and `task controller:pin VERSION=<pinned version>`

### Requirement: Install refuses a target version the CLI cannot drive or that disagrees with itself

Before any object changes, install SHALL refuse a target module version (each refusal naming the module version and the rule it failed, exit 2):

- whose stated controller `MAJOR.MINOR` is above the CLI's own, naming upgrading the CLI as the fix; a CLI whose own version is not a released semver SHALL skip this rule with a warning;
- whose render does not run the controller Deployment `opm-controller-manager` with an image tagged with the controller version the module states;
- whose rendered ModuleInstance CRD does not carry the fields the CLI requires of that CRD before it applies an instance, naming the floor.

Install SHALL NOT be refused because the controller the cluster Platform reports has a `MAJOR.MINOR` above the CLI's, since install replaces the running controller rather than driving it. A target below the recorded module version or the running controller SHALL NOT be refused by this requirement. Source: 0021:D9:R3, 0021:D9:R4.

#### Scenario: Controller newer than the CLI

- **WHEN** the target module states controller `1.1.0` and the CLI is `1.0.0-beta.9`
- **THEN** install exits 2 naming the module version, the controller version and upgrading the CLI, and nothing changed

#### Scenario: Rendered image disagrees with the stated controller version

- **WHEN** the target module states controller `1.0.0-beta.6` and its rendered controller image is tagged `v1.0.0-beta.5`
- **THEN** install exits 2 naming the module version and both versions, and nothing changed

#### Scenario: Rendered CRD below the floor

- **WHEN** the target's rendered ModuleInstance CRD lacks `spec.owner`
- **THEN** install exits 2 naming the module version and the floor, before the CRD step

#### Scenario: Newer running controller does not block repair

- **WHEN** the cluster Platform reports controller `1.1.0`, the CLI is `1.0.0`, and the target module states controller `1.0.2`
- **THEN** the running-controller ceiling does not refuse install, and install proceeds

### Requirement: The controller's settings are recorded instance values that survive reinstall

Install SHALL take values for the controller module from `-f/--values` files, layered in order over the values recorded on the controller's instance: maps merge, and a scalar or list given in a later source replaces the earlier one. `--reset-values` SHALL drop the recorded values for that run. Re-running install, for the same or another module version, SHALL keep every recorded value the run does not change. The module's `debugValues` SHALL NOT be used. When the merged values are not accepted by the target version's `#config`, install SHALL refuse before any object changes and name each rejected value. `-f` and `--reset-values` SHALL be refused with `--crds-only` before any registry or cluster call. Source: 0006:D19.

#### Scenario: A recorded value survives reinstall

- **WHEN** install ran with a values file setting the registry mapping, and install runs again with no `-f`
- **THEN** the second render carries the same registry mapping and the record still holds it

#### Scenario: Changing one value keeps the others

- **WHEN** the record holds a registry mapping and a replica count, and install runs with a file setting only the replica count
- **THEN** the render carries the new replica count and the recorded registry mapping

#### Scenario: Recorded value the target rejects

- **WHEN** the record holds a value the target module version's `#config` does not accept
- **THEN** install exits 2 naming that value's path and `--reset-values`, and nothing changed

### Requirement: Install succeeds when the installed controller has rolled out

Install SHALL report success only after the rendered CRDs are `Established` and the controller Deployment has completed its rollout, both within the one `--timeout` budget that also covers waiting out terminating objects. When the rollout does not complete in time, install SHALL keep every applied object and the record, roll nothing back, and exit non-zero naming the Deployment and that re-running install completes it. Install SHALL NOT wait for the cluster Platform's readiness.

#### Scenario: Upgrade waits for the rollout

- **WHEN** install upgrades the controller to another module version
- **THEN** install reports success only after the controller Deployment's new ReplicaSet has completed its rollout

#### Scenario: Rollout does not complete

- **WHEN** the new controller pod cannot become ready before `--timeout` elapses
- **THEN** install exits non-zero naming `opm-controller-manager` and the timeout, and every applied object and the record remain

### Requirement: Platform seeding and the opt-in user role stay install steps

After the controller's rollout, and unless `--skip-platform` or `--crds-only` is given, install SHALL create the singleton `cluster` Platform subscribing to the catalog build it resolved before contacting the cluster, with a plain create as field manager `opm-cli`, never server-side apply or update. An existing Platform SHALL be reported and left untouched, an `AlreadyExists` response SHALL be a success-noop, and a create denied by RBAC SHALL degrade to a warning. Neither the Platform nor the `opm-cli-user` role of `--rbac` SHALL be part of the controller module's render or the instance's inventory; `--rbac` SHALL apply on the CRDs-only path as on the full one. Source: 0006:D12, 0006:D22, 0006:D23, 0021:D11:R8.

#### Scenario: Platform is seeded on a full install

- **WHEN** install completes against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` exists, subscribing to the resolved catalog build, and the output names the catalog module path and version

#### Scenario: Existing Platform is left untouched

- **WHEN** install runs against a cluster that already carries a `Platform` CR
- **THEN** the stored Platform is unchanged and the command reports it was already present

#### Scenario: Platform create denied by RBAC

- **WHEN** the installing user may apply the controller but may not create `platforms`
- **THEN** the command warns that the Platform was not seeded, names the missing permission, and does not fail because of it

#### Scenario: Platform and role are not recorded

- **WHEN** install with `--rbac --user alice` completes
- **THEN** neither `Platform/cluster` nor the `opm-cli-user` ClusterRole or its binding appears in the controller instance's inventory

### Requirement: Uninstall deletes the controller instance's recorded inventory

`opm controller uninstall` SHALL read the controller's instance record `opm-controller` in `opm-controller-system`. When none exists, it SHALL delete nothing and exit 2 naming `opm controller install` as the step that records the running controller. Otherwise, after the finalizer guard, it SHALL delete every object the record's inventory lists except CRDs and the Namespace, in descending resource-weight order, leaving behind any object that no longer carries this instance's identity, then, when no object failed, delete the record. It SHALL delete no object the inventory does not list, SHALL NOT wait for deletion to complete, and SHALL treat an already absent object as deleted. Source: 0006:D34, 0021:D11:R11.

#### Scenario: Uninstall after a module install

- **WHEN** `opm controller uninstall` runs on a cluster with no armed ModuleInstance
- **THEN** every recorded object except the four CRDs and the Namespace is deleted, then the record, and the CRDs and the Namespace still exist

#### Scenario: Object an older release installed is removed

- **WHEN** the record lists a ClusterRole the CLI's pinned module no longer renders
- **THEN** uninstall deletes it

#### Scenario: No record

- **WHEN** `opm controller uninstall` runs on a cluster whose controller was applied with kubectl and has no instance record
- **THEN** it exits 2 naming `opm controller install`, and no object is deleted

#### Scenario: Re-running uninstall

- **WHEN** uninstall runs again after an earlier run deleted the objects but failed to delete the record
- **THEN** it treats the absent objects as deleted, deletes the record and exits zero

### Requirement: Uninstall keeps the record when a recorded object could not be read

When `opm controller uninstall` reads the objects its instance record lists and the read of one fails with an error other than NotFound, that object SHALL count as a failure for that object: it SHALL be listed with its read error and not deleted, the other recorded objects SHALL still be deleted, the instance record SHALL NOT be deleted, and the command SHALL exit 1 and SHALL NOT report the controller uninstalled. A recorded `Namespace` or `CustomResourceDefinition` that could not be read SHALL instead be left behind, as it would be if read, and SHALL NOT count as a failure. Re-running uninstall after the cause is fixed SHALL delete the remaining objects and the record.

#### Scenario: Unreadable recorded object keeps the record

- **WHEN** `opm controller uninstall` runs and reading the recorded ClusterRole fails with Forbidden
- **THEN** the recorded ServiceAccount and Deployment SHALL be deleted
- **AND** the ClusterRole SHALL be listed with the Forbidden error and SHALL NOT be deleted
- **AND** the instance record SHALL NOT be deleted
- **AND** the command SHALL exit 1 without reporting the controller uninstalled

### Requirement: Install runs every refusing check before its first write

Install SHALL run, before it writes any object, every check that can refuse it: resolving the module and reading the controller version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the controller version against the CLI's, the rendered image against the module's stated controller version, and the rendered ModuleInstance CRD against the CLI's floor), the status-subresource permission check, and, on a cluster with no record of the controller's instance, the apply guard over every object the render names. The apply guard SHALL admit no object it would refuse for any other instance. With a record, install runs no apply guard: the instance apply takes over any object the render names, including one new to the inventory that already exists and OPM does not manage. Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied. Only after every check passed SHALL install write, in this order: the CRD step, then the instance apply, then the Platform seed. The CRDs-only form resolves, fetches, renders and reads nothing of the module and reads no `ModuleInstance`: its checks are the flag validation, the wait for terminating CRDs within `--timeout`, the apply guard over the embedded CRDs, which runs whether or not a record exists, and the refusal of a CRD a newer module version installed. An install that refuses SHALL change no object in the cluster.

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm controller uninstall` has returned and `opm controller install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies only after it is gone, and completes

#### Scenario: Terminating object outlives the budget

- **WHEN** an object of the render stays terminating for longer than `--timeout`
- **THEN** the command exits non-zero naming that object and the elapsed timeout, and nothing has been applied

#### Scenario: An existing Namespace OPM does not manage is refused

- **WHEN** install runs on a cluster with no record of the controller's instance, and `Namespace opm-controller-system` exists without OPM's managed-by label
- **THEN** the apply guard refuses naming that Namespace, and nothing has been applied

### Requirement: Install resolves another module version as the registry serves it

`opm controller install --version <selector>` SHALL resolve the controller module version through the CLI's configured registry, a release version pinning it and a major (`v0`) floating to that major's newest release, and SHALL install it through the same render, checks and steps as the default. Install SHALL report the module version and the controller version it deploys. A selector the registry cannot satisfy, or that is not a version selector, SHALL refuse before any cluster call. With the CLI's registry mapping pointed at a mirror holding the module and its dependencies, install SHALL need no other registry. Source: 0021:D11:R1, 0021:D11:R2.

#### Scenario: Selecting another version

- **WHEN** `opm controller install --version 0.2.0` runs and the registry serves that version
- **THEN** install applies it and its output names module `0.2.0` and the controller version it deploys

#### Scenario: Unserved version

- **WHEN** `opm controller install --version 1.0.0-beta.5` runs and no module version matches
- **THEN** the command exits 2 before contacting the cluster

#### Scenario: Mirror only

- **WHEN** the registry mapping routes `opmodel.dev` to a mirror holding the module and its dependencies, and the public registry is unreachable
- **THEN** install succeeds

### Requirement: The running-controller check finds the controller by its fixed names alone

Every CLI command that checks for a running controller before it acts SHALL locate the controller by its fixed names, reading no instance record.

The controller's objects SHALL be its fixed names, which every install of the controller module keeps: the `Deployment` `opm-controller-manager` in the namespace `opm-controller-system`, and the `CustomResourceDefinition`s `moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev` and `transformerregistrations.opmodel.dev`. The `Namespace` is checked only as the `Deployment`'s namespace; the check reads no `Namespace` object and no instance record. A controller that lacks one of these CRDs SHALL be reported as not ready.

The controller SHALL count as running only when every one of these `CustomResourceDefinition`s reports `Established=True` and the `Deployment` has completed its rollout. Otherwise the check SHALL report the controller as not ready, name each object that failed, and point at `opm controller install`. A read of any of these objects that fails SHALL count that object as not ready, so the command proceeds only on a positive finding.

#### Scenario: Module-installed controller is found by its fixed names

- **WHEN** the cluster holds the `ModuleInstance` `opm-controller` in `opm-controller-system` whose render created the four CRDs and the `Deployment` `opm-controller-manager` in `opm-controller-system`
- **AND** every CRD is `Established` and the `Deployment` has rolled out
- **THEN** the running-controller check SHALL report the controller as running

#### Scenario: Controller without the fourth CRD is not ready

- **WHEN** the cluster serves `moduleinstances`, `modulepackages` and `platforms` in `opmodel.dev` but not `transformerregistrations.opmodel.dev`, and `opm-controller-manager` has rolled out
- **THEN** the running-controller check SHALL report the controller as not ready and name `transformerregistrations.opmodel.dev`

#### Scenario: No controller at all

- **WHEN** the cluster holds none of the four CRDs and no `opm-controller-manager` Deployment
- **THEN** the running-controller check SHALL report the controller as not ready, naming the missing objects, and point at `opm controller install`

### Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry

`opm controller install --crds-only` SHALL apply the `CustomResourceDefinition` manifests the CLI embeds, which are the CRDs the pinned module version renders for the instance `opm-controller` in `opm-controller-system`, with server-side apply as `opm-cli`, and wait for each to be `Established`. It SHALL make no registry request: it SHALL NOT resolve, fetch or render the controller module and SHALL perform no catalog lookup. It SHALL NOT read any `ModuleInstance`, the controller's instance record included, and SHALL write no instance record, no workload and no Platform, so its only network peer is the Kubernetes API server. With `--rbac` it SHALL also apply the `opm-cli-user` role as a full install does. A later full install of the pinned module version SHALL record those CRDs without recreating them. When an existing CRD it would apply carries a `module.opmodel.dev/version` label newer than the pinned module version, it SHALL refuse with exit 2 before any write, naming that CRD, that version and a full install of that version as the remedy, since applying older CRDs would change the schemas the running controller was deployed with; it reads those CRDs from the API server only. `--version`, `-f/--values` and `--reset-values` SHALL be rejected with `--crds-only` as flag-validation errors before any cluster call, because the form installs only the pinned version and records no values. Source: 0021:D11:R7.

#### Scenario: Solo-cluster CRD install with no registry

- **WHEN** `opm controller install --crds-only` runs against an empty cluster while no registry can be reached
- **THEN** exactly the four CRDs of the pinned module version exist and are `Established`, applied by field manager `opm-cli`, and no ModuleInstance, Deployment or Platform exists
- **AND** the command exits 0 and names the pinned module version

#### Scenario: The controller's instance record is not read

- **WHEN** `opm controller install --crds-only` runs on a cluster whose record `opm-controller` in `opm-controller-system` has `spec.owner: controller`
- **THEN** the CRDs are applied, and no `ModuleInstance` is read or changed

#### Scenario: Full install after CRDs-only

- **WHEN** a full install of the pinned module version follows a CRDs-only install
- **THEN** the four CRDs keep their uid and appear in the new record's inventory

#### Scenario: A CRD OPM does not manage is refused

- **WHEN** `opm controller install --crds-only` runs and `moduleinstances.opmodel.dev` exists without OPM's managed-by label
- **THEN** the command exits 2 naming that CRD, and nothing has been applied

#### Scenario: CRDs a newer module version installed are not replaced

- **WHEN** a full install of module version 0.2.0 recorded the CRDs and `opm controller install --crds-only` runs from a CLI that pins 0.1.0
- **THEN** the command exits 2 naming the CRD, module version 0.2.0 and `opm controller install --version 0.2.0`, and nothing has been applied

#### Scenario: CRDs-only with a values file

- **WHEN** `opm controller install --crds-only -f values.cue` runs
- **THEN** the command fails flag validation before any registry or cluster call

#### Scenario: CRDs-only with a version

- **WHEN** `opm controller install --crds-only --version 0.2.0` runs
- **THEN** the command fails flag validation before any registry or cluster call, naming the module version the CLI pins
