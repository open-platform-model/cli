## Why

The shipped opm-operator has, by design, no permissions on workload kinds: it applies an instance's resources by impersonating `ModuleInstance.spec.serviceAccountName`, a field the CLI never writes. On the `kind-opm-dev` cluster the gap is closed by `hack/kind-operator-rbac.yaml`, which only `task cluster:operator` applies. An operator installed with `opm operator install` alone is complete for every purpose except the handoff e2e tests, which then fail 90 seconds later with `cannot patch resource "services"`, a message that points at the operator, not at the missing dev grant. The suite should check its own precondition and say what to run.

## What Changes

- The e2e operator tests gain a precondition probe: before any test that hands an instance to the operator or otherwise needs the operator to apply workloads, the harness checks (via a `SelfSubjectAccessReview`-equivalent, `kubectl auth can-i … --as=<operator ServiceAccount>`) that the operator's ServiceAccount may `patch` `services` and `deployments.apps` in the test namespace. If not, the test fails immediately with a one-line diagnosis naming the missing grant and the remedy: `task cluster:operator`.
- The probe is a `require*` helper beside `requireKindCluster`, used by the handoff, thin-editor and delete tests; the operator lifecycle test (which installs and uninstalls the operator itself) keeps its existing preconditions.
- `hack/kind-operator-rbac.yaml`'s header and the cli `CLAUDE.md` dev-cluster note state that `opm operator install` alone does not install the dev grant.

Not in scope: changing what `opm operator install` applies (the dev grant stays dev-only), the product-level question of how a handed-off instance gets an applier identity (a candidate enhancement, not this change), and the install terminating guard (sibling change `operator-install-terminating-guard`).

## Capabilities

### New Capabilities

- `e2e-cluster-preconditions`: what the cluster-backed e2e suite verifies about the dev cluster before running, and how it reports a cluster that is not prepared.

### Modified Capabilities

None.

## Impact

- Files: `tests/e2e/operator_test.go` (helper), `tests/e2e/instance_handoff_test.go` (call sites), `hack/kind-operator-rbac.yaml` (comment), `CLAUDE.md` (dev-cluster note).
- No product code; `test`/`chore` commits, no release.
- Enhancement: none.
