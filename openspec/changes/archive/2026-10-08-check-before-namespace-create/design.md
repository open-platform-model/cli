## Context

See proposal.md for the motivation. `apply.Execute` (`internal/workflow/apply/apply.go`) is the
one workflow behind `opm instance apply` and `opm module apply`; `opm operator install` calls it
too, always with `CreateNS: false`, because the operator module renders its own Namespace.

Order today, on a real run:

```text
1  EnsureNamespaceIfRequested     GET namespace, CREATE when missing        <- write
2  RunClusterGates                CRD present, field floor, operator ceiling   (exit 2)
3  LoadPreviousInventory          GET ModuleInstance                           (exit 4, 3, 1)
4  ownership                      operator-owned: thin-editor path, return
5  GateStatusRBAC                 SelfSubjectAccessReview                      (exit 4)
6  GuardEmptyRender               needs a record with entries
7  RunPreApplyExistenceCheck      GET every rendered object, no record only    (exit 4, 3, 1)
8  kubernetes.Apply, prune, record write
```

Steps 2, 3, 5 and 7 can refuse on a cluster where step 1 has just created the namespace.

## Goals / Non-Goals

**Goals:**

- No write before the last check that can refuse.
- The same final cluster state on every successful apply, and the same dry-run output.
- A stated rule for checks that read in a namespace that does not exist yet.

**Non-Goals:**

- No new check, no changed exit code, no changed flag.
- No change to `opm operator install` behaviour; it gets tests and a comment only.
- No rollback of a namespace when the apply fails later, during the writes. That is a failed
  apply, not a refusal, and the next run needs the namespace.

## Decisions

### Split the namespace step into a read and a create

`EnsureNamespaceIfRequested` becomes two unexported steps in the same file:

```go
// namespaceToCreate reports whether --create-namespace has a namespace to
// create: the flag is set and the instance namespace is missing. Read only.
func namespaceToCreate(ctx context.Context, c *kubernetes.Client, namespace string, createNS, dryRun bool, l *log.Logger) (bool, error)

// ensureNamespace creates the namespace when create is set, which the caller
// derives from namespaceToCreate on a real run.
func ensureNamespace(ctx context.Context, c *kubernetes.Client, namespace string, create bool, l *log.Logger) error
```

New order on a real run:

```text
1  namespaceToCreate              GET namespace                                (exit by cause)
2  RunClusterGates
3  LoadPreviousInventory          skipped when step 1 found the namespace missing
4  ownership                      a record implies the namespace exists
5  GateStatusRBAC
6  GuardEmptyRender
7  RunPreApplyExistenceCheck      objects in the missing namespace are skipped
8  ensureNamespace                CREATE, only when step 1 found it missing    <- first write
                                  (exit 1 when the namespace exists by now)
9  kubernetes.Apply, prune, record write
```

The read stays first for two reasons. A failure of it (no permission to read namespaces) stops
the apply at the same place and with the same text as today. And a dry run prints
`namespace "<ns>" would be created` as its first line, as today, and hands the name to
`kubernetes.Apply` as a namespace the apply creates.

Both steps use the existing `kubernetes.Client.EnsureNamespace`: with `dryRun` true it is the
read (it reports a missing namespace and creates nothing), with `dryRun` false it is the create.
The create reads once more before it writes, which costs one GET, and reports whether it
created the namespace; "not created" is the refusal described below.

**Alternatives considered**

1. Move the whole step, read and create, to after the checks. Smaller diff, but a dry run would
   print the "would be created" line after the first-install warning, and a failed namespace
   read would surface after four other checks instead of first.
2. Create first and delete the namespace on a refusal. Rejected: a delete is a second write
   that can fail and leave the namespace anyway, and it cannot tell "I created it" from "it
   appeared between my read and my create".
3. Add a `NamespaceExists` method to the client. Not needed: the dry-run form of
   `EnsureNamespace` is that read already.

### A missing namespace holds nothing, so nothing in it is read

When step 1 found the namespace missing, the apply sends no read into it:

| Check | In a missing namespace | Effect |
| --- | --- | --- |
| Record read | skipped | no record: a first install, which is correct, since a record cannot exist without its namespace |
| Existence check | rendered objects in that namespace are skipped | cluster-scoped objects and other namespaces are checked as usual |
| Status permission | unchanged: a `SelfSubjectAccessReview` naming the namespace | allowed or denied from the rules that exist; see the limit below |

