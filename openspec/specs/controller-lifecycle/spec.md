## Purpose

Controller lifecycle surface: `opm controller install` / `opm controller uninstall`, the pinned controller module the CLI installs as its own instance, readiness waits, uninstall safety, and the CLI's server-side-apply field-manager identity. Slice B2 of enhancement 0006.

## Requirements

### Requirement: All CLI server-side-apply writes use field manager `opm-cli`

Every server-side-apply write the CLI performs — instance/module applies and controller installs alike — SHALL use the field manager `opm-cli`. The former manager name `opm` is retired; ownership of resources previously applied under `opm` transfers on their next apply via the existing forced-conflicts behavior, with no user-visible change.

#### Scenario: One manager identity across the CLI

- **WHEN** any resource is applied by any CLI command after this change
- **THEN** its `managedFields` attribute the CLI's fields to manager `opm-cli`
- **AND** no CLI code path applies with manager `opm`

### Requirement: Catalog resolution precedes every cluster write

When `opm controller install` will seed a Platform, it SHALL resolve the catalog version from the registry before it resolves the kubeconfig, contacts the cluster, or applies any document. A resolution that cannot produce a version SHALL abort the command with nothing applied, so a registry problem can never leave a partially installed cluster.

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

### Requirement: Install refuses a controller instance record that is not CLI-owned

Before any object changes, install SHALL refuse when the controller's instance record `opm-controller` in `opm-system` exists and its `spec.owner` is not `cli`, an absent owner included, since the CLI treats a record without an owner as controller-owned. The refusal SHALL exit 2, name the record, state that its `spec.owner` is not `cli`, and give the command that sets `spec.owner: cli`.

#### Scenario: Controller-owned record

- **WHEN** install runs on a cluster whose record `opm-controller` in `opm-system` has `spec.owner: controller`
- **THEN** install exits 2 naming the record, saying its `spec.owner` is not `cli` and how to set it, and nothing changed

#### Scenario: Record without an owner

- **WHEN** install runs on a cluster whose record `opm-controller` in `opm-system` has no `spec.owner`
- **THEN** install refuses in the same way, and nothing changed

### Requirement: Install succeeds when the installed controller has rolled out

Install SHALL report success only after the rendered CRDs are `Established` and the controller Deployment has completed its rollout, both within the one `--timeout` budget that also covers waiting out terminating objects. When the rollout does not complete in time, install SHALL keep every applied object and the record, roll nothing back, and exit non-zero naming the Deployment and that re-running install completes it. Install SHALL NOT wait for the cluster Platform's readiness.

#### Scenario: Upgrade waits for the rollout

- **WHEN** install upgrades the controller to another module version
- **THEN** install reports success only after the controller Deployment's new ReplicaSet has completed its rollout

#### Scenario: Rollout does not complete

- **WHEN** the new controller pod cannot become ready before `--timeout` elapses
- **THEN** install exits non-zero naming `opm-controller-manager` and the timeout, and every applied object and the record remain

### Requirement: Uninstall keeps the record when a recorded object could not be read

When `opm controller uninstall` reads the objects its instance record lists and the read of one fails with an error other than NotFound, that object SHALL count as a failure for that object: it SHALL be listed with its read error and not deleted, the other recorded objects SHALL still be deleted, the instance record SHALL NOT be deleted, and the command SHALL exit 1 and SHALL NOT report the controller uninstalled.

#### Scenario: Unreadable recorded object keeps the record

- **WHEN** `opm controller uninstall` runs and reading the recorded ClusterRole fails with Forbidden
- **THEN** the recorded ServiceAccount and Deployment SHALL be deleted
- **AND** the ClusterRole SHALL be listed with the Forbidden error and SHALL NOT be deleted
- **AND** the instance record SHALL NOT be deleted
- **AND** the command SHALL exit 1 without reporting the controller uninstalled

### Requirement: Uninstall fails when the instance record cannot be deleted

When `opm controller uninstall` deleted every recorded object and the delete of the controller's instance record then fails with an error other than NotFound, the command SHALL NOT report the controller uninstalled and SHALL exit non-zero. The error SHALL name the record and the cause, and SHALL say that the recorded objects were deleted, that the record remains and that re-running is safe.

#### Scenario: Forbidden record delete

- **WHEN** `opm controller uninstall` runs and the delete of the instance record fails with Forbidden
- **THEN** the recorded ClusterRole, ServiceAccount and Deployment SHALL be deleted
- **AND** the instance record SHALL still exist
- **AND** the command SHALL exit 4 without reporting the controller uninstalled

### Requirement: Install with `--rbac` creates the `opm-cli-user` role

`opm controller install --rbac` SHALL additionally create a ClusterRole `opm-cli-user` granting full verbs on `moduleinstances`, `get/patch/update` on `moduleinstances/status`, and `get/list` on `platforms`. When `--user <U>` or `--group <G>` is supplied alongside `--rbac`, the command SHALL also create a ClusterRoleBinding binding that subject to the role. Without `--rbac`, no RBAC objects beyond those of the controller module's render are created.

#### Scenario: RBAC off by default

- **WHEN** `opm controller install --crds-only` is run
- **THEN** no `opm-cli-user` ClusterRole or ClusterRoleBinding is created

#### Scenario: Role plus binding for a user

- **WHEN** `opm controller install --crds-only --rbac --user alice` is run
- **THEN** the `opm-cli-user` ClusterRole and a ClusterRoleBinding for user `alice` are applied with field manager `opm-cli`

### Requirement: The `--user` and `--group` flags require `--rbac`

`--user`/`--group` without `--rbac` SHALL be rejected as a flag-validation error before any cluster interaction.

#### Scenario: Subject flags require --rbac

- **WHEN** `opm controller install --user alice` is run without `--rbac`
- **THEN** the command fails flag validation with an error stating `--user`/`--group` require `--rbac`, before contacting the cluster

### Requirement: Uninstall refuses and names controller instances that carry the cleanup finalizer

Before deleting anything, `opm controller uninstall` SHALL list `ModuleInstance` resources cluster-wide; if any carries the `opmodel.dev/cleanup` finalizer, the command SHALL refuse (exit non-zero) and name each such instance as `namespace/name`. If the cluster-wide list fails (including RBAC denial), the command SHALL fail closed without deleting anything.

#### Scenario: Refusal names the armed instances

- **WHEN** `opm controller uninstall` is run while `default/jellyfin` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command exits non-zero without deleting anything
- **AND** the error names `default/jellyfin` and points at `--remove-finalizers` and its orphaning consequence

#### Scenario: List failure fails closed

- **WHEN** the uninstalling user lacks permission to list `moduleinstances` cluster-wide
- **THEN** the command exits non-zero without deleting anything, mapping the RBAC error to the standard permission-denied exit code

