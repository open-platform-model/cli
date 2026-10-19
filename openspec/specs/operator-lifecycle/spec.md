## Purpose

Operator lifecycle surface: `opm operator install` / `opm operator uninstall`, the pinned operator module the CLI installs as its own instance, readiness waits, uninstall safety, and the CLI's server-side-apply field-manager identity. Slice B2 of enhancement 0006.

## Requirements

### Requirement: All CLI server-side-apply writes use field manager `opm-cli`

Every server-side-apply write the CLI performs — instance/module applies and operator installs alike — SHALL use the field manager `opm-cli`. The former manager name `opm` is retired; ownership of resources previously applied under `opm` transfers on their next apply via the existing forced-conflicts behavior, with no user-visible change.

#### Scenario: One manager identity across the CLI

- **WHEN** any resource is applied by any CLI command after this change
- **THEN** its `managedFields` attribute the CLI's fields to manager `opm-cli`
- **AND** no CLI code path applies with manager `opm`

### Requirement: Catalog resolution precedes every cluster write

When `opm operator install` will seed a Platform, it SHALL resolve the catalog version from the registry before it resolves the kubeconfig, contacts the cluster, or applies any document. A resolution that cannot produce a version SHALL abort the command with nothing applied, so a registry problem can never leave a partially installed cluster.

#### Scenario: No selectable release aborts before install

- **WHEN** `opm operator install` is run and the subscribed catalog major has published no stable release
- **THEN** the command SHALL exit 2 naming `--catalog-prerelease` as the flag that selects a prerelease
- **AND** no operator resource, CRD included, SHALL have been created

#### Scenario: Unreachable registry aborts before install

- **WHEN** the catalog registry cannot be reached
- **THEN** the command SHALL exit 3 naming the lookup and the registry
- **AND** no operator resource SHALL have been created

### Requirement: Platform flags are validated before any lookup or cluster contact

`--catalog-prerelease` SHALL be rejected as a flag-validation error when combined with `--crds-only` or `--skip-platform`, because neither path seeds a Platform and the flag would have no effect. The rejection SHALL occur before any registry lookup, kubeconfig resolution, or cluster call, mirroring the existing rule that `--user`/`--group` require `--rbac`.

#### Scenario: Prerelease flag with CRDs-only

- **WHEN** `opm operator install --crds-only --catalog-prerelease` is run
- **THEN** the command SHALL fail flag validation with an error stating the flag has no effect without Platform seeding
- **AND** no registry lookup and no cluster call SHALL be made

#### Scenario: Prerelease flag with skipped platform

- **WHEN** `opm operator install --skip-platform --catalog-prerelease` is run
- **THEN** the command SHALL fail flag validation before contacting the registry or the cluster

### Requirement: Install refuses an operator instance record that is not CLI-owned

Before any object changes, install SHALL refuse when the operator's instance record `opm-operator` in `opm-operator-system` exists and its `spec.owner` is not `cli`, an absent owner included, since the CLI treats a record without an owner as operator-owned. The refusal SHALL exit 2, name the record, state that its `spec.owner` is not `cli`, and give the command that sets `spec.owner: cli`.

#### Scenario: Operator-owned record

- **WHEN** install runs on a cluster whose record `opm-operator` in `opm-operator-system` has `spec.owner: operator`
- **THEN** install exits 2 naming the record, saying its `spec.owner` is not `cli` and how to set it, and nothing changed

#### Scenario: Record without an owner

- **WHEN** install runs on a cluster whose record `opm-operator` in `opm-operator-system` has no `spec.owner`
- **THEN** install refuses in the same way, and nothing changed

### Requirement: Install succeeds when the installed operator has rolled out

Install SHALL report success only after the rendered CRDs are `Established` and the operator Deployment has completed its rollout, both within the one `--timeout` budget that also covers waiting out terminating objects. When the rollout does not complete in time, install SHALL keep every applied object and the record, roll nothing back, and exit non-zero naming the Deployment and that re-running install completes it. Install SHALL NOT wait for the cluster Platform's readiness.

#### Scenario: Upgrade waits for the rollout

- **WHEN** install upgrades the operator to another module version
- **THEN** install reports success only after the controller Deployment's new ReplicaSet has completed its rollout

#### Scenario: Rollout does not complete

- **WHEN** the new controller pod cannot become ready before `--timeout` elapses
- **THEN** install exits non-zero naming `opm-operator-controller-manager` and the timeout, and every applied object and the record remain

### Requirement: Uninstall keeps the record when a recorded object could not be read