The first draft of this change sent the reads and relied on a NotFound answer. Review showed
that this holds only for a caller with cluster-wide read: the API server authorizes a request
before it looks the object up, so a caller whose rights come from a RoleBinding in the
namespace gets Forbidden for a read in a namespace that does not exist yet. Skipping the reads
does not depend on which of the two the server answers.

Because nothing inside the namespace was checked, a namespace that exists at the create
(somebody else made it in between, with or without a record in it) stops the apply with exit 1
and "run the command again". The second run finds the namespace and reads it.

Limit, not removed by this change: the status-permission check still runs before the create. A
caller whose right to patch `moduleinstances/status` comes only from a RoleBinding that some
automation adds after the namespace appears is refused (exit 4) and the namespace is never
created by the apply. Before this change the first run created the namespace and was refused
too, and a later run passed. Whether that check may move after the create is a question for the
owner.

A module that renders its own instance Namespace, applied with `--create-namespace` into a
cluster without it, was refused before this change (the namespace the flag had just created was
found as an untracked Namespace). It now passes: the check runs while the namespace is missing.

The paths that need a record (the thin-editor path of an operator-owned instance and the
empty-render guard) cannot meet a missing namespace.

### An apply with nothing to apply still creates the namespace

With no rendered resource and no record, `Execute` returns early with "no resources to apply".
Today the namespace exists by then. To keep the successful result unchanged, the early return
creates the namespace first. Nothing can refuse after that point.

### Refusal text

Both refusals run before step 8. `opm instance apply` and `opm module apply` write nothing
before `Execute`, so for them both texts say that the apply stopped before any change.
`opm operator install` applies its CRDs and its migration before it calls `Execute`; it sets
the new `Options.AfterCallerWrites`, and the sentence is left out. The sentence is added by
the workflow, not by `internal/inventory`, and follows every refusal of the existence check.

```text
pre-apply existence check failed: cannot check whether ClusterRole/app in namespace "" already exists: <read error>
Check that you can read that resource, then run the command again
apply stopped before any change
```

```text
cannot read the ModuleInstance record "demo" in namespace "media": <read error>
apply stopped before any change: without the record it cannot tell a first install from an existing instance.
Check that you can read moduleinstances.opmodel.dev in that namespace, then run the command again
```

Exit codes are unchanged: 4 for Forbidden or Unauthorized, 3 for a server timeout or service
unavailable, 1 otherwise; a failed cluster gate exits 2. The success output is unchanged except
that `namespace "<ns>" created` now follows the checks.

### `opm operator install`: what the install-level test found

Command: `opm operator install [flags]`; no flag changes. `PlanInstall` reads each object it
applies three times before any write: in the terminating wait, in the migration proof and in the
apply guard (the shared existence check). `installError` assigns the exit code by error type:

| Read that fails | Error | Exit |
| --- | --- | --- |
| Terminating wait (first) | wrapped API error | by cause: 4 for Forbidden |
| Migration proof (second) | `MigrationReadError` | by cause: 4 for Forbidden |
| Apply guard (third) | `GuardError` | 2 |

So an object that is unreadable from the start is refused with exit 4 by the first read, and
exit 2 is reached only when the guard's read fails after the first two were answered. All three
refuse before any write. The tests pin all three; the numbering is not changed here (out of
scope) and is raised as a question for the owner.

## Risks / Trade-offs

- [A window between the read and the create grows by the duration of the checks] → a namespace
  that appears in it stops the apply (exit 1, run again). A namespace that was present at the
  read and is deleted in the window is not created; the apply then fails at its first
  namespaced object, as it would without the flag.
- [A user who may create namespaces but may not read `moduleinstances` gets no namespace] → that
  is the intent: the apply could not have gone on.
- [No run against a live cluster in this change] → the apply no longer depends on the answer
  to a read in a missing namespace; the access review for a missing namespace is not observed.

## Research & Decisions

### Which checks can refuse before the first write

**Context**: "every read-only check" needed a list.
**Explored**: `apply.Execute` top to bottom, and the callers in `internal/cmd/instance/apply.go`
and `internal/cmd/module/apply.go` (no cluster write before `Execute`).
**Options considered**:
1. Reorder only around the two refusals that were reported - leaves the gates and the
   permission check creating a namespace.
2. Put the create after the last refusing step - covers all of them with one move.
**Decision**: option 2.
**Rationale**: one rule, "first write after the last check", is easy to keep true when a check
is added.
