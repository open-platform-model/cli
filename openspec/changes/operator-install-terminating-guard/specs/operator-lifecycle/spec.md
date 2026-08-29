## MODIFIED Requirements

### Requirement: Full operator install from the embedded manifest

`opm operator install` SHALL server-side-apply every document of the embedded operator manifest (`dist/install.yaml` of the pinned opm-operator release) with field manager `opm-cli`, ordered ascending by the resource weights of `pkg/resourceorder` (CRDs before Namespace before RBAC before Deployment). The command SHALL then wait, bounded by `--timeout` (default 5m), for every applied CRD to reach the `Established=True` condition and for the operator Deployment to complete its rollout, and SHALL exit non-zero with an actionable error if the timeout elapses first.

Before applying, the command SHALL inspect every object in its plan on the cluster. An object that exists with `metadata.deletionTimestamp` set is terminating; the command SHALL wait for each such object to disappear before applying anything, report the wait per resource, and charge the wait against the same `--timeout` budget as the readiness wait. If a terminating object has not disappeared when the budget runs out, the command SHALL exit non-zero naming that resource and the elapsed timeout, and SHALL NOT have applied any document. An object that does not exist, or exists without a deletion timestamp, SHALL NOT delay the apply. The guard applies to every plan shape (`--crds-only`, `--rbac`).

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

#### Scenario: Install right after uninstall waits out the terminating objects

- **WHEN** `opm operator uninstall` has returned and `opm operator install` runs before the deleted Deployment has been garbage-collected
- **THEN** the command reports that it is waiting for the terminating Deployment, applies the plan only after it is gone
- **AND** the installed Deployment completes its rollout and the command exits zero

#### Scenario: Terminating object outlives the budget

- **WHEN** an object in the plan stays terminating (for example, a finalizer nobody removes) for longer than `--timeout`
- **THEN** the command exits non-zero naming that resource and the elapsed timeout
- **AND** no manifest document has been applied

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
- **THEN** the operator install SHALL still exit zero
- **AND** the command SHALL warn that the Platform was not seeded and name the permission that was missing

#### Scenario: Seeding suppressed by --skip-platform

- **WHEN** `opm operator install --skip-platform` completes
- **THEN** no `Platform` CR SHALL be created
- **AND** the command SHALL report that seeding was skipped
