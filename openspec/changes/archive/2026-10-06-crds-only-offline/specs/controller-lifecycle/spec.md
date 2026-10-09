## ADDED Requirements

### Requirement: CRDs-only install applies the CRDs pinned with the CLI, without a registry

`opm controller install --crds-only` SHALL apply the `CustomResourceDefinition` manifests the CLI embeds, which are the CRDs the pinned module version renders for the instance `opm-controller` in `opm-system`, with server-side apply as `opm-cli`, and wait for each to be `Established`. It SHALL make no registry request: it SHALL NOT resolve, fetch or render the controller module and SHALL perform no catalog lookup. It SHALL read one `ModuleInstance` at most, the controller's instance record, and SHALL use only that record's inventory: the record's owner and values SHALL play no part. It SHALL write no instance record, no workload and no Platform, so its only network peer is the Kubernetes API server. With `--rbac` it SHALL also apply the `opm-cli-user` role as a full install does. A later full install of the pinned module version SHALL record those CRDs without recreating them. It SHALL NOT adopt, relabel or migrate a CRD by itself: every existing CRD it would apply is judged by the apply guard (capability `apply-pruning`, "Ownership guard judges every apply and dry run") as on a full install: with the instance UUID the embedded CRDs carry, and with the inventory of the controller's instance record when that record exists. A CRD that the record's inventory lists SHALL therefore be applied whatever identity label it carries, as a full install applies it. With no record, or for a CRD the record does not list, the CRD is judged as on a first install: when it carries no OPM managed-by label, or carries the identity of another instance, the command SHALL refuse with exit 2 before any write, with the guard's message: it names that CRD and the `opmodel.dev/adopt` annotation with the UUID to set, which is what the user does to let the instance take the CRD over. A cluster that does not serve `ModuleInstance` has no record. When the record cannot be read for any other reason, the command SHALL fail before any write with the read error. When an existing CRD it would apply carries the embedded CRD's `module.opmodel.dev/name` and a `module.opmodel.dev/version` label newer than the pinned module version, it SHALL refuse with exit 2 before any write, naming that CRD, that version and both remedies, `--crds-only` from a CLI that pins that module version or newer and a full install of that version, since that version's CRDs came from either one and applying older CRDs would silently downgrade the schemas that version installed. It reads those CRDs from the API server only. `--version`, `-f/--values` and `--reset-values` SHALL be rejected with `--crds-only` as flag-validation errors before any cluster call, because the form installs only the pinned version and records no values. Source: 0021:D11:R7.

#### Scenario: Solo-cluster CRD install with no registry

- **WHEN** `opm controller install --crds-only` runs against an empty cluster while no registry can be reached
- **THEN** exactly the four CRDs of the pinned module version exist and are `Established`, applied by field manager `opm-cli`, and no ModuleInstance, Deployment or Platform exists
- **AND** the command exits 0 and names the pinned module version

#### Scenario: The controller's instance record is read for its inventory only

- **WHEN** `opm controller install --crds-only` runs on a cluster whose record `opm-controller` in `opm-system` has `spec.owner: controller`
- **THEN** the CRDs are applied, and no `ModuleInstance` is changed

#### Scenario: A CRD the record lists is applied whatever identity it carries

- **WHEN** `opm controller install --crds-only` runs, the record `opm-controller` in `opm-system` lists `moduleinstances.opmodel.dev` in its inventory, and that CRD carries OPM's managed-by label and the UUID of another instance
- **THEN** the CRD is applied and carries the UUID of the controller's instance, and the record is unchanged

#### Scenario: A CRD the record does not list is judged as on a first install

- **WHEN** `opm controller install --crds-only` runs, the record lists `moduleinstances.opmodel.dev` only, and `platforms.opmodel.dev` carries the UUID of another instance
- **THEN** the command exits 2 naming `platforms.opmodel.dev` alone, and nothing has been applied

#### Scenario: A CRD of another instance without a record is refused

