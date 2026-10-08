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

// createNamespace creates the namespace namespaceToCreate found missing.
func createNamespace(ctx context.Context, c *kubernetes.Client, namespace string, l *log.Logger) error
```

New order on a real run:

```text
1  namespaceToCreate              GET namespace                                (exit by cause)
2  RunClusterGates
3  LoadPreviousInventory          a missing namespace answers NotFound: no record
4  ownership                      a record implies the namespace exists
5  GateStatusRBAC
6  GuardEmptyRender
7  RunPreApplyExistenceCheck      a missing namespace answers NotFound: nothing there
8  createNamespace                CREATE, only when step 1 found it missing    <- first write
9  kubernetes.Apply, prune, record write
```

The read stays first for two reasons. A failure of it (no permission to read namespaces) stops
the apply at the same place and with the same text as today. And a dry run prints
`namespace "<ns>" would be created` as its first line, as today, and hands the name to
`kubernetes.Apply` as a namespace the apply creates.

Both steps use the existing `kubernetes.Client.EnsureNamespace`: with `dryRun` true it is the
read (it reports a missing namespace and creates nothing), with `dryRun` false it is the create.
The create reads once more before it writes, which costs one GET and keeps its handling of a
namespace somebody else created in between (AlreadyExists is not an error).

**Alternatives considered**

1. Move the whole step, read and create, to after the checks. Smaller diff, but a dry run would
   print the "would be created" line after the first-install warning, and a failed namespace
   read would surface after four other checks instead of first.
2. Create first and delete the namespace on a refusal. Rejected: a delete is a second write
   that can fail and leave the namespace anyway, and it cannot tell "I created it" from "it
   appeared between my read and my create".
3. Add a `NamespaceExists` method to the client. Not needed: the dry-run form of
   `EnsureNamespace` is that read already.

### A missing namespace reads as "nothing there"

Three checks name the instance namespace before it exists:

| Check | Call | Answer in a missing namespace | Effect |
| --- | --- | --- | --- |
| Record read | GET `moduleinstances/<name>` in the namespace | NotFound | no record: first install, which is correct, since a record cannot exist without its namespace |
| Status permission | `SelfSubjectAccessReview` naming the namespace | allowed or denied from cluster-wide rules; no RoleBinding can exist in a missing namespace, and creating the namespace adds none | the same answer the check gives right after the create |
| Existence check | GET each rendered object | NotFound for every object in the namespace | passes; objects in other namespaces and cluster-scoped objects are checked as usual |

No code treats the missing namespace specially: `inventory.GetRecord` and
`inventory.FirstInstallCheck` already map NotFound to "absent". The API server answers a GET of
a namespaced object in a missing namespace with 404, reason NotFound; `apierrors.IsNotFound`
is true for it. This assumption is not checked against a live cluster in this change (the unit
tests use fake clients, which answer NotFound as well); the existing integration program
`tests/integration/module-apply` runs the path on a cluster.

The paths that need a record (the thin-editor path of an operator-owned instance and the
empty-render guard) cannot meet a missing namespace: step 1 then finds the namespace present and
there is nothing to create.

### An apply with nothing to apply still creates the namespace

With no rendered resource and no record, `Execute` returns early with "no resources to apply".
Today the namespace exists by then. To keep the successful result unchanged, the early return
creates the namespace first. Nothing can refuse after that point.

### Refusal text

Both refusals run before step 8 for every caller, so both can say it:

```text
cannot check whether ConfigMap/app in namespace "media" already exists: <read error>
apply stopped before any change. Check that you can read that resource, then run the command again
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

- [A window between the read and the create grows by the duration of the checks] → the create
  tolerates AlreadyExists; a namespace deleted in that window is created again.
- [A user who may create namespaces but may not read `moduleinstances` gets no namespace] → that
  is the intent: the apply could not have gone on.
- [Fake clients stand in for the 404 of a missing namespace] → stated above as unverified
  against a live cluster.

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