### Requirement: Uninstall removes only the cleanup finalizer on request

With `--remove-finalizers`, `opm controller uninstall` SHALL attempt to remove exactly the `opmodel.dev/cleanup` finalizer (leaving all other finalizers intact) from every instance that carries it, state that the instances and their workloads are now orphaned, and then proceed to delete only if every such instance was successfully stripped.

#### Scenario: Override strips only the controller's finalizer

- **WHEN** `opm controller uninstall --remove-finalizers` is run while an instance carries both `opmodel.dev/cleanup` and a foreign finalizer
- **THEN** `opmodel.dev/cleanup` is removed from the instance, the foreign finalizer remains
- **AND** the command states the orphaning consequence and proceeds with the uninstall

#### Scenario: Partial finalizer-removal failure blocks the delete step

- **WHEN** `--remove-finalizers` is given and stripping the finalizer fails for one of several armed instances
- **THEN** the command still attempts every armed instance rather than stopping at the first failure
- **AND** the returned error names every instance that failed
- **AND** the controller's own resources are not deleted unless every armed instance was successfully stripped

### Requirement: Catalog resolution failures map to the standard exit codes

When `opm controller install` will seed a Platform, resolution failures SHALL map to the CLI's standard exit codes: a catalog major with no selectable version is a validation refusal (exit 2) that names the flag which would select a prerelease, and an unreachable or unreadable registry is a connectivity failure (exit 3).

#### Scenario: Unreadable registry is a connectivity failure

- **WHEN** the catalog registry is unreadable
- **THEN** the command SHALL exit 3 as a connectivity failure
- **AND** no controller resource SHALL have been created

### Requirement: Install renders the controller module as the instance `opm-controller`

`opm controller install` SHALL obtain the controller module `opmodel.dev/modules/opm_controller` from the CLI's configured module registry and render it as an instance named `opm-controller` in the namespace `opm-system`. Apart from the render, install SHALL create only the cluster Platform and the opt-in `opm-cli-user` role. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: Install creates the render plus the Platform and the role

- **WHEN** `opm controller install` runs
- **THEN** it renders `opmodel.dev/modules/opm_controller` from the CLI's configured module registry as the instance `opm-controller` in the namespace `opm-system`
- **AND** apart from the render it creates only the cluster Platform and the opt-in `opm-cli-user` role

### Requirement: Install applies the CRDs before it applies the instance

Install SHALL apply the rendered `CustomResourceDefinition` objects with server-side apply as field manager `opm-cli` and wait until each reports `Established=True`, and then apply the whole render as a CLI-owned ModuleInstance (`spec.owner: cli`) through the same apply the CLI uses for every instance. The instance record SHALL be written only after the CRDs are served.

#### Scenario: Fresh cluster

- **WHEN** `opm controller install` runs against a cluster with none of OPM's CRDs
- **THEN** the four CRDs are applied and `Established` before any other object is applied
- **AND** the ModuleInstance `opm-controller` in `opm-system` exists with `spec.owner: cli`, and its inventory lists every rendered object, the four CRDs and the Namespace included
- **AND** the controller Deployment is running

### Requirement: The controller instance records every object of the render

Every object the module renders, the CRDs and the Namespace included, SHALL be recorded in the instance's inventory, and no step SHALL refuse an object the same install applied. Install SHALL NOT create the Namespace outside the render.

#### Scenario: The module's Namespace is not refused

- **WHEN** install runs on a cluster where `opm-system` does not exist
- **THEN** the Namespace is created by the instance apply as one of its objects, and the apply does not refuse it as a foreign object

### Requirement: Reinstall changes only what the module version changes

Re-running install with an unchanged module version and unchanged values SHALL change no live object, and installing another module version over the instance SHALL apply what changed and remove every object the previous version rendered and the new one does not, except CRDs and the Namespace. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: Unchanged reinstall

- **WHEN** install runs a second time with the same module version and no new values
- **THEN** every object keeps its uid and resourceVersion, and the command reports every resource unchanged

#### Scenario: Upgrade drops an object the new version no longer renders

- **WHEN** install of module version B runs over an instance of version A, and A rendered a ClusterRole that B does not
- **THEN** that ClusterRole is deleted, the CRDs and the Namespace stay, and the record lists B's objects

### Requirement: Install does not change the owner of the controller instance record

Install SHALL NOT change the owner itself: the instance apply would otherwise only edit the record's spec, and the controller never reconciles the instance that deploys it, so nothing would install.

#### Scenario: Refusal leaves the owner as it was

- **WHEN** install refuses a record whose `spec.owner` is not `cli`
- **THEN** install has not changed the owner of the record

### Requirement: Each CLI release pins a default controller module version

Each CLI release SHALL name one default controller module version and SHALL record the controller version that module deploys, both readable from the CLI's source without a registry. The recorded controller version SHALL be the version the module's own source states it deploys. Install without `--version` SHALL install exactly that module version. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: Default install uses the pin

- **WHEN** `opm controller install` runs without `--version`
- **THEN** install renders the pinned module version and its output names that module version and the recorded controller version

### Requirement: Only `task controller:pin` writes the pin, from the module's own statement

The pin SHALL be produced by `task controller:pin VERSION=<v>`, which reads the controller version from the module's own statement of the controller it deploys, without a render. No other task SHALL write the pin or the manifests, except that the pin tool MAY write the manifests alone from a local checkout of the module before that version is published. The module is trusted as the configured registry serves it, as every other module the CLI installs is.

#### Scenario: Pin refresh is one reviewable diff

- **WHEN** `task controller:pin VERSION=0.2.0` runs and module `0.2.0` states it deploys controller `1.0.0-beta.6`
- **THEN** only the pin file and the embedded CRD manifests change: the pin carries module version `0.2.0` and controller version `v1.0.0-beta.6`, and the manifests are the CRDs that version renders

#### Scenario: A module without a readable controller version is refused by the pin task

- **WHEN** `task controller:pin VERSION=<v>` runs against a module version whose source states no controller version, or a non-semver one
- **THEN** the task exits non-zero naming the module version, and the pin file and the manifests are unchanged

### Requirement: The pin task writes the CRD manifests the CLI embeds

`task controller:pin` SHALL render the pinned module version as install renders the controller's instance (the instance `opm-controller` in `opm-system`, no values, against the module's own dependency pins). It SHALL write the render's `CustomResourceDefinition` objects as the manifests the CLI embeds for `--crds-only`, one file per CRD, replacing the previous set. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: A failed render leaves the pin and the manifests unchanged

- **WHEN** the render fails, or a manifest write fails
- **THEN** the task exits non-zero, and the pin file and the manifests are unchanged

