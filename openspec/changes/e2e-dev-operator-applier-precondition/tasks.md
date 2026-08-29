## 1. Precondition helper

- [ ] 1.1 `tests/e2e/operator_test.go`: add `requireDevOperatorApplier(t, kubeconfig)` running `kubectl auth can-i patch services` and `patch deployments.apps` in the test namespace with `--as=system:serviceaccount:opm-operator-system:opm-operator-controller-manager`; on "no", `t.Fatalf` naming the ServiceAccount, the denied verb/resource, and `task cluster:operator` as the remedy.
- [ ] 1.2 Call it at the top of the five handoff-family tests in `tests/e2e/instance_handoff_test.go`, after `requireKindCluster`.

## 2. Documentation

- [ ] 2.1 `hack/kind-operator-rbac.yaml` header: state that `opm operator install` does not apply this file and name the symptom (`cannot patch resource "services"`).
- [ ] 2.2 `CLAUDE.md` dev-cluster note: `task cluster:operator` is the complete path for `kind-opm-dev`; `opm operator install` alone leaves handoff tests failing at the applier precondition.

## 3. Verification

- [ ] 3.1 On a cluster with the operator installed by `opm operator install` only, a handoff test fails at the precondition with the documented message and creates nothing.
- [ ] 3.2 After `task cluster:operator`, the full e2e suite passes.
