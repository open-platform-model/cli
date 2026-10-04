## ADDED Requirements

### Requirement: Install deploys the operator module as a CLI-owned instance in two steps

`opm operator install` SHALL obtain the operator module `opmodel.dev/modules/opm_operator` from the CLI's configured module registry, render it as an instance named `opm-operator` in the namespace `opm-operator-system`, apply the rendered `CustomResourceDefinition` objects with server-side apply as field manager `opm-cli` and wait until each reports `Established=True`, and then apply the whole render as a CLI-owned ModuleInstance (`spec.owner: cli`) through the same apply the CLI uses for every instance. The instance record SHALL be written only after the CRDs are served. Every object the module renders, the CRDs and the Namespace included, SHALL be recorded in the instance's inventory, and no step SHALL refuse an object the same install applied. Install SHALL NOT create the Namespace outside the render. Apart from the render, install SHALL create only the cluster Platform and the opt-in `opm-cli-user` role. Re-running install with an unchanged module version and unchanged values SHALL change no live object, and installing another module version over the instance SHALL apply what changed and remove every object the previous version rendered and the new one does not, except CRDs and the Namespace. Source: 0021:D11:R1, 0021:D11:R2.

#### Scenario: Fresh cluster

- **WHEN** `opm operator install` runs against a cluster with none of OPM's CRDs
- **THEN** the four CRDs are applied and `Established` before any other object is applied
- **AND** the ModuleInstance `opm-operator` in `opm-operator-system` exists with `spec.owner: cli`, and its inventory lists every rendered object, the four CRDs and the Namespace included
- **AND** the operator Deployment is running

#### Scenario: The module's Namespace is not refused

- **WHEN** install runs on a cluster where `opm-operator-system` does not exist
- **THEN** the Namespace is created by the instance apply as one of its objects, and the apply does not refuse it as a foreign object

#### Scenario: Unchanged reinstall

- **WHEN** install runs a second time with the same module version and no new values
- **THEN** every object keeps its uid and resourceVersion, and the command reports every resource unchanged

#### Scenario: Upgrade drops an object the new version no longer renders

- **WHEN** install of module version B runs over an instance of version A, and A rendered a ClusterRole that B does not
- **THEN** that ClusterRole is deleted, the CRDs and the Namespace stay, and the record lists B's objects

### Requirement: Every check that can refuse install runs before its first write

Install SHALL run, before it writes any object, every check that can refuse it: resolving the module and reading the operator version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the operator version against the CLI's, the rendered image against the module's stated operator version, and the rendered ModuleInstance CRD against the CLI's floor), the status-subresource permission check, and the apply guard over every object the render names. Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied. Only after every check passed SHALL install write, in this order: the CRD step, then the instance apply, then the Platform seed. An install that refuses SHALL change no object in the cluster.

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm operator uninstall` has returned and `opm operator install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies only after it is gone, and completes

#### Scenario: Terminating object outlives the budget

- **WHEN** an object of the render stays terminating for longer than `--timeout`
- **THEN** the command exits non-zero naming that object and the elapsed timeout, and nothing has been applied

### Requirement: Install refuses an operator instance record that is not CLI-owned

Before any object changes, install SHALL refuse when the operator's instance record `opm-operator` in `opm-operator-system` exists and its `spec.owner` is not `cli`, an absent owner included, since the CLI treats a record without an owner as operator-owned. The refusal SHALL exit 2, name the record, state that its `spec.owner` is not `cli`, and give the command that sets `spec.owner: cli`. Install SHALL NOT change the owner itself: the instance apply would otherwise only edit the record's spec, and the operator never reconciles the instance that deploys it, so nothing would install.

#### Scenario: Operator-owned record

- **WHEN** install runs on a cluster whose record `opm-operator` in `opm-operator-system` has `spec.owner: operator`
- **THEN** install exits 2 naming the record, saying its `spec.owner` is not `cli` and how to set it, and nothing changed

#### Scenario: Record without an owner

- **WHEN** install runs on a cluster whose record `opm-operator` in `opm-operator-system` has no `spec.owner`
- **THEN** install refuses in the same way, and nothing changed

### Requirement: The CLI pins a default operator module version and the operator it deploys

