## Context

`apply.Execute` runs the ownership guard (`inventory.Guard`) only when `!dryRun`. On a dry run it calls the guard once, for the first-install warning, and drops the refusal (`previewAlreadyManaged`). The prune preview (`previewPrune`) prints the stale set without a read. Both gaps were left on purpose by the change that put the guard on every apply.

The guard only reads. The prune runs the library's deletion plan through `kubernetes.RunDeletion`, which already has a `DryRun` option: it performs the reads, asks the delete verdict and sends no delete. `opm instance delete --dry-run` uses it.

## Goals / Non-Goals

**Goals:**

- One judgement for the dry run and the real run: the same function, the same input, so the two cannot drift.
- A dry run whose exit code a CI job can gate on.
- No write of any kind on a dry run.

**Non-Goals:**

- A `--dry-run` flag for `opm operator install`. The command has none.
- A change to the dry run of an operator-managed instance. The operator applies that instance, so the CLI guard does not apply to it.
- A change to `opm instance delete`.
- Running the cluster gates (CRD presence, field floor, operator ceiling, status RBAC) on a dry run.

## Decisions

### The dry run calls the same guard

`Execute` calls `inventory.Guard` with the same `GuardInput` on both paths. Only the wording of the result differs:

```go
// real run: unchanged
guard, err = RunOwnershipGuard(ctx, client, guardInput, firstApply, nothingChanged)
// dry run: same guard, per-object lines, then the error
guard, err = previewOwnershipGuard(ctx, client, guardInput, firstApply, instanceLog)
```

`previewAlreadyManaged` is deleted: the first-install warning reads `guard.Managed` from the same call.

Alternative: a second, "lenient" judgement for the dry run. Rejected: two judgements drift, which is the defect this change removes.

### A previewed refusal exits 1 and stops the dry run

The dry run exits with the code of the real refusal (1). A read failure exits with the code of the real read failure (4 denied, 3 unavailable, 1 other).

- A dry run exists to answer "will the apply work". Exit 0 for an apply that will be refused answers wrongly, and a CI job reads only the exit code.
- `opm instance delete --dry-run` already reports its refusal with the same outcome as the real run, and `opm module publish --dry-run` exits 2 on a refusal. The apply dry run was the exception.
- The same code as the real run means a script needs one table.

After the refusal the dry run stops: no server-side dry run of the other objects and no prune preview. The real run stops there too, and what would follow a refused apply is not something the real run does. All refused objects are listed in one pass, so the user does not fix them one run at a time.

Alternative: print the lines and exit 0. Rejected for the first reason above.
Alternative: a distinct exit code for "previewed refusal". Rejected: a new code is a new contract for one case, and the real code already says what would happen.

### Output

Lines go to the log stream (stderr), like every other resource line of the apply. The samples below are the lines of a unit-test run of `apply.Execute` for an instance named `demo`, with the UUIDs shortened; the `cannot check` line and the two closing lines are composed from the code. Each uses the existing resource-line format with a new status word and the library's message as the reason. The message names the object, the owner where there is one, and the adopt annotation to set.

Dry run, guard refuses (exit 1):

```text
ERRO m:demo: r:ConfigMap/default/settings                      ! would refuse reason="ConfigMap/default/settings exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c0a52-..."
dry run: a real apply would be refused: 1 object(s) cannot be applied by this instance (listed above)
the dry run changed nothing
```

Dry run, an object is adopted elsewhere (exit 0):

```text
WARN m:demo: r:ConfigMap/default/shared                        ! would skip reason="ConfigMap/default/shared is being adopted by module instance 9a40c1de-...; this instance does not apply it; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c0a52-..."
```

Dry run, prune preview:

```text
INFO m:demo: would prune 1 stale resource(s)
INFO m:demo: r:ConfigMap/default/old-a                           would prune
WARN m:demo: r:ConfigMap/default/old-b                         ! would keep reason="ConfigMap/default/old-b belongs to module instance 9a40c1de-..., not this one; left in place"
WARN m:demo: r:ConfigMap/default/old-c                         ! would let go reason="ConfigMap/default/old-c is being adopted by module instance 9a40c1de-..., not this one; left in place"
ERRO m:demo: r:ConfigMap/default/old-d                         ! cannot check error="..."
```

Everything adopted elsewhere:

```text
WARN m:demo: nothing applied: all 2 rendered resource(s) are adopted by another instance          (real run, exit 0)
INFO m:demo: dry run complete: nothing would be applied: all 2 rendered resource(s) are adopted by another instance
```

Status words and their meaning, one word per outcome:

| Status | Where | Meaning | Exit |
| --- | --- | --- | --- |
| `would refuse` | dry run, rendered object | the real apply refuses | 1 |
| `would skip` | dry run, rendered object | the real apply lets it go (adopted elsewhere) | 0 |
| `would prune` | dry run, stale object | the real prune deletes it | 0 |
| `would keep` | dry run, stale object | not managed by OPM, or of another instance: the real prune leaves it | 0 |
| `would let go` | dry run, stale object | another instance is adopting it: the real prune leaves it | 0 |
| `cannot check` | dry run, stale object | the read failed: the real prune fails | 1, or 4 or 3 for a failed discovery request |

The real prune prints `left behind` for both `would keep` and `would let go`. The preview splits them because the cause differs and so does what the user does next. `kept` stays the word for a PersistentVolumeClaim kept without `--delete-data`.

### The prune preview runs the deletion plan read-only

`inventory.PruneStaleResources` and a new `inventory.PreviewPruneStaleResources` share one function that takes `dryRun` and passes it to `kubernetes.RunDeletion`. The preview therefore gets the plan's order, the same verdict and the same failure rules, and adds no call site that can send a DELETE.

```go
func PreviewPruneStaleResources(ctx, client, stale, instanceUUID) (wouldPrune []k8sinventory.Entry, leftBehind []LeftBehind, err error)
```

`LeftBehind` gains the verdict's skip reason, so the caller picks the status word without reading text.

A failed read in the preview is a failed step, as in the real prune. The dry run then exits with the code the real apply would use. The reason is the same as for the guard: the real apply fails there.

### Error handling

| Situation on a dry run | Output | Exit |
| --- | --- | --- |
| Guard refuses | one `would refuse` line each, then the error | 1 |
| Guard cannot read a rendered object | the real run's error, with "the dry run changed nothing" | 4, 3 or 1 |
| Object adopted elsewhere | `would skip` line | 0 |
| Stale object fails the verdict | `would keep` or `would let go` line | 0 |
| Stale object cannot be read | `cannot check` line, then an error | 1, 4 or 3 |
| Server-side dry run has errors | as before | 1 |

When both the server-side dry run and the prune preview fail, the apply error is returned, as the real run skips the prune after a failed apply.

## Risks / Trade-offs

- [A CI job that ran `--dry-run` against a cluster with a foreign object passed before and fails now] → That is the intent. The real apply fails in the same place. The docs page and the PR body say so.
- [The dry run sends more reads: one per stale object] → The real prune sends the same reads. The count is bounded by the inventory.
- [A reader with rights to patch but not to get a kind] → The dry run now fails where it passed. The real run fails the same way, with the same message.
- [The preview can still differ from the real run when the cluster changes between the two] → Unavoidable; the real run judges again.

## Migration Plan

No step for a user. A dry run that fails after the upgrade names the objects and the fix.

## Open Questions

None that changes the specs or the tasks.
