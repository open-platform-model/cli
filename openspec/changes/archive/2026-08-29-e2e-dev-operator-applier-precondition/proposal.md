## Why

The shipped opm-operator has, by design, no permissions on workload kinds: it applies an instance's resources by impersonating `ModuleInstance.spec.serviceAccountName`, a field the CLI never writes. On the `kind-opm-dev` cluster the gap is closed by `hack/kind-operator-rbac.yaml`, which only `task cluster:operator` applies. An operator installed with `opm operator install` alone is complete for every purpose except the handoff e2e tests, which then fail 90 seconds later with `cannot patch resource "services"`, a message that points at the operator, not at the missing dev grant. The suite should check its own precondition and say what to run.

## What Changes

- The e2e operator tests gain a precondition probe: before any test that hands an instance to the operator or otherwise needs the operator to apply workloads, the harness checks (via a `SelfSubjectAccessReview`-equivalent, `kubectl auth can-i … --as=<operator ServiceAccount>`) that the operator's ServiceAccount may `patch` `services` and `deployments.apps` in the test namespace. If not, the test fails immediately with a one-line diagnosis naming the missing grant and the remedy: `task cluster:operator`.
- The probe extends `requireReconcilingOperator` (the helper the five handoff-family tests already call, which today checks only that a replica is available); the operator lifecycle test (which installs and uninstalls the operator itself) keeps its existing preconditions.
- `hack/kind-operator-rbac.yaml`'s header and the cli `CLAUDE.md` dev-cluster note state that `opm operator install` alone does not install the dev grant.

This change is a stopgap for the dev loop, not a fix of the underlying defect. The defect is in the product: `opm instance handoff` flips ownership without checking that the operator's effective applier identity can apply the instance's inventory kinds, and the CLI never writes `spec.serviceAccountName`, so on a stock `opm operator install` the flip (which is irreversible) strands the instance operator-owned and unreconcilable. The dev grant hides that from the e2e suite, which therefore never exercises the operator's impersonation path. The root fix is an enhancement with two parts: a pre-flip handoff gate (SubjectAccessReview of the effective applier identity against the inventory's GVRs) and a dev cluster that runs the production pattern (`--default-service-account=opm-applier` plus a fixture-shipped SA and namespace Role) in place of `hack/kind-operator-rbac.yaml`. When that lands, this probe is replaced by the fixture creating its own applier SA.

Not in scope: changing what `opm operator install` applies (the dev grant stays dev-only), the handoff gate and dev-cluster impersonation described above (the enhancement), and the install terminating guard (sibling change `operator-install-terminating-guard`).

## Capabilities

### New Capabilities

- `e2e-cluster-preconditions`: what the cluster-backed e2e suite verifies about the dev cluster before running, and how it reports a cluster that is not prepared.

### Modified Capabilities

None.

## Impact

- Files: `tests/e2e/instance_handoff_test.go` (`requireReconcilingOperator` and the identity helper), `hack/kind-operator-rbac.yaml` (comment), `CLAUDE.md` (dev-cluster note).
- No product code; `test`/`chore` commits, no release.
- Enhancement: none.
