## Context

See `proposal.md` for the motivation. The state of the code at the base of this change:

- `internal/inventory/stale.go`: `FirstInstallCheck` reads every rendered entry and refuses a terminating, an untracked or an unreadable object. It passes every OPM-managed object, whichever instance its UUID label names. `RunPreApplyExistenceCheck` in `internal/workflow/apply/apply.go` skips it when a record exists and on a dry run.
- `internal/inventory/stale.go`: `PruneStaleResources` deletes each stale entry by its recorded name, with foreground propagation, no read and no precondition.
- `internal/kubernetes/delete.go`: `checkDeletable` reads each object again and compares the managed-by label and the UUID label; `deleteResource` sends no precondition.
- `internal/operator/plan_install.go`: the apply guard runs only when the operator instance has no record. `internal/operator/migration_execute.go`: `deleteProven` deletes with a precondition on the UID the migration proof read.
- `internal/kubernetes/protected.go`: `IsProtectedKind` restates the library's `ownership.SafetyExcluded`.

Constraints:

- The library verdicts are pure. The caller reads the live object and hands it in; the library reads no cluster and words the refusal (`opm/k8s/ownership` package doc).
- The contract is 0012:D4:R2 and 0012:D8:R1 to R8, with the "annotation only" rule of 0012:D8:R8.
- The fail-safe rules stay: an unreadable record or object stops the apply (cli#332), a refusal before the first write changes nothing (cli#334), a failed prune keeps the entry (cli#332), a failed record delete is an error (cli#338), reads resolve kinds by discovery (cli#342), PersistentVolumeClaims are kept unless `--delete-data` (cli#345).
- No command syntax changes. No flag is added, removed or renamed.

Reversibility: costly two-way. The cli can change how it calls the verdict at any release. What users see (which applies are refused, the adopt annotation) is the contract of 0012:D8, already accepted by the owner; this change does not decide it.

## Goals / Non-Goals

**Goals:**

- One function decides apply ownership and one decides delete ownership, for every cli path, and both are the library's.
- Prune never deletes an object without a live read and a verdict.
- Every delete after a verdict carries the UID precondition.
- An object adopted by another instance leaves the inventory and stays in the cluster.

**Non-Goals:**

- The deletion protocol: `opm/k8s/lifecycle`, the order, the finalizer hold, the propagation policy. The next change replaces the two delete loops with the library's deletion plan; this change keeps the loops and only changes the per-object decision and the precondition.
- A command or flag that sets the adopt annotation (0012:D8:R6 excludes it).
- A change to what a dry run refuses.
- Ownership of the `--rbac` objects of `opm operator install`.

## Decisions

### Paths and verdicts

The cli MUST call the verdict named here on each path and MUST act on each answer as stated.

| Path | Input to the verdict | Answer | cli action |
| --- | --- | --- | --- |
| Apply guard (`instance apply`, `module apply`, the instance apply inside `operator install`) | `CanApply{Object, Live, InInventory, InstanceUUID, Admit}` for every rendered object. `InInventory` is true when the previous record lists the same object (group, kind, namespace, name). `Admit` is true only for an object in the operator install's admit set. | allowed | Apply. |
| | | `terminating`, `foreign-object`, `other-instance` | Refuse the apply before its first write. Report every refused object, one line each, with the library's message. |
| | | `adopted-elsewhere` | Do not apply the object. One warning line with the library's message. Leave it out of the record. The apply goes on and this refusal alone does not change the exit code (0012:D8:R8). |
| | read error other than NotFound and other than "kind not served" | (no verdict) | Refuse the apply before its first write, as cli#332 set for the first install. |
| Prune | `CanDelete{Object, Live, InstanceUUID}` for every prunable stale entry, after the protected-kind split and the claim split | proceed | DELETE, foreground, with `verdict.Preconditions()`. |
| | | `already-absent` | Done; the entry leaves the record. |
| | | `not-opm-managed`, `owner-mismatch`, `adopted-elsewhere` | Leave behind: one `left behind` warning line with the library's message; the entry leaves the record. |
| | | `safety-excluded` | Not reached for a split set; when reached, left behind as today. |
| | read error other than NotFound; DELETE answers Conflict on the precondition | (no verdict) | A failed prune: the entry stays in the record and the command fails after the write (cli#332). |
| `instance delete`, `operator uninstall` | `CanDelete{Object, Live, InstanceUUID}` on the existing re-read; `InstanceUUID` is the record's | proceed | DELETE, foreground, with `verdict.Preconditions()`. |
| | | `already-absent` | Neither deleted nor an error, as today. |
| | | any other skip | `left behind` with the library's message; not an error, as today. |
| | read error; Conflict on the precondition | (no verdict) | A per-resource error: the record is kept and the command fails, as today for a failed re-read or delete. |
| `operator install`, check phase guard | The apply guard above over every object the plan applies, on every install | refusals | `GuardError`, exit 2, as today. `adopted-elsewhere` is not a refusal here either: the CRD step and the instance apply both skip the object. |
| `operator install`, migration deletes | `CanDelete{Object, Live, InstanceUUID, Admit: true}` on the object the migration proof read, in the check phase | proceed | The delete in the write phase carries `verdict.Preconditions()`. This is the UID precondition `deleteProven` sends today. |
| | | any skip | A block of the migration refusal (exit 2), with the library's message; nothing is written. |

`IsProtectedKind` MUST delegate to `ownership.SafetyExcluded`, so the protected-kind rule has one definition. The split of protected kinds and of claims before prune stays as it is.

### Shape in the code

```go
// internal/inventory: the apply guard, replacing FirstInstallCheck.
type GuardInput struct {
    Entries      []k8sinventory.Entry  // every rendered object, in render order
    Previous     []k8sinventory.Entry  // the record's inventory; nil on a first apply
    InstanceUUID string
    Admit        AdmitSet              // keyed by ownership.Object; nil for every caller but operator install
}

type GuardResult struct {
    Managed []k8sinventory.Entry // allowed, exists, OPM-managed, not in Previous: the cli#333 warning counts these
    LetGo   []LetGo              // refused as adopted-elsewhere: entry and library message
}

// Guard returns a *GuardRefusal (every terminating, foreign-object and
// other-instance verdict) or the read error of the first object it could not
// read. It writes nothing.
func Guard(ctx context.Context, client *kubernetes.Client, in GuardInput) (GuardResult, error)

// internal/kubernetes: one judged delete, shared by instance delete and prune.
// It reads the live object, asks ownership.CanDelete, and on proceed sends the
// DELETE with the verdict's preconditions.
type DeleteOutcome struct {
    Deleted bool                 // the DELETE was accepted
    Skip    ownership.SkipReason // set when the verdict skipped
    Message string               // the library's message for the skip
}

func JudgedDelete(ctx context.Context, client *Client, obj ownership.Object, version, instanceUUID string, dryRun bool) (DeleteOutcome, error)
```

`AdmitSet` becomes `map[ownership.Object]struct{}`; `K8sIdentity` and `IdentityOf` go, as their comment at `internal/inventory/admit.go` announces. `checkDeletable`, `deleteResource`, `deleteEntry`, `reasonNotManaged` and `reasonOtherInstance` go.

Apply flow, real run (`internal/workflow/apply/apply.go`):

```text
cluster gates -> read record -> ownership mode -> status RBAC -> empty-render guard
  -> Guard(all rendered entries, previous inventory, UUID, admit)     refuses here, before any write
  -> cli#333 warning from GuardResult.Managed (first apply only)
  -> create namespace -> Apply(rendered minus LetGo)
  -> stale = StaleSet(previous, rendered)                              a LetGo object is rendered, so never stale and never pruned
  -> split protected, split claims -> prune through JudgedDelete
  -> write record: (rendered minus LetGo) + failed prunes + kept claims
```

With `--create-namespace` and a missing namespace the guard skips the objects in that namespace, as the existence check does today.

### Research & Decisions

#### Where the apply verdict runs

**Context**: 0012:D8:R1 needs the verdict on every apply. The cli must also keep "a refusal changes nothing" (cli#334).
**Explored**: `internal/workflow/apply/apply.go` (gate order), `internal/kubernetes/apply.go` (`applyOne` reads the object just before its patch).
**Options considered**:
1. Status quo: first install only. Leaves the takeover on later applies; contradicts 0012:D8:R1.
2. One pass before the first write, over every rendered object. All-or-nothing refusal; one extra GET per object on later applies.
3. Judge inside `applyOne` on its own read. No extra GET and the smallest window between read and write, but a refusal arrives after earlier objects were applied.
**Decision**: Option 2.
**Rationale**: It keeps the fail-safe rule that a refused apply has written nothing, and it is the shape the existence check already has. The cost is one read per rendered object on applies that had none.

#### What a hand-over does to the inventory and the prune

**Context**: 0012:D8:R8 says the instance drops the object and never deletes it; 0012:D7:R1 words this as "in the stale set, and the deletion plan skips it".
**Options considered**:
1. Remove the object from the rendered set before the stale set is computed, so prune sees it as stale and `CanDelete` skips it.
2. Keep it out of the apply and out of the record, and compute the stale set from the full render, so prune never sees it.
**Decision**: Option 2.
**Rationale**: The outcome is the one 0012:D7:R1 asks for (not recorded, not deleted), with no path on which a prune of that object is even attempted. If the annotation goes away later, the next apply judges the object as one outside the inventory.

#### Delete precondition

**Context**: The brief asks for the UID and the resourceVersion "where the library asks for them".
**Explored**: `DeleteVerdict.Preconditions()` returns the UID only; its doc says a resourceVersion precondition fails on any status update or finalizer write between the read and the DELETE.
**Options considered**:
1. Status quo: no precondition (prune, instance delete).
2. UID only, from `verdict.Preconditions()`.
3. UID and resourceVersion.
**Decision**: Option 2.
**Rationale**: The UID closes the case that matters, an object deleted and recreated under the same name since the read. Option 3 would fail deletes of healthy workloads whose controllers write status. A relabel between the read and the DELETE stays possible; see Risks.

#### Read errors and precondition conflicts

**Context**: The library leaves the policy for a failed read to the frontend.
**Decision**: Fail closed everywhere. Apply: refuse before any write, with the exit code of the read error (4 denied, 3 unavailable, 1 other), the rule cli#332 set. Prune: a failed prune (entry kept, exit 1 after the record write, or the discovery failure's code as cli#342 set). Instance delete and uninstall: a per-resource error (record kept), as today. A DELETE that answers Conflict on the UID precondition is reported as not deleted and handled as a failed delete on each path; the next run reads the new object and judges it.
**Rationale**: Each of these is the outcome the path already has for the nearest existing failure, so no new exit code appears.

#### Operator install

**Context**: Install applies CRDs itself before the instance apply, proves which earlier-manifest objects it may take (0012:D8:R6) and deletes the earlier Deployment and bindings (0012:D8:R7).
**Options considered**:
1. Status quo: guard only without a record; with a record, the instance apply takes over anything.
2. The guard on every install; admitted objects pass `Admit: true`; migration deletes pass `CanDelete` with `Admit: true` in the check phase, on the object the proof read.
**Decision**: Option 2. A skip verdict on an object the migration would delete refuses the install in the check phase.
**Rationale**: 0012:D8:R1 has no exception for an instance with a record. Judging the migration deletes in the check phase keeps "every refusing check runs before the first write". Refusing on a skip, in place of leaving the object, is the safe side: the earlier Deployment must go before the module's Deployment can apply, and an install that cannot remove it cannot complete.

#### The dry run

**Context**: Spec `apply-pruning` says a dry run refuses nothing and that its first-install look ends silently on an object a real run would refuse.
**Options considered**:
1. Unchanged.
2. The dry run runs the guard and the delete verdicts and prints "would refuse" and "would leave behind".
**Decision**: Option 1 in this change.
**Rationale**: Option 2 is a visible change to dry-run output on its own and fits a follow-up. The gap is real: a dry run can list an apply or a prune that the real run refuses or skips. It is recorded as a question for the owner in the proposal gate report.

### Exit codes and messages

Codes: 0 success, 1 general, 2 validation, 3 connectivity, 4 permission denied, 5 not found. Cases not listed keep their code and their text; that includes every case cli#332, cli#334, cli#338, cli#341, cli#342 and cli#345 settled.

| # | Case | Before | After | Reason |
| --- | --- | --- | --- | --- |
| 1 | First apply; a rendered object exists and OPM does not manage it | exit 1; "already exists and is not managed by OPM ... remove or rename it" | exit 1; the library's `foreign-object` message, which names the adopt annotation | Same code. The remedy changes because an override now exists (0012:D8:R3). |
| 2 | First apply; a rendered object exists, OPM-managed, UUID label of another instance | exit 0; applied over, counted in the cli#333 warning | exit 1; `other-instance` | 0012:D8:R1. The old pass is the takeover the audit confirmed. |
| 3 | Later apply; an object new to the inventory exists and OPM does not manage it, or it carries another instance's UUID | exit 0; applied over | exit 1; `foreign-object` or `other-instance` | 0012:D8:R1: the guard runs on every apply. |
| 4 | Later apply; a rendered object is terminating | exit 0; the patch is accepted and the garbage collector then removes the object | exit 1; `terminating`, before any write | 0012:D8:R5. |
| 5 | Later apply; the read of a rendered object fails with an error other than NotFound | no read was made; the apply went on | exit 4, 3 or 1 by the read error, before any write | The rule of cli#332 for the first install now holds on every apply, because the guard reads on every apply. |
| 6 | Any apply; an existing object carries the adopt annotation with this instance's UUID | exit 1 on a first apply when OPM does not manage it | exit 0; applied and recorded | 0012:D8:R2. |
| 7 | Any apply; an object's adopt annotation names another instance (in the inventory, or outside it when OPM manages it) | exit 0; applied over | exit 0; not applied, one warning, not recorded | 0012:D8:R8. No code change. |
| 8 | Prune; the live stale object is not OPM-managed, carries another instance's UUID, or is adopted elsewhere | exit 0; deleted | exit 0; `left behind`, entry dropped | Audit finding B1; 0012:D4:R1. No code change. |
| 9 | Prune; the live read fails with an error other than NotFound | no read was made | exit 1 after the record write, entry kept; a discovery failure keeps its code (4 or 3) | The failed-prune rule of cli#332 and cli#342, applied to the new read. |
| 10 | Prune or delete; the DELETE answers Conflict on the UID precondition | not possible | prune: as row 9; `instance delete` and `operator uninstall`: a per-resource error, record kept, the command's existing failure code | The library's rule: a failed precondition is never reported as deleted. |
| 11 | `instance delete`, `operator uninstall`; the live object's adopt annotation names another instance | deleted | `left behind`; exit code unchanged | 0012:D8:R8. |
| 12 | `instance delete`, `operator uninstall`; left-behind reasons | "no longer managed by OPM", "owned by another instance" | the library's messages | One wording for both frontends. No code change. |
| 13 | `operator install` with a record; a rendered object new to the inventory exists and is foreign | exit 0; applied over | exit 2; guard refusal, nothing written | 0012:D8:R1. Exit 2 is the code install's guard has today. |
| 14 | `operator install`; a proven earlier Deployment or binding gets a skip verdict (in practice: its adopt annotation names another instance) | deleted | exit 2; migration refusal, nothing written | 0012:D8:R8: no instance deletes an object annotated for another. |

Example, row 3 (`opm instance apply`, UUID shortened):

```text
ERRO ConfigMap/default/settings exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c...e2
ERRO apply refused: 1 object(s) belong to someone else
apply stopped before any change
```

Example, row 7:

```text
WARN ConfigMap/default/settings was adopted by module instance 9a40...17; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=6f1c...e2
```

Example, row 8:

```text
WARN ConfigMap/default/old  left behind  reason="ConfigMap/default/old is not managed by OPM; left in place"
```

The first line of each ownership message is the library's text and MUST NOT be reworded by the cli. The lines around it are the cli's.

### Security

- Assets: objects in the cluster that this instance did not create, and their data.
- Trust boundary: the cluster API. Live labels and annotations are input that any principal with patch rights on the object can set. This is by design: the adopt annotation is an act of someone who may already change the object (0012:D8).
- Threats and mitigations: taking over a foreign object on apply (the guard on every apply); deleting an object another party recreated under a recorded name (the live read, the verdict and the UID precondition); deciding on a read that failed (every check fails closed). Baseline: the contract of 0012:D8 and the library's verdict tests.
- The cli decides nothing about ownership itself after this change. Its own policy is limited to read errors, exit codes and output.
- Residual risk, owner: the cli maintainers. A principal who can patch an object can set the adopt annotation and so hand it to an instance, or take it out of one. The same principal can already edit or delete the object.
- Refusal messages carry object names and instance UUIDs. Neither is a secret.

## Risks / Trade-offs

- [An apply that passed now refuses, for example two instances that have shared an object] -> The message names the annotation that resolves it; a migration note and a docs page say what to do; the refusal comes before any write.
- [The window between the guard's read and the patch, and between the delete read and the DELETE] -> The UID precondition closes the recreate case on delete. A label or annotation change inside the window is not closed, on either path; the server-side apply has no matching precondition. The operator has the same window. Accepted, and stated in the docs page.
- [One more GET per rendered object on every apply, one per stale object on prune] -> The first apply and `instance delete` already pay it. `applyOne` reads the object once more for its status line; reusing the guard's read there is allowed and not required.
- [An instance with no UUID in its render or its record] -> The library then compares no identity inside the inventory and refuses every non-empty UUID outside it. The cli passes what it has and adds no rule.
- [`operator install` over a record now reads and judges every object] -> The reads already happen in the terminating wait; the guard adds one pass.
- [The next change replaces the delete loops with `opm/k8s/lifecycle`] -> `JudgedDelete` is the single seam: that change removes it and its two call sites. Outcomes and messages do not change again, because the lifecycle package uses the same verdict and the same precondition.

## Migration Plan

No data migration and no change to the record's shape. The release note says: the guard now runs on every apply; how to adopt an object (`kubectl annotate <kind> <name> opmodel.dev/adopt=<instance uuid>`, the UUID being the value of the instance's `module-instance.opmodel.dev/uuid` label, printed in the refusal); prune and delete leave behind what they do not own. Rollback is a revert of the change; no stored state depends on it.

## Open Questions

- Whether the dry run should preview guard refusals and left-behind objects (see "The dry run"). A later change can add it without changing this one.
- Whether the `--rbac` objects of `opm operator install` come under the guard. They carry no OPM label and belong to no inventory, so every honest answer needs a labelling or recording decision first.
