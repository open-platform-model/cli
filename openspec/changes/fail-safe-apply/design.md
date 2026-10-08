## Context

See proposal.md for the three defects. The apply flow is `Execute` in `internal/workflow/apply/apply.go`; the cluster reads and deletes it relies on are in `internal/inventory` (`GetRecord` in `store.go`, `PreApplyExistenceCheck` and `PruneStaleResources` in `stale.go`).

Observed at the base commit:

- `GetRecord` already separates NotFound (`nil, nil`) from every other error (`store.go:31-40`). The fail-open step is the caller: `LoadPreviousInventory` warns and returns no record (`apply.go:327-331`), and the dry-run ownership read does the same (`apply.go:111-118`).
- `PruneStaleResources` returns one summary error and no list of what failed (`stale.go:94-138`). `Execute` warns and writes the record from the current entries only (`apply.go:216-230`).
- `PreApplyExistenceCheck` skips any read error other than NotFound with a debug line (`stale.go:44-52`).
- Exit codes are the constants in `internal/exit/exit.go` (0 success, 1 general, 2 validation, 3 connectivity, 4 permission denied, 5 not found). `exitCodeFromK8sError` in `apply.go` maps an API error to them.

## Goals / Non-Goals

**Goals:**

- Each of the three failures stops or fails the command, with a unit test that fails before the fix.
- No change to the success path, to flags, or to the inventory format.

**Non-Goals:**

- A live label or instance check before a prune delete.
- The legacy inventory Secret read (`FindLegacySecretInventory`), which keeps its warn-and-go-on behavior. See Open Questions.
- A change to how a non-API error (a refused connection, for one) maps to an exit code.

## Research & Decisions

### Where the record read fails

**Context**: Two reads of the same record exist, one for the dry-run ownership preview and one in `LoadPreviousInventory`.
**Options considered**:
1. Return the error from both reads. Small diff, both sites fail closed.
2. Merge the two reads into one. Fewer calls, but it is a refactor inside a bug fix.
**Decision**: Option 1. `LoadPreviousInventory` MUST return `(record, legacy, error)`; `Execute` MUST return an `*exit.ExitError` with the code from `exitCodeFromK8sError` on both reads.
**Rationale**: A fix and a refactor in one commit hide each other. A dry run fails too, because a preview built without the record promises a first install that a real run would not perform.

```go
func LoadPreviousInventory(ctx context.Context, k8sClient *kubernetes.Client,
    name, namespace, instanceID string, dryRun bool, instanceLog *log.Logger,
) (*inventory.Record, *inventory.LegacyInventory, error)
```

### What a failed prune leaves in the record

**Context**: After a failed delete the object is still in the cluster. The record must keep tracking it.
**Options considered**:
1. Write nothing, as on an apply failure. The previous record keeps the stale entry, but the resources this apply created stay untracked until a later apply succeeds.
2. Write the current entries plus the entries that failed to delete, then fail the command.
3. Fail before the write and leave the user to re-run.
**Decision**: Option 2. `PruneStaleResources` keeps its signature and returns a `*inventory.PruneError` that carries the failed entries and their errors; `Execute` appends those entries to the entries it writes and returns exit 1 after the write.
**Rationale**: Only option 2 tracks both the new objects and the objects that are still there. A kept entry is stale again on the next apply, so a re-run converges; a delete that answers NotFound counts as done. Keeping the function's `error` result leaves the two integration programs that call it unchanged.

```go
type PruneError struct {
    Failed []k8sinventory.Entry // not deleted, input order of the delete loop
    Errs   []error              // one per failed entry, same order
}
```

An error from `PruneStaleResources` that is not a `*PruneError` cannot happen today; `Execute` treats it as a failure before the write (exit 1), so an unknown failure never drops entries.

### Exit codes

**Decision**: An unreadable record and an unreadable object in the existence check use `exitCodeFromK8sError`: 4 for Forbidden or Unauthorized, 3 for a server timeout or an unavailable server, 1 otherwise. A failed prune exits 1, as a failed resource apply does, because several deletes can fail for different reasons. The untracked and terminating refusals keep exit 1.

### Error text

Each message says what failed, the cause, and the next step, and names no internal symbol:

```text
cannot read the ModuleInstance record "demo" in namespace "default": <cause>
apply stopped: without the record it cannot tell a first install from an existing instance.
Check that you can read moduleinstances.opmodel.dev in that namespace, then run the command again.
```

```text
cannot check whether ConfigMap/app in namespace "default" already exists: <cause>
apply stopped before any change. Check that you can read that resource, then run the command again.
```

```text
r:ConfigMap/default/stale    prune failed  error=<cause>
1 stale resource(s) could not be pruned and stay in the inventory; fix the cause and run apply again to retry
```

## Risks / Trade-offs

- [A user who may patch a kind but not read it can no longer run a first install] -> The error names the resource and the missing read; granting `get` is the fix. Applying blind was the defect.
- [A flaky API server now fails an apply that passed with a warning] -> Intended. Re-running is safe: nothing was applied or written.
- [A record that keeps a stale entry reports one more resource than the render] -> It is the truth about the cluster, and the next successful prune removes it.

## Open Questions

- The legacy inventory Secret read has the same warn-and-go-on shape. Failing closed there would stop every first install for a user who cannot read Secrets, so it is left for an owner decision.
