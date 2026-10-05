## RENAMED Requirements

- FROM: `### Requirement: cluster:operator builds the CLI it installs with`
- TO: `### Requirement: cluster:controller builds the CLI it installs with`

## REMOVED Requirements

### Requirement: Operator install task needs no local registry
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-controller-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## MODIFIED Requirements

### Requirement: Local registry is an explicit opt-in for the kind flow

Pointing the kind flow at a local registry SHALL require setting `KIND_CUE_REGISTRY` explicitly. That variable SHALL default to empty. When it is set, and only then, the task SHALL verify the registry container is running, join it to kind's docker network, and pass the mapping to `opm controller install` as the controller module's registry value in a values file, so the mapping is recorded on the controller's instance and survives every later install. When it is unset the task SHALL pass no registry value, so a mapping an earlier opt-in run recorded stays until `opm controller install --reset-values`. The task SHALL NOT patch the controller Deployment.

The controller's registry is a command-line flag the module renders from that value, not an environment variable: `--registry` wins over `OPM_REGISTRY`, which is read only when the flag is empty, and the controller overwrites `CUE_REGISTRY` in its own process environment. Setting either as a pod environment variable has no effect.

#### Scenario: Opt-in with the registry running

- **WHEN** `task cluster:controller KIND_CUE_REGISTRY='testing.opmodel.dev=opm-registry:5000+insecure,...'` runs and the registry container is up
- **THEN** the container is joined to kind's network, the controller's instance records the mapping as its registry value, and the controller Deployment carries the matching `--registry` argument

#### Scenario: Opt-in without the registry running

- **WHEN** `KIND_CUE_REGISTRY` is set and no registry container is running
- **THEN** the task SHALL fail with a message naming `task registry:start` and the option of unsetting the variable to use GHCR

#### Scenario: Default path reports its registry source

- **WHEN** `task cluster:controller` runs with `KIND_CUE_REGISTRY` unset
- **THEN** the task SHALL state that it sets no registry value, so the controller uses its built-in GHCR default unless an earlier run recorded a mapping, and SHALL NOT patch the Deployment

### Requirement: cluster:controller builds the CLI it installs with

The `cluster:controller` task SHALL ensure the CLI binary it invokes exists and is current before
invoking it, by depending on the build task rather than assuming a previous build. The dependency
SHALL reuse the build task's existing up-to-date check, so a tree whose binary is already current
performs no rebuild and the task stays idempotent.

The task SHALL therefore succeed from a clean checkout with no manual preparation beyond a running
cluster, and SHALL be callable unattended — a caller that cannot answer a prompt or run a suggested
command by hand, such as an automated test cleanup, SHALL NOT need one.

#### Scenario: Clean tree with no binary

- **WHEN** a developer runs `task cluster:controller` in a checkout that has never been built, against
  a running kind cluster
- **THEN** the CLI binary is built first and the controller install proceeds
- **AND** the task SHALL NOT fail reporting that the binary does not exist

#### Scenario: Binary already current

- **WHEN** `task cluster:controller` runs again with no source changes since the last build
- **THEN** no rebuild is performed
- **AND** the task completes as it did before, without duplicating any controller configuration

#### Scenario: Called unattended by a test cleanup

- **WHEN** a destructive e2e test invokes `task cluster:controller` to rebuild the dev controller it tore
  down, in a worktree where the binary was never built
- **THEN** the task builds what it needs and restores the controller without developer intervention

## ADDED Requirements

### Requirement: Controller install task needs no local registry

The `cluster:controller` task SHALL install the pinned controller into the kind cluster and SHALL NOT require a local registry container to do so. Everything the flow resolves — core, the catalogs, and this repository's fixture modules — is published on GHCR, and the pinned controller's built-in `--registry` default routes both `opmodel.dev/*` and `testing.opmodel.dev/*` there.

The task SHALL remain idempotent, SHALL apply the cluster `Platform` singleton and the dev-only RBAC grant, and SHALL confirm the controller can actually serve the cluster by waiting for the `Ready` condition of `Platform/cluster` to be `True` for the Platform's current generation, that is with `status.observedGeneration` equal to `metadata.generation`, rather than for pod readiness or for `status.controllerVersion`, which the controller stamps on every reconcile whatever its outcome. A `Ready=True` observed for an earlier generation (the Platform the install created, before the task applied the pinned one) SHALL NOT satisfy the wait. If `Ready` is not `True` for the current generation within the task's wait, the task SHALL fail, and its message SHALL print the Platform's generation and observed generation, its `Ready` and `Stalled` conditions with their reasons and messages, and name the controller's log as the next place to look.

#### Scenario: Install with no registry container running

- **WHEN** a developer runs `task cluster:controller` against a running kind cluster with no `opm-registry` container
- **THEN** the task SHALL install the controller, apply the Platform and RBAC, and report the reconciling controller version

#### Scenario: Re-run is a no-op

- **WHEN** `task cluster:controller` is run a second time
- **THEN** it SHALL complete without error and without duplicating container arguments

#### Scenario: Controller that cannot build the Platform

- **WHEN** the installed controller stamps `status.controllerVersion` on `Platform/cluster` but sets `Ready=False` with `Stalled=True`, for example reason `BuildFailed` (`MaterializeFailed` on controller alpha.14)
- **THEN** the task SHALL fail once its wait expires, printing the `Stalled` reason and message
- **AND** it SHALL NOT report the controller as reconciling

#### Scenario: Ready left over from an earlier generation

- **WHEN** `Platform/cluster` still carries `Ready=True` from the generation the install created, and the generation the task applied has not been observed or is stalling
- **THEN** the task SHALL keep waiting, and SHALL fail once its wait expires, printing both generations and the `Stalled` reason
