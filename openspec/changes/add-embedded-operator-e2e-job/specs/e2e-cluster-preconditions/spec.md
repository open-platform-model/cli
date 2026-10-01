## ADDED Requirements

### Requirement: A cluster-required run fails instead of skipping

When the environment variable `OPM_E2E_REQUIRE_CLUSTER` is set to `1`, every precondition in the e2e suite that would otherwise skip a test for want of a usable cluster SHALL fail that test instead, with the same message the skip would have printed. This covers a missing kubeconfig, an unreachable `kind-opm-dev` context, and an applier-grant check that could not be performed. When the variable is unset or has any other value, those preconditions SHALL behave exactly as they do without this requirement: they skip.

The variable SHALL NOT change any test that does not use the cluster, and SHALL NOT change the outcome of a precondition that already fails (an explicit applier-grant denial, a missing reconciling operator).

#### Scenario: Required cluster unreachable

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER=1` is set and the `kind-opm-dev` context does not answer
- **THEN** each cluster-backed test fails, its message naming the context and `task cluster:create`

#### Scenario: Developer machine without a cluster

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER` is unset and no kind cluster is running
- **THEN** the cluster-backed tests skip as before, and the rest of the suite runs

#### Scenario: Grant check cannot be performed in a required run

- **WHEN** `OPM_E2E_REQUIRE_CLUSTER=1` is set and `kubectl auth can-i --as=...` errors instead of answering
- **THEN** the operator-owned test fails naming the kubectl error, rather than skipping