When `opm operator uninstall` reads the objects its instance record lists and the read of one fails with an error other than NotFound, that object SHALL count as a failure for that object: it SHALL be listed with its read error and not deleted, the other recorded objects SHALL still be deleted, the instance record SHALL NOT be deleted, and the command SHALL exit 1 and SHALL NOT report the operator uninstalled.

#### Scenario: Unreadable recorded object keeps the record

- **WHEN** `opm operator uninstall` runs and reading the recorded ClusterRole fails with Forbidden
- **THEN** the recorded ServiceAccount and Deployment SHALL be deleted
- **AND** the ClusterRole SHALL be listed with the Forbidden error and SHALL NOT be deleted
- **AND** the instance record SHALL NOT be deleted
- **AND** the command SHALL exit 1 without reporting the operator uninstalled

### Requirement: Uninstall fails when the instance record cannot be deleted

When `opm operator uninstall` deleted every recorded object and the delete of the operator's instance record then fails with an error other than NotFound, the command SHALL NOT report the operator uninstalled and SHALL exit non-zero. The error SHALL name the record and the cause, and SHALL say that the recorded objects were deleted, that the record remains and that re-running is safe.

#### Scenario: Forbidden record delete

- **WHEN** `opm operator uninstall` runs and the delete of the instance record fails with Forbidden
- **THEN** the recorded ClusterRole, ServiceAccount and Deployment SHALL be deleted
- **AND** the instance record SHALL still exist
- **AND** the command SHALL exit 4 without reporting the operator uninstalled

### Requirement: Install with `--rbac` creates the `opm-cli-user` role

`opm operator install --rbac` SHALL additionally create a ClusterRole `opm-cli-user` granting full verbs on `moduleinstances`, `get/patch/update` on `moduleinstances/status`, and `get/list` on `platforms`. When `--user <U>` or `--group <G>` is supplied alongside `--rbac`, the command SHALL also create a ClusterRoleBinding binding that subject to the role. Without `--rbac`, no RBAC objects beyond those of the operator module's render are created.

#### Scenario: RBAC off by default

- **WHEN** `opm operator install --crds-only` is run
- **THEN** no `opm-cli-user` ClusterRole or ClusterRoleBinding is created

#### Scenario: Role plus binding for a user

- **WHEN** `opm operator install --crds-only --rbac --user alice` is run
- **THEN** the `opm-cli-user` ClusterRole and a ClusterRoleBinding for user `alice` are applied with field manager `opm-cli`

### Requirement: The `--user` and `--group` flags require `--rbac`

`--user`/`--group` without `--rbac` SHALL be rejected as a flag-validation error before any cluster interaction.

#### Scenario: Subject flags require --rbac

- **WHEN** `opm operator install --user alice` is run without `--rbac`
- **THEN** the command fails flag validation with an error stating `--user`/`--group` require `--rbac`, before contacting the cluster

### Requirement: Uninstall refuses and names operator instances that carry the cleanup finalizer

Before deleting anything, `opm operator uninstall` SHALL list `ModuleInstance` resources cluster-wide; if any carries the `opmodel.dev/cleanup` finalizer, the command SHALL refuse (exit non-zero) and name each such instance as `namespace/name`. If the cluster-wide list fails (including RBAC denial), the command SHALL fail closed without deleting anything.

#### Scenario: Refusal names the armed instances

- **WHEN** `opm operator uninstall` is run while `default/jellyfin` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command exits non-zero without deleting anything
- **AND** the error names `default/jellyfin` and points at `--remove-finalizers` and its orphaning consequence

#### Scenario: List failure fails closed

- **WHEN** the uninstalling user lacks permission to list `moduleinstances` cluster-wide
- **THEN** the command exits non-zero without deleting anything, mapping the RBAC error to the standard permission-denied exit code

### Requirement: Uninstall removes only the cleanup finalizer on request

With `--remove-finalizers`, `opm operator uninstall` SHALL attempt to remove exactly the `opmodel.dev/cleanup` finalizer (leaving all other finalizers intact) from every instance that carries it, state that the instances and their workloads are now orphaned, and then proceed to delete only if every such instance was successfully stripped.

#### Scenario: Override strips only the operator's finalizer

- **WHEN** `opm operator uninstall --remove-finalizers` is run while an instance carries both `opmodel.dev/cleanup` and a foreign finalizer
- **THEN** `opmodel.dev/cleanup` is removed from the instance, the foreign finalizer remains
- **AND** the command states the orphaning consequence and proceeds with the uninstall

#### Scenario: Partial finalizer-removal failure blocks the delete step

- **WHEN** `--remove-finalizers` is given and stripping the finalizer fails for one of several armed instances
- **THEN** the command still attempts every armed instance rather than stopping at the first failure
- **AND** the returned error names every instance that failed
- **AND** the operator's own resources are not deleted unless every armed instance was successfully stripped

