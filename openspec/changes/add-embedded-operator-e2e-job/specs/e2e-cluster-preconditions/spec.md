## ADDED Requirements

### Requirement: A cluster-required run fails instead of skipping

When the environment variable `OPM_E2E_REQUIRE_CLUSTER` is set to `1`, every precondition in the e2e suite that would otherwise skip a test for want of a usable cluster SHALL fail that test instead. This covers a missing kubeconfig, an unreachable `kind-opm-dev` context, and an applier-grant check that could not be performed. The failure message SHALL start with `OPM_E2E_REQUIRE_CLUSTER=1:` and SHALL name the cause; for a missing kubeconfig or an unreachable context it SHALL also name the `kind-opm-dev` context and the remedy `task cluster:create`, and for a grant check that could not be performed it SHALL name the kubectl error. Precondition messages SHALL read correctly as either a skip or a failure, so none of them says "skipping". When the variable is unset or has any other value, those preconditions SHALL skip as they do without this requirement.

The variable SHALL NOT change any test that does not use the cluster, and SHALL NOT change the outcome of a precondition that already fails (an explicit applier-grant denial, a missing reconciling operator). When the variable is `1`, this requirement takes precedence over the skip clause of `Operator applier grant is verified before operator-owned tests`.

#### Scenario: Required cluster has no kubeconfig

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER=1` is set and the user's home has no `.kube/config`
- **THEN** each cluster-backed test fails, its message starting `OPM_E2E_REQUIRE_CLUSTER=1:` and naming the missing kubeconfig, the `kind-opm-dev` context and `task cluster:create`

#### Scenario: Required cluster unreachable

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER=1` is set and the `kind-opm-dev` context does not answer
- **THEN** each cluster-backed test fails, its message starting `OPM_E2E_REQUIRE_CLUSTER=1:` and naming the context and `task cluster:create`

#### Scenario: Developer machine without a cluster

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER` is unset and no kind cluster is running
- **THEN** the cluster-backed tests skip as before, and the rest of the suite runs

#### Scenario: Grant check cannot be performed in a required run

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER=1` is set and `kubectl auth can-i --as=...` errors instead of answering
- **THEN** the operator-owned test fails naming the kubectl error, rather than skipping

## MODIFIED Requirements

### Requirement: Operator applier grant is verified before operator-owned tests

Every e2e test that relies on the operator applying workload resources for a `ModuleInstance` SHALL first verify that the operator's effective applier identity is permitted to `patch` `services` (core API group) and `deployments` (`apps`) in the test namespace, using the cluster's own authorization check. The effective applier identity is the ServiceAccount named by the controller Deployment's `--default-service-account` argument, resolved in the test namespace, when that argument is present; otherwise the controller's own ServiceAccount. If the check reports denial, the test SHALL fail before performing any cluster mutation, with a message that names the identity probed, the denied verb and resource, and the remedy `task cluster:operator` (which applies `hack/kind-operator-rbac.yaml`). A denial SHALL be distinguished from a check that could not be performed: only an explicit `no` answer is a denial. If the check itself cannot be performed (no cluster, kubectl failure, no `impersonate` permission for the test user) and `OPM_E2E_REQUIRE_CLUSTER` is not `1`, the existing cluster-reachability precondition applies and the test skips as it does today, with the check's error in the skip message; with `OPM_E2E_REQUIRE_CLUSTER=1` it fails instead (`A cluster-required run fails instead of skipping`).

#### Scenario: Operator installed without the dev grant

- **WHEN** the operator was installed with `opm operator install` alone and an operator-owned e2e test starts
- **THEN** the test fails immediately, its message names `system:serviceaccount:opm-operator-system:opm-operator-controller-manager`, the denied `patch services`, and `task cluster:operator` as the remedy
- **AND** no `ModuleInstance` or workload has been created by the test

#### Scenario: Check cannot be performed

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER` is unset and the test user may not impersonate ServiceAccounts, so `kubectl auth can-i --as=...` errors instead of answering
- **THEN** the test skips, naming the kubectl error, rather than reporting a denial

#### Scenario: Cluster prepared by task cluster:operator

- **WHEN** `task cluster:operator` has been run and an operator-owned e2e test starts
- **THEN** the precondition passes silently and the test proceeds

#### Scenario: Lifecycle test is exempt

- **WHEN** the operator lifecycle e2e test runs on a cluster with no operator installed
- **THEN** it does not require the applier grant, since it installs and uninstalls the operator itself and applies no workloads