- **WHEN** `opm controller install --crds-only` runs on a cluster with no record of the controller's instance and `moduleinstances.opmodel.dev` carries OPM's managed-by label and the UUID of another instance
- **THEN** the command exits 2 naming that CRD, that instance and the `opmodel.dev/adopt` annotation with the UUID to set, and nothing has been applied

#### Scenario: An annotated CRD is applied

- **WHEN** `opm controller install --crds-only` runs and `moduleinstances.opmodel.dev` exists without OPM's managed-by label, annotated `opmodel.dev/adopt` with the UUID of the controller's instance
- **THEN** the CRD is applied and carries that UUID

#### Scenario: An unreadable record refuses

- **WHEN** `opm controller install --crds-only` runs and the read of the record `opm-controller` in `opm-system` is denied
- **THEN** the command fails naming the record, and nothing has been applied

#### Scenario: Full install after CRDs-only

- **WHEN** a full install of the pinned module version follows a CRDs-only install
- **THEN** the four CRDs keep their uid and appear in the new record's inventory

#### Scenario: CRDs OPM does not manage are refused, not adopted

- **WHEN** `opm controller install --crds-only` runs and `moduleinstances.opmodel.dev` exists without OPM's managed-by label and without an adopt annotation
- **THEN** the command exits 2 with `refusing to install:`, naming that CRD as not managed by OPM and the `opmodel.dev/adopt` annotation with the UUID to set
- **AND** nothing has been applied and the CRD's labels are unchanged

#### Scenario: CRDs a newer module version installed are not replaced

- **WHEN** module version 0.2.0 installed the CRDs, by a full install or by `--crds-only` from a CLI that pins 0.2.0, and `opm controller install --crds-only` runs from a CLI that pins 0.1.0
- **THEN** the command exits 2 naming the CRD, module version 0.2.0, `--crds-only` from a CLI that pins 0.2.0 or newer, and `opm controller install --version 0.2.0`, and nothing has been applied

#### Scenario: CRDs-only with a values file

- **WHEN** `opm controller install --crds-only -f values.cue` runs
- **THEN** the command fails flag validation before any registry or cluster call

#### Scenario: CRDs-only with a version

- **WHEN** `opm controller install --crds-only --version 0.2.0` runs
- **THEN** the command fails flag validation before any registry or cluster call, naming the module version the CLI pins

## MODIFIED Requirements

### Requirement: The CLI pins a default controller module version and the controller it deploys

