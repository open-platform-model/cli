# kind-cluster-tasks — Delta

## ADDED Requirements

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