### Requirement: The release pin check fails on drifted CRD manifests

The release pin check (G1) SHALL fail when the embedded manifests differ from a fresh render of the pinned version. (Requirement: The pin task writes the CRD manifests the CLI embeds)

#### Scenario: The pin check finds drifted CRD manifests

- **WHEN** an embedded CRD manifest differs from the render of the pinned module version, or one is missing or extra
- **THEN** the release pin check fails naming each such manifest and `task controller:pin VERSION=<pinned version>`

### Requirement: Install refuses a target controller newer than the CLI

Before any object changes, install SHALL refuse a target module version whose stated controller `MAJOR.MINOR` is above the CLI's own, with exit 2, naming the module version and the rule it failed and naming upgrading the CLI as the fix. A CLI whose own version is not a released semver SHALL skip this rule with a warning. Source: enhancement 0021 ("The CLI's ceiling on controller versions compares `MAJOR.MINOR` only").

#### Scenario: Controller newer than the CLI

- **WHEN** the target module states controller `1.1.0` and the CLI is `1.0.0-beta.9`
- **THEN** install exits 2 naming the module version, the controller version and upgrading the CLI, and nothing changed

### Requirement: Install refuses a target whose render disagrees with its stated controller version

Before any object changes, install SHALL refuse a target module version whose render does not run the controller Deployment `opm-controller-manager` with an image tagged with the controller version the module states, with exit 2, naming the module version and the rule it failed.

#### Scenario: Rendered image disagrees with the stated controller version

- **WHEN** the target module states controller `1.0.0-beta.6` and its rendered controller image is tagged `v1.0.0-beta.5`
- **THEN** install exits 2 naming the module version and both versions, and nothing changed

### Requirement: Install refuses a target whose ModuleInstance CRD lacks the required fields

Before any object changes, install SHALL refuse a target module version whose rendered ModuleInstance CRD does not carry the fields the CLI requires of that CRD before it applies an instance, with exit 2, naming the module version and the rule it failed and naming the floor.

#### Scenario: Rendered CRD below the floor

- **WHEN** the target's rendered ModuleInstance CRD lacks `spec.owner`
- **THEN** install exits 2 naming the module version and the floor, before the CRD step

### Requirement: A newer running controller does not block install

Install SHALL NOT be refused because the controller the cluster Platform reports has a `MAJOR.MINOR` above the CLI's, because install replaces, not drives, the running controller. A target below the recorded module version or the running controller SHALL NOT be refused by "Install refuses a target controller newer than the CLI", "Install refuses a target whose render disagrees with its stated controller version" or "Install refuses a target whose ModuleInstance CRD lacks the required fields".

**Source**: enhancement 0021 ("The CLI's ceiling on controller versions compares `MAJOR.MINOR` only").

#### Scenario: Newer running controller does not block repair

- **WHEN** the cluster Platform reports controller `1.1.0`, the CLI is `1.0.0`, and the target module states controller `1.0.2`
- **THEN** the running-controller ceiling does not refuse install, and install proceeds

### Requirement: Install layers values over the values recorded on the controller instance

Install SHALL take values for the controller module from `-f/--values` files, layered in order over the values recorded on the controller's instance: maps merge, and a scalar or list given in a later source replaces the earlier one. `--reset-values` SHALL drop the recorded values for that run. The module's `debugValues` SHALL NOT be used. Source: enhancement 0006 ("`spec.values` is the sole authoritative render input; the CLI unifies all value inputs into it; resolves OQ10").

#### Scenario: Changing one value keeps the others

- **WHEN** the record holds a registry mapping and a replica count, and install runs with a file setting only the replica count
- **THEN** the render carries the new replica count and the recorded registry mapping

### Requirement: Recorded controller values survive reinstall

Re-running install, for the same or another module version, SHALL keep every recorded value the run does not change.

#### Scenario: A recorded value survives reinstall

- **WHEN** install ran with a values file setting the registry mapping, and install runs again with no `-f`
- **THEN** the second render carries the same registry mapping and the record still holds it

### Requirement: Install refuses values it cannot use

When the merged values are not accepted by the target version's `#config`, install SHALL refuse before any object changes and name each rejected value. `-f` and `--reset-values` SHALL be refused with `--crds-only` before any registry or cluster call.

#### Scenario: Recorded value the target rejects

- **WHEN** the record holds a value the target module version's `#config` does not accept
- **THEN** install exits 2 naming that value's path and `--reset-values`, and nothing changed

#### Scenario: CRDs-only with a values file

- **WHEN** `opm controller install --crds-only -f values.cue` runs
- **THEN** the command fails flag validation before any registry or cluster call

### Requirement: Install seeds the cluster Platform after the controller rolls out

After the controller's rollout, and unless `--skip-platform` or `--crds-only` is given, install SHALL create the singleton `cluster` Platform subscribing to the catalog build it resolved before contacting the cluster, with a plain create as field manager `opm-cli`, never server-side apply or update. Source: enhancement 0006 ("`Platform` carries no owner marker; the operator always owns the singleton; the CLI writes an un-owned `cluster` Platform in solo clusters").

#### Scenario: Platform is seeded on a full install