### Requirement: Catalog resolution failures map to the standard exit codes

When `opm operator install` will seed a Platform, resolution failures SHALL map to the CLI's standard exit codes: a catalog major with no selectable version is a validation refusal (exit 2) that names the flag which would select a prerelease, and an unreachable or unreadable registry is a connectivity failure (exit 3).

#### Scenario: Unreadable registry is a connectivity failure

- **WHEN** the catalog registry is unreadable
- **THEN** the command SHALL exit 3 as a connectivity failure
- **AND** no operator resource SHALL have been created

### Requirement: Install renders the operator module as the instance `opm-operator`

`opm operator install` SHALL obtain the operator module `opmodel.dev/modules/opm_operator` from the CLI's configured module registry and render it as an instance named `opm-operator` in the namespace `opm-operator-system`. Apart from the render, install SHALL create only the cluster Platform and the opt-in `opm-cli-user` role. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Install creates the render plus the Platform and the role

- **WHEN** `opm operator install` runs
- **THEN** it renders `opmodel.dev/modules/opm_operator` from the CLI's configured module registry as the instance `opm-operator` in the namespace `opm-operator-system`
- **AND** apart from the render it creates only the cluster Platform and the opt-in `opm-cli-user` role

### Requirement: Install applies the CRDs before it applies the instance

Install SHALL apply the rendered `CustomResourceDefinition` objects with server-side apply as field manager `opm-cli` and wait until each reports `Established=True`, and then apply the whole render as a CLI-owned ModuleInstance (`spec.owner: cli`) through the same apply the CLI uses for every instance. The instance record SHALL be written only after the CRDs are served.

#### Scenario: Fresh cluster

- **WHEN** `opm operator install` runs against a cluster with none of OPM's CRDs
- **THEN** the four CRDs are applied and `Established` before any other object is applied
- **AND** the ModuleInstance `opm-operator` in `opm-operator-system` exists with `spec.owner: cli`, and its inventory lists every rendered object, the four CRDs and the Namespace included
- **AND** the operator Deployment is running

### Requirement: The operator instance records every object of the render

Every object the module renders, the CRDs and the Namespace included, SHALL be recorded in the instance's inventory, and no step SHALL refuse an object the same install applied. Install SHALL NOT create the Namespace outside the render.

#### Scenario: The module's Namespace is not refused

- **WHEN** install runs on a cluster where `opm-operator-system` does not exist
- **THEN** the Namespace is created by the instance apply as one of its objects, and the apply does not refuse it as a foreign object

### Requirement: Reinstall changes only what the module version changes

Re-running install with an unchanged module version and unchanged values SHALL change no live object, and installing another module version over the instance SHALL apply what changed and remove every object the previous version rendered and the new one does not, except CRDs and the Namespace. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Unchanged reinstall

- **WHEN** install runs a second time with the same module version and no new values
- **THEN** every object keeps its uid and resourceVersion, and the command reports every resource unchanged

#### Scenario: Upgrade drops an object the new version no longer renders

- **WHEN** install of module version B runs over an instance of version A, and A rendered a ClusterRole that B does not
- **THEN** that ClusterRole is deleted, the CRDs and the Namespace stay, and the record lists B's objects

### Requirement: Install does not change the owner of the operator instance record

Install SHALL NOT change the owner itself: the instance apply would otherwise only edit the record's spec, and the operator never reconciles the instance that deploys it, so nothing would install.

#### Scenario: Refusal leaves the owner as it was

- **WHEN** install refuses a record whose `spec.owner` is not `cli`
- **THEN** install has not changed the owner of the record

### Requirement: Each CLI release pins a default operator module version

Each CLI release SHALL name one default operator module version and SHALL record the operator version that module deploys, both readable from the CLI's source without a registry. The recorded operator version SHALL be the version the module's own source states it deploys. Install without `--version` SHALL install exactly that module version. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Default install uses the pin

- **WHEN** `opm operator install` runs without `--version`
- **THEN** install renders the pinned module version and its output names that module version and the recorded operator version

### Requirement: Only `task operator:pin` writes the pin, from the module's own statement

The pin SHALL be produced by `task operator:pin VERSION=<v>` from the module's own statement of the operator it deploys, without a render, and no other task SHALL write it. The module is trusted as the configured registry serves it, as every other module the CLI installs is.

#### Scenario: Pin refresh is one reviewable diff

- **WHEN** `task operator:pin VERSION=0.2.0` runs and module `0.2.0` states it deploys operator `1.0.0-beta.6`
- **THEN** only the pin file changes, carrying module version `0.2.0` and operator version `v1.0.0-beta.6`

