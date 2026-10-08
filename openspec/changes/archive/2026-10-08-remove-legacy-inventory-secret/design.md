## Context

`Execute` (`internal/workflow/apply/apply.go`) loads the previous inventory with `LoadPreviousInventory`: the `ModuleInstance` record, and when there is none, a legacy Secret through `inventory.FindLegacySecretInventory` (a GET by name, then a LIST by label). The legacy inventory then feeds the stale set, the revision counter and the "has a previous inventory" switch that skips the first-install existence check. After the status write, `cleanupLegacySecret` issues a DELETE on the Secret name on every apply of every instance, whether a Secret exists or not.

Releases `v0.1.0` to `v1.0.0-alpha.1` wrote the Secret. The record backend landed in 57cd28d9, first released in `v1.0.0-alpha.2`. `v1.0.0-beta.10` is the newest release and still migrates.

## Goals / Non-Goals

**Goals:**

- No non-test code reads, lists, decodes, migrates or deletes a legacy inventory Secret.
- An apply makes no request on Secrets for inventory purposes.
- The outcome for an instance recorded only in a Secret is stated, tested and safe: nothing deleted, nothing pruned, nothing lost, and not silent.
- A migration note a user can act on.

**Non-Goals:**

- A stricter first-install existence check (see the decision below).
- The order of namespace creation, the record format, the operator manifest migration.

## Research & Decisions

### Delete, do not probe

**Context**: After the removal the cli cannot know that a legacy Secret exists.
**Options considered**:
1. Delete every Secret request. No permission needed; the cli cannot detect a legacy instance.
2. Keep one GET by name and refuse when it finds a Secret. Detects the case, but a Forbidden answer must either stop every first apply (the defect of the first attempt) or be ignored (fail open again). It also keeps a Secret read in the cli.
**Decision**: option 1.
**Rationale**: the task requires that no code reads the Secret and that apply needs no permission on Secrets. Option 2 keeps the exact trade-off the owner rejected.

### What happens to an instance recorded only in a Secret

**Context**: The expected outcome was a refusal by the first-install existence check. The check does not refuse. `PreApplyExistenceCheck` (`internal/inventory/stale.go`) refuses a resource that exists *without* an OPM managed-by label. Resources of a Secret-era instance carry `app.kubernetes.io/managed-by: opm-cli`: `pkg/core/labels.go` at 57cd28d9^ already stamps that value. So they pass.
**Observed result after the removal**: the apply runs as a first install. It server-side applies the rendered resources over the existing ones (same instance, same field manager), writes a record at revision 1 with the current entries, prunes nothing (no previous inventory, so the stale set is empty) and never touches the Secret. A resource that only the Secret recorded and that the render no longer produces stays in the cluster, labelled but in no record.
**Options considered**:
1. Leave it silent. Nothing is deleted, but the user is not told that the old inventory was not used.
2. Refuse a first install when a rendered resource already exists with an OPM managed-by label. It cannot tell a Secret-era instance from a first apply that failed part way: `Execute` writes no record when a resource fails to apply ("apply had errors: skipping pruning and inventory write"), so the retry finds its own resources without a record. A refusal there makes every failed first apply impossible to retry without deleting resources by hand. It also changes `opm operator install`.
3. Warn. The existence check already reads every rendered resource; it reports the ones that exist under OPM management, and `Execute` prints one warning before it applies.
**Decision**: option 3.
**Rationale**: it meets "not silent, nothing lost, nothing pruned" with no new request and no regression of the retry path. Option 2 is a change of the first-install contract for all users and belongs to the owner; it is listed as an open question.

### Shape of the check result

`inventory.PreApplyExistenceCheck` keeps its signature: `internal/operator/plan_install.go` calls it and is out of scope. Its body moves to

```go
// FirstInstallCheck is PreApplyExistenceCheck that also reports the entries
// that already exist under OPM management.
func FirstInstallCheck(ctx context.Context, client *kubernetes.Client, entries []k8sinventory.Entry, admit AdmitSet) (managed []k8sinventory.Entry, err error)
```

and `PreApplyExistenceCheck` calls it and drops the list. `RunPreApplyExistenceCheck` in the apply workflow returns the list; `Execute` warns when it is not empty. An admitted resource is not OPM-managed and is not in the list.

The warning is opt-in: `Options.WarnUnrecorded`, set by `opm instance apply` and `opm module apply`. `opm operator install` leaves it off, because `Install` applies the render's CRDs (`internal/operator/install.go`) before it calls `Execute`, so the check of a fresh install always finds four objects OPM manages.

A dry run skips the existence check and keeps doing so for its refusals. With `WarnUnrecorded` and no record it runs the same reads once through `previewAlreadyManaged`, which returns the managed list and turns an error into a debug line: a dry run refuses nothing it did not refuse before.

### Warning text

One WARN line on the log stream (standard error), exit code unchanged. The two runs differ in their last sentence.

Dry run, nothing written yet:

```text
WARN 2 of 3 rendered resource(s) already exist and are managed by OPM, but the instance has no ModuleInstance record. A real apply would update them in place and record them; it would prune nothing, so a resource an earlier apply created and this render no longer produces would stay in the cluster untracked. If opm v1.0.0-alpha.1 or older last applied this instance, its inventory is in a Secret this release does not read: apply the instance once with opm v1.0.0-beta.10 before you apply it with this release
```

Real run. It writes the record next. `v1.0.0-beta.10` then finds the record, skips the Secret and deletes it by name, so sending the user there would destroy the only list of the old inventory. The text names the Secret (built from the instance name and id, not read) and says to keep it:

```text
WARN 2 of 3 rendered resource(s) already exist and are managed by OPM, but the instance has no ModuleInstance record. This apply updates them in place and records them; it prunes nothing, so a resource an earlier apply created and this render no longer produces stays in the cluster untracked. If opm v1.0.0-alpha.1 or older last applied this instance, the Secret "opm.demo.<id>" in this namespace still lists what it owned: keep it, do not apply this instance with an older opm, and remove the leftovers as the opm docs page "Legacy inventory Secret" says
```

The warning also prints on the retry of a first apply that failed part way, and after a record was deleted by hand. Its first two sentences are true there too; the last one is conditional.

### Removal of the code

- `internal/inventory/legacy.go`, `legacy_test.go`: deleted.
- `LoadPreviousInventory` returns `(*inventory.Record, error)` and loses `dryRun` and the logger, which only worded the migration message.
- `WriteInstanceRecord`, `nextRevision`, `previousEntries` lose the `legacy` parameter; `cleanupLegacySecret` is deleted. `previousEntries` becomes a nil-safe read of the record.
- `tests/integration/migration/main.go` exists only for the path: deleted, with its `go run` line in `Taskfile.yml` and `.github/workflows/pr.yml`.

### Error handling and exit codes

No new error. The warning does not change the exit code. Every refusal of cli#332 keeps its code (4, 3, 1).

### The release the note names

`v1.0.0-beta.10` (`.release-please-manifest.json`, newest tag). If a release is cut from `main` before this change merges, that release migrates too and the note must name it.

## Risks / Trade-offs

- A user who skips the migration step keeps leftovers: the Secret, and resources the module no longer renders. Nothing running is deleted. The docs page gives the manual cleanup.
- A warning is not a stop: the apply goes on. See the open question.
- The warning is noise on the retry of a failed first apply. It is one line and accurate.

## Open Questions

- Should a first install refuse (not warn) when rendered resources already exist under OPM management without a record? That needs an answer for the failed-first-apply retry first (for example a record written before the resources). Owner decision; not in this change.