- **WHEN** install completes against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` exists, subscribing to the resolved catalog build, and the output names the catalog module path and version

### Requirement: An existing Platform is left alone and a denied create is a warning

An existing Platform SHALL be reported and left untouched, an `AlreadyExists` response SHALL be a success-noop, and a create denied by RBAC SHALL degrade to a warning. Source: enhancement 0006 ("Solo-cluster `Platform` write-if-absent uses a plain `Create` that tolerates `AlreadyExists`, not SSA apply; refines D12's mechanism, resolves OQ13").

#### Scenario: Existing Platform is left untouched

- **WHEN** install runs against a cluster that already carries a `Platform` CR
- **THEN** the stored Platform is unchanged and the command reports it was already present

#### Scenario: Platform create denied by RBAC

- **WHEN** the installing user may apply the controller but may not create `platforms`
- **THEN** the command warns that the Platform was not seeded, names the missing permission, and does not fail because of it

### Requirement: The Platform and the opt-in user role are not part of the render or the inventory

Neither the Platform nor the `opm-cli-user` role of `--rbac` SHALL be part of the controller module's render or the instance's inventory. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: Platform and role are not recorded

- **WHEN** install with `--rbac --user alice` completes
- **THEN** neither `Platform/cluster` nor the `opm-cli-user` ClusterRole or its binding appears in the controller instance's inventory

### Requirement: `--rbac` applies on the CRDs-only path as on the full one

`--rbac` SHALL apply on the CRDs-only path as on the full one. Source: enhancement 0006 ("Status-subresource RBAC (opt-in `--rbac` emission + a pre-flight access check, both as edge-case safety nets); resolves OQ3").

#### Scenario: `--rbac` on the CRDs-only path

- **WHEN** `opm controller install --crds-only --rbac --user alice` is run
- **THEN** `--rbac` applies as on the full path and the `opm-cli-user` ClusterRole and a ClusterRoleBinding for user `alice` are applied

### Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry

`opm controller install --crds-only` SHALL apply the `CustomResourceDefinition` manifests the CLI embeds, which are the CRDs the pinned module version renders for the instance `opm-controller` in `opm-system`, with server-side apply as `opm-cli`, and wait for each to be `Established`. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: Solo-cluster CRD install with no registry

- **WHEN** `opm controller install --crds-only` runs against an empty cluster while no registry can be reached
- **THEN** exactly the four CRDs of the pinned module version exist and are `Established`, applied by field manager `opm-cli`, and no ModuleInstance, Deployment or Platform exists
- **AND** the command exits 0 and names the pinned module version

### Requirement: CRDs-only install makes no registry request

`opm controller install --crds-only` SHALL make no registry request: it SHALL NOT resolve, fetch or render the controller module and SHALL perform no catalog lookup. It SHALL read the CRDs already in the cluster from the API server only. (Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry)

#### Scenario: CRDs-only install makes no registry request

- **WHEN** `opm controller install --crds-only` runs while no registry can be reached
- **THEN** the command makes no registry request and applies the embedded CRDs

### Requirement: CRDs-only install writes no record, workload or Platform

`opm controller install --crds-only` SHALL write no instance record, no workload and no Platform, so its only network peer is the Kubernetes API server. (Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry)

#### Scenario: CRDs-only install writes only CRDs

- **WHEN** `opm controller install --crds-only` runs against an empty cluster
- **THEN** no ModuleInstance, Deployment or Platform exists afterwards

### Requirement: A full install records the CRDs a CRDs-only install applied

A later full install of the pinned module version SHALL record those CRDs without recreating them. (Requirement: CRDs-only install writes no record, workload or Platform)

#### Scenario: Full install after CRDs-only

- **WHEN** a full install of the pinned module version follows a CRDs-only install
- **THEN** the four CRDs keep their uid and appear in the new record's inventory

### Requirement: CRDs-only install reads the controller's instance record for its inventory only

`opm controller install --crds-only` SHALL read one `ModuleInstance` at most, the controller's instance record, and SHALL use only that record's inventory: the record's owner and values SHALL play no part. A cluster that does not serve `ModuleInstance` has no record. (Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry)

#### Scenario: The controller's instance record is read for its inventory only

- **WHEN** `opm controller install --crds-only` runs on a cluster whose record `opm-controller` in `opm-system` has `spec.owner: controller`
- **THEN** the CRDs are applied, and no `ModuleInstance` is changed

### Requirement: CRDs-only install judges each CRD by the apply guard

`opm controller install --crds-only` SHALL NOT adopt, relabel or migrate a CRD by itself. Every existing CRD it would apply is judged by the apply guard (capability `apply-pruning`, "The ownership guard judges every apply") as on a full install: with the instance UUID the embedded CRDs carry, and with the inventory of the controller's instance record when that record exists. (Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry)

#### Scenario: CRDs OPM does not manage are refused, not adopted

- **WHEN** `opm controller install --crds-only` runs and `moduleinstances.opmodel.dev` exists without OPM's managed-by label and without an adopt annotation
- **THEN** the command exits 2 with `refusing to install:`, naming that CRD as not managed by OPM and the `opmodel.dev/adopt` annotation with the UUID to set
- **AND** nothing has been applied and the CRD's labels are unchanged

#### Scenario: An annotated CRD is applied

- **WHEN** `opm controller install --crds-only` runs and `moduleinstances.opmodel.dev` exists without OPM's managed-by label, annotated `opmodel.dev/adopt` with the UUID of the controller's instance
- **THEN** the CRD is applied and carries that UUID

### Requirement: A CRD the record lists is applied whatever identity it carries

A CRD that the record's inventory lists SHALL be applied by `opm controller install --crds-only` whatever identity label it carries, as a full install applies it. (Requirement: CRDs-only install judges each CRD by the apply guard)

#### Scenario: A CRD the record lists is applied whatever identity it carries

- **WHEN** `opm controller install --crds-only` runs, the record `opm-controller` in `opm-system` lists `moduleinstances.opmodel.dev` in its inventory, and that CRD carries OPM's managed-by label and the UUID of another instance
- **THEN** the CRD is applied and carries the UUID of the controller's instance, and the record is unchanged

### Requirement: A CRD the record does not list is judged as on a first install

With no record, or for a CRD the record does not list, the CRD SHALL be judged as on a first install. When it carries no OPM managed-by label, or carries the identity of another instance, `opm controller install --crds-only` SHALL refuse with exit 2 before any write, with the guard's message: it names that CRD and the `opmodel.dev/adopt` annotation with the UUID to set. (Requirement: CRDs-only install judges each CRD by the apply guard)

#### Scenario: A CRD the record does not list is judged as on a first install

- **WHEN** `opm controller install --crds-only` runs, the record lists `moduleinstances.opmodel.dev` only, and `platforms.opmodel.dev` carries the UUID of another instance
- **THEN** the command exits 2 naming `platforms.opmodel.dev` alone, and nothing has been applied

#### Scenario: A CRD of another instance without a record is refused

- **WHEN** `opm controller install --crds-only` runs on a cluster with no record of the controller's instance and `moduleinstances.opmodel.dev` carries OPM's managed-by label and the UUID of another instance
- **THEN** the command exits 2 naming that CRD, that instance and the `opmodel.dev/adopt` annotation with the UUID to set, and nothing has been applied

### Requirement: A denied read of the record is no record on the CRDs-only path

When the read of the record is denied (the API server answers Forbidden, judged by the typed error and never by message text), `opm controller install --crds-only` SHALL treat it as no record, SHALL print one line that says the record could not be read and that the CRDs are judged without it, and SHALL go on. (Requirement: CRDs-only install reads the controller's instance record for its inventory only)

#### Scenario: A denied record read is no record

- **WHEN** `opm controller install --crds-only` runs on a cluster with no `opmodel.dev` CRD, by a caller the API server denies the read of the record `opm-controller` in `opm-system`
- **THEN** the four CRDs are applied and the command exits 0
- **AND** one line says that the record could not be read and that the CRDs are judged without it

#### Scenario: A denied record read opens nothing

- **WHEN** `opm controller install --crds-only` runs, the read of the record is denied, and `moduleinstances.opmodel.dev` exists without OPM's managed-by label and without an adopt annotation
- **THEN** the command exits 2 naming that CRD and the `opmodel.dev/adopt` annotation with the UUID to set, and nothing has been applied

#### Scenario: A CRD of the controller's instance passes with the record read denied

- **WHEN** `opm controller install --crds-only` runs, the read of the record is denied, and `moduleinstances.opmodel.dev` carries OPM's managed-by label and the UUID of the controller's instance
- **THEN** the CRDs are applied

### Requirement: A caller without access to the controller's namespace can still install the CRDs

The record only widens what the guard allows, so a caller with no permission in the controller's namespace SHALL still be able to install the CRDs with `--crds-only`. (Requirement: A denied read of the record is no record on the CRDs-only path)

#### Scenario: A caller without access to the controller's namespace installs the CRDs

- **WHEN** `opm controller install --crds-only` runs on a cluster with no `opmodel.dev` CRD, by a caller with no permission in `opm-system`
- **THEN** the four CRDs are applied

### Requirement: Another failed read of the record stops the CRDs-only install

When the read of the record fails for any other reason (Unauthorized, a server error, a timeout), `opm controller install --crds-only` SHALL fail before any write with the read error. A full install is not changed by this: it needs the record and SHALL fail when it cannot read it. (Requirement: A denied read of the record is no record on the CRDs-only path)

#### Scenario: A failed record read stops the CRDs-only install

- **WHEN** `opm controller install --crds-only` runs and the read of the record `opm-controller` in `opm-system` fails with a server error, Unauthorized or a timeout
- **THEN** the command fails naming the record, and nothing has been applied

#### Scenario: A full install stops on a denied record read

- **WHEN** a full `opm controller install` runs and the read of the record `opm-controller` in `opm-system` is denied
- **THEN** the command fails naming the record, and nothing has been applied

### Requirement: CRDs-only install does not replace the CRDs of a newer module version

When an existing CRD it would apply carries the embedded CRD's `module.opmodel.dev/name` and a `module.opmodel.dev/version` label newer than the pinned module version, `opm controller install --crds-only` SHALL refuse with exit 2 before any write, naming that CRD, that version and both remedies: `--crds-only` from a CLI that pins that version or newer, and a full install of it. (Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry)

#### Scenario: CRDs a newer module version installed are not replaced

- **WHEN** module version 0.2.0 installed the CRDs, by a full install or by `--crds-only` from a CLI that pins 0.2.0, and `opm controller install --crds-only` runs from a CLI that pins 0.1.0
- **THEN** the command exits 2 naming the CRD, module version 0.2.0, `--crds-only` from a CLI that pins 0.2.0 or newer, and `opm controller install --version 0.2.0`, and nothing has been applied

### Requirement: CRDs-only install rejects the flags that need a registry or record values

`--version`, `-f/--values` and `--reset-values` SHALL be rejected with `--crds-only` as flag-validation errors before any cluster call, because the form installs only the pinned version and records no values. (Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry)

#### Scenario: CRDs-only with a version

- **WHEN** `opm controller install --crds-only --version 0.2.0` runs
- **THEN** the command fails flag validation before any registry or cluster call, naming the module version the CLI pins

### Requirement: Uninstall without an instance record deletes nothing

`opm controller uninstall` SHALL read the controller's instance record `opm-controller` in `opm-system`. When none exists, it SHALL delete nothing and exit 2 naming `opm controller install` as the step that records the running controller. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: No record

- **WHEN** `opm controller uninstall` runs on a cluster whose controller was applied with kubectl and has no instance record
- **THEN** it exits 2 naming `opm controller install`, and no object is deleted

### Requirement: Uninstall deletes the recorded objects except the CRDs and the Namespace

After the finalizer guard, `opm controller uninstall` SHALL delete every object the record's inventory lists except CRDs and the Namespace, in descending resource-weight order, leaving behind any object that the delete verdict of `opm instance delete` leaves behind (capability `deploy`, "Instance delete re-checks live ownership before each delete"), then, when no object failed, delete the record. (Requirement: Uninstall leaves behind what the instance delete verdict leaves behind)

**Source**: enhancement 0006 ("`opm operator uninstall` preserves the namespace; refuses while cleanup finalizers are armed, with `--remove-finalizers` as the override").

#### Scenario: Uninstall after a module install

- **WHEN** `opm controller uninstall` runs on a cluster with no armed ModuleInstance
- **THEN** every recorded object except the four CRDs and the Namespace is deleted, then the record, and the CRDs and the Namespace still exist

#### Scenario: Object an older release installed is removed

- **WHEN** the record lists a ClusterRole the CLI's pinned module no longer renders
- **THEN** uninstall deletes it

### Requirement: Uninstall leaves behind what the instance delete verdict leaves behind

Uninstall SHALL leave behind any object that the delete verdict of `opm instance delete` leaves behind (capability `deploy`, "Instance delete re-checks live ownership before each delete"): one that OPM no longer manages, one that carries another instance's identity, or one whose `opmodel.dev/adopt` annotation names another instance.

#### Scenario: Recorded object adopted by another instance is left behind

- **WHEN** a recorded ClusterRole carries `opmodel.dev/adopt` with the UUID of another instance
- **THEN** uninstall SHALL NOT delete it, SHALL list it as `left behind`, and SHALL count it in the closing line
- **AND** the record SHALL still be deleted when no object failed

### Requirement: Uninstall deletes nothing outside the inventory and treats an absent object as deleted

`opm controller uninstall` SHALL delete no object the inventory does not list, SHALL NOT wait for deletion to complete, and SHALL treat an already absent object as deleted.

#### Scenario: Re-running uninstall

- **WHEN** uninstall runs again after an earlier run deleted the objects but failed to delete the record
- **THEN** it treats the absent objects as deleted, deletes the record and exits zero

### Requirement: Uninstall leaves an unreadable Namespace or CRD behind without failing

When the read of a recorded `Namespace` or `CustomResourceDefinition` fails with an error other than NotFound, `opm controller uninstall` SHALL leave that object behind, as it would if the read had worked, and it SHALL NOT count as a failure.

#### Scenario: Unreadable recorded Namespace is left behind

- **WHEN** `opm controller uninstall` runs and reading the recorded `Namespace` fails with Forbidden
- **THEN** the `Namespace` SHALL be left behind, as it would be if read
- **AND** it SHALL NOT count as a failure

### Requirement: Re-running uninstall after a read failure deletes the rest

Re-running `opm controller uninstall` after the cause of a failed read is fixed SHALL delete the remaining objects and the record.

#### Scenario: Re-running uninstall after the read cause is fixed

- **WHEN** `opm controller uninstall` is run again after the cause of the failed read is fixed
- **THEN** it SHALL delete the remaining objects and the record

### Requirement: A failed record delete exits by the cause of the failure

When `opm controller uninstall` deleted every recorded object and the delete of the controller's instance record then fails with an error other than NotFound, the exit code SHALL be 4 when the API server denied the delete (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure.

#### Scenario: Record delete times out

- **WHEN** `opm controller uninstall` runs and the delete of the instance record fails with a server timeout
- **THEN** the command SHALL exit 3 without reporting the controller uninstalled

#### Scenario: Record delete fails for another reason

- **WHEN** `opm controller uninstall` runs and the delete of the instance record fails with an error that is not a denial, a server timeout or service unavailable
- **THEN** the command SHALL exit 1 without reporting the controller uninstalled

### Requirement: Uninstall takes every action on a recorded object from the shared deletion plan

`opm controller uninstall` SHALL take every action on a recorded object from the deletion plan `opm instance delete` uses (capability `deploy`, "Instance delete follows the shared deletion plan"), built from the objects of the controller's instance record that the command found or could not read, and from the identity stored in the record. The output lines and the exit codes of the command SHALL stay as the other requirements of this capability state them.

#### Scenario: Recorded objects are deleted in the plan's order

- **WHEN** `opm controller uninstall` runs on a cluster with no armed ModuleInstance
- **THEN** the delete request for the controller Deployment SHALL be sent before the ones for its ServiceAccount and ClusterRole
- **AND** each delete request SHALL carry Foreground propagation and a precondition on the UID of the object that was read

### Requirement: Uninstall acts on an object and on the record only as the plan allows

`opm controller uninstall` SHALL read a recorded object, delete it or leave it in place only when the plan names that action as the next one, and SHALL delete the instance record only when the plan's release verdict allows it. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Failed object holds the record

- **WHEN** the delete of the recorded ClusterRole fails with Forbidden
- **THEN** the other recorded objects SHALL still be deleted
- **AND** the instance record SHALL NOT be deleted
- **AND** the command SHALL exit 1 without reporting the controller uninstalled

### Requirement: The finalizer guard and the no-record refusal run before the plan is built

The finalizer guard and the no-record refusal of `opm controller uninstall` SHALL run before it builds the deletion plan.

#### Scenario: Guard refuses before any plan

- **WHEN** a ModuleInstance still carries the controller's cleanup finalizer and `--remove-finalizers` is not set
- **THEN** no recorded object SHALL be read for a delete verdict and none SHALL be deleted

### Requirement: Install runs every check that can refuse it before it writes

Install SHALL run, before any write, every check that can refuse it: resolving the module and reading the controller version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the controller version against the CLI's, the rendered image against the module's stated controller version, and the rendered ModuleInstance CRD against the CLI's floor). (Requirement: Install runs the permission check and apply guard before writing)

#### Scenario: The module, render, values and target rules are checked before the first write

- **WHEN** install runs
- **THEN** it SHALL run the resolving of the module, the rendering, the check of the values against the target version's `#config` and the target rules of install before it writes any object