#### Scenario: A module without a readable operator version is refused by the pin task

- **WHEN** `task operator:pin VERSION=<v>` runs against a module version whose source states no operator version, or a non-semver one
- **THEN** the task exits non-zero naming the module version, and the pin file is unchanged

### Requirement: Install refuses a target operator newer than the CLI

Before any object changes, install SHALL refuse a target module version whose stated operator `MAJOR.MINOR` is above the CLI's own, with exit 2, naming the module version and the rule it failed and naming upgrading the CLI as the fix. A CLI whose own version is not a released semver SHALL skip this rule with a warning. Source: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Operator newer than the CLI

- **WHEN** the target module states operator `1.1.0` and the CLI is `1.0.0-beta.9`
- **THEN** install exits 2 naming the module version, the operator version and upgrading the CLI, and nothing changed

### Requirement: Install refuses a target whose render disagrees with its stated operator version

Before any object changes, install SHALL refuse a target module version whose render does not run the controller Deployment `opm-operator-controller-manager` with an image tagged with the operator version the module states, with exit 2, naming the module version and the rule it failed.

#### Scenario: Rendered image disagrees with the stated operator version

- **WHEN** the target module states operator `1.0.0-beta.6` and its rendered controller image is tagged `v1.0.0-beta.5`
- **THEN** install exits 2 naming the module version and both versions, and nothing changed

### Requirement: Install refuses a target whose ModuleInstance CRD lacks the required fields

Before any object changes, install SHALL refuse a target module version whose rendered ModuleInstance CRD does not carry the fields the CLI requires of that CRD before it applies an instance, with exit 2, naming the module version and the rule it failed and naming the floor.

#### Scenario: Rendered CRD below the floor

- **WHEN** the target's rendered ModuleInstance CRD lacks `spec.owner`
- **THEN** install exits 2 naming the module version and the floor, before the CRD step

### Requirement: A newer running operator does not block install

Install SHALL NOT be refused because the operator the cluster Platform reports has a `MAJOR.MINOR` above the CLI's, since install replaces the running operator rather than driving it. A target below the recorded module version or the running operator SHALL NOT be refused by the target-version rules of install. Source: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Newer running operator does not block repair

- **WHEN** the cluster Platform reports operator `1.1.0`, the CLI is `1.0.0`, and the target module states operator `1.0.2`
- **THEN** the running-operator ceiling does not refuse install, and install proceeds

### Requirement: Install layers values over the values recorded on the operator instance

Install SHALL take values for the operator module from `-f/--values` files, layered in order over the values recorded on the operator's instance: maps merge, and a scalar or list given in a later source replaces the earlier one. `--reset-values` SHALL drop the recorded values for that run. The module's `debugValues` SHALL NOT be used. Source: enhancement 0006 ("`spec.values` is the sole authoritative render input; the CLI unifies all value inputs into it; resolves OQ10").

#### Scenario: Changing one value keeps the others

- **WHEN** the record holds a registry mapping and a replica count, and install runs with a file setting only the replica count
- **THEN** the render carries the new replica count and the recorded registry mapping

### Requirement: Recorded operator values survive reinstall

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

- **WHEN** `opm operator install --crds-only -f values.cue` runs
- **THEN** the command fails flag validation before any registry or cluster call

### Requirement: Install seeds the cluster Platform after the operator rolls out

After the operator's rollout, and unless `--skip-platform` or `--crds-only` is given, install SHALL create the singleton `cluster` Platform subscribing to the catalog build it resolved before contacting the cluster, with a plain create as field manager `opm-cli`, never server-side apply or update. Source: enhancement 0006 ("`Platform` carries no owner marker; the operator always owns the singleton; the CLI writes an un-owned `cluster` Platform in solo clusters").

#### Scenario: Platform is seeded on a full install

