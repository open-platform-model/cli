## 1. Precondition helper

- [x] 1.1 `tests/e2e/instance_handoff_test.go`: add `operatorApplierIdentity(t, kubeconfig)` reading the controller Deployment's container args; `--default-service-account=<sa>` yields `system:serviceaccount:default:<sa>`, otherwise `system:serviceaccount:opm-operator-system:opm-operator-controller-manager`.
- [x] 1.2 `tests/e2e/instance_handoff_test.go`: extend `requireReconcilingOperator` so that, after the replica check, it runs `kubectl auth can-i patch services` and `patch deployments.apps` in the test namespace with `--as=<identity>`, parsing stdout: `no` calls `t.Fatalf` naming the identity, the denied verb/resource, and `task cluster:operator`; `yes` passes; anything else calls `t.Skipf` with the kubectl error (the check could not be performed).
- [x] 1.3 No call-site changes: the five handoff-family tests already call `requireReconcilingOperator`.

## 2. Documentation

- [x] 2.1 `hack/kind-operator-rbac.yaml` header: state that `opm operator install` does not apply this file and name the symptom (`cannot patch resource "services"`).
- [x] 2.2 `CLAUDE.md` dev-cluster note: `task cluster:operator` is the complete path for `kind-opm-dev`; `opm operator install` alone leaves handoff tests failing at the applier precondition.

## 3. Verification

- [x] 3.1 On a cluster with the operator installed by `opm operator install` only, a handoff test fails at the precondition with the documented message and creates nothing.
- [x] 3.2 After `task cluster:operator`, the full e2e suite passes.

## 4. Follow-up

- [x] 4.1 File the enhancement named in the proposal (handoff pre-flip applier gate; dev cluster on `--default-service-account` with a fixture-shipped applier SA and Role) and reference it from the proposal before this change is archived.
