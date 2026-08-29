## Purpose

Taskfile tasks for managing the lifecycle of a local kind (Kubernetes in Docker) cluster used for development and testing.

## Requirements

### Requirement: Cluster creation task

The Taskfile SHALL provide a `cluster:create` task that provisions a single-node kind cluster using a version-controlled configuration file.

The task SHALL use the `CLUSTER_NAME` Taskfile variable (default: `opm-dev`) as the kind cluster name and SHALL read the cluster configuration from `hack/kind-config.yaml`.

The task SHALL use the `K8S_VERSION` Taskfile variable (default: `1.34.0`) to select the node image via `--image kindest/node:v{{K8S_VERSION}}`. Developers MAY override this to test against a different Kubernetes version.

#### Scenario: Create cluster with defaults

- **WHEN** a developer runs `task cluster:create` with no variable overrides
- **THEN** a kind cluster named `opm-dev` is created using the configuration in `hack/kind-config.yaml` and the `kindest/node:v1.34.0` image

#### Scenario: Create cluster with custom Kubernetes version

- **WHEN** a developer runs `task cluster:create K8S_VERSION=1.33.0`
- **THEN** a kind cluster named `opm-dev` is created using the `kindest/node:v1.33.0` image

#### Scenario: Create cluster with custom name

- **WHEN** a developer runs `task cluster:create CLUSTER_NAME=my-test`
- **THEN** a kind cluster named `my-test` is created

#### Scenario: Kind not installed

- **WHEN** a developer runs `task cluster:create` and the `kind` binary is not on PATH
- **THEN** the task SHALL fail immediately with an error message that includes installation instructions (link to kind quickstart)

### Requirement: Cluster deletion task

The Taskfile SHALL provide a `cluster:delete` task that destroys the kind cluster identified by `CLUSTER_NAME`.

The deletion task SHALL be idempotent — deleting a cluster that does not exist MUST NOT produce an error.

#### Scenario: Delete an existing cluster

- **WHEN** a developer runs `task cluster:delete` and the `opm-dev` cluster exists
- **THEN** the kind cluster named `opm-dev` is destroyed and its resources are freed

#### Scenario: Delete a nonexistent cluster

- **WHEN** a developer runs `task cluster:delete` and no cluster named `opm-dev` exists
- **THEN** the task SHALL complete without error

#### Scenario: Kind not installed

- **WHEN** a developer runs `task cluster:delete` and the `kind` binary is not on PATH
- **THEN** the task SHALL fail immediately with an error message that includes installation instructions

### Requirement: Cluster status task

The Taskfile SHALL provide a `cluster:status` task that reports whether the named cluster is running and displays its connection details.

The task SHALL first check whether the cluster exists via `kind get clusters`. If the cluster exists, the task SHALL display connection information via `kubectl cluster-info --context kind-{{CLUSTER_NAME}}`.

#### Scenario: Status of a running cluster

- **WHEN** a developer runs `task cluster:status` and the `opm-dev` cluster is running
- **THEN** the task SHALL display the cluster name and the output of `kubectl cluster-info` for that cluster's context

#### Scenario: Status when cluster does not exist

- **WHEN** a developer runs `task cluster:status` and no cluster named `opm-dev` exists
- **THEN** the task SHALL display a message indicating the cluster is not running (e.g., "Cluster 'opm-dev' is not running")

#### Scenario: Kind not installed

- **WHEN** a developer runs `task cluster:status` and the `kind` binary is not on PATH
- **THEN** the task SHALL fail immediately with an error message that includes installation instructions

### Requirement: Cluster recreate task

The Taskfile SHALL provide a `cluster:recreate` task that destroys and re-creates the cluster in a single command.

The task SHALL compose `cluster:delete` followed by `cluster:create` using Taskfile's task composition (the `task:` directive in `cmds`).

#### Scenario: Recreate an existing cluster

- **WHEN** a developer runs `task cluster:recreate` and the `opm-dev` cluster exists
- **THEN** the existing cluster is destroyed and a fresh cluster is created with the same name and configuration

#### Scenario: Recreate when no cluster exists

- **WHEN** a developer runs `task cluster:recreate` and no cluster named `opm-dev` exists
- **THEN** the delete step completes without error and a new cluster is created

### Requirement: Kind cluster configuration file