- **WHEN** install completes against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` exists, subscribing to the resolved catalog build, and the output names the catalog module path and version

### Requirement: An existing Platform is left alone and a denied create is a warning

An existing Platform SHALL be reported and left untouched, an `AlreadyExists` response SHALL be a success-noop, and a create denied by RBAC SHALL degrade to a warning. Source: enhancement 0006 ("Solo-cluster `Platform` write-if-absent uses a plain `Create` that tolerates `AlreadyExists`, not SSA apply; refines D12's mechanism, resolves OQ13").

#### Scenario: Existing Platform is left untouched

- **WHEN** install runs against a cluster that already carries a `Platform` CR
- **THEN** the stored Platform is unchanged and the command reports it was already present

#### Scenario: Platform create denied by RBAC

- **WHEN** the installing user may apply the operator but may not create `platforms`
- **THEN** the command warns that the Platform was not seeded, names the missing permission, and does not fail because of it

### Requirement: The Platform and the opt-in user role are not part of the render or the inventory

Neither the Platform nor the `opm-cli-user` role of `--rbac` SHALL be part of the operator module's render or the instance's inventory. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Platform and role are not recorded

- **WHEN** install with `--rbac --user alice` completes
- **THEN** neither `Platform/cluster` nor the `opm-cli-user` ClusterRole or its binding appears in the operator instance's inventory

### Requirement: `--rbac` applies on the CRDs-only path as on the full one

`--rbac` SHALL apply on the CRDs-only path as on the full one. Source: enhancement 0006 ("Status-subresource RBAC (opt-in `--rbac` emission + a pre-flight access check, both as edge-case safety nets); resolves OQ3").

#### Scenario: `--rbac` on the CRDs-only path

- **WHEN** `opm operator install --crds-only --rbac --user alice` is run
- **THEN** `--rbac` applies as on the full path and the `opm-cli-user` ClusterRole and a ClusterRoleBinding for user `alice` are applied

### Requirement: CRDs-only install applies exactly the rendered CRDs

`opm operator install --crds-only` SHALL render the same module version as a full install would, apply exactly the rendered `CustomResourceDefinition` objects with server-side apply as `opm-cli`, and wait for each to be `Established`. It SHALL write no instance record, no workload and no Platform, and SHALL perform no catalog lookup. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Solo-cluster CRD install

- **WHEN** `opm operator install --crds-only` runs against an empty cluster
- **THEN** exactly the four rendered CRDs exist and are `Established`, and no ModuleInstance, Deployment or Platform exists

### Requirement: CRDs-only install refuses what a full install refuses before it applies a CRD

`opm operator install --crds-only` SHALL refuse under the operator-newer-than-CLI rule, the rendered-image rule and the CRD floor before applying any CRD. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: CRDs-only refuses a target newer than the CLI

- **WHEN** `opm operator install --crds-only` runs and the target module states an operator `MAJOR.MINOR` above the CLI's
- **THEN** the command refuses before applying any CRD

### Requirement: A full install records the CRDs a CRDs-only install applied

A later full install of the same module version SHALL record those CRDs without recreating them. (Requirement: CRDs-only install applies exactly the rendered CRDs) Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Full install after CRDs-only

- **WHEN** a full install of the same module version follows a CRDs-only install
- **THEN** the four CRDs keep their uid and appear in the new record's inventory

### Requirement: Uninstall without an instance record deletes nothing

`opm operator uninstall` SHALL read the operator's instance record `opm-operator` in `opm-operator-system`. When none exists, it SHALL delete nothing and exit 2 naming `opm operator install` as the step that records the running operator. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: No record

- **WHEN** `opm operator uninstall` runs on a cluster whose operator was applied with kubectl and has no instance record
- **THEN** it exits 2 naming `opm operator install`, and no object is deleted

### Requirement: Uninstall deletes the recorded objects except the CRDs and the Namespace

After the finalizer guard, `opm operator uninstall` SHALL delete every object the record's inventory lists except CRDs and the Namespace, in descending resource-weight order, then, when no object failed, delete the record. Source: enhancement 0006 ("`opm operator uninstall` preserves the namespace; refuses while cleanup finalizers are armed, with `--remove-finalizers` as the override").

#### Scenario: Uninstall after a module install

- **WHEN** `opm operator uninstall` runs on a cluster with no armed ModuleInstance
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

`opm operator uninstall` SHALL delete no object the inventory does not list, SHALL NOT wait for deletion to complete, and SHALL treat an already absent object as deleted.

#### Scenario: Re-running uninstall

- **WHEN** uninstall runs again after an earlier run deleted the objects but failed to delete the record
- **THEN** it treats the absent objects as deleted, deletes the record and exits zero

### Requirement: Uninstall leaves an unreadable Namespace or CRD behind without failing

When the read of a recorded `Namespace` or `CustomResourceDefinition` fails with an error other than NotFound, `opm operator uninstall` SHALL leave that object behind, as it would if the read had worked, and it SHALL NOT count as a failure.

#### Scenario: Unreadable recorded Namespace is left behind

- **WHEN** `opm operator uninstall` runs and reading the recorded `Namespace` fails with Forbidden
- **THEN** the `Namespace` SHALL be left behind, as it would be if read
- **AND** it SHALL NOT count as a failure

### Requirement: Re-running uninstall after a read failure deletes the rest

Re-running `opm operator uninstall` after the cause of a failed read is fixed SHALL delete the remaining objects and the record.

#### Scenario: Re-running uninstall after the read cause is fixed

- **WHEN** `opm operator uninstall` is run again after the cause of the failed read is fixed
- **THEN** it SHALL delete the remaining objects and the record

### Requirement: A failed record delete exits by the cause of the failure

When `opm operator uninstall` deleted every recorded object and the delete of the operator's instance record then fails with an error other than NotFound, the exit code SHALL be 4 when the API server denied the delete (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure.

#### Scenario: Record delete times out

- **WHEN** `opm operator uninstall` runs and the delete of the instance record fails with a server timeout
- **THEN** the command SHALL exit 3 without reporting the operator uninstalled

#### Scenario: Record delete fails for another reason

- **WHEN** `opm operator uninstall` runs and the delete of the instance record fails with an error that is not a denial, a server timeout or service unavailable
- **THEN** the command SHALL exit 1 without reporting the operator uninstalled

### Requirement: Uninstall takes every action on a recorded object from the shared deletion plan

`opm operator uninstall` SHALL take every action on a recorded object from the deletion plan `opm instance delete` uses (capability `deploy`, "Instance delete follows the shared deletion plan"), built from the objects of the operator's instance record that the command found or could not read, and from the identity stored in the record. The output lines and the exit codes of the command SHALL stay as the other requirements of this capability state them.

#### Scenario: Recorded objects are deleted in the plan's order

- **WHEN** `opm operator uninstall` runs on a cluster with no armed ModuleInstance
- **THEN** the delete request for the controller Deployment SHALL be sent before the ones for its ServiceAccount and ClusterRole
- **AND** each delete request SHALL carry Foreground propagation and a precondition on the UID of the object that was read

### Requirement: Uninstall acts on an object and on the record only as the plan allows

`opm operator uninstall` SHALL read a recorded object, delete it or leave it in place only when the plan names that action as the next one, and SHALL delete the instance record only when the plan's release verdict allows it. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Failed object holds the record

- **WHEN** the delete of the recorded ClusterRole fails with Forbidden
- **THEN** the other recorded objects SHALL still be deleted
- **AND** the instance record SHALL NOT be deleted
- **AND** the command SHALL exit 1 without reporting the operator uninstalled

### Requirement: The finalizer guard and the no-record refusal run before the plan is built

The finalizer guard and the no-record refusal of `opm operator uninstall` SHALL run before it builds the deletion plan.

#### Scenario: Guard refuses before any plan

- **WHEN** a ModuleInstance still carries the operator's cleanup finalizer and `--remove-finalizers` is not set
- **THEN** no recorded object SHALL be read for a delete verdict and none SHALL be deleted

### Requirement: Install runs every check that can refuse it before it writes

Install SHALL run, before it writes any object, every check that can refuse it: resolving the module and reading the operator version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the operator version against the CLI's, the rendered image against the module's stated operator version, and the rendered ModuleInstance CRD against the CLI's floor).

#### Scenario: The module, render, values and target rules are checked before the first write

- **WHEN** install runs
- **THEN** it SHALL run the resolving of the module, the rendering, the check of the values against the target version's `#config` and the target rules of install before it writes any object