### Requirement: The CRDs-only form runs the checks that need no module

The CRDs-only form SHALL resolve, fetch, render and read nothing of the module. Its checks SHALL be the flag validation, the wait for terminating CRDs within `--timeout`, the refusal of a CRD a newer module version installed, and the same apply guard over the embedded CRDs, asked with the UUID of the controller's instance and with the inventory of the controller's instance record when that record exists. (Requirement: Install runs every check that can refuse it before it writes)

#### Scenario: The CRDs-only form needs no registry for its checks

- **WHEN** `opm controller install --crds-only` runs against an empty cluster while no registry can be reached
- **THEN** every check of the form runs without a registry request

### Requirement: Install runs the permission check and apply guard before writing

Install SHALL run, before it writes any object, every check that can refuse it: the status-subresource permission check, and the apply guard over every object the render names (capability `apply-pruning`, "The ownership guard judges every apply"). An install that refuses SHALL change no object in the cluster. (Requirement: Install runs every check that can refuse it before it writes)

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

### Requirement: Install writes in a fixed order after every check passed

Only after every check passed SHALL install write, in this order: the CRD step, then the instance apply, then the Platform seed. Outside the instance apply and its prune, install SHALL send no delete and no patch of managed fields.

#### Scenario: Install writes the CRD step, the instance apply and the Platform seed in order