Each CLI release SHALL name one default operator module version and SHALL record the operator version that module deploys, both readable from the CLI's source without a registry. The recorded operator version SHALL be the version the module's own source states it deploys. Install without `--version` SHALL install exactly that module version. The pin SHALL be produced by `task operator:pin VERSION=<v>` from the module's own statement of the operator it deploys, without a render, and no other task SHALL write it. The module is trusted as the configured registry serves it, as every other module the CLI installs is. Source: 0021:D11:R6.

#### Scenario: Default install uses the pin

- **WHEN** `opm operator install` runs without `--version`
- **THEN** install renders the pinned module version and its output names that module version and the recorded operator version

#### Scenario: Pin refresh is one reviewable diff

- **WHEN** `task operator:pin VERSION=0.2.0` runs and module `0.2.0` states it deploys operator `1.0.0-beta.6`
- **THEN** only the pin file changes, carrying module version `0.2.0` and operator version `v1.0.0-beta.6`

#### Scenario: A module without a readable operator version is refused by the pin task

- **WHEN** `task operator:pin VERSION=<v>` runs against a module version whose source states no operator version, or a non-semver one
- **THEN** the task exits non-zero naming the module version, and the pin file is unchanged

### Requirement: Another module version is installed from the registry as served

`opm operator install --version <selector>` SHALL resolve the operator module version through the CLI's configured registry, a release version pinning it and a major (`v0`) floating to that major's newest release, and SHALL install it through the same render, checks and steps as the default. Install SHALL report the module version and the operator version it deploys. A selector the registry cannot satisfy SHALL refuse before any cluster call, and when the selector has the shape of an opm-operator release tag, the message SHALL say that `--version` now takes an operator module version. With the CLI's registry mapping pointed at a mirror holding the module and its dependencies, install SHALL need no other registry. Source: 0021:D11:R1, 0021:D11:R2.

#### Scenario: Selecting another version

- **WHEN** `opm operator install --version 0.2.0` runs and the registry serves that version
- **THEN** install applies it and its output names module `0.2.0` and the operator version it deploys

#### Scenario: Old-style operator tag

- **WHEN** `opm operator install --version v1.0.0-beta.5` runs and no module version matches
- **THEN** the command fails before contacting the cluster, saying that `--version` takes an operator module version

#### Scenario: Mirror only

- **WHEN** the registry mapping routes `opmodel.dev` to a mirror holding the module and its dependencies, and the public registry is unreachable
- **THEN** install succeeds

### Requirement: Install refuses a target version the CLI cannot drive or that disagrees with itself

Before any object changes, install SHALL refuse a target module version (each refusal naming the module version and the rule it failed, exit 2):

- whose stated operator `MAJOR.MINOR` is above the CLI's own, naming upgrading the CLI as the fix; a CLI whose own version is not a released semver SHALL skip this rule with a warning;
- whose render does not run the controller Deployment `opm-operator-controller-manager` with an image tagged with the operator version the module states;
- whose rendered ModuleInstance CRD does not carry the fields the CLI requires of that CRD before it applies an instance, naming the floor.

Install SHALL NOT be refused because the operator the cluster Platform reports has a `MAJOR.MINOR` above the CLI's, since install replaces the running operator rather than driving it. A target below the recorded module version or the running operator SHALL NOT be refused by this requirement. Source: 0021:D9:R3, 0021:D9:R4.

#### Scenario: Operator newer than the CLI

- **WHEN** the target module states operator `1.1.0` and the CLI is `1.0.0-beta.9`
- **THEN** install exits 2 naming the module version, the operator version and upgrading the CLI, and nothing changed

#### Scenario: Rendered image disagrees with the stated operator version

- **WHEN** the target module states operator `1.0.0-beta.6` and its rendered controller image is tagged `v1.0.0-beta.5`
- **THEN** install exits 2 naming the module version and both versions, and nothing changed

#### Scenario: Rendered CRD below the floor

- **WHEN** the target's rendered ModuleInstance CRD lacks `spec.owner`
- **THEN** install exits 2 naming the module version and the floor, before the CRD step

#### Scenario: Newer running operator does not block repair

- **WHEN** the cluster Platform reports operator `1.1.0`, the CLI is `1.0.0`, and the target module states operator `1.0.2`
- **THEN** the running-operator ceiling does not refuse install, and install proceeds