### Requirement: Install runs its status-subresource permission check and the apply guard before it writes

Install SHALL run, before it writes any object, every check that can refuse it: the status-subresource permission check, and the apply guard over every object the render names (capability `apply-pruning`, "The ownership guard judges every apply"). An install that refuses SHALL change no object in the cluster.

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

### Requirement: Install writes in a fixed order after every check passed

Only after every check passed SHALL install write, in this order: the CRD step, then the instance apply, then the Platform seed. Outside the instance apply and its prune, install SHALL send no delete and no patch of managed fields.

#### Scenario: Install writes the CRD step, the instance apply and the Platform seed in order

- **WHEN** every check of install passed
- **THEN** install writes in this order: the CRD step, then the instance apply, then the Platform seed

### Requirement: The apply guard runs on every install

The apply guard over every object the render names (capability `apply-pruning`, "The ownership guard judges every apply") SHALL run on every install, with or without a record of the operator's instance. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: CRDs-only refuses unlabelled CRDs

- **WHEN** `opm operator install --crds-only` runs and a CRD the render names exists without an OPM managed-by label and without an adopt annotation
- **THEN** the command SHALL exit 2 naming that CRD, and no CRD has changed

### Requirement: Install passes the guard no admission set and no proof of origin

Install SHALL pass the guard no admission set and SHALL run no proof of where an existing object came from: an object that an earlier opm-operator release manifest created is judged as any other existing object.

#### Scenario: Unlabelled CRDs of an earlier manifest install are refused

