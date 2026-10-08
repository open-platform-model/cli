## Context

The library package `opm/k8s/lifecycle` (library `v1.0.0-beta.7`, `opm/k8s/lifecycle/doc.go`) is the deletion protocol of enhancement 0012:D4. It has three parts:

- `NewDeletionPlan(entries, policy, ownerUUID)` orders inventory entries by descending kind weight (stable) and marks every CustomResourceDefinition and Namespace as skipped before any read.
- `Advance(plan, state, event)` names one action per call: `read`, `delete`, `skip` or `done`. The caller performs a read or a delete with its own client and hands back the object or the raw error. Every judged step goes through `ownership.CanDelete`. Every delete it names carries Foreground propagation and a precondition on the UID of the judged object. A failed step is recorded and the plan goes on.
- `MayReleaseHold(plan, state, input)` says whether the instance's deletion hold may come off: yes when every step was deleted or skipped, no when a step failed or the plan is unfinished.

Two facts about the package shape this design. Both were read from the source, not from the package doc:

1. **The hold is the instance's hold, not a hold on an object.** `MayReleaseHold` (`hold.go:83`) answers one question: may the thing that keeps the instance alive be removed. For the operator that is its cleanup finalizer on the ModuleInstance. For the cli it is the ModuleInstance record, which the cli deletes last. The package has no verdict about a finalizer on a tracked object.
2. **The package never waits.** A delete the API server accepted is `ResultDeleted` at once (`answerDelete`, `advance.go:234`). The package reads no clock. An object that stays Terminating after an accepted delete is, to the plan, deleted.

The cli today (base `4bf7f04b`):

- `kubernetes.JudgedDelete` (`internal/kubernetes/delete.go:278`) reads one object, asks `ownership.CanDelete`, and deletes with Foreground propagation and the verdict's UID precondition. cli#347 added it.
- `kubernetes.Delete` (`delete.go:135`) is the loop of `opm instance delete` and `opm operator uninstall`: sort descending, keep claims, skip protected kinds, `JudgedDelete`, collect errors.
- `inventory.PruneStaleResources` (`internal/inventory/stale.go:197`) is the prune loop: sort descending, `JudgedDelete`, stop at a failed discovery request.
- `workflowapply.DeleteRecorded` (`internal/workflow/apply/delete.go:71`) prints the lines and deletes the record when no object failed.
- `deleteProven` (`internal/operator/migration_execute.go:101`) sends the migration's deletes.
- `TestDeleteCallSites` (`internal/kubernetes/deletesites_test.go`) allows a DELETE in exactly three functions: `JudgedDelete`, `deleteProven`, `DeleteCR`.

So the cli already behaves as the protocol asks, object by object. What it lacks is the protocol as the driver.

Reversibility: two-way door. The change is internal to the cli, behind unchanged commands, flags, output and exit codes. Reverting it is one revert commit.

## Goals / Non-Goals

**Goals:**

