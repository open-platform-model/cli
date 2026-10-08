## Context

See `proposal.md` for the motivation. The state of the code at the base of this change (cli `main` at 4d4884fa):

- `internal/inventory/stale.go`: `PruneStaleResources` deletes each stale entry by its recorded name, with foreground propagation, no read and no precondition.
- `internal/kubernetes/delete.go`: `checkDeletable` reads each object again and compares the managed-by label and the UUID label; `deleteResource` sends no precondition. `opm instance delete` and `opm operator uninstall` share it through `internal/workflow/apply/delete.go`.
- `internal/operator/migration_execute.go`: `deleteProven` deletes with a precondition on the UID the migration proof read, after no ownership verdict.
- `internal/kubernetes/protected.go`: `IsProtectedKind` restates the library's `ownership.SafetyExcluded`.

Constraints:

- The library verdict is pure. The caller reads the live object and hands it in; the library reads no cluster and words the skip (`opm/k8s/ownership` package doc).
- The contract is 0012:D4:R1, 0012:D7:R1 and 0012:D8:R7 and R8.
- The fail-safe rules stay: a failed prune keeps the entry (cli#332), a failed record delete is an error (cli#338), reads resolve kinds by discovery (cli#342), PersistentVolumeClaims are kept unless `--delete-data` (cli#345), errors are told apart by type and never by message text (cli#346).
- No command syntax changes. No flag is added, removed or renamed.

Reversibility: costly two-way. The cli can change how it calls the verdict at any release. What users see (which objects a prune or delete leaves) is the contract of 0012:D8, already accepted by the owner.

## Goals / Non-Goals

**Goals:**

- One function, the library's, decides delete ownership for every object the cli deletes on behalf of an instance: prune, `instance delete`, `operator uninstall` and the migration's deletes. A test keeps it so.
- Prune never deletes an object without a live read and a verdict.
- Every delete after a verdict carries the UID precondition.
- No instance deletes an object whose adopt annotation names another instance.

**Non-Goals:**

- The apply guard, the adopt annotation on apply, and the hand-over on apply: change `guard-every-apply-by-ownership`.
- The deletion protocol: `opm/k8s/lifecycle`, the order, the finalizer hold, the propagation policy. A later change replaces the two delete loops with the library's deletion plan; this change keeps the loops and changes only the per-object decision and the precondition.
- The delete of the cli's own `ModuleInstance` record, and the removal of the cleanup finalizer from other instances' records by `operator uninstall`. Neither is a delete of an instance's object under 0012:D8.
- A change to the dry run.

## Decisions

### Paths, identities and verdicts

The cli MUST call `ownership.CanDelete` on each path below, with the identity named, and MUST act on each answer as stated.

| Path | Identity given to the verdict | Answer | cli action |
| --- | --- | --- | --- |
| Prune | The `InstanceUUID` of the record the apply read before it rendered anything into the cluster (`status.instanceUUID`), never the render's | proceed | DELETE, foreground, with `verdict.Preconditions()`. |
| | | `already-absent` | Done; the entry leaves the record. |
| | | `not-opm-managed`, `owner-mismatch`, `adopted-elsewhere` | Leave behind: one `left behind` warning line with the library's message; the entry leaves the record. |
| | | `safety-excluded` | Not reached for a split set; when reached, left behind as today. |
| | | (no verdict) read error other than NotFound; DELETE answers Conflict on the precondition | A failed prune: the entry stays in the record and the command fails after the write (cli#332). |
| `instance delete`, `operator uninstall` | The record's `InstanceUUID`, as today | proceed | DELETE, foreground, with `verdict.Preconditions()`. |
| | | `already-absent` | Neither deleted nor an error, as today. |
| | | any other skip | `left behind` with the library's message; not an error, as today; counted in the closing line, as today. |
| | | (no verdict) read error; Conflict on the precondition | A per-resource error: the record is kept and the command fails with the code it has today for a failed delete. |
| `operator install`, migration deletes | The operator instance's UUID from the render, with `Admit: true`, on the object the migration proof read, in the check phase | proceed | The delete in the write phase carries `verdict.Preconditions()`. This is the UID precondition `deleteProven` sends today. |
| | | any skip | A block of the migration refusal (exit 2), naming the object and the instance in the library's message; nothing is written. |

`IsProtectedKind` MUST delegate to `ownership.SafetyExcluded`, so the protected-kind rule has one definition. The split of protected kinds and of claims before prune stays as it is.

Deletes outside the verdict, on purpose: the `ModuleInstance` record itself (`internal/inventory/store.go`), which the cli owns by `spec.owner`. Section 2 of `tasks.md` adds a test that lists the allowed call sites of `.Delete(` on a resource client, so a new delete cannot bypass the verdict unseen.

### Shape in the code

```go
// internal/kubernetes: one judged delete, shared by instance delete and prune.
// It resolves the kind by discovery, reads the live object, asks
// ownership.CanDelete, and on proceed sends the DELETE with foreground
// propagation and the verdict's preconditions. A dry run stops after the verdict.
type DeleteOutcome struct {
    Deleted bool                 // the DELETE was accepted
    Skip    ownership.SkipReason // set when the verdict skipped
    Message string               // the library's message for the skip
}

// ErrReplaced reports a DELETE the API server refused on the UID
// precondition: the object was replaced since the read. Callers test it with
// errors.Is; it is never reported as deleted.
var ErrReplaced = errors.New("object was replaced since it was read")

func JudgedDelete(ctx context.Context, client *Client, obj ownership.Object, version, instanceUUID string, dryRun bool) (DeleteOutcome, error)
```

`checkDeletable`, `deleteResource`, `deleteEntry`, `reasonNotManaged` and `reasonOtherInstance` go. `PruneStaleResources` takes the recorded identity as an argument and returns the entries left behind with their messages beside the `*PruneError`.

Apply flow, real run (`internal/workflow/apply/apply.go`); only the marked steps change:

```text
read record (identity R) -> ... -> apply rendered objects (identity N stamped by the render)
  -> stale = StaleSet(previous, rendered)
  -> split protected, split claims
  -> prune through JudgedDelete with identity R                  * changed
  -> write record: rendered + failed prunes + kept claims,
     status.instanceUUID = N                                     (order unchanged: after the prune)
```

### Research & Decisions

#### Which identity prune judges with

**Context**: `CanDelete` skips as `owner-mismatch` when the live UUID label and the given identity are both set and differ. An instance's identity is a UUID v5 of `<module path without the major>:<instance name>:<namespace>` (core `src/module_instance.cue`). The record is found by name and namespace, so within one record the identity changes only when the module's path changes; a major bump and a move to another hosting registry do not change it. After such a change the rendered objects are relabelled by the apply, and every stale object still carries the old identity.
**Options considered**:
1. The render's identity. After a path change every stale object is skipped as `owner-mismatch` and dropped from the record at exit 0: abandoned.
2. The recorded identity (`status.instanceUUID` of the record the apply read).
3. Either identity.
**Decision**: Option 2 (owner decision of 2026-10-08: prune judges with the recorded identity, the one that applied the objects, also when the rendered identity differs).
**Rationale**: The recorded identity is the one the stale objects carry, and it is what `instance delete` uses, so prune and delete judge one object alike. The record takes the new identity only in the write after the prune, so the stale objects are always judged with the identity that applied them. A stale object that carries neither identity is another instance's and is left behind. A record with no stored identity disables the comparison, as the library defines.

#### Delete precondition

**Context**: The task asks for the UID and the resourceVersion "where the library asks for them".
**Explored**: `DeleteVerdict.Preconditions()` returns the UID only; its doc says a resourceVersion precondition fails on any status update or finalizer write between the read and the DELETE.
**Options considered**:
1. Status quo: no precondition (prune, instance delete).
2. UID only, from `verdict.Preconditions()`.
3. UID and resourceVersion.
**Decision**: Option 2.
**Rationale**: The UID closes the case that matters, an object deleted and recreated under the same name since the read. Option 3 would fail deletes of healthy workloads whose controllers write status. A relabel between the read and the DELETE stays possible; see Risks.

#### Read errors and precondition conflicts

**Context**: The library leaves the policy for a failed read to the frontend.
**Decision**: Fail closed everywhere. Prune: a failed prune (entry kept, exit 1 after the record write, or the discovery failure's code as cli#342 set). Instance delete and uninstall: a per-resource error (record kept), as today. A DELETE that answers Conflict on the UID precondition is `ErrReplaced`: reported as not deleted and handled as a failed delete on each path; the next run reads the new object and judges it. The Conflict is recognised by the API status reason, not by text (cli#346).
**Rationale**: Each of these is the outcome the path already has for the nearest existing failure, so no new exit code appears.

#### The migration's deletes: refuse, not skip

**Context**: Install deletes the earlier Deployment and the superseded bindings (0012:D8:R7). `CanDelete` can skip one, in practice when its adopt annotation names another instance. 0012:D8:R8 words that case as a skip.
**Options considered**:
1. Status quo: delete on the proof alone.
2. Skip the object and go on.
3. Refuse the install in the check phase, exit 2, naming the object and its owner.
**Decision**: Option 3 (owner decision of 2026-10-08: when `opm operator install` meets an object it needs that another instance owns, it refuses with exit 2 before changing anything).
**Rationale**: The earlier Deployment must go before the module's Deployment can apply, so an install that skips it cannot complete, and would report success on an operator it did not replace. This is the one place where the cli refuses on an object annotated for another instance, and it departs from the letter of 0012:D8:R8; the enhancement needs a revision note for it.

#### The stale set (0012:D7:R1)

**Context**: 0012:D7:R1 binds the end state: the stale set is exactly what the instance recorded and no longer renders.
**Decision**: The stale set is computed as today (`inventory.StaleSet`) and is not filtered. Ownership is decided per stale object by the verdict; an object the verdict skips is left in the cluster and leaves the record.
**Rationale**: The requirement's end state holds, and the mechanism is the library's.

#### The dry run

**Decision**: Unchanged in this change. The prune preview reads no stale object for a verdict, so it can list as `would prune` an object the real run leaves behind. `instance delete --dry-run` already runs the per-object check and now shows the library's reasons.
**Rationale**: A change to dry-run output is a visible change on its own. A follow-up change prints "would leave behind" lines; the migration note states the gap until then.

### Exit codes and messages

Codes: 0 success, 1 general, 2 validation, 3 connectivity, 4 permission denied, 5 not found. Cases not listed keep their code and their text; that includes every case cli#332, cli#334, cli#338, cli#341, cli#342, cli#345 and cli#346 settled.

| # | Case | Before | After | Reason |
| --- | --- | --- | --- | --- |
| 1 | Prune; the live stale object is not OPM-managed, carries a UUID other than the recorded one, or is adopted elsewhere | exit 0; deleted | exit 0; `left behind`, entry dropped | Audit finding B1; 0012:D4:R1, 0012:D8:R8. No code change. |
| 2 | Prune; the live read fails with an error other than NotFound | no read was made; the delete was tried | exit 1 after the record write, entry kept; a discovery failure keeps its code (4 or 3) | The failed-prune rule of cli#332 and cli#342, applied to the new read. |
| 3 | Prune; the DELETE answers Conflict on the UID precondition | not possible | as row 2: exit 1, entry kept | The library's rule: a failed precondition is never reported as deleted. |
| 4 | Prune on the first apply after the module's path changed; stale objects carry the recorded identity | exit 0; deleted | exit 0; deleted | No change: prune judges with the recorded identity. |
| 5 | `instance delete`, `operator uninstall`; the live object's adopt annotation names another instance | deleted | `left behind`; exit code unchanged (0 when nothing failed) | 0012:D8:R8. |
| 6 | `instance delete`, `operator uninstall`; left-behind reasons | "no longer managed by OPM", "owned by another instance" | the library's messages | One wording for both frontends. No code change. |
| 7 | `instance delete`, `operator uninstall`; the DELETE answers Conflict on the UID precondition | not possible | a per-resource error, record kept; `instance delete` exits with the code of `deleteFailureExitCode` (`internal/cmd/instance/delete_exit.go`), as for any failed delete | As row 3. |
| 8 | `operator install`; a proven earlier Deployment or binding gets a skip verdict (in practice: its adopt annotation names another instance) | deleted | exit 2; migration refusal naming the object and the instance, nothing written | Owner decision of 2026-10-08; 0012:D8:R8: no instance deletes an object annotated for another. |

Example, row 1 (`opm instance apply`):

```text
WARN ConfigMap/default/old  left behind  reason="ConfigMap/default/old is not managed by OPM; left in place"
```

Example, row 5 (`opm instance delete`, UUID shortened; the closing line that follows counts one resource left behind, as today):

```text
WARN ClusterRole/viewer  left behind  reason="ClusterRole/viewer is being adopted by module instance 9a40...17, not this one; left in place"
```

The reason is the library's text and MUST NOT be reworded by the cli. The lines around it are the cli's.

### Security

- Assets: objects in the cluster that this instance does not own, and their data.
- Trust boundary: the cluster API. Live labels and annotations are input that any principal with patch rights on the object can set.
- Threats and mitigations: deleting an object another party created or recreated under a recorded name (the live read, the verdict and the UID precondition); deciding on a read that failed (every check fails closed). Baseline: the contract of 0012:D8 and the library's verdict tests.
- The cli decides nothing about delete ownership itself after this change. Its own policy is limited to read errors, exit codes and output.
- Residual risk, owner: the cli maintainers. Enhancement 0012 records two risks of the adopt annotation (a mistyped UUID, an annotation removed too early) in its `05-risks.md`; the one below is not there yet and belongs beside them. A principal with patch rights on an object, and no delete rights, can set the adopt annotation to another instance's UUID. The object then survives `opm instance delete` and `opm operator uninstall`, and a prune. For a role binding or a network policy that is a grant or a rule that outlives the removal of its instance. On a proven earlier binding it blocks `opm operator install` (row 8).
- Detection: the object is never left silently. Each is printed as a `left behind` warning that names the instance in the annotation, and the closing line of `instance delete` and `operator uninstall` gives the number left behind. The exit code stays 0, because a left-behind object is the contract's outcome and not a failure.
- Refusal and left-behind messages carry object names and instance UUIDs. Neither is a secret.

## Risks / Trade-offs

- [A failed prune on the apply that follows a module path change] -> The record written by that apply holds the new identity and keeps the failed entries (cli#332). The retry judges them with the new identity, so they are left behind and dropped, with a warning each. The user deletes them by hand. The same holds for a PersistentVolumeClaim kept across a path change and pruned later with `--delete-data`. The same outcome follows when, after a path change, the apply relabels the rendered objects and its record write then fails: the record keeps the old identity, and an object a later render drops carries the new one. All three fail on the safe side: nothing is deleted that should stay. The owner accepted this limit on 2026-10-08.
- [The window between the delete read and the DELETE] -> The UID precondition closes the recreate case. A label or annotation change inside the window is not closed. The operator has the same window. Accepted.
- [One more GET per stale object on prune] -> `instance delete` already pays it.
- [Between this change and `guard-every-apply-by-ownership` the annotation is honoured on delete and not on apply] -> Stated in the migration note; both changes are meant for the same beta.
- [The fake dynamic client does not enforce delete preconditions] -> Unit tests assert the request's precondition and inject the Conflict; the integration script of section 2 checks the real API server's answer.
- [A later change replaces the delete loops with `opm/k8s/lifecycle`] -> `JudgedDelete` is the single seam: that change removes it and its two call sites. Outcomes and messages do not change again, because the lifecycle package uses the same verdict and the same precondition.

## Migration Plan

No data migration and no change to the record's shape. Rollback is a revert of the change; no stored state depends on it.

Migration note for the PR body and the release:

- Prune now reads each stale object before it deletes it. It leaves in the cluster, and removes from the inventory, an object that OPM does not manage, that belongs to another instance, or whose `opmodel.dev/adopt` annotation names another instance. Each is printed as `left behind` with the reason. Delete it with `kubectl delete` when nothing else needs it.
- `opm instance delete` and `opm operator uninstall` leave behind an object whose `opmodel.dev/adopt` annotation names another instance. The left-behind reasons have new wording.
- Prune judges a stale object with the identity stored in the instance's record. If a prune fails on the first apply after the module moved to a new path, the next apply leaves that object behind with a warning, because the record then holds the new identity. Delete it with `kubectl delete`.
- `opm instance apply --dry-run` does not run this check: it can list a stale object as `would prune` that the real run leaves behind.
- This release honours the adopt annotation when it deletes and not yet when it applies: an instance whose object is annotated for another instance still re-applies it until the release that carries the apply guard.
- `opm operator install` refuses, with exit 2 and nothing changed, when the earlier operator Deployment or a superseded role binding is annotated for another instance. Remove the annotation, then run the install again.

## Open Questions

- Follow-up change: the dry-run prune preview prints "would leave behind" lines.
- Follow-up after `guard-every-apply-by-ownership`: whether the `--rbac` objects of `opm operator install` come under any ownership rule. They are applied, never deleted by the cli, so this change does not meet them.
