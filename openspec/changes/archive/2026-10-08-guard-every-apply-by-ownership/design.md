## Context

See `proposal.md` for the motivation. This change builds on `adopt-kubernetes-ownership-package` (the delete half), merged as cli#347 and archived: it added the import of `opm/k8s/ownership`, `kubernetes.JudgedDelete`, the prune that judges with the recorded identity, and the call-site test `internal/kubernetes/deletesites_test.go`. The state of the apply side at cli `main` 4bf7f04b:

- `internal/inventory/stale.go`: `FirstInstallCheck` reads every rendered entry and refuses a terminating, an untracked or an unreadable object. It passes every OPM-managed object, whichever instance its UUID label names. `RunPreApplyExistenceCheck` in `internal/workflow/apply/apply.go` skips it when a record exists and on a dry run.
- `internal/kubernetes/apply.go`: `applyOne` reads the object, ignores a read error, and patches with `Force: true`.
- `internal/operator/plan_install.go`: the apply guard runs only when the operator instance has no record. `internal/operator/install.go` then writes in this order: the CRD step (`ApplyOne`), `MoveOwnership` (a managedFields patch), `DeleteSuperseded`, the instance apply, the `--rbac` objects (`ApplyOne`).

Constraints:

- The library verdict is pure; the cli reads the live object and hands it in. The contract is 0012:D4:R2 and 0012:D8:R1 to R8, with the "annotation only" rule of 0012:D8:R8.
- The fail-safe rules stay: an unreadable record or object stops the apply (cli#332), a refusal before the first write changes nothing (cli#334), reads resolve kinds by discovery (cli#342), errors are told apart by type (cli#346).
- No command syntax changes. No flag is added, removed or renamed.

Reversibility: costly two-way. What users see (which applies are refused, the adopt annotation) is the contract of 0012:D8, already accepted by the owner; this change does not decide it.

## Goals / Non-Goals

**Goals:**

- One function, the library's, decides apply ownership for every object the cli applies on behalf of an instance, on every apply. A test keeps it so.
- A refused apply has written nothing.
- An object adopted by another instance leaves the inventory and stays in the cluster.
- `opm operator install` never writes to an object another instance owns or adopts.

**Non-Goals:**

- The delete side and the deletion protocol.
- A command or flag that sets the adopt annotation (0012:D8:R6 excludes it).
- A change to what a dry run refuses.
- Ownership of the `--rbac` objects of `opm operator install`. They carry no OPM label and are in no inventory. They stay outside the guard, as the one named exception, until a follow-up decides how they are labelled or recorded.
- Create-only writes: the namespace of `--create-namespace` and the Platform seed. A create cannot take an existing object over.

## Decisions

### Paths and verdicts

| Path | Input to the verdict | Answer | cli action |
| --- | --- | --- | --- |
| Apply guard (`instance apply`, `module apply`) | `CanApply{Object, Live, InInventory, InstanceUUID, Admit}` for every rendered object. `InstanceUUID` is the render's. `InInventory` is true when the previous record lists the same object (group, kind, namespace, name). | allowed | Apply. |
| | | `terminating`, `foreign-object`, `other-instance` | Refuse the apply before its first write, exit 1. Report every refused object, one line each, with the library's message. |
| | | `adopted-elsewhere` | Do not apply the object. One warning line with the library's message. Leave it out of the record. The apply goes on and this alone does not change the exit code (0012:D8:R8). |
| | | (no verdict) read error other than NotFound and other than "kind not served" | Refuse the apply before its first write, with the exit code of the read error, as cli#332 set for the first install. |
| `operator install`, check phase | The same guard over every object the plan applies, on every install, with or without a record; `Admit` is true for the objects of the migration's admit set | `terminating`, `foreign-object`, `other-instance`, `adopted-elsewhere`, or a failed read | `GuardError`, exit 2, nothing written. `adopted-elsewhere` refuses here, unlike in any other apply. |
| `operator install`, CRD step and `MoveOwnership` | none of their own | | They run after the check phase passed, on objects the guard allowed. Because the guard refuses on `adopted-elsewhere`, the plan holds no object that was let go, so neither step can write to one. |
| `operator install`, instance apply | The guard again, inside the apply workflow, with the option that makes `adopted-elsewhere` a refusal | a refusal (the object changed since the check phase) | The instance apply fails with its own code and does not say that nothing was changed, as today. |
| `operator install`, `--rbac` objects | none | | Applied as today. Outside the guard; see Non-Goals. |

### Shape in the code

```go
// internal/inventory: the apply guard, replacing FirstInstallCheck.
type GuardInput struct {
    Entries      []k8sinventory.Entry  // every rendered object, in render order
    Previous     []k8sinventory.Entry  // the record's inventory; nil on a first apply
    InstanceUUID string                // the render's identity
    Admit        AdmitSet              // keyed by ownership.Object; nil for every caller but operator install
    RefuseLetGo  bool                  // operator install: adopted-elsewhere is a refusal
}

type GuardResult struct {
    Managed []k8sinventory.Entry // allowed, exists, OPM-managed, not in Previous: the cli#333 warning counts these
    LetGo   []LetGo              // refused as adopted-elsewhere: entry and library message
}

// Guard returns a *GuardRefusalError holding every refused object with its
// ownership.ApplyRefusal and message, or the read error of the first object
// it could not read. It writes nothing.
func Guard(ctx context.Context, client *kubernetes.Client, in GuardInput) (GuardResult, error)
```

`AdmitSet` becomes `map[ownership.Object]struct{}`; `K8sIdentity` and `IdentityOf` go, as their comment in `internal/inventory/admit.go` announces. `FirstInstallCheck` and `PreApplyExistenceCheck` go.

Apply flow, real run:

```text
cluster gates -> read record -> ownership mode -> status RBAC -> empty-render guard
  -> Guard(all rendered entries, previous inventory, render identity, admit)   refuses here, before any write
  -> cli#333 warning from GuardResult.Managed (first apply only)
  -> create namespace -> Apply(rendered minus LetGo)
  -> stale = StaleSet(previous, rendered)
  -> prune with the recorded identity (the delete half)
  -> write record: (rendered minus LetGo) + failed prunes + kept claims
```

With `--create-namespace` and a missing namespace the guard skips the objects in that namespace, as the existence check does today. A dry run does not refuse; its first-install look that feeds the cli#333 warning uses the guard's reads and ignores its refusals.

A test in `internal/kubernetes` lists the allowed call sites of `ApplyOne` and of `.Patch(` on a resource client (the apply workflow, the CRD step, `MoveOwnership`, the `--rbac` loop, the record writes), so a new apply cannot bypass the guard unseen.

### Research & Decisions

#### Where the apply verdict runs

**Context**: 0012:D8:R1 needs the verdict on every apply. The cli must also keep "a refusal changes nothing" (cli#334).
**Options considered**:
1. Status quo: first install only. Leaves the takeover on later applies; contradicts 0012:D8:R1.
2. One pass before the first write, over every rendered object. All-or-nothing refusal; one extra GET per object on later applies.
3. Judge inside `applyOne` on its own read. No extra GET and the smallest window between read and write, but a refusal arrives after earlier objects were applied.
**Decision**: Option 2.
**Rationale**: It keeps the fail-safe rule that a refused apply has written nothing, and it is the shape the existence check already has.

#### What a hand-over does to the inventory and the prune

**Context**: 0012:D8:R8 says the instance drops the object and never deletes it. 0012:D7:R1 describes this as "in the stale set, and the deletion plan skips it"; its end state binds, not its mechanism.
**Options considered**:
1. Remove the object from the rendered set before the stale set is computed, so prune sees it as stale and the delete verdict skips it.
2. Keep it out of the apply and out of the record, and compute the stale set from the full render, so prune never sees it.
**Decision**: Option 2.
**Rationale**: The end state is the one 0012:D7:R1 binds: the object is not recorded and not deleted, and the stale set is exactly what the instance recorded and no longer renders. There is no path on which a delete of that object is attempted. The one visible difference from option 1 is the line printed: the apply's warning in place of a `left behind` line. If the annotation goes away later, the next apply judges the object as one outside the inventory.

#### `opm operator install` refuses where another apply lets go

**Context**: For every other instance an `adopted-elsewhere` object is skipped and the apply goes on. Install then waits for the rollout of a controller Deployment it did not apply, and can report success on an operator it did not upgrade. It also patches managedFields (`MoveOwnership`) before its instance apply.
**Options considered**:
1. Let go and go on, as any apply does.
2. Refuse only for a CRD, the Namespace and the controller Deployment.
3. Refuse, in the check phase, for every object the plan applies.
**Decision**: Option 3, exit 2, naming the object and the instance that owns or adopts it (owner decision of 2026-10-08: when `opm operator install` meets an object it needs that another instance owns, it refuses with exit 2 before changing anything).
**Rationale**: The operator module renders nothing optional: its roles, bindings and service account are needed as much as the three kinds the decision names, so the rule is stated for every rendered object. Refusing in the check phase also settles the managedFields patch: no write of the install reaches an object that was let go, because no object is let go. This departs from the letter of 0012:D8:R8 ("does not stop the apply"), for this one command; the enhancement needs a revision note.

#### The first install that meets its own objects

**Context**: With no record and a changed identity, every object is refused as `other-instance`, and the library's remedy says "remove it from module instance <old UUID>", which is the same instance. The cli#333 warning, which carries the legacy Secret hint, is printed after the guard and so is not reached.
**Decision**: On a first apply (no record), when any refusal is `other-instance`, the cli adds one line of its own after the library's messages: the objects may be the instance's own under an earlier identity; annotate each as shown; nothing has to be removed first. The line also names the legacy Secret case.
**Rationale**: The library cannot know that the other identity is this instance's past. The cli does not reword the library's line; it adds its own.

#### The dry run

**Decision**: Unchanged. A dry run refuses nothing and prints no refusal. The spec says "SHALL refuse nothing because of the guard" and does not forbid the reads, which the first-install warning needs.
**Rationale**: Every refusal the dry run hides stops the real run before its first write, so the real run is a safe probe. A follow-up change prints "would refuse" and "would let go" lines; the migration note states the gap until then.

### Exit codes and messages

Codes: 0 success, 1 general, 2 validation, 3 connectivity, 4 permission denied, 5 not found. Cases not listed keep their code and their text; that includes every case cli#332, cli#334, cli#338, cli#341, cli#342, cli#345 and cli#346 settled.

| # | Case | Before | After | Reason |
| --- | --- | --- | --- | --- |
| 1 | First apply; a rendered object exists and OPM does not manage it | exit 1; "already exists and is not managed by OPM ... remove or rename it" | exit 1; the library's `foreign-object` message, which names the adopt annotation | Same code. The remedy changes because an override now exists (0012:D8:R3). |
| 2 | First apply; a rendered object exists, OPM-managed, UUID label of another instance | exit 0; applied over, counted in the cli#333 warning | exit 1; `other-instance`, plus the cli's line on an earlier identity | 0012:D8:R1. |
| 3 | Later apply; an object new to the inventory exists and OPM does not manage it, or it carries another instance's UUID | exit 0; applied over | exit 1; `foreign-object` or `other-instance` | 0012:D8:R1: the guard runs on every apply. |
| 4 | Later apply; a rendered object is terminating | exit 0; the patch is accepted | exit 1; `terminating`, before any write | 0012:D8:R5. |
| 5 | Later apply; the read of a rendered object fails with an error other than NotFound | `applyOne` read the object and ignored the error; the apply went on and the patch decided | exit 4, 3 or 1 by the read error, before any write | The rule of cli#332 for the first install now holds on every apply. |
| 6 | Any apply; an existing object carries the adopt annotation with this instance's UUID | exit 1 on a first apply when OPM does not manage it | exit 0; applied and recorded | 0012:D8:R2. |
| 7 | Any apply but `operator install`; an object's adopt annotation names another instance (in the inventory, or outside it when OPM manages it) | exit 0; applied over | exit 0; not applied, one warning, not recorded | 0012:D8:R8. No code change. |
| 8 | `operator install`, no record; a rendered object exists, OPM-managed, UUID label of another instance | exit 0; applied over. For an object on the proof list of an earlier operator manifest (the CRDs, the Namespace, the controller Deployment and others) the migration proof already refused it, exit 2 | exit 2; guard refusal, nothing written; the proof-list objects are still refused by the migration proof first, with its message | 0012:D8:R1; exit 2 is the code install's guard has today. |
| 9 | `operator install` with a record; a rendered object new to the inventory exists and is foreign or another instance's | exit 0; no guard ran, applied over | exit 2; guard refusal, nothing written | 0012:D8:R1. |
| 10 | `operator install` with a record; the guard's read of a rendered object fails | no guard ran | exit 2, nothing written | The rule the guard has today without a record, now with one. |
| 11 | `operator install`; a rendered object's adopt annotation names another instance | exit 0; applied over | exit 2; guard refusal naming the object and the instance, nothing written | Owner decision of 2026-10-08. |

Example, row 3 (`opm instance apply`, UUID shortened):

```text
apply refused: 1 object(s) cannot be applied by this instance:
  ConfigMap/default/settings exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c...e2
apply stopped before any change
```

Example, row 7:

```text
WARN ConfigMap/default/settings was adopted by module instance 9a40...17; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=6f1c...e2
```

Each object line of a refusal, and the warning of row 7, is the library's text and MUST NOT be reworded by the cli. The lines around them are the cli's. The refusal is one error, printed once by the command, as the existence check's was. A failed read keeps the text cli#332 gave it ("pre-apply existence check failed: cannot check whether ...").

### Security

- Assets: objects in the cluster that this instance did not create, and their data.
- Trust boundary: the cluster API. Live labels and annotations are input that any principal with patch rights on the object can set.
- Threats and mitigations: taking over a foreign object on apply (the guard on every apply, before the first write); deciding on a read that failed (fails closed). Baseline: the contract of 0012:D8 and the library's verdict tests.
- Residual risk, owner: the cli maintainers. A principal with patch rights on an object can set the adopt annotation. With this instance's UUID, the object is taken over by the instance on its next apply. With another UUID, the instance stops applying the object, so it no longer receives the module's updates, and `opm operator install` is blocked while one of its objects carries such an annotation. Patch rights do not include delete rights, so for some principals this is more than they could do before.
- Detection: every let-go prints a warning on every apply that names the object and the instance in the annotation; every refusal names the object and the owner.
- Refusal messages carry object names and instance UUIDs. Neither is a secret.

## Risks / Trade-offs

Refusals that hit a user's own objects. Each follows 0012:D8:R1 or R5; each is in the migration note and the docs page with its remedy.

- [Leftover CRDs and Namespaces] `instance delete` and `operator uninstall` never delete them and leave their labels. A reinstall under another instance name, namespace or module path meets objects of an instance that no longer exists and is refused as `other-instance` (exit 1; 2 in `operator install`). Today it passes with a warning. -> Remedy: annotate each leftover with `opmodel.dev/adopt=<new UUID>`, as the refusal prints, or delete the leftovers when nothing else uses them.
- [No record and a changed identity] The record was deleted, or the instance was last recorded in a legacy Secret, and its module path changed. Every object is refused as `other-instance`. -> Remedy: annotate each object; nothing has to be removed. The cli's extra line says so.
- [An inventoried object stuck terminating] For example a PersistentVolumeClaim held by pvc-protection while the instance's own pods mount it. Every later apply refuses, so opm cannot change the workload that would release it. Today the apply goes through. -> Remedy outside opm: release the object with `kubectl` (remove the pod or the finalizer's cause), wait until it is gone, apply again.

Other risks:

- [Two instances that have shared an object] -> The refusal names the annotation that resolves it; the refusal comes before any write.
- [The window between the guard's read and the patch] -> A label or annotation change inside the window is not closed; the server-side apply has no matching precondition. The operator has the same window. Accepted.
- [One more GET per rendered object on every apply] -> The first apply already pays it. `applyOne` reads the object once more for its status line; reusing the guard's read there is allowed and not required.
- [An instance with no UUID in its render] -> The library then compares no identity inside the inventory and refuses every non-empty UUID outside it. The cli passes what it has and adds no rule.
- [This change lands without the delete half] -> Cannot happen: the delete half is on `main` (cli#347), and the delta builds on its spec text.

## Migration Plan

No data migration and no change to the record's shape. Rollback is a revert of the change; no stored state depends on it.

Migration note for the PR body and the release:

- The ownership guard now runs on every apply, not only the first. An apply refuses, before any change, when an object it renders exists and OPM does not manage it, belongs to another instance, or is being deleted.
- To adopt an existing object: `kubectl annotate <kind> <name> opmodel.dev/adopt=<instance UUID>`. The refusal prints the exact annotation. No flag overrides the guard.
- An object whose `opmodel.dev/adopt` annotation names another instance is no longer applied by this instance and leaves its inventory; it is not deleted.
- You may meet a refusal on your own objects in three cases: CRDs and Namespaces left by a deleted instance that you reinstall under another name, namespace or module path (annotate them, or delete them); an instance with no record whose module path changed (annotate each object; nothing has to be removed); an object of the instance that is stuck terminating (release it with `kubectl`, then apply again).
- `opm operator install` now checks every object on every install and refuses with exit 2, with nothing changed, when one is owned or adopted by another instance.
- `--dry-run` does not show these refusals: a dry run can succeed where the real run refuses. The real run refuses before it changes anything.

## Open Questions

- Follow-up change: the dry run prints "would refuse" and "would let go" lines.
- Follow-up issue: the `--rbac` objects of `opm operator install` (label them, record them, or keep them outside the guard for good).
