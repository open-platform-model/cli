## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: The CLI pins a default controller module version and the controller it deploys

Each CLI release SHALL name one default controller module version and SHALL record the controller version that module deploys, both readable from the CLI's source without a registry. The recorded controller version SHALL be the version the module's own source states it deploys. Install without `--version` SHALL install exactly that module version. The pin SHALL be produced by `task controller:pin VERSION=<v>`, which reads the controller version from the module's own statement of the controller it deploys, without a render. The same task SHALL render that module version as install renders the controller's instance (the instance `opm-controller` in `opm-controller-system`, no values, against the module's own dependency pins) and write the render's `CustomResourceDefinition` objects as the manifests the CLI embeds for `--crds-only`, one file per CRD, replacing the previous set. A render that fails, or a manifest write that fails, SHALL leave the pin and the manifests unchanged. No other task SHALL write the pin or the manifests, except that the pin tool MAY write the manifests alone from a local checkout of the module before that version is published. The release pin check (G1) SHALL fail when the embedded manifests differ from a fresh render of the pinned version. The module is trusted as the configured registry serves it, as every other module the CLI installs is. Source: 0021:D11:R6.

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

## REMOVED Requirements

### Requirement: CRDs-only install applies the CRDs of the same module render

**Reason**: The CRDs-only form no longer renders the module at install time. It applies CRDs the pin task rendered from the pinned module version and the CLI embeds, so it works with no registry (0021:D11:R1/R7, revised 2026-10-06).

**Migration**: `opm controller install --crds-only` keeps its name and result for the pinned version. Pass no `--version` with it; to install another version's CRDs, run a full install of that version.
