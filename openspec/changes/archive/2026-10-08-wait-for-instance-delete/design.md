## Context

`opm instance delete` of a CLI-owned instance drives the library's deletion plan (`lifecycle.Advance`) through `kubernetes.RunDeletion`, then deletes the `ModuleInstance` record when the hold verdict releases it. Every delete uses Foreground propagation and a UID precondition. The command returns when every planned action is done; it does not look at the resources again.

The command already has `--timeout` (default 5 minutes). Today it bounds only the operator-managed path, where the command deletes the `ModuleInstance` and always waits for the operator's finalizer to let it go.

`opm instance apply` already has `--wait` with `--timeout`: opt-in, skipped on `--dry-run`, exit 1 on timeout, and "operator-managed instances always wait for the operator".

## Goals / Non-Goals

**Goals:**

- An opt-in, bounded wait that ends when the deleted resources are gone.
- A timeout that names what is still terminating and why (finalizers).
- No change at all without the flag.

**Non-Goals:**

- Removing finalizers, or forcing a delete.
- Waiting for resources the command did not delete.
- Any change to the apply path or to `opm operator uninstall`.

## Decisions

### 1. Flag names: `--wait`, bounded by the existing `--timeout`

`opm instance apply` uses the same pair with the same meaning ("wait for the end state, bounded by `--timeout`"), so a flag keeps one meaning across commands. `--timeout` exists on `instance delete` with a default of `5m0s`; a second timeout flag would give one command two budgets. The budget starts when the wait starts, as in apply.

Alternative: wait by default. Rejected: the task requires that nothing changes without a flag, and scripts rely on the fast return.

### 2. What "gone" means

A deleted resource is gone when its read returns NotFound, or when the live object has a UID other than the one the delete was sent with. The second case is an object that something created again under the same name: the object this run deleted no longer exists, and waiting for the new one would never end. The UID is the one in the delete's precondition, so the run records it per step. A delete sent without a UID precondition is waited for by NotFound only.

Any other read error keeps the resource pending; the last error is shown at timeout. A failed read is never taken as "gone".

### 3. Which resources are waited for

Only the steps the plan recorded as `deleted` on a real run. Kept PersistentVolumeClaims are not in the plan. Skipped steps (protected kind, not managed, another instance, adopted, already absent) had no delete sent. Failed steps had no accepted delete.

### 4. A timeout holds the record and exits 1

Options for the record when the wait times out:

- **Delete it anyway** (the wait is only a report). A re-run then finds no instance (exit 5), so the user cannot wait again through `opm`, and OPM has stopped tracking resources that still exist.
- **Keep it** (chosen). The record still lists resources that still exist, which is true. A re-run reads them, sends the deletes again (the API server accepts a delete of a terminating object), and waits again. A re-run without `--wait` behaves as today and deletes the record. This matches how every other per-resource failure of delete behaves: the record is kept and re-running is safe.

So the wait runs after the deletes and before the record delete.

Exit code: 1. Code 3 is the connectivity class: the API server did not answer in time or was unavailable (cli#332). Here the server answered every poll and said that the resources exist. Nothing is wrong with the connection, and the remedy differs (remove a finalizer, wait longer). The two existing waits of the cli agree: `instance apply --wait` and the operator-managed delete wait both exit 1 on timeout.

### 5. A run with per-resource failures does not wait

Such a run already fails and keeps the record. Waiting up to the timeout for the other resources delays the failure report and adds nothing: the re-run that follows the fix waits.

### 6. Operator-managed instance: the flag changes nothing

There the cli deletes only the `ModuleInstance`, and it already waits for it to be gone, bounded by `--timeout`, on every run. That is what `--wait` would mean, so `--wait` is accepted and adds nothing. Refusing the flag would break a script that passes `--wait` to every delete without knowing the owner; making the existing wait depend on the flag would change today's behaviour. The help states it, in the words `instance apply` uses. The cli does not check the pruned resources of an operator-managed instance.

### 7. `opm operator uninstall` does not get the flag

Uninstall shares the delete routine, so the option is reachable. It is not wired: the flag also needs a timeout flag, a timeout report, help, a spec requirement and tests there, which is more than a few lines; and the race it would close is closed on the other side, because `opm operator install` waits for terminating objects before it applies. It is a question for the owner.

### 8. Where the code goes

- `kubernetes.StepResult.UID`: set from the delete precondition when a delete was accepted.
- `kubernetes.WaitDeleted`: the poll helper. It takes its budget from the context deadline, like `Wait` and `WaitAbsent`, and uses `WaitPollInterval`. On the deadline it returns a `*TerminatingError` that lists the pending objects with their finalizers or last read error. On cancellation it returns the context error.
- `kubernetes.DeleteResult.WaitUntilGone`: waits for the objects the run of the result deleted and puts the pending ones in `DeleteResult.Terminating`. `Delete` itself does not wait.
- `workflowapply.DeleteRecorded`: with `DeleteRequest.Wait`, on a real run without per-resource errors, it calls `WaitUntilGone` after it printed the kept, left-behind and error lines, so the line that says the command is waiting is the last one before the silence. It does not release the record when `Terminating` is not empty.
- The command prints the report.

While it waits, the command prints how many resources are left each time the number drops and every 15 polls (30 seconds) otherwise, on any stream: at most one line per resource plus one per 30 seconds. A read that answers after the deadline still counts; an object that no read answered for is reported as "not read before the timeout".

`WaitAbsent` is not reused: it knows no UID, and its timeout is one error string with no finalizers.

## Risks / Trade-offs

- A resource with a finalizer that no controller removes makes every `--wait` run time out and keep the record. → The timeout names the finalizers and says that a run without `--wait` deletes the record.
- A read that keeps failing (for example Forbidden on get) keeps the resource pending until the timeout. → The report shows the read error for that resource.
- The poll sends one GET per pending resource every 2 seconds. → Bounded by the timeout; the set only shrinks.

## Open Questions

- Should a timeout delete the record anyway (decision 4)? Built: the record is kept.
- Should `opm operator uninstall` get `--wait` (decision 7)? Built: no.
