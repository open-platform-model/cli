## Purpose

The cluster-backed e2e suite runs against a developer's `kind-opm-dev` cluster whose preparation is a set of manual tasks. This capability states which preconditions the suite verifies before running and how it reports an unprepared cluster, so a missing preparation step fails fast with its remedy instead of surfacing as an unrelated product error.

## ADDED Requirements

### Requirement: Operator applier grant is verified before handoff tests

Every e2e test that hands a `ModuleInstance` to the operator, or otherwise relies on the operator applying workload resources, SHALL first verify that the operator's ServiceAccount is permitted to `patch` `services` (core API group) and `deployments` (`apps`) in the test namespace, using the cluster's own authorization check. If the check reports denial, the test SHALL fail before performing any cluster mutation, with a message that names the ServiceAccount, the denied verb and resource, and the remedy `task cluster:operator` (which applies `hack/kind-operator-rbac.yaml`). If the check itself cannot be performed (no cluster, kubectl failure), the existing cluster-reachability precondition applies and the test skips as it does today.

#### Scenario: Operator installed without the dev grant

- **WHEN** the operator was installed with `opm operator install` alone and a handoff e2e test starts
- **THEN** the test fails immediately, its message names `system:serviceaccount:opm-operator-system:opm-operator-controller-manager`, the denied `patch services`, and `task cluster:operator` as the remedy
- **AND** no `ModuleInstance` or workload has been created by the test

#### Scenario: Cluster prepared by task cluster:operator

- **WHEN** `task cluster:operator` has been run and a handoff e2e test starts
- **THEN** the precondition passes silently and the test proceeds

#### Scenario: Lifecycle test is exempt

- **WHEN** the operator lifecycle e2e test runs on a cluster with no operator installed
- **THEN** it does not require the applier grant, since it installs and uninstalls the operator itself and applies no workloads

### Requirement: Dev grant is documented as a separate step

The dev-only RBAC grant SHALL be documented, in its own file header and in the repository's dev-cluster guidance, as applied by `task cluster:operator` and NOT by `opm operator install`, together with the symptom of its absence.

#### Scenario: Reader installs the operator by hand

- **WHEN** a developer reads the dev-cluster guidance before installing the operator on `kind-opm-dev`
- **THEN** the guidance states that `opm operator install` alone leaves the operator unable to apply workloads for handed-off instances and names `task cluster:operator` as the complete path