### Requirement: The operator's settings are recorded instance values that survive reinstall

Install SHALL take values for the operator module from `-f/--values` files, layered in order over the values recorded on the operator's instance: maps merge, and a scalar or list given in a later source replaces the earlier one. `--reset-values` SHALL drop the recorded values for that run. Re-running install, for the same or another module version, SHALL keep every recorded value the run does not change. The module's `debugValues` SHALL NOT be used. When the merged values are not accepted by the target version's `#config`, install SHALL refuse before any object changes and name each rejected value. `-f` and `--reset-values` SHALL be refused with `--crds-only` before any registry or cluster call. Source: 0006:D19.

#### Scenario: A recorded value survives reinstall

- **WHEN** install ran with a values file setting the registry mapping, and install runs again with no `-f`
- **THEN** the second render carries the same registry mapping and the record still holds it

#### Scenario: Changing one value keeps the others

- **WHEN** the record holds a registry mapping and a replica count, and install runs with a file setting only the replica count
- **THEN** the render carries the new replica count and the recorded registry mapping

#### Scenario: Recorded value the target rejects

- **WHEN** the record holds a value the target module version's `#config` does not accept
- **THEN** install exits 2 naming that value's path and `--reset-values`, and nothing changed

### Requirement: Install succeeds when the installed operator has rolled out

Install SHALL report success only after the rendered CRDs are `Established` and the operator Deployment has completed its rollout, both within the one `--timeout` budget that also covers waiting out terminating objects. When the rollout does not complete in time, install SHALL keep every applied object and the record, roll nothing back, and exit non-zero naming the Deployment and that re-running install completes it. Install SHALL NOT wait for the cluster Platform's readiness.

#### Scenario: Upgrade waits for the rollout

- **WHEN** install upgrades the operator to another module version
- **THEN** install reports success only after the controller Deployment's new ReplicaSet has completed its rollout

#### Scenario: Rollout does not complete

- **WHEN** the new controller pod cannot become ready before `--timeout` elapses
- **THEN** install exits non-zero naming `opm-operator-controller-manager` and the timeout, and every applied object and the record remain

### Requirement: Platform seeding and the opt-in user role stay install steps

After the operator's rollout, and unless `--skip-platform` or `--crds-only` is given, install SHALL create the singleton `cluster` Platform subscribing to the catalog build it resolved before contacting the cluster, with a plain create as field manager `opm-cli`, never server-side apply or update. An existing Platform SHALL be reported and left untouched, an `AlreadyExists` response SHALL be a success-noop, and a create denied by RBAC SHALL degrade to a warning. Neither the Platform nor the `opm-cli-user` role of `--rbac` SHALL be part of the operator module's render or the instance's inventory; `--rbac` SHALL apply on the CRDs-only path as on the full one. Source: 0006:D12, 0006:D22, 0006:D23, 0021:D11:R8.

#### Scenario: Platform is seeded on a full install