- **WHEN** every check of install passed
- **THEN** install writes in this order: the CRD step, then the instance apply, then the Platform seed

### Requirement: The apply guard runs on every install

The apply guard over every object the render names (capability `apply-pruning`, "The ownership guard judges every apply") SHALL run on every install, with or without a record of the controller's instance. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: CRDs-only refuses unlabelled CRDs

- **WHEN** `opm controller install --crds-only` runs and a CRD the CLI embeds exists without an OPM managed-by label and without an adopt annotation
- **THEN** the command SHALL exit 2 naming that CRD, and no CRD has changed

### Requirement: Install passes the guard no admission set and no proof of origin

Install SHALL pass the guard no admission set and SHALL run no proof of where an existing object came from: an object that an earlier release manifest created is judged as any other existing object. (Requirement: The apply guard runs on every install)

#### Scenario: Unlabelled CRDs of an earlier manifest install are refused

- **WHEN** install runs on a cluster whose four `opmodel.dev` CRDs, Namespace and Deployment were applied from an earlier release manifest and carry no OPM managed-by label and no adopt annotation
- **THEN** the command SHALL exit 2 with `refusing to install:` and one line for each such object that says it exists and is not managed by OPM, and names `opmodel.dev/adopt` with the UUID to set
- **AND** install SHALL adopt none of them by itself, and no object in the cluster has changed

### Requirement: The guard judges an object the record lists as recorded

An object the record lists is judged as recorded and SHALL pass whatever its UUID label says. (Requirement: The apply guard runs on every install)

#### Scenario: Reinstall over its own recorded objects passes the guard

- **WHEN** install runs on a cluster with a record of the controller's instance and every existing object the render names is in that record
- **THEN** the apply guard refuses none of them

#### Scenario: Recorded objects that carry another identity are applied

- **WHEN** install runs on a cluster with a record of the controller's instance that lists every object the render names
- **AND** every one of those objects, the four CRDs and the Namespace included, carries an OPM managed-by label and the UUID of another instance
- **THEN** the apply guard refuses none of them and install applies them

### Requirement: The guard judges an existing object the record does not list

An existing object the record does not list SHALL pass when its `opmodel.dev/adopt` annotation holds the UUID of the controller's instance, or when it carries an OPM managed-by label and no UUID label of another instance; it SHALL refuse the install when OPM does not manage it or when it carries another instance's identity. (Requirement: The apply guard runs on every install)

#### Scenario: Annotated objects that carry another identity are applied

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** every object the render names exists with the UUID of another instance and `opmodel.dev/adopt` holding the UUID of the controller's instance
- **THEN** the apply guard refuses none of them, and install applies and records them

