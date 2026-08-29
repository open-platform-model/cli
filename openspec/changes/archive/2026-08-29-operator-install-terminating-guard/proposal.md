## Why

`opm operator uninstall` is fire-and-report: it issues foreground deletes and returns before the objects are gone. An `opm operator install` that follows immediately server-side-applies onto a Deployment that still carries the `foregroundDeletion` finalizer; the apply reports `configured`, the garbage collector then finishes the delete, and the install waits out its full `--timeout` for a resource that no longer exists. A second run fixes it, which is exactly the kind of "run it again" folklore the operator commands were built to remove.

## What Changes

- `opm operator install` gains a pre-apply terminating guard: before applying, every object in the plan that exists on the cluster with `metadata.deletionTimestamp` set is waited on until it is gone (bounded by `--timeout`, sharing the budget with the readiness wait), and only then applied. The wait is reported per resource. If the object is still terminating when the budget runs out, the command exits non-zero naming it, and applies nothing.
- The guard covers `--crds-only` and `--rbac` plans alike (it runs on the resolved plan).
- No new flag: waiting is the sensible default; there is no case where applying onto a terminating object is what the user wants.
- The readiness wait fails fast when an object it just applied reads NotFound, naming it, instead of polling until `--timeout`. The guard removes the known trigger; this removes the silent timeout for any trigger.

Not in scope: making `uninstall` wait for deletion (its fire-and-report contract stays; the guard makes install robust to any prior delete, not only uninstall), and the e2e precondition probe (sibling change `e2e-dev-operator-applier-precondition`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `operator-lifecycle`: "Full operator install from the embedded manifest" gains the terminating guard requirement and scenarios (terminating object waited out, terminating object outlives the budget).

## Impact

- Packages: `internal/operator` (`install.go`, `wait.go`; a new absence predicate), `internal/cmd/operator/install.go` (no flag change; help text mentions the wait).
- Tests: unit tests in `internal/operator` with a fake dynamic client returning a terminating object; `tests/e2e/operator_test.go` lifecycle test gains an uninstall-then-install step with no pause, which is the regression this fixes.
- SemVer: PATCH (`fix(operator)`); behavior only changes in the case that previously failed.
- Complexity: one pre-apply pass over the plan plus one predicate; justified by removing a non-deterministic install failure.
- Enhancement: none.
