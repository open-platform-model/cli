## 1. Absence wait

- [ ] 1.1 `internal/operator/wait.go`: add `AbsentPredicate` and an absence mode of the poll loop in which a NotFound read satisfies the predicate; keep the existing readiness semantics unchanged.
- [ ] 1.2 Unit tests: absence mode returns when the object disappears; times out naming the object while it persists; readiness mode unaffected.

## 2. Terminating guard in Install

- [ ] 2.1 `internal/operator/install.go`: after the plan is resolved and before the apply loop, read each planned object; collect those with `metadata.deletionTimestamp` set; log one `waiting for … to finish terminating` line per object; wait for their absence under the shared `--timeout` deadline; on failure return an error naming the resource and the elapsed timeout without applying.
- [ ] 2.2 Unit tests with a fake dynamic client: terminating object delays apply until gone; terminating object beyond the budget fails with nothing applied; absent and live objects do not delay; `--crds-only` and `--rbac` plans are covered.
- [ ] 2.3 `internal/cmd/operator/install.go`: help text mentions that install waits for terminating objects left by a previous uninstall.

## 3. End-to-end

- [ ] 3.1 `tests/e2e/operator_test.go` lifecycle test: after the uninstall step, run `install` immediately (no pause) and assert it exits zero with the Deployment rolled out.
- [ ] 3.2 `task lint`, `task test:unit`, and the operator lifecycle e2e pass on a kind cluster.