A kind cluster configuration file SHALL exist at `hack/kind-config.yaml` and SHALL be used by the `cluster:create` task.

The configuration SHALL define a single-node cluster. The file SHALL be valid kind configuration (kind: Cluster, apiVersion: kind.x-k8s.io/v1alpha4).

#### Scenario: Configuration file is valid

- **WHEN** a developer inspects `hack/kind-config.yaml`
- **THEN** the file contains a valid kind Cluster resource with `apiVersion: kind.x-k8s.io/v1alpha4` and a single node entry with role `control-plane`

#### Scenario: Create uses configuration file

- **WHEN** a developer runs `task cluster:create`
- **THEN** kind receives the `--config hack/kind-config.yaml` flag, applying the settings defined in the configuration file

### Requirement: Precondition checks on all cluster tasks

Every task in the `cluster:*` namespace SHALL include a Taskfile precondition that verifies the `kind` binary is available on PATH before executing any commands.

The precondition failure message SHALL include a URL to the kind installation documentation.

#### Scenario: Precondition format

- **WHEN** a `cluster:*` task checks for the kind binary
- **THEN** it uses a Taskfile `preconditions` entry with `command -v kind` and a message containing `https://kind.sigs.k8s.io/docs/user/quick-start/#installation`

### Requirement: Integration test task documentation

The existing `test:integration` task SHALL include a `summary` field documenting that a running cluster is required and how to create one using `task cluster:create`.

The task's runtime behavior (the `cmds` list) MUST NOT change.

#### Scenario: Test integration summary mentions cluster

- **WHEN** a developer runs `task test:integration --summary`
- **THEN** the output includes guidance that a running cluster is needed and references `task cluster:create`

#### Scenario: Test integration behavior unchanged

- **WHEN** a developer runs `task test:integration`
- **THEN** it executes `go test ./tests/integration/... -v` exactly as before, with no additional preconditions or dependencies on cluster tasks

### Requirement: Operator install task needs no local registry

The `cluster:operator` task SHALL install the pinned operator into the kind cluster and SHALL NOT require a local registry container to do so. Everything the flow resolves — core, the catalogs, and this repository's fixture modules — is published on GHCR, and the pinned operator's built-in `--registry` default routes both `opmodel.dev/*` and `testing.opmodel.dev/*` there.

The task SHALL remain idempotent, SHALL apply the cluster `Platform` singleton and the dev-only RBAC grant, and SHALL confirm the operator is genuinely reconciling by waiting for `status.operatorVersion` on `Platform/cluster` rather than for pod readiness alone.

#### Scenario: Install with no registry container running

- **WHEN** a developer runs `task cluster:operator` against a running kind cluster with no `opm-registry` container
- **THEN** the task SHALL install the operator, apply the Platform and RBAC, and report the reconciling operator version

#### Scenario: Re-run is a no-op

- **WHEN** `task cluster:operator` is run a second time
- **THEN** it SHALL complete without error and without duplicating container arguments

### Requirement: Local registry is an explicit opt-in for the kind flow

Pointing the kind flow at a local registry SHALL require setting `KIND_CUE_REGISTRY` explicitly. That variable SHALL default to empty. When it is set, and only then, the task SHALL verify the registry container is running, join it to kind's docker network, and append `--registry` to the operator Deployment's container arguments — appending only if no `--registry` argument is already present.

The operator's registry is a command-line flag, not an environment variable: `--registry` wins over `OPM_REGISTRY`, which is read only when the flag is empty, and the operator overwrites `CUE_REGISTRY` in its own process environment. Setting either as a pod environment variable has no effect.

#### Scenario: Opt-in with the registry running

- **WHEN** `task cluster:operator KIND_CUE_REGISTRY='testing.opmodel.dev=opm-registry:5000+insecure,...'` runs and the registry container is up
- **THEN** the container is joined to kind's network and the operator Deployment is patched with the matching `--registry` argument

#### Scenario: Opt-in without the registry running

- **WHEN** `KIND_CUE_REGISTRY` is set and no registry container is running
- **THEN** the task SHALL fail with a message naming `task registry:start` and the option of unsetting the variable to use GHCR

#### Scenario: Default path reports its registry source

- **WHEN** `task cluster:operator` runs with `KIND_CUE_REGISTRY` unset
- **THEN** the task SHALL state that the operator's built-in GHCR default is in use and SHALL NOT patch the Deployment
