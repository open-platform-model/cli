## MODIFIED Requirements

### Requirement: Operator applier grant is verified before handoff tests

Every e2e test that relies on the operator applying workload resources for a `ModuleInstance` SHALL first verify that the operator's effective applier identity is permitted to `patch` `services` (core API group) and `deployments` (`apps`) in the test namespace, using the cluster's own authorization check. The effective applier identity is the ServiceAccount named by the controller Deployment's `--default-service-account` argument, resolved in the test namespace, when that argument is present; otherwise the controller's own ServiceAccount. If the check reports denial, the test SHALL fail before performing any cluster mutation, with a message that names the identity probed, the denied verb and resource, and the remedy `task cluster:operator` (which applies `hack/kind-operator-rbac.yaml`). A denial SHALL be distinguished from a check that could not be performed: only an explicit `no` answer is a denial. If the check itself cannot be performed (no cluster, kubectl failure, no `impersonate` permission for the test user), the existing cluster-reachability precondition applies and the test skips as it does today, with the check's error in the skip message.

#### Scenario: Operator installed without the dev grant

- **WHEN** the operator was installed with `opm operator install` alone and an operator-owned e2e test starts
- **THEN** the test fails immediately, its message names `system:serviceaccount:opm-operator-system:opm-operator-controller-manager`, the denied `patch services`, and `task cluster:operator` as the remedy
- **AND** no `ModuleInstance` or workload has been created by the test

#### Scenario: Check cannot be performed

- **WHEN** the test user may not impersonate ServiceAccounts, so `kubectl auth can-i --as=...` errors instead of answering
- **THEN** the test skips, naming the kubectl error, rather than reporting a denial

#### Scenario: Cluster prepared by task cluster:operator

- **WHEN** `task cluster:operator` has been run and an operator-owned e2e test starts
- **THEN** the precondition passes silently and the test proceeds

#### Scenario: Lifecycle test is exempt

- **WHEN** the operator lifecycle e2e test runs on a cluster with no operator installed
- **THEN** it does not require the applier grant, since it installs and uninstalls the operator itself and applies no workloads

### Requirement: Dev grant is documented as a separate step

The dev-only RBAC grant SHALL be documented, in its own file header and in the repository's dev-cluster guidance, as applied by `task cluster:operator` and NOT by `opm operator install`, together with the symptom of its absence.

#### Scenario: Reader installs the operator by hand

- **WHEN** a developer reads the dev-cluster guidance before installing the operator on `kind-opm-dev`
- **THEN** the guidance states that `opm operator install` alone leaves the operator unable to apply workloads for operator-owned instances and names `task cluster:operator` as the complete path
