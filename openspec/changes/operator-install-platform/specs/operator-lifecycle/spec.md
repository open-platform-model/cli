# operator-lifecycle: Delta

## MODIFIED Requirements

### Requirement: Full operator install from the embedded manifest

`opm operator install` SHALL server-side-apply every document of the embedded operator manifest (`dist/install.yaml` of the pinned opm-operator release) with field manager `opm-cli`, ordered ascending by the resource weights of `pkg/resourceorder` (CRDs before Namespace before RBAC before Deployment). The command SHALL then wait, bounded by `--timeout` (default 5m), for every applied CRD to reach the `Established=True` condition and for the operator Deployment to complete its rollout, and SHALL exit non-zero with an actionable error if the timeout elapses first.

After the readiness wait completes, and unless `--skip-platform` is given, the command SHALL create the singleton `cluster` Platform subscribing to the catalog build it resolved before contacting the cluster, under the write-if-absent contract of the `platform-resolution` capability: a plain create with field manager `opm-cli`, never server-side apply and never update. The seeding step SHALL run only after readiness, so the Platform CRD is `Established` before the write is attempted. The command SHALL report the Platform outcome (created with its pinned catalog coordinate, already present and left untouched, or skipped) beside the operator install summary.

Seeding SHALL NOT be able to fail an otherwise successful install: an `AlreadyExists` response is a success-noop, and a create denied by RBAC SHALL degrade to a warning while the command still exits zero.

#### Scenario: Install onto an empty cluster

- **WHEN** `opm operator install` is run against a cluster with no OPM components
- **THEN** all manifest documents are applied via SSA with field manager `opm-cli`
- **AND** the command reports a per-resource status line (created/configured/unchanged) for each document
- **AND** the command waits until the CRDs are `Established` and the operator Deployment rollout completes, then exits zero

#### Scenario: Install is idempotent

- **WHEN** `opm operator install` is run a second time with no cluster-side changes in between
- **THEN** every document reports `unchanged` and the command exits zero

#### Scenario: Readiness timeout

- **WHEN** the operator Deployment cannot become ready within `--timeout`
- **THEN** the command exits non-zero with an error naming the resource still unready and the elapsed timeout

#### Scenario: Platform is seeded on a full install

- **WHEN** `opm operator install` completes its readiness wait against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` SHALL exist, subscribing to the resolved catalog build at the resolved version
- **AND** the command output SHALL name the catalog module path and the version it pinned

#### Scenario: Existing Platform is left untouched

- **WHEN** `opm operator install` runs against a cluster that already carries a `Platform` CR
- **THEN** the stored Platform SHALL be unchanged, including its subscribed catalog version
- **AND** the command SHALL report that the Platform was already present and exit zero

#### Scenario: Platform create denied by RBAC

- **WHEN** the installing user may apply the operator manifest but may not create `platforms`
- **THEN** the command SHALL warn that the Platform could not be created and SHALL still exit zero

#### Scenario: Seeding suppressed by --skip-platform

- **WHEN** `opm operator install --skip-platform` is run against an empty cluster
- **THEN** the operator SHALL be installed and no `Platform` resource SHALL exist afterwards
- **AND** no registry lookup SHALL be performed

### Requirement: CRDs-only install via `--crds-only`

`opm operator install --crds-only` SHALL apply only the `CustomResourceDefinition` documents filtered from the same embedded manifest, and SHALL wait only for the `Established=True` condition on those CRDs. No Namespace, RBAC, Deployment, or Service objects are created, and no `Platform` resource is created. `--crds-only` SHALL NOT perform a catalog registry lookup.

#### Scenario: Solo-cluster CRD install

- **WHEN** `opm operator install --crds-only` is run against an empty cluster
- **THEN** exactly the manifest's `CustomResourceDefinition` documents are applied
- **AND** the command waits for each to report `Established=True` and exits zero
- **AND** no other kinds from the manifest exist on the cluster afterwards

#### Scenario: No Platform after a CRDs-only install

- **WHEN** `opm operator install --crds-only` completes against an empty cluster
- **THEN** no `Platform` resource SHALL exist
- **AND** the command SHALL succeed with no registry access

## ADDED Requirements

### Requirement: Catalog resolution precedes every cluster write

When `opm operator install` will seed a Platform, it SHALL resolve the catalog version from the registry before it resolves the kubeconfig, contacts the cluster, or applies any document. A resolution that cannot produce a version SHALL abort the command with nothing applied, so a registry problem can never leave a partially installed cluster.

Resolution failures SHALL map to the CLI's standard exit codes: a catalog major with no selectable version is a validation refusal (exit 2) that names the flag which would select a prerelease, and an unreachable or unreadable registry is a connectivity failure (exit 3).

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