Each CLI release SHALL name one default controller module version and SHALL record the controller version that module deploys, both readable from the CLI's source without a registry. The recorded controller version SHALL be the version the module's own source states it deploys. Install without `--version` SHALL install exactly that module version. The pin SHALL be produced by `task controller:pin VERSION=<v>`, which reads the controller version from the module's own statement of the controller it deploys, without a render. The same task SHALL render that module version as install renders the controller's instance (the instance `opm-controller` in `opm-system`, no values, against the module's own dependency pins) and write the render's `CustomResourceDefinition` objects as the manifests the CLI embeds for `--crds-only`, one file per CRD, replacing the previous set. A render that fails, or a manifest write that fails, SHALL leave the pin and the manifests unchanged. No other task SHALL write the pin or the manifests, except that the pin tool MAY write the manifests alone from a local checkout of the module before that version is published. The release pin check (G1) SHALL fail when the embedded manifests differ from a fresh render of the pinned version. The module is trusted as the configured registry serves it, as every other module the CLI installs is. Source: 0021:D11:R6.

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

### Requirement: Install runs every refusing check before its first write

Install SHALL run, before it writes any object, every check that can refuse it: resolving the module and reading the controller version it deploys, rendering it, the check of the values against the target version's `#config`, the target rules of install (the controller version against the CLI's, the rendered image against the module's stated controller version, and the rendered ModuleInstance CRD against the CLI's floor), the status-subresource permission check, and the apply guard over every object the render names (capability `apply-pruning`, "Ownership guard judges every apply and dry run"). Install SHALL pass the guard no admission set and SHALL run no proof of where an existing object came from: an object that an earlier release manifest created is judged as any other existing object. The guard SHALL run on every install, with or without a record of the controller's instance. An object the record lists is judged as recorded and SHALL pass whatever its UUID label says. An existing object the record does not list SHALL pass when its `opmodel.dev/adopt` annotation holds the UUID of the controller's instance, or when it carries an OPM managed-by label and no UUID label of another instance; it SHALL refuse the install when OPM does not manage it or when it carries another instance's identity. Install needs every object its render names, so, unlike any other apply, it SHALL also refuse when the `opmodel.dev/adopt` annotation of an object names another instance, whether or not the record lists the object: it SHALL NOT leave the object unapplied and go on (owner decision of 2026-10-08 on `opm controller install`). Every refusal of the guard in this phase SHALL exit 2, SHALL name the object and the instance that owns or adopts it, SHALL name the `opmodel.dev/adopt` annotation and the UUID to set where the annotation lifts the refusal, and SHALL leave the cluster unchanged; that includes a failed read by the guard, with or without a record. No refusal of install SHALL tell the user to delete or rename an object. Source: 0012:D8:R1, 0012:D8:R2, 0012:D8:R3, 0012:D8:R8. Objects of the render that exist with a deletion timestamp SHALL be waited out within `--timeout` before the apply guard runs, so install right after uninstall waits instead of refusing; a terminating object that outlives the budget SHALL fail the command with nothing applied. Only after every check passed SHALL install write, in this order: the CRD step, then the instance apply, then the Platform seed. Outside the instance apply and its prune, install SHALL send no delete and no patch of managed fields. The CRDs-only form resolves, fetches, renders and reads nothing of the module: its checks are the flag validation, the wait for terminating CRDs within `--timeout`, the refusal of a CRD a newer module version installed, and the same apply guard over the embedded CRDs, asked with the UUID of the controller's instance and with the inventory of the controller's instance record when that record exists. An install that refuses SHALL change no object in the cluster. Two of these checks read every object install applies, in this order: the wait for terminating objects and the apply guard. A read that fails with an error other than NotFound SHALL refuse the install in the check that made it. When the wait for terminating objects fails a read, the command SHALL exit with the code of the read error: 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. When the apply guard fails a read, the command SHALL exit 2, the code of the guard's other refusals. An object that install cannot read from the start is therefore refused by the wait for terminating objects, with exit code 4 when the read is denied.

#### Scenario: A refused install leaves the CRDs untouched

- **WHEN** install refuses because an object the render names already exists and is not managed by OPM
- **THEN** no CRD, label or other object in the cluster has changed

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm controller uninstall` has returned and `opm controller install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies only after it is gone, and completes

#### Scenario: Terminating object outlives the budget

- **WHEN** an object of the render stays terminating for longer than `--timeout`
- **THEN** the command exits non-zero naming that object and the elapsed timeout, and nothing has been applied

#### Scenario: An object unreadable from the start exits by the read error

- **WHEN** install runs on a cluster with no record of the controller's instance and every read of a ClusterRole the render names fails with Forbidden
- **THEN** the command SHALL exit 4 naming that ClusterRole and the read error, and no object in the cluster has changed

#### Scenario: A read the apply guard fails exits with the guard code

- **WHEN** the earlier read of a rendered object was answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed

#### Scenario: Reinstall refuses a foreign object new to the inventory

- **WHEN** install runs on a cluster with a record of the controller's instance
- **AND** the render names a ClusterRole the record does not list, which exists without an OPM managed-by label
- **THEN** the command SHALL exit 2 naming that ClusterRole and the adopt annotation, and no object in the cluster has changed

#### Scenario: First install refuses an object of another instance

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** a CustomResourceDefinition the render names exists with an OPM managed-by label and the UUID of another instance
- **THEN** the command SHALL exit 2 naming that CustomResourceDefinition and the other instance, and no object in the cluster has changed

