## MODIFIED Requirements

### Requirement: Operator install task needs no local registry

The `cluster:operator` task SHALL install the pinned operator into the kind cluster and SHALL NOT require a local registry container to do so. Everything the flow resolves — core, the catalogs, and this repository's fixture modules — is published on GHCR, and the pinned operator's built-in `--registry` default routes both `opmodel.dev/*` and `testing.opmodel.dev/*` there.

The task SHALL remain idempotent, SHALL apply the cluster `Platform` singleton and the dev-only RBAC grant, and SHALL confirm the operator can actually serve the cluster by waiting for the `Ready` condition of `Platform/cluster` to be `True` for the Platform's current generation, that is with `status.observedGeneration` equal to `metadata.generation`, rather than for pod readiness or for `status.operatorVersion`, which the operator stamps on every reconcile whatever its outcome. A `Ready=True` observed for an earlier generation (the Platform the install created, before the task applied the pinned one) SHALL NOT satisfy the wait. If `Ready` is not `True` for the current generation within the task's wait, the task SHALL fail, and its message SHALL print the Platform's generation and observed generation, its `Ready` and `Stalled` conditions with their reasons and messages, and name the operator's log as the next place to look.

#### Scenario: Install with no registry container running

- **WHEN** a developer runs `task cluster:operator` against a running kind cluster with no `opm-registry` container
- **THEN** the task SHALL install the operator, apply the Platform and RBAC, and report the reconciling operator version

#### Scenario: Re-run is a no-op

- **WHEN** `task cluster:operator` is run a second time
- **THEN** it SHALL complete without error and without duplicating container arguments

#### Scenario: Operator that cannot materialize the Platform

- **WHEN** the installed operator stamps `status.operatorVersion` on `Platform/cluster` but sets `Ready=False` with `Stalled=True`, for example reason `MaterializeFailed`
- **THEN** the task SHALL fail once its wait expires, printing the `Stalled` reason and message
- **AND** it SHALL NOT report the operator as reconciling

#### Scenario: Ready left over from an earlier generation

- **WHEN** `Platform/cluster` still carries `Ready=True` from the generation the install created, and the generation the task applied has not been observed or is stalling
- **THEN** the task SHALL keep waiting, and SHALL fail once its wait expires, printing both generations and the `Stalled` reason