- **WHEN** install runs on a cluster whose four `opmodel.dev` CRDs, Namespace and Deployment were applied from an opm-operator release manifest and carry no OPM managed-by label and no adopt annotation
- **THEN** the command SHALL exit 2 with `refusing to install:` and one line for each such object that says it exists and is not managed by OPM, and names `opmodel.dev/adopt` with the UUID to set
- **AND** install SHALL adopt none of them by itself, and no object in the cluster has changed

### Requirement: The guard judges an object the record lists as recorded

An object the record lists is judged as recorded and SHALL pass whatever its UUID label says. (Requirement: The apply guard runs on every install)

#### Scenario: Reinstall over its own recorded objects passes the guard

- **WHEN** install runs on a cluster with a record of the operator's instance and every existing object the render names is in that record
- **THEN** the apply guard refuses none of them

#### Scenario: Recorded objects that carry another identity are applied

- **WHEN** install runs on a cluster with a record of the operator's instance that lists every object the render names
- **AND** every one of those objects, the four CRDs and the Namespace included, carries an OPM managed-by label and the UUID of another instance
- **THEN** the apply guard refuses none of them and install applies them

### Requirement: The guard judges an existing object the record does not list

An existing object the record does not list SHALL pass when its `opmodel.dev/adopt` annotation holds the UUID of the operator's instance, or when it carries an OPM managed-by label and no UUID label of another instance; it SHALL refuse the install when OPM does not manage it or when it carries another instance's identity.

#### Scenario: Annotated objects that carry another identity are applied

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** every object the render names exists with the UUID of another instance and `opmodel.dev/adopt` holding the UUID of the operator's instance
- **THEN** the apply guard refuses none of them, and install applies and records them

#### Scenario: The instance's own objects without a record pass the guard

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** every existing object the render names carries an OPM managed-by label and the UUID of the operator's instance (a run that stopped before it wrote the record, a CRDs-only install, or a `kubectl apply` of the module's render)
- **THEN** the apply guard refuses none of them

#### Scenario: An unrecorded object with the instance's UUID and no OPM managed-by label is refused

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** an object the render names carries the UUID of the operator's instance, and its managed-by label is absent or names another tool
- **THEN** the command SHALL exit 2 naming that object as not managed by OPM, and `opmodel.dev/adopt` with the UUID to set

#### Scenario: Reinstall refuses a foreign object new to the inventory

- **WHEN** install runs on a cluster with a record of the operator's instance
- **AND** the render names a ClusterRole the record does not list, which exists without an OPM managed-by label
- **THEN** the command SHALL exit 2 naming that ClusterRole and the adopt annotation, and no object in the cluster has changed

#### Scenario: First install refuses an object of another instance

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** a CustomResourceDefinition the render names exists with an OPM managed-by label and the UUID of another instance
- **THEN** the command SHALL exit 2 naming that CustomResourceDefinition and the other instance, and no object in the cluster has changed

### Requirement: Install refuses an object adopted by another instance

Install needs every object its render names, so, unlike any other apply, it SHALL also refuse when the `opmodel.dev/adopt` annotation of an object names another instance, whether or not the record lists the object: it SHALL NOT leave the object unapplied and go on (owner decision of 2026-10-08 on `opm operator install`).

#### Scenario: Install refuses an object adopted by another instance

- **WHEN** the controller Deployment, a rendered CustomResourceDefinition or the operator Namespace carries `opmodel.dev/adopt` with the UUID of another instance
- **THEN** the command SHALL exit 2 naming that object and that instance
- **AND** no object in the cluster has changed, the managed fields of that object included

### Requirement: Every refusal of the guard exits 2 and names the object, the instance and the annotation

Every refusal of the guard before the first write SHALL exit 2, SHALL name the object and the instance that owns or adopts it, SHALL name the `opmodel.dev/adopt` annotation and the UUID to set where the annotation lifts the refusal, and SHALL leave the cluster unchanged; that includes a failed read by the guard, with or without a record. No refusal of install SHALL tell the user to delete or rename an object.

#### Scenario: Objects that carry another identity, with no record and no annotation, are refused

- **WHEN** install runs on a cluster with no record of the operator's instance
- **AND** every object the render names exists with an OPM managed-by label and the UUID of another instance, and none carries an adopt annotation
- **THEN** the command SHALL exit 2 naming each object, the other instance and `opmodel.dev/adopt` with the UUID to set
- **AND** the output SHALL NOT contain `operator migration refused` or `remove or rename`
- **AND** no object in the cluster has changed

