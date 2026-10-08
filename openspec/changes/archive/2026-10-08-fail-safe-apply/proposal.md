## Why

`opm instance apply` and `opm module apply` fail open in three places. Each one can lose track of cluster objects or take over an instance while the command still exits 0:

- A `ModuleInstance` record read that fails (a 5xx, a timeout, an RBAC denial) is logged as a warning and treated as "no record". The apply then runs as a first install: it applies resources directly even when the operator owns the instance, prunes nothing, and writes a record with owner `cli` at revision 1. The earlier inventory is gone.
- A stale resource that prune fails to delete is logged as a warning. The inventory is then written with the current render only, so the object stays in the cluster and no later apply or delete knows about it.
- On a first install, the existence check skips an object it cannot read, with a debug line. The forced server-side apply that follows can take over an object the check never saw.

## What Changes

- An apply (real or dry run) whose `ModuleInstance` record read fails with anything other than NotFound stops with an error before any rendered resource is applied and before any record is written. A namespace that `--create-namespace` created earlier in the run stays.
- When prune fails to delete a stale resource, the apply still writes the record, and the record keeps every entry that was not deleted. The output names each such resource with its error. The command exits non-zero and prints no success line. The next apply retries the prune.
- On a first install, an object the existence check cannot read (any error other than NotFound) refuses the apply. The same check guards `opm operator install`, which refuses too, with the exit code of its other apply-guard refusals (2).
- Exit codes come from the existing table (`internal/exit`): a failed read exits 4 on Forbidden or Unauthorized, 3 on a server timeout or an unavailable server, 1 otherwise; a failed prune exits 1.

No flag, no command and no inventory field changes. The success path does not change. The existence check still runs on a first install only.

Not in this change: the live label and instance check before a prune delete; the legacy inventory Secret read, which still falls back to a first install when it fails; resource-name resolution.

SemVer class: PATCH (bug fix). Before GA it ships in the next `1.0.0-beta.N`. Not breaking: only runs that already hit one of the three failures change outcome, from a warning and exit 0 to an error and a non-zero exit.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `apply-pruning`: the first-install existence check refuses on an unreadable object; a new requirement stops the apply on an unreadable record; a new requirement keeps entries that prune failed to delete and fails the command; the apply flow sequence names both outcomes.

## Impact

- `internal/workflow/apply/apply.go`: `Execute`, `LoadPreviousInventory` (gains an error result), `RunPreApplyExistenceCheck`, the prune step and the record write.
- `internal/inventory/stale.go`: `PreApplyExistenceCheck`, `PruneStaleResources` (its error now carries the entries it failed to delete).
- `tests/integration/migration/main.go`: one call site of `LoadPreviousInventory` follows the new signature.
- `internal/operator/plan_install.go` calls `PreApplyExistenceCheck` and inherits the refusal; no code change there.
- Unit tests in both packages. No dependency change.