#### Scenario: The instance's own objects without a record pass the guard

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** every existing object the render names carries an OPM managed-by label and the UUID of the controller's instance (a run that stopped before it wrote the record, a CRDs-only install, or a `kubectl apply` of the module's render)
- **THEN** the apply guard refuses none of them

#### Scenario: An unrecorded object with the instance's UUID and no OPM managed-by label is refused

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** an object the render names carries the UUID of the controller's instance, and its managed-by label is absent or names another tool
- **THEN** the command SHALL exit 2 naming that object as not managed by OPM, and `opmodel.dev/adopt` with the UUID to set

#### Scenario: Reinstall refuses a foreign object new to the inventory

- **WHEN** install runs on a cluster with a record of the controller's instance
- **AND** the render names a ClusterRole the record does not list, which exists without an OPM managed-by label
- **THEN** the command SHALL exit 2 naming that ClusterRole and the adopt annotation, and no object in the cluster has changed

#### Scenario: First install refuses an object of another instance

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** a CustomResourceDefinition the render names exists with an OPM managed-by label and the UUID of another instance
- **THEN** the command SHALL exit 2 naming that CustomResourceDefinition and the other instance, and no object in the cluster has changed

### Requirement: Install refuses an object adopted by another instance

Install needs every object its render names, so, unlike any other apply, it SHALL also refuse when the `opmodel.dev/adopt` annotation of an object names another instance, whether or not the record lists the object: it SHALL NOT leave the object unapplied and go on (owner decision of 2026-10-08 on `opm controller install`).

#### Scenario: Install refuses an object adopted by another instance

- **WHEN** the controller Deployment, a rendered CustomResourceDefinition or the controller Namespace carries `opmodel.dev/adopt` with the UUID of another instance
- **THEN** the command SHALL exit 2 naming that object and that instance
- **AND** no object in the cluster has changed, the managed fields of that object included

### Requirement: Every refusal of the guard exits 2 and names the object, the instance and the annotation

Every refusal of the guard in this phase SHALL exit 2, SHALL name the object and the instance that owns or adopts it, SHALL name the `opmodel.dev/adopt` annotation and the UUID to set where the annotation lifts the refusal, and SHALL leave the cluster unchanged; that includes a failed read by the guard, with or without a record. No refusal of install SHALL tell the user to delete or rename an object. (Requirement: Install runs the permission check and apply guard before writing)

#### Scenario: Objects that carry another identity, with no record and no annotation, are refused

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** every object the render names exists with an OPM managed-by label and the UUID of another instance, and none carries an adopt annotation
- **THEN** the command SHALL exit 2 naming each object, the other instance and `opmodel.dev/adopt` with the UUID to set
- **AND** the output SHALL NOT contain `controller migration refused` or `remove or rename`
- **AND** no object in the cluster has changed

### Requirement: Install waits out terminating objects before the guard runs

Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied.

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm controller uninstall` has returned and `opm controller install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies only after it is gone, and completes

#### Scenario: Terminating object outlives the budget

- **WHEN** an object of the render stays terminating for longer than `--timeout`
- **THEN** the command exits non-zero naming that object and the elapsed timeout, and nothing has been applied

### Requirement: A failed read refuses the install in the check that made it

Two of the checks of install read every object install applies, in this order: the wait for terminating objects and the apply guard. A read that fails with an error other than NotFound SHALL refuse the install in the check that made it. When the apply guard fails a read, the command SHALL exit 2, the code of the guard's other refusals.

#### Scenario: A read the apply guard fails exits with the guard code

- **WHEN** the earlier read of a rendered object was answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed

#### Scenario: Reinstall with a failed guard read exits with the guard code

- **WHEN** install runs on a cluster with a record of the controller's instance, the earlier read of a rendered object was answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed

### Requirement: A failed read in the wait for terminating objects exits by the read error

When the wait for terminating objects fails a read with an error other than NotFound, the command SHALL exit with the code of the read error: 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. An object that install cannot read from the start is therefore refused by the wait for terminating objects, with exit code 4 when the read is denied.

#### Scenario: An object unreadable from the start exits by the read error

- **WHEN** install runs on a cluster with no record of the controller's instance and every read of a ClusterRole the render names fails with Forbidden
- **THEN** the command SHALL exit 4 naming that ClusterRole and the read error, and no object in the cluster has changed

### Requirement: `--version` resolves the controller module through the configured registry

`opm controller install --version <selector>` SHALL resolve the controller module version through the CLI's configured registry, a release version pinning it and a major (`v0`) floating to that major's newest release, and SHALL install it through the same render, checks and steps as the default. Install SHALL report the module version and the controller version it deploys. Source: enhancement 0021 ("The controller's install artifact is a registry module on its own version train").

#### Scenario: Selecting another version

- **WHEN** `opm controller install --version 0.2.0` runs and the registry serves that version
- **THEN** install applies it and its output names module `0.2.0` and the controller version it deploys

### Requirement: Install refuses a selector the registry cannot satisfy before any cluster call

A selector the registry cannot satisfy, or that is not a version selector, SHALL refuse before any cluster call; a selector that has the shape of an opm-controller release tag SHALL get no message of its own. (Requirement: `--version` resolves the controller module through the configured registry)

#### Scenario: Unserved version

- **WHEN** `opm controller install --version 1.0.0-beta.5` runs and no module version matches
- **THEN** the command exits 2 before contacting the cluster

### Requirement: A registry mirror is the only registry install needs

With the CLI's registry mapping pointed at a mirror holding the module and its dependencies, install SHALL need no other registry.

#### Scenario: Mirror only

- **WHEN** the registry mapping routes `opmodel.dev` to a mirror holding the module and its dependencies, and the public registry is unreachable
- **THEN** install succeeds

### Requirement: The running-controller check locates the controller by its fixed names

Every CLI command that checks for a running controller before it acts SHALL locate the controller by its fixed names, reading no instance record. The controller's objects SHALL be its fixed names, which every install of the controller module keeps: the `Deployment` `opm-controller-manager` in the namespace `opm-system`, and the `CustomResourceDefinition`s `moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev` and `transformerregistrations.opmodel.dev`.

#### Scenario: Module-installed controller is found by the same names

- **WHEN** the cluster holds the `ModuleInstance` `opm-controller` in `opm-system` whose render created the four CRDs and the `Deployment` `opm-controller-manager` in `opm-system`
- **AND** every CRD is `Established` and the `Deployment` has rolled out
- **THEN** the running-controller check SHALL report the controller as running

### Requirement: The running-controller check reads no Namespace and reports a missing CRD as not ready