- **WHEN** install completes against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` exists, subscribing to the resolved catalog build, and the output names the catalog module path and version

#### Scenario: Existing Platform is left untouched

- **WHEN** install runs against a cluster that already carries a `Platform` CR
- **THEN** the stored Platform is unchanged and the command reports it was already present

#### Scenario: Platform create denied by RBAC

- **WHEN** the installing user may apply the operator but may not create `platforms`
- **THEN** the command warns that the Platform was not seeded, names the missing permission, and does not fail because of it

#### Scenario: Platform and role are not recorded

- **WHEN** install with `--rbac --user alice` completes
- **THEN** neither `Platform/cluster` nor the `opm-cli-user` ClusterRole or its binding appears in the operator instance's inventory

### Requirement: CRDs-only install applies the CRDs of the same module render

`opm operator install --crds-only` SHALL render the same module version as a full install would, refuse under the operator-newer-than-CLI rule, the rendered-image rule and the CRD floor before applying any CRD, apply exactly the rendered `CustomResourceDefinition` objects with server-side apply as `opm-cli`, and wait for each to be `Established`. It SHALL write no instance record, no workload and no Platform, and SHALL perform no catalog lookup. A later full install of the same module version SHALL record those CRDs without recreating them. Source: 0021:D11:R7.

#### Scenario: Solo-cluster CRD install

- **WHEN** `opm operator install --crds-only` runs against an empty cluster
- **THEN** exactly the four rendered CRDs exist and are `Established`, and no ModuleInstance, Deployment or Platform exists

#### Scenario: Full install after CRDs-only

- **WHEN** a full install of the same module version follows a CRDs-only install
- **THEN** the four CRDs keep their uid and appear in the new record's inventory

#### Scenario: CRDs-only with a values file

- **WHEN** `opm operator install --crds-only -f values.cue` runs
- **THEN** the command fails flag validation before any registry or cluster call

### Requirement: Uninstall deletes the operator instance's recorded inventory

`opm operator uninstall` SHALL read the operator's instance record `opm-operator` in `opm-operator-system`. When none exists, it SHALL delete nothing and exit 2 naming `opm operator install` as the step that records the running operator. Otherwise, after the finalizer guard, it SHALL delete every object the record's inventory lists except CRDs and the Namespace, in descending resource-weight order, leaving behind any object that no longer carries this instance's identity, then delete the record. It SHALL delete no object the inventory does not list, SHALL NOT wait for deletion to complete, and SHALL treat an already absent object as deleted. Source: 0006:D34, 0021:D11:R11.

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

## MODIFIED Requirements

### Requirement: Opt-in RBAC emission via `--rbac`

`opm operator install --rbac` SHALL additionally create a ClusterRole `opm-cli-user` granting full verbs on `moduleinstances`, `get/patch/update` on `moduleinstances/status`, and `get/list` on `platforms`. When `--user <U>` or `--group <G>` is supplied alongside `--rbac`, the command SHALL also create a ClusterRoleBinding binding that subject to the role. Without `--rbac`, no RBAC objects beyond those of the operator module's render are created. `--user`/`--group` without `--rbac` SHALL be rejected as a flag-validation error before any cluster interaction.

#### Scenario: RBAC off by default

- **WHEN** `opm operator install --crds-only` is run
- **THEN** no `opm-cli-user` ClusterRole or ClusterRoleBinding is created

#### Scenario: Role plus binding for a user

- **WHEN** `opm operator install --crds-only --rbac --user alice` is run
- **THEN** the `opm-cli-user` ClusterRole and a ClusterRoleBinding for user `alice` are applied with field manager `opm-cli`

#### Scenario: Subject flags require --rbac

- **WHEN** `opm operator install --user alice` is run without `--rbac`
- **THEN** the command fails flag validation with an error stating `--user`/`--group` require `--rbac`, before contacting the cluster

## REMOVED Requirements

### Requirement: Full operator install from the embedded manifest
**Reason**: Install no longer applies an embedded manifest; it deploys the operator module as a CLI-owned instance.
**Migration**: "Install deploys the operator module as a CLI-owned instance in two steps", "Every check that can refuse install runs before its first write", "Install succeeds when the installed operator has rolled out" and "Platform seeding and the opt-in user role stay install steps" carry its behaviour, the terminating wait and the Platform seeding included.

### Requirement: CRDs-only install via `--crds-only`
**Reason**: The CRDs come from the module render, not the embedded manifest, and the form now needs the registry.
**Migration**: "CRDs-only install applies the CRDs of the same module render".

### Requirement: Single embedded artifact with a pinned version
**Reason**: The CLI embeds no operator manifest; it pins a module version and records the operator version that module deploys.
**Migration**: "The CLI pins a default operator module version and the operator it deploys"; `task operator:pin VERSION=<v>` replaces `task operator:sync VERSION=<tag>`.

### Requirement: `--version` fetches the release asset instead of the embed
**Reason**: `--version` selects an operator module version from the registry; no GitHub release asset is fetched.
**Migration**: "Another module version is installed from the registry as served". Pass a module version (`--version 0.2.0`); install prints the operator version it deploys.

### Requirement: Uninstall preserves CRDs and the Namespace
**Reason**: Uninstall deletes the instance's recorded inventory, not the embedded manifest's documents.
**Migration**: "Uninstall deletes the operator instance's recorded inventory", which keeps CRDs and the Namespace and stays idempotent.
