## Purpose

The cluster-backed e2e suite runs against a developer's `kind-opm-dev` cluster whose preparation is a set of manual tasks. This capability states which preconditions the suite verifies before running and how it reports an unprepared cluster, so a missing preparation step fails fast with its remedy instead of surfacing as an unrelated product error.

## Requirements

### Requirement: Operator applier grant is verified before operator-owned tests

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

### Requirement: The e2e suite resolves modules through the shipped default registry

The cluster-backed e2e suite SHALL resolve published modules through the registry mapping the CLI
ships as its default, and SHALL NOT depend on a registry process that no part of the suite starts.
The suite's stub home configuration SHALL take its registry from that shipped default by reference
rather than restating an address, so that an invocation which does not choose a registry for itself
resolves exactly as an end user's would and the suite cannot drift from what the CLI ships.

A test that needs to write to a registry — publishing a module, or scaffolding from a template it
publishes first — SHALL provide one it owns for the duration of that test, routing only the domains
it writes to at that registry and leaving `opmodel.dev` resolving from the shipped default. Such a
registry SHALL require no external process, port reservation or container, and SHALL be released
when the test ends.

A registry that a test does not itself provide SHALL NOT be a precondition of the suite: no test may
require a developer to start one by hand.

#### Scenario: A test that chooses no registry uses the shipped default

- **WHEN** an e2e test runs the CLI without selecting a registry of its own, and no local registry
  process is running
- **THEN** the CLI resolves `opmodel.dev` modules through the shipped default mapping
- **AND** the invocation SHALL NOT fail reporting that a published catalog has no published release

#### Scenario: The operator lifecycle test seeds a Platform

- **WHEN** the operator lifecycle e2e test installs the operator on a prepared cluster with no local
  registry running
- **THEN** the catalog version the install seeds the cluster `Platform` with resolves successfully
- **AND** the test proceeds to its assertions rather than failing on registry resolution

#### Scenario: A publishing test owns its registry

- **WHEN** an e2e test publishes a module
- **THEN** it publishes to a registry it started for that test, needing no external process
- **AND** that registry is released when the test ends, leaving nothing running

#### Scenario: No unreachable registry address is named

- **WHEN** the e2e suite's own configuration and helpers are inspected for a hardcoded local
  registry address
- **THEN** no such address is named as a default any test would fall through to
- **AND** the default the suite does fall through to is the shipped one, taken by reference, so it
  cannot be changed in one place and not the other

### Requirement: A destructive test that cannot restore the cluster fails its own run

An e2e test that tears down the shared dev cluster's operator SHALL restore it before the run ends,
and SHALL fail the run when it cannot. Reporting the failure without failing SHALL NOT satisfy this
requirement: a run that leaves the cluster unable to serve the operator-owned tests MUST NOT exit
successfully, because the next run's failure would otherwise appear as an unrelated product error
far from its cause.

The failure message SHALL name the restore step that did not complete, the underlying error, and
the command a developer runs to repair the cluster by hand.

#### Scenario: Restore fails

- **WHEN** a destructive e2e test completes its assertions but the step that rebuilds the dev
  operator fails
- **THEN** the test run SHALL fail
- **AND** the message names the failed restore, the underlying error, and the repair command

#### Scenario: Restore succeeds

- **WHEN** the same test completes and the rebuild step succeeds
- **THEN** the run reports no failure from the restore
- **AND** a subsequent operator-owned test finds a reconciling operator

#### Scenario: A poisoned cluster is never reported as success

- **WHEN** a destructive e2e run ends with the cluster's operator or CRDs absent because the restore
  did not complete
- **THEN** that run's exit status SHALL be non-zero
