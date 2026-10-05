## MODIFIED Requirements

### Requirement: Local registry is an explicit opt-in for the kind flow

Pointing the kind flow at a local registry SHALL require setting `KIND_CUE_REGISTRY` explicitly. That variable SHALL default to empty. When it is set, and only then, the task SHALL verify the registry container is running, join it to kind's docker network, and pass the mapping to `opm operator install` as the operator module's registry value in a values file, so the mapping is recorded on the operator's instance and survives every later install. When it is unset the task SHALL pass no registry value, so a mapping an earlier opt-in run recorded stays until `opm operator install --reset-values`. The task SHALL NOT patch the operator Deployment.

The operator's registry is a command-line flag the module renders from that value, not an environment variable: `--registry` wins over `OPM_REGISTRY`, which is read only when the flag is empty, and the operator overwrites `CUE_REGISTRY` in its own process environment. Setting either as a pod environment variable has no effect.

#### Scenario: Opt-in with the registry running

- **WHEN** `task cluster:operator KIND_CUE_REGISTRY='testing.opmodel.dev=opm-registry:5000+insecure,...'` runs and the registry container is up
- **THEN** the container is joined to kind's network, the operator's instance records the mapping as its registry value, and the operator Deployment carries the matching `--registry` argument

#### Scenario: Opt-in without the registry running

- **WHEN** `KIND_CUE_REGISTRY` is set and no registry container is running
- **THEN** the task SHALL fail with a message naming `task registry:start` and the option of unsetting the variable to use GHCR

#### Scenario: Default path reports its registry source

- **WHEN** `task cluster:operator` runs with `KIND_CUE_REGISTRY` unset
- **THEN** the task SHALL state that it sets no registry value, so the operator uses its built-in GHCR default unless an earlier run recorded a mapping, and SHALL NOT patch the Deployment