#### Scenario: Install refuses an object adopted by another instance

- **WHEN** the controller Deployment, a rendered CustomResourceDefinition or the controller Namespace carries `opmodel.dev/adopt` with the UUID of another instance
- **THEN** the command SHALL exit 2 naming that object and that instance
- **AND** no object in the cluster has changed, the managed fields of that object included

#### Scenario: Reinstall with a failed guard read exits with the guard code

- **WHEN** install runs on a cluster with a record of the controller's instance, the earlier read of a rendered object was answered and the apply guard's read of it fails with Forbidden
- **THEN** the command SHALL exit 2 naming that object and the read error, and no object in the cluster has changed

#### Scenario: Reinstall over its own recorded objects passes the guard

- **WHEN** install runs on a cluster with a record of the controller's instance and every existing object the render names is in that record
- **THEN** the apply guard refuses none of them

#### Scenario: Recorded objects that carry another identity are applied

- **WHEN** install runs on a cluster with a record of the controller's instance that lists every object the render names
- **AND** every one of those objects, the four CRDs and the Namespace included, carries an OPM managed-by label and the UUID of another instance
- **THEN** the apply guard refuses none of them and install applies them

#### Scenario: Annotated objects that carry another identity are applied

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** every object the render names exists with the UUID of another instance and `opmodel.dev/adopt` holding the UUID of the controller's instance
- **THEN** the apply guard refuses none of them, and install applies and records them

#### Scenario: Objects that carry another identity, with no record and no annotation, are refused

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** every object the render names exists with an OPM managed-by label and the UUID of another instance, and none carries an adopt annotation
- **THEN** the command SHALL exit 2 naming each object, the other instance and `opmodel.dev/adopt` with the UUID to set
- **AND** the output SHALL NOT contain `controller migration refused` or `remove or rename`
- **AND** no object in the cluster has changed

#### Scenario: Unlabelled CRDs of an earlier manifest install are refused

- **WHEN** install runs on a cluster whose four `opmodel.dev` CRDs, Namespace and Deployment were applied from an earlier release manifest and carry no OPM managed-by label and no adopt annotation
- **THEN** the command SHALL exit 2 with `refusing to install:` and one line for each such object that says it exists and is not managed by OPM, and names `opmodel.dev/adopt` with the UUID to set
- **AND** install SHALL adopt none of them by itself, and no object in the cluster has changed

#### Scenario: CRDs-only refuses unlabelled CRDs

- **WHEN** `opm controller install --crds-only` runs and a CRD the CLI embeds exists without an OPM managed-by label and without an adopt annotation
- **THEN** the command SHALL exit 2 naming that CRD, and no CRD has changed

#### Scenario: The instance's own objects without a record pass the guard

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** every existing object the render names carries an OPM managed-by label and the UUID of the controller's instance (a run that stopped before it wrote the record, a CRDs-only install, or a `kubectl apply` of the module's render)
- **THEN** the apply guard refuses none of them

#### Scenario: An unrecorded object with the instance's UUID and no OPM managed-by label is refused

- **WHEN** install runs on a cluster with no record of the controller's instance
- **AND** an object the render names carries the UUID of the controller's instance, and its managed-by label is absent or names another tool
- **THEN** the command SHALL exit 2 naming that object as not managed by OPM, and `opmodel.dev/adopt` with the UUID to set

## REMOVED Requirements

### Requirement: CRDs-only install applies the CRDs of the same module render

**Reason**: The CRDs-only form no longer renders the module at install time. It applies CRDs the pin task rendered from the pinned module version and the CLI embeds, so it works with no registry (0021:D11:R1/R7, revised 2026-10-06).

**Migration**: `opm controller install --crds-only` keeps its name and result for the pinned version. Pass no `--version` with it; to install another version's CRDs, run a full install of that version.