- Every delete of an instance's object is sent because `lifecycle.Advance` named it (0012:D4:R1).
- The record of a deleted instance is removed only on a release verdict of `lifecycle.MayReleaseHold`.
- One function in the cli sends those deletes.
- Output, exit codes, the PersistentVolumeClaim rule (cli#345), the discovery resolver and its stop rule (cli#342), the failed-read rules (cli#332, cli#334, cli#338) and the typed error rule (cli#346) stay as they are.

**Non-Goals:**

- Waiting for deleted objects to disappear, or any other new behaviour on a finalizer.
- The apply path (`guard-every-apply-by-ownership`).
- The dry-run prune preview. It still reads no stale object.
- The migration deletes of `opm operator install`.
- New flags, new commands, a library bump.

## Research & Decisions

### One runner drives every plan

**Context**: `Advance` names actions; something in the cli must perform them. Two loops exist today.

**Explored**: `opm/k8s/lifecycle/advance.go`, `plan.go`; the two cli loops; `deletesites_test.go`.

**Options considered**:

1. Status quo: keep `JudgedDelete` and both loops. No work, but 0012:D4:R1 is not met and a rule added to the library plan never reaches the cli.
2. Each loop calls `Advance` itself. Meets R1, but the read, the delete and the error mapping are written twice.
3. One runner in `internal/kubernetes` that takes a plan and returns one result per step; the two callers only build the plan and report.

**Decision**: option 3.

**Rationale**: the runner becomes the single DELETE site of the cli, which the call-site test can pin by name. The callers keep what differs between them: which entries enter the plan, what is printed, what is written afterwards.

```go
// internal/kubernetes/deletion.go

// DeletionOptions configures one run of a deletion plan.
type DeletionOptions struct {
	// DryRun performs the reads and sends no delete. A delete the plan
	// names is recorded as it would be after a successful delete.
	DryRun bool
	// StopOnDiscoveryFailure answers every read after the first failed
	// API discovery request with that failure, without a request.
	StopOnDiscoveryFailure bool
	// Unreadable holds the read error of entries the caller already
	// failed to read. The runner answers the plan's read of such an entry
	// with that error and sends no second request.
	Unreadable map[k8sinventory.Entry]error
}

// StepResult is what happened to one step of the plan.
type StepResult struct {
	Entry   k8sinventory.Entry
	Outcome lifecycle.Outcome
	// Failed is the action that failed (read or delete); empty otherwise.
	Failed lifecycle.ActionKind
	// Err is the raw error of the failed action, so callers classify it by
	// type (IsDiscoveryFailure, IsKindNotServed, ErrReplaced).
	Err error
}

// DeletionRun is a finished plan: its final state and one result per step,
// in plan order.
type DeletionRun struct {
	Plan  lifecycle.DeletionPlan
	State lifecycle.State
	Steps []StepResult
}

// RunDeletion drives plan to done with the cli's client. It is the only
// function of the cli that deletes an object of an instance. It returns an
// error only when Advance refuses the state, which a run that starts from
// the zero State cannot cause.
func RunDeletion(ctx context.Context, client *Client, plan lifecycle.DeletionPlan, opts DeletionOptions) (DeletionRun, error)
```

The loop, in outline:

```go
state, ev := lifecycle.State{}, lifecycle.Event{}
for {
	next, act, err := lifecycle.Advance(plan, state, ev)
	if err != nil { return run, err }
	record(run, state, next)          // outcomes appended by this call
	state, ev = next, lifecycle.Event{}
	switch act.Kind {
	case lifecycle.ActionDone:
		return run, nil
	case lifecycle.ActionSkip:
		// already recorded
	case lifecycle.ActionRead:
		ev.Live, ev.Err = read(ctx, client, act.Entry, opts) // resolver, then GET
	case lifecycle.ActionDelete:
		if !opts.DryRun {
			ev.Err = resource.Delete(ctx, act.Entry.Name, metav1.DeleteOptions{
				PropagationPolicy: &act.Propagation,
				Preconditions:     act.Preconditions,
			})
		}
	}
}
```

The runner sets no propagation and no precondition of its own: it passes the action's.

### Which entries enter the plan

**Context**: a plan is built from inventory entries. `kubernetes.Delete` takes live objects today, read by `inventory.DiscoverResourcesFromInventory` before the prompt.

**Options considered**:

1. Convert the live objects back to entries. Keeps the input, loses nothing visible, but builds the plan from a second-hand list, and 0012 says the plan comes from the persisted inventory.
2. Build the plan from the record's entries. The discovery read stays, because the prompt needs it to name the claims `--delete-data` deletes.

**Decision**: option 2. Instance delete and uninstall: `rec.Inventory.Entries`, minus every PersistentVolumeClaim unless `--delete-data`. Prune: the prunable stale entries, exactly the slice `pruneStale` passes today (protected entries and kept claims are already split off by `SplitProtected` and `SplitDataClaims`).

**Rationale**: the claim rule is the cli's (cli#345), so it is applied before the plan and the plan never sees a kept claim. A kept claim is reported from the discovery read as today, and a claim that is already gone is not reported, as today. The policy is `lifecycle.Policy{Prune: true}` on every path: the user asked for the delete. The owner UUID is the record's `InstanceUUID`, as cli#347 settled, also for prune after an identity change.

### An entry the discovery read could not read

**Context**: cli#338 made such an entry a per-object failure of instance delete and uninstall, without a second read (`deploy`: "Instance delete keeps the ModuleInstance when a tracked resource could not be read").

**Options considered**:

1. Let the plan read it again. Simpler: the `Unreadable` plumbing goes. But an entry that failed at discovery and reads fine a second later is then deleted, which changes a rule cli#338 specified.
2. Answer the plan's read with the discovery error (`DeletionOptions.Unreadable`).

**Decision**: option 2.

**Rationale**: no behaviour change. The outcome is a failed step, so the hold verdict holds and the record is kept, as today. An unreadable CRD or Namespace is skipped by the plan before any read, so it is left behind, as today. Option 1 is a fair follow-up, with its own spec change.

### The stop at a failed discovery request (prune)

**Context**: when an API discovery request fails, prune stops: that entry and every entry not yet tried are reported as failed with the discovery error, and the exit code is the code of that failure (cli#342). `Advance` does not stop at a failure.

**Decision**: the runner keeps the rule as its own read policy (`StopOnDiscoveryFailure`, set by prune only). After the first `*DiscoveryError`, it answers every later read with the same error and sends no request. The plan then records each remaining step as failed, which is the result today's `break` builds by hand.

**Rationale**: the rule is about how the cli resolves resources, so it belongs to the cli's performer. The plan still names every action.

### The dry run of instance delete

**Context**: `opm instance delete --dry-run` reads and judges every object and deletes nothing. The package has no dry-run mode.

**Options considered**:

1. A second code path that calls `ownership.CanDelete` directly. The preview and the real run can then drift.
2. Drive the same plan; when it names a delete, send nothing and hand back an empty event.

**Decision**: option 2.

**Rationale**: the preview lists what the real run would do by construction. The state is held in memory for one command (0012:D4) and thrown away; on a dry run the cli asks no hold verdict and deletes no record.

### The record delete follows the hold verdict

**Decision**: `DeleteRecorded` deletes the record when the run is real, a record exists and `lifecycle.MayReleaseHold(run.Plan, run.State, lifecycle.HoldInput{})` releases. The identity is always available: the cli deletes with the user's own credentials.

**Rationale**: with `Policy.Prune` true the verdict releases exactly when no step failed, or the plan is empty. That is today's `len(deleteResult.Errors) == 0`. The verdict's message is not printed: the cli keeps its own lines.

### The migration deletes stay outside the plan

**Decision**: `deleteProven` is unchanged and stays an allowed delete site.

**Rationale**: 0012:D4:R1 names it as the one exception. Its objects are proven earlier-manifest objects, not inventory entries of an OPM instance; their verdict needs `ownership.DeleteInput.Admit`, which `NewDeletionPlan` does not take; and the verdict must be taken in the check phase, before the install's first write, so that a refusal changes nothing.

## Data flow

```text
opm instance delete (CLI-owned) / opm operator uninstall
  record ──► entries − kept claims ──► NewDeletionPlan(Prune, record UUID)
                                             │
                    ┌────────────────────────▼─────────────────────────┐
                    │ RunDeletion: Advance ─► read / delete / skip ─► … │
                    └────────────────────────┬─────────────────────────┘
                                             ▼
      StepResults ──► lines (deleted, left behind, kept, errors)
      MayReleaseHold ──► release: DeleteCR(record)   hold: record kept, exit 1

opm instance apply / opm module apply (prune)
  stale set ──► SplitDataClaims, SplitProtected ──► prunable
  prunable ──► NewDeletionPlan(Prune, recorded UUID) ──► RunDeletion(StopOnDiscoveryFailure)
  StepResults ──► left-behind lines; failed entries stay in the written record
```

## What a user sees

### Lines and codes that change

Command syntax, flags, the prompt, every status word, every reason and every closing line stay. These are the only differences:

| # | Command | Today | With this change | Reason |
| --- | --- | --- | --- | --- |
| 1 | `opm instance delete`, `opm operator uninstall` | The error line of an object the discovery read could not read is printed first, before any `deleted` line | The same line, in the object's place in the delete order | The plan reports steps in plan order |
| 2 | `opm instance delete`, `opm operator uninstall`, debug level only | A recorded object that was already gone at the discovery read prints nothing | The debug line `resource already gone` | The plan is built from the record, so it reads that entry and finds it absent |
| 3 | all three | Cannot occur | `reading <Kind>/<name>: reading <object> returned <other object>; not deleted`, counted as a failed object | The plan refuses to judge a read that returned another object than the step's. A conforming API server never does this |

No exit code changes:

| Case | Exit code, today and after |
| --- | --- |
| Instance delete or uninstall, every object deleted, left behind or kept | 0 |
| A read or delete of an object fails (Forbidden, Conflict on the UID precondition, any other error) | 1, record kept |
| A failed object whose error is a failed discovery request | 4 when denied, 3 when unavailable, record kept |
| The record delete fails after the objects are gone | 4, 3 or 1 by cause |
| A guard refuses (operator instance, armed finalizers, no record) | 2 |
| Prune: a stale object fails to delete | 1 after the record write; 4 or 3 when a failed discovery request stopped the prune |

### Propagation, order and waiting

Nothing here changes, and a script sees no difference:

- **Foreground propagation.** The cli has sent it since its first delete command. The delete call returns when the API server has accepted it; the object is then Terminating until the garbage collector has removed its dependents.
- **Order.** Descending kind weight, stable, as today. Both the cli and the plan sort with the library's `object.Sort`.
- **No wait.** `opm instance delete` and `opm operator uninstall` report an object as `deleted` when its delete is accepted, delete the record, and exit. They do not wait for the object to disappear (`operator-lifecycle`: "SHALL NOT wait for deletion to complete"). A script that needs the objects gone after `opm instance delete` must wait itself, for example with `kubectl wait --for=delete`. That was true before this change.

### An object that holds a finalizer nobody removes

The delete of such an object is accepted: the API server sets its `deletionTimestamp` and keeps the object.

- **How long the cli waits**: it does not wait.
- **What it prints**: the object's line with the status `deleted`, then the closing line `Instance deleted` (or the prune's normal lines).
- **Exit code**: 0.
- **What is left**: the object stays Terminating. After `opm instance delete` the record is gone, so no opm command tracks or retries it. After a prune the entry is out of the written record. A later first apply of an instance that renders the same object stops with the "is terminating (deletionTimestamp set)" error of the first-install check (`internal/inventory/stale.go:72`); `opm operator install` waits for it under `--timeout`.

This is today's behaviour, and the package gives no other: its hold verdict is about the instance, and it counts an accepted delete as deleted. Whether the cli should wait is owner question 1.

## What of cli#347 this replaces, and what stays

Replaced (removed by this change):

- `kubernetes.JudgedDelete`, `kubernetes.DeleteOutcome`, `outcomeOf` (`internal/kubernetes/delete.go`)
- `liveReadError` and `kubernetes.IsLiveReadFailure`: the runner knows which action failed (`StepResult.Failed`)
- the per-object switch of `kubernetes.Delete` and the loop of `inventory.PruneStaleResources`
- `judgeddelete_test.go`, whose cases move to the runner's test
- the entry `internal/kubernetes/delete.go:JudgedDelete` of `allowedDeleteSites`, replaced by the runner

Kept:

- `kubernetes.ErrReplaced`: the runner wraps a Conflict on a preconditioned delete in it, so the line and its type stay
- `kubernetes.IsProtectedKind` and `ProtectedKindReason`: the cli's wording for a step the plan skips as safety-excluded
- prune judged with the recorded identity; the left-behind lines; entries left behind dropped from the record
- the migration's check-phase verdict and `deleteProven`
- `TestDeleteCallSites` and `tests/integration/delete-ownership`
- every spec requirement cli#347 wrote: none is modified

The two changes do not fight over files as long as `guard-every-apply-by-ownership` lands first: it works in `internal/workflow/apply` (the apply stages) and `internal/operator` (install), this one in the delete and prune functions of the same packages. The implementing agent merges the base that holds it before the first edit.

## Error handling

- A failed read or delete is a failed step; the plan goes on. The caller prints `reading <Kind>/<name>: <err>` or `deleting <Kind>/<name>: <err>` from `StepResult.Failed` and `StepResult.Err`, as today.
- A Conflict on a delete that carried a precondition is wrapped in `ErrReplaced`, recognised by `apierrors.IsConflict`, never by text.
- A kind the cluster does not serve, and a failed discovery request, are read errors with their own types; `hintUnservedKinds` and the exit-code functions keep working on `StepResult.Err`.
- No error is matched by its text (`TestNoErrorTextMatch`).
- `RunDeletion` returns an error only when `Advance` refuses the state. The caller treats it as an internal failure: nothing more is deleted, the record is kept, exit 1.

## Security

- Assets: the objects of other instances and of users that share a name with a recorded entry; the instance record.
- Trust boundary: the API server's answers. The cluster content is untrusted input to the verdict; the verdict itself is the library's.
- Threats and mitigations: a delete of an object the instance does not own (every step is judged by `ownership.CanDelete` inside `Advance`; the cli cannot skip it); a delete of an object recreated since the read (UID precondition from the action); a delete added later that bypasses the plan (`TestDeleteCallSites`, with the runner as the only instance delete site); loss of the record while objects remain (release verdict).
- The change adds no credential, no network endpoint and no input. It removes a place where the cli could decide a delete on its own.
- Residual risk: an object held by a finalizer is reported as deleted and is no longer tracked. Unchanged by this change; owner: the cli maintainers, through owner question 1.

## Risks / Trade-offs

- [The runner changes a line or a count the tests do not pin] → the existing suites of cli#332 to cli#347 (`delete_test.go`, `keptclaims_test.go`, `prunefail_test.go`, `prunediscovery_test.go`, `pruneownership_test.go`, `uninstall_test.go`, `unreadable_test.go`) run unchanged except where a removed symbol is named; the status goldens stay.
- [The plan's order differs from today's for objects of equal weight] → both are stable sorts over record order with the same weight table; section 1 adds a test that compares the plan's order with `SortObjects` over the same entries.
- [The fake dynamic client does not enforce preconditions] → as in cli#347, unit tests assert the request and inject the Conflict; the real answer is step 3 of `tests/integration/delete-ownership`, in CI.
- [A double read per object: discovery, then the plan's read] → already so today (`JudgedDelete` reads again). No new request.
- [The library package may still change before v1] → the runner is the only importer of `lifecycle` besides the two callers that build plans; a break is absorbed in three files.

## Migration Plan

No user migration. Rollback is a revert of the PR. The change is three sections, each green on its own: the runner (unused), then instance delete and uninstall, then prune and the removal of `JudgedDelete`.

## Open Questions

Owner decisions, each with a recommendation:

1. **Should `opm instance delete` wait for the deleted objects to disappear?**
   - a. No, as today. The command stays fire-and-report. (Recommended for this change: the brief keeps output and exit codes, and flags are out of scope.)
   - b. Yes, in a follow-up change: wait under `--timeout`, name the objects still Terminating, exit non-zero and keep the record. This is the only way a stuck finalizer becomes visible; it changes output, exit codes and the meaning of `--timeout`, so it needs its own proposal. (Recommended as the next step.)
   - c. Yes, in this change. Not recommended: it makes this refactor a behaviour change.
2. **An entry the discovery read could not read: keep it a failure, or let the plan read it again?**
   - a. Keep it a failure with no second read, as cli#338 specified. (Recommended: no behaviour change.)
   - b. Read again; a transient read error then no longer fails the delete. Needs a `deploy` and `operator-lifecycle` spec change.
3. **Does this change claim 0012:D4 in the delivery log?**
   - a. No decision claimed; the log takes 0012:D4 when the operator's deletion adoption has merged too. (Recommended; it follows the ownership change.)
   - b. Claim 0012:D4 now for the cli half.