### Requirement: Install waits out terminating objects before the guard runs

Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied.

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm operator uninstall` has returned and `opm operator install` runs before the deleted Deployment has been garbage-collected
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

- **WHEN** install runs on a cluster with a record of the operator's instance, the earlier read of a rendered object was answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed

### Requirement: A failed read in the wait for terminating objects exits by the read error

When the wait for terminating objects fails a read with an error other than NotFound, the command SHALL exit with the code of the read error: 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. An object that install cannot read from the start is therefore refused by the wait for terminating objects, with exit code 4 when the read is denied.

#### Scenario: An object unreadable from the start exits by the read error

- **WHEN** install runs on a cluster with no record of the operator's instance and every read of a ClusterRole the render names fails with Forbidden
- **THEN** the command SHALL exit 4 naming that ClusterRole and the read error, and no object in the cluster has changed

### Requirement: `--version` resolves the operator module through the configured registry

`opm operator install --version <selector>` SHALL resolve the operator module version through the CLI's configured registry, a release version pinning it and a major (`v0`) floating to that major's newest release, and SHALL install it through the same render, checks and steps as the default. Install SHALL report the module version and the operator version it deploys. Source: enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Selecting another version

- **WHEN** `opm operator install --version 0.2.0` runs and the registry serves that version
- **THEN** install applies it and its output names module `0.2.0` and the operator version it deploys

### Requirement: Install refuses a selector the registry cannot satisfy before any cluster call

A selector the registry cannot satisfy, or that is not a version selector, SHALL refuse before any cluster call; a selector that has the shape of an opm-operator release tag SHALL get no message of its own.

#### Scenario: Unserved version

- **WHEN** `opm operator install --version 1.0.0-beta.5` runs and no module version matches
- **THEN** the command exits 2 before contacting the cluster

### Requirement: A registry mirror is the only registry install needs

With the CLI's registry mapping pointed at a mirror holding the module and its dependencies, install SHALL need no other registry.

#### Scenario: Mirror only

- **WHEN** the registry mapping routes `opmodel.dev` to a mirror holding the module and its dependencies, and the public registry is unreachable
- **THEN** install succeeds

### Requirement: The running-operator check locates the operator by its fixed names

Every CLI command that checks for a running operator before it acts SHALL locate the operator by its fixed names, reading no instance record. The operator's objects SHALL be its fixed names, which every install of the operator module keeps: the `Deployment` `opm-operator-controller-manager` in the namespace `opm-operator-system`, and the `CustomResourceDefinition`s `moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev` and `transformerregistrations.opmodel.dev`.

#### Scenario: Module-installed operator is found by the same names

- **WHEN** the cluster holds the `ModuleInstance` `opm-operator` in `opm-operator-system` whose render created the four CRDs and the `Deployment` `opm-operator-controller-manager` in `opm-operator-system`
- **AND** every CRD is `Established` and the `Deployment` has rolled out
- **THEN** the running-operator check SHALL report the operator as running

### Requirement: The running-operator check reads no Namespace and reports a missing CRD as not ready

The `Namespace` is checked only as the namespace of the `Deployment` `opm-operator-controller-manager`; the check reads no `Namespace` object and no instance record. An operator that lacks one of these CRDs (as listed in requirement "The running-operator check locates the operator by its fixed names") SHALL be reported as not ready.

#### Scenario: Operator older than the fourth CRD is not ready

- **WHEN** the cluster serves `moduleinstances`, `modulepackages` and `platforms` in `opmodel.dev` but not `transformerregistrations.opmodel.dev`, and `opm-operator-controller-manager` has rolled out
- **THEN** the running-operator check SHALL report the operator as not ready and name `transformerregistrations.opmodel.dev`

### Requirement: The operator counts as running only on a positive finding

The operator SHALL count as running only when every one of these `CustomResourceDefinition`s reports `Established=True` and the `Deployment` has completed its rollout. (Requirement: The running-operator check locates the operator by its fixed names) Otherwise the check SHALL report the operator as not ready, name each object that failed, and point at `opm operator install`.

#### Scenario: No operator at all

- **WHEN** the cluster holds none of the four CRDs and no `opm-operator-controller-manager` Deployment
- **THEN** the running-operator check SHALL report the operator as not ready, naming the missing objects, and point at `opm operator install`

### Requirement: A failed read counts the object as not ready

A read of any of these objects that fails SHALL count that object as not ready, so the command proceeds only on a positive finding. (Requirement: The running-operator check locates the operator by its fixed names)

#### Scenario: A failed read counts the object as not ready

- **WHEN** a read of any of the fixed-name objects of the running-operator check fails
- **THEN** the check SHALL count that object as not ready, so the command proceeds only on a positive finding
