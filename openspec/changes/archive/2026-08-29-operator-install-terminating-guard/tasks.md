## 1. Absence wait

- [x] 1.1 `internal/operator/wait.go`: add `AbsentPredicate` and an absence mode of the poll loop in which a NotFound read (`apierrors.IsNotFound`, nothing else) satisfies the predicate; any other Get error stays pending in both modes.
- [x] 1.2 `internal/operator/wait.go`: in readiness mode, a NotFound read fails the wait immediately with an error naming the object as applied and since disappeared (D2a). `CheckReady` keeps its single-shot "pending" semantics for NotFound; only the post-apply wait changes.
- [x] 1.3 Timeout text: derive the "timed out after" duration from the deadline actually in force (or the elapsed time), so the shared budget of D3 reports honestly.
- [x] 1.4 Unit tests: absence mode returns when the object disappears; times out naming the object while it persists; readiness mode fails fast on NotFound; readiness mode otherwise unaffected; a non-NotFound Get error is pending in both modes.

## 2. Terminating guard in Install

- [x] 2.1 `internal/operator/install.go`: after the plan is resolved and before the apply loop, read each planned object; collect those with `metadata.deletionTimestamp` set; log one `waiting for … to finish terminating` line per object; wait for their absence under the shared `--timeout` deadline; on failure return an error naming the resource and the elapsed timeout without applying.
- [x] 2.2 Unit tests with a fake dynamic client: terminating object delays apply until gone; terminating object beyond the budget fails with nothing applied; absent and live objects do not delay; `--crds-only` and `--rbac` plans are covered.
- [x] 2.3 `internal/cmd/operator/install.go`: help text mentions that install waits for terminating objects left by a previous uninstall.

## 3. End-to-end

- [x] 3.1 `tests/e2e/operator_test.go` lifecycle test: after the uninstall step, run `install` immediately (no pause) and assert it exits zero with the Deployment rolled out.
- [x] 3.2 `task lint`, `task test:unit`, and the operator lifecycle e2e pass on a kind cluster.