The `Namespace` is checked only as the namespace of the `Deployment` `opm-controller-manager`; the check reads no `Namespace` object and no instance record. A controller that lacks one of these CRDs (as listed in requirement "The running-controller check locates the controller by its fixed names") SHALL be reported as not ready.

#### Scenario: Controller older than the fourth CRD is not ready

- **WHEN** the cluster serves `moduleinstances`, `modulepackages` and `platforms` in `opmodel.dev` but not `transformerregistrations.opmodel.dev`, and `opm-controller-manager` has rolled out
- **THEN** the running-controller check SHALL report the controller as not ready and name `transformerregistrations.opmodel.dev`

### Requirement: The controller counts as running only on a positive finding

The controller SHALL count as running only when every one of these `CustomResourceDefinition`s reports `Established=True` and the `Deployment` has completed its rollout. (Requirement: The running-controller check locates the controller by its fixed names) Otherwise the check SHALL report the controller as not ready, name each object that failed, and point at `opm controller install`.

#### Scenario: No controller at all

- **WHEN** the cluster holds none of the four CRDs and no `opm-controller-manager` Deployment
- **THEN** the running-controller check SHALL report the controller as not ready, naming the missing objects, and point at `opm controller install`

### Requirement: A failed read counts the object as not ready

A read of any of these objects that fails SHALL count that object as not ready, so the command proceeds only on a positive finding. (Requirement: The running-controller check locates the controller by its fixed names)

#### Scenario: A failed read counts the object as not ready

- **WHEN** a read of any of the fixed-name objects of the running-controller check fails
- **THEN** the check SHALL count that object as not ready, so the command proceeds only on a positive finding

### Requirement: Install tells the user when the earlier controller is on the cluster

After a refusal of the apply guard, or when every check passed and before the first write, `opm controller install` (full form and `--crds-only`) SHALL print one line when a successful read finds the Deployment `opm-operator-controller-manager` in the namespace `opm-operator-system`. The line says the cluster still runs the controller of the earlier install and names the docs page "Install over the earlier controller". The refusal, its text and the exit codes SHALL NOT change.

#### Scenario: A refusal on a cluster that holds the earlier controller

- **WHEN** `opm controller install` is refused by the apply guard on a cluster that holds the earlier controller Deployment
- **THEN** it exits 2 with the guard's refusal unchanged
- **AND** the output ends with one line that starts with `hint:` and names the Deployment, the namespace and the docs page

#### Scenario: An install that goes on

- **WHEN** the checks pass on a cluster that holds the earlier controller Deployment, for example because its CRDs carry the adopt annotation
- **THEN** one warning with the same words is printed before the first write
- **AND** the install runs as before and its exit code is unchanged

#### Scenario: The `--crds-only` form

- **WHEN** `opm controller install --crds-only` is refused, or goes on, on the same cluster
- **THEN** it prints the hint line, or the warning, and its exit code is unchanged

### Requirement: The line is a hint after a refusal and a warning when the install goes on

After a refusal of the apply guard the line SHALL start with `hint:` and follow the refusal. When the checks pass and the install goes on, the line SHALL be one warning printed before the first write.

#### Scenario: Placement of the line

- **WHEN** the guard refuses, or the checks pass
- **THEN** the line is the last line of the refusal, or one warning before the first write

### Requirement: Nothing is printed when the earlier Deployment is not found

The line SHALL NOT be printed when the earlier Deployment is absent, when the read of it fails for any reason, or when the install fails before the apply guard gives its verdict. A failed read of the Deployment SHALL NOT change the exit code or the text of any outcome.

#### Scenario: A refusal with no earlier Deployment

- **WHEN** the apply guard refuses a foreign CustomResourceDefinition and the cluster has no Deployment `opm-operator-controller-manager`
- **THEN** the output holds the refusal and no `hint:` line

#### Scenario: A failure before the guard verdict

- **WHEN** the render, the owned-record check or the wait for terminating objects fails on a cluster that holds the earlier controller Deployment
- **THEN** the output holds that failure and no hint and no warning

#### Scenario: A denied read of the Deployment

- **WHEN** the read of the earlier Deployment is forbidden
- **THEN** the output holds no hint and no warning, and the exit code is the one the install would have had

### Requirement: A later failure leaves the printed warning and its exit code

A warning printed before the first write SHALL stay when a later step of the install fails, and the exit code SHALL be the one of the later failure.

#### Scenario: The rollout fails after the warning

- **WHEN** the checks pass on a cluster that holds the earlier controller Deployment, the warning is printed, and the controller rollout then times out
- **THEN** the warning is in the output and the exit code is the one the rollout failure gives

### Requirement: The line does not tell the user to delete an object

The hint line and the warning SHALL point to the docs page and SHALL NOT tell the user to delete or rename an object. The steps that delete the earlier controller SHALL be on the docs page only.

#### Scenario: The line holds no deletion advice

- **WHEN** the hint line or the warning is printed
- **THEN** it does not contain `delete`, `remove or rename` or a `kubectl` command

### Requirement: A docs page says how to move off the earlier controller

The docs site SHALL have a page `Install over the earlier controller` under `diagnostics`. It SHALL say that the earlier install is not migrated and that its controller keeps running. It SHALL give the steps to remove the earlier controller by hand before `opm controller install`, name the earlier objects, and say what happens to the four CustomResourceDefinitions and the ModuleInstances.

#### Scenario: The page names the objects

- **WHEN** a user reads the page
- **THEN** it names the namespace, the Deployment, the four CRDs and the cluster roles and bindings of the earlier install
- **AND** it keeps the CRDs by default and warns that deleting a CRD deletes every object of that kind

### Requirement: The page has a step for the owner value of kept ModuleInstances

After the step that removes the earlier controller and before the step that runs `opm controller install`, the page SHALL tell the user to remove the `spec.owner` field from existing ModuleInstances that hold `operator` and from GitOps manifests. It SHALL say that an absent owner is valid in both schemas and means controller-owned in the new one, and that the install replaces the CRD schema, which then rejects `operator`.

#### Scenario: The order of the steps

- **WHEN** a user reads the steps of the page
- **THEN** the owner-value step comes after the removal of the earlier controller and before `opm controller install`
- **AND** the page says the install replaces the schema, after which `owner: operator` is rejected

### Requirement: The page says that uninstall deletes nothing over an earlier install

The page SHALL say that `opm controller uninstall` deletes nothing on a cluster whose controller has no record, as with an earlier install, and SHALL give the `kubectl` steps instead.

#### Scenario: Uninstall over an earlier install

- **WHEN** a user reads the page's section on uninstall
- **THEN** it states that the command exits 2 and deletes nothing without a record
- **AND** it lists the `kubectl delete` steps
