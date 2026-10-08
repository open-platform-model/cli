## Why

`opm instance apply --create-namespace` and `opm module apply --create-namespace` create the
namespace as the first step of the apply, before the read-only checks that can refuse it (the
cluster gates, the `ModuleInstance` record read, the status-permission check, the first-install
existence check). A refused apply therefore leaves an empty namespace behind, and the refusal
text had to be narrowed to "no rendered resource was applied". A refusal must leave the cluster
as it found it.

Separately, `opm operator install` inherits the unreadable-object refusal of the first-install
existence check, and no test at the install level shows it.

## What Changes

- With `--create-namespace`, the apply reads first whether the namespace exists, runs every
  check that can refuse, and creates the missing namespace only after the last check passed,
  directly before the first write. A refused apply has created nothing.
- A check that reads inside the namespace that is still missing treats it as holding nothing:
  no record and no resource. This needs no new code path, because the API answers NotFound.
- The two refusal texts say again that the apply stopped before any change: the unreadable
  object of the existence check, and the unreadable `ModuleInstance` record.
- The success path keeps its result: the namespace is created and every resource is applied.
  Only the position of the line `namespace "<ns>" created` in the log moves, to after the
  checks. A dry run prints what it printed before, in the same order.
- Tests at the level of `opm operator install` show that it refuses an object it cannot read
  and writes nothing, and which exit code each of its three reads of that object leads to. The
  comment on its guard error names the unreadable case.

SemVer class: PATCH (a bug fix; no flag, no command and no exit code changes). On the beta line
it ships in the next `1.0.0-beta.N`.

Not changed: the first-install warning for unrecorded resources, every exit code, every flag,
and the checks themselves.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: a new requirement states where `--create-namespace` creates the namespace in the
  order of an apply, and that a refused apply creates none.
- `apply-pruning`: the existence check and the unreadable-record requirement drop the caveat
  that the namespace was already created, and say that nothing was changed.

## Impact

- `internal/workflow/apply/apply.go`: the order inside `Execute`; `EnsureNamespaceIfRequested`
  is split into a read and a create.
- `internal/inventory/stale.go`: one sentence of the unreadable-object refusal.
- `internal/operator/plan_install.go`: the comment on `GuardError`.
- Tests: `internal/workflow/apply/`, `internal/operator/`, and one mapping row in
  `internal/cmd/operator/operator_test.go` (the exit code is assigned there).
- No dependency, no API, no docs page: no user page states the order.
