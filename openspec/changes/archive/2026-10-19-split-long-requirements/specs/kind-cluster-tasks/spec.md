## REMOVED Requirements

### Requirement: Cluster creation task
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`", "`cluster:create` selects the node image from the `K8S_VERSION` Taskfile variable".

### Requirement: Operator install task needs no local registry
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The `cluster:operator` task installs the pinned operator without requiring a local registry container", "The `cluster:operator` task remains idempotent and applies the cluster `Platform` singleton and the dev-only RBAC grant", "The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation", "A `Ready=True` observed for an earlier generation does not satisfy the wait of `cluster:operator`", "The `cluster:operator` task fails when `Ready` is not `True` for the current generation and prints the Platform's state".

### Requirement: Local registry is an explicit opt-in for the kind flow
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly", "When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file", "The operator's registry is a command-line flag rendered from the registry value, not an environment variable".

### Requirement: cluster:operator builds the CLI it installs with
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current", "The `cluster:operator` task succeeds from a clean checkout and is callable unattended".

## ADDED Requirements

### Requirement: `cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`
The Taskfile SHALL provide a `cluster:create` task that provisions a single-node kind cluster using a version-controlled configuration file. The task SHALL use the `CLUSTER_NAME` Taskfile variable (default: `opm-dev`) as the kind cluster name and SHALL read the cluster configuration from `hack/kind-config.yaml`.

#### Scenario: Create cluster with defaults

- **WHEN** a developer runs `task cluster:create` with no variable overrides
- **THEN** a kind cluster named `opm-dev` is created using the configuration in `hack/kind-config.yaml` and the `kindest/node:v1.34.0` image

#### Scenario: Create cluster with custom name

- **WHEN** a developer runs `task cluster:create CLUSTER_NAME=my-test`
- **THEN** a kind cluster named `my-test` is created

#### Scenario: Kind not installed

- **WHEN** a developer runs `task cluster:create` and the `kind` binary is not on PATH
- **THEN** the task SHALL fail immediately with an error message that includes installation instructions (link to kind quickstart)

### Requirement: `cluster:create` selects the node image from the `K8S_VERSION` Taskfile variable
The `cluster:create` task SHALL use the `K8S_VERSION` Taskfile variable (default: `1.34.0`) to select the node image via `--image kindest/node:v{{K8S_VERSION}}`. Developers MAY override this to test against a different Kubernetes version.

#### Scenario: Create cluster with custom Kubernetes version

- **WHEN** a developer runs `task cluster:create K8S_VERSION=1.33.0`
- **THEN** a kind cluster named `opm-dev` is created using the `kindest/node:v1.33.0` image

### Requirement: The `cluster:operator` task installs the pinned operator without requiring a local registry container
The `cluster:operator` task SHALL install the pinned operator into the kind cluster and SHALL NOT require a local registry container to do so. Everything the flow resolves: core, the catalogs, and this repository's fixture modules, is published on GHCR, and the pinned operator's built-in `--registry` default routes both `opmodel.dev/*` and `testing.opmodel.dev/*` there.

#### Scenario: Install with no registry container running

- **WHEN** a developer runs `task cluster:operator` against a running kind cluster with no `opm-registry` container
- **THEN** the task SHALL install the operator, apply the Platform and RBAC, and report the reconciling operator version

### Requirement: The `cluster:operator` task remains idempotent and applies the cluster `Platform` singleton and the dev-only RBAC grant
The `cluster:operator` task SHALL remain idempotent, SHALL apply the cluster `Platform` singleton and the dev-only RBAC grant.

#### Scenario: Re-run is a no-op

- **WHEN** `task cluster:operator` is run a second time
- **THEN** it SHALL complete without error and without duplicating container arguments

### Requirement: The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation
The `cluster:operator` task SHALL confirm the operator can actually serve the cluster by waiting for the `Ready` condition of `Platform/cluster` to be `True` for the Platform's current generation, that is with `status.observedGeneration` equal to `metadata.generation`, rather than for pod readiness or for `status.operatorVersion`, which the operator stamps on every reconcile whatever its outcome.

#### Scenario: Ready for the current generation confirms the operator can serve the cluster

- **WHEN** `Platform/cluster` has `Ready` `True` and `status.observedGeneration` equal to `metadata.generation`
- **THEN** the task SHALL confirm the operator can actually serve the cluster, rather than waiting for pod readiness or for `status.operatorVersion`

### Requirement: A `Ready=True` observed for an earlier generation does not satisfy the wait of `cluster:operator`
A `Ready=True` observed for an earlier generation (the Platform the install created, before the task applied the pinned one) SHALL NOT satisfy the wait. (Requirement: The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation)

#### Scenario: Ready left over from an earlier generation

- **WHEN** `Platform/cluster` still carries `Ready=True` from the generation the install created, and the generation the task applied has not been observed or is stalling
- **THEN** the task SHALL keep waiting, and SHALL fail once its wait expires, printing both generations and the `Stalled` reason

### Requirement: The `cluster:operator` task fails when `Ready` is not `True` for the current generation and prints the Platform's state
If `Ready` is not `True` for the current generation within the task's wait, the task SHALL fail, and its message SHALL print the Platform's generation and observed generation, its `Ready` and `Stalled` conditions with their reasons and messages, and name the operator's log as the next place to look. (Requirement: The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation)

#### Scenario: Operator that cannot build the Platform

- **WHEN** the installed operator stamps `status.operatorVersion` on `Platform/cluster` but sets `Ready=False` with `Stalled=True`, for example reason `BuildFailed` (`MaterializeFailed` on operator alpha.14)
- **THEN** the task SHALL fail once its wait expires, printing the `Stalled` reason and message
- **AND** it SHALL NOT report the operator as reconciling

### Requirement: A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly
Pointing the kind flow at a local registry SHALL require setting `KIND_CUE_REGISTRY` explicitly. That variable SHALL default to empty. When it is unset the `cluster:operator` task SHALL pass no registry value, so a mapping an earlier opt-in run recorded stays until `opm operator install --reset-values`. The `cluster:operator` task SHALL NOT patch the operator Deployment.

#### Scenario: Default path reports its registry source

- **WHEN** `task cluster:operator` runs with `KIND_CUE_REGISTRY` unset
- **THEN** the task SHALL state that it sets no registry value, so the operator uses its built-in GHCR default unless an earlier run recorded a mapping, and SHALL NOT patch the Deployment

### Requirement: When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file
When `KIND_CUE_REGISTRY` is set, and only then, the `cluster:operator` task SHALL verify the registry container is running, join it to kind's docker network, and pass the mapping to `opm operator install` as the operator module's registry value in a values file, so the mapping is recorded on the operator's instance and survives every later install. (Requirement: A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly)

#### Scenario: Opt-in with the registry running

- **WHEN** `task cluster:operator KIND_CUE_REGISTRY='testing.opmodel.dev=opm-registry:5000+insecure,...'` runs and the registry container is up
- **THEN** the container is joined to kind's network, the operator's instance records the mapping as its registry value, and the operator Deployment carries the matching `--registry` argument

#### Scenario: Opt-in without the registry running

- **WHEN** `KIND_CUE_REGISTRY` is set and no registry container is running
- **THEN** the task SHALL fail with a message naming `task registry:start` and the option of unsetting the variable to use GHCR

### Requirement: The operator's registry is a command-line flag rendered from the registry value, not an environment variable
The operator's registry is a command-line flag the module renders from that value, not an environment variable: `--registry` wins over `OPM_REGISTRY`, which is read only when the flag is empty, and the operator overwrites `CUE_REGISTRY` in its own process environment. Setting either as a pod environment variable SHALL have no effect. (Requirement: When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file)

#### Scenario: A pod environment variable does not set the operator's registry

- **WHEN** `OPM_REGISTRY` or `CUE_REGISTRY` is set as a pod environment variable on the operator
- **THEN** setting either as a pod environment variable SHALL have no effect

### Requirement: The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current
The `cluster:operator` task SHALL ensure the CLI binary it invokes exists and is current before invoking it, by depending on the build task rather than assuming a previous build. The dependency SHALL reuse the build task's existing up-to-date check, so a tree whose binary is already current performs no rebuild and the task stays idempotent.

#### Scenario: Clean tree with no binary

- **WHEN** a developer runs `task cluster:operator` in a checkout that has never been built, against
  a running kind cluster
- **THEN** the CLI binary is built first and the operator install proceeds
- **AND** the task SHALL NOT fail reporting that the binary does not exist

#### Scenario: Binary already current

- **WHEN** `task cluster:operator` runs again with no source changes since the last build
- **THEN** no rebuild is performed
- **AND** the task completes as it did before, without duplicating any operator configuration

### Requirement: The `cluster:operator` task succeeds from a clean checkout and is callable unattended
The `cluster:operator` task SHALL therefore succeed from a clean checkout with no manual preparation beyond a running cluster, and SHALL be callable unattended: a caller that cannot answer a prompt or run a suggested command by hand, such as an automated test cleanup, SHALL NOT need one. (Requirement: The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current)

#### Scenario: Called unattended by a test cleanup

- **WHEN** a destructive e2e test invokes `task cluster:operator` to rebuild the dev operator it tore
  down, in a worktree where the binary was never built
- **THEN** the task builds what it needs and restores the operator without developer intervention
