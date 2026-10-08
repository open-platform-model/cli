## Why

The cli decides who owns a Kubernetes object in three places, and the three disagree. `opm instance delete` reads each object again and checks the managed-by label and the instance UUID. Prune deletes a stale entry by its recorded name with no live read, so it can delete an object that another instance or a user created under that name since. The apply guard runs only on an instance's first apply, passes any OPM-managed object whichever instance it belongs to, and is skipped once a record exists, so a later apply takes over a foreign object that a new release of the module happens to name.

The library now holds one rule for this, `opm/k8s/ownership`, shipped in library `v1.0.0-beta.7`, which the cli already pins. Enhancement 0012 decided that both frontends ask it on every apply, prune and delete path (0012:D4:R2, 0012:D8). The cli imports neither it nor `opm/k8s/lifecycle` today.

## What Changes

Every cli path that applies or deletes an object asks the library for a verdict and acts on the answer. The cli keeps its own reads, its own server-side apply and its own exit codes.

| Path | Verdict | What the cli does with the answer |
| --- | --- | --- |
| `opm instance apply`, `opm module apply`: every rendered object, on every apply | `ownership.CanApply` | Allowed: apply. `terminating`, `foreign-object`, `other-instance`: refuse the whole apply before any change. `adopted-elsewhere`: do not apply the object, warn, drop it from the inventory, go on. |
| The first-install existence check and its warning (cli#333) | the same `CanApply` pass | The check becomes that pass. The warning stays, and counts the existing OPM-managed objects the verdict allowed. |
| Prune of stale entries | `ownership.CanDelete` on a live read | Proceed: delete with the verdict's UID precondition. `already-absent`: done. `not-opm-managed`, `owner-mismatch`, `adopted-elsewhere`: leave behind, report, drop from the inventory. |
| `opm instance delete` | `ownership.CanDelete` on its existing re-read | The same outcomes; the delete carries the UID precondition. |
| `opm operator install`: CRD step and instance apply | `ownership.CanApply` with `Admit` for the objects the migration proved | The guard runs on every install, with or without a record. Refusals keep exit code 2. |
| `opm operator install`: delete of the earlier Deployment and the superseded bindings | `ownership.CanDelete` with `Admit`, in the check phase | Proceed: delete with the UID precondition. Any skip: refuse the install before any change. |
| `opm operator uninstall` | `ownership.CanDelete`, through the code `instance delete` uses | The same outcomes as `instance delete`. |

- **BREAKING** The apply guard runs on every apply, not only the first (0012:D8:R1). An apply that used to take over an existing object, because a record existed or because the object carried another instance's OPM labels, now refuses and exits non-zero. A terminating object refuses every apply (0012:D8:R5).
- **BREAKING** The refusal message is the library's. It no longer says "remove or rename it"; it names the adopt annotation and the UUID to set.
- **Adopt rule, annotation only (0012:D8:R8).** An object is adopted only when its live `opmodel.dev/adopt` annotation holds the instance's UUID. The user sets it, for example with `kubectl annotate`. The cli never sets it and offers no flag for it (0012:D8:R3, 0012:D8:R6); it honours it through the verdict and prints the exact annotation in every ownership refusal.
- **Hand-over.** An object whose adopt annotation names another instance is not applied, is dropped from the inventory the apply records, and is never deleted, by prune, by `instance delete` or by `operator uninstall`.
- **Prune reads before it deletes.** Prune leaves behind an object that OPM does not manage, that carries another instance's UUID or that is adopted elsewhere (audit finding B1).
- **Delete preconditions.** Every delete the cli sends after a verdict carries the UID of the object that was judged, so an object recreated under the same name since the read is not deleted. The library asks for the UID only; the cli sends no resourceVersion precondition.
- **Exit codes.** `design.md` holds the table of every case whose code or message changes, against the codes cli#332 to cli#342 fixed. No code changes for a case those changes settled.
- **Built on, not replaced.** The PersistentVolumeClaim rule of cli#345 (kept unless `--delete-data`) runs before the delete verdict, unchanged. Every read and delete goes through the discovery resolver of cli#342, unchanged. The fail-safe rules of cli#332, cli#334 and cli#338 stay.
- No new command and no new flag. The dry run is unchanged: it refuses nothing.

Not in this change: the deletion protocol (order, finalizer holds, propagation, `opm/k8s/lifecycle`), which is the next change; flags; publish; the operator repo; the `--rbac` objects of `opm operator install`, which belong to no instance.

SemVer: MAJOR after GA (an apply that succeeded now refuses). Before GA it ships as the next beta, with `!` in the PR title and a migration note in the PR body.

Size: four `tasks.md` sections, sized for one agent in about 90 turns. If the proposal gate wants two changes, the cut is after section 2: the delete side (sections 1 and 2) first, then the apply side (sections 3 and 4). Each half leaves `main` releasable.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `apply-pruning`: the first-install existence check becomes an ownership guard on every apply; an object adopted by another instance is let go; prune asks the delete verdict on a live read and sends a UID precondition.
- `deploy`: `opm instance delete` takes its per-object decision from the shared delete verdict, leaves an object adopted elsewhere behind, and sends a UID precondition.
- `operator-lifecycle`: `opm operator install` runs the apply guard on every install, with or without a record.
- `operator-migration`: the migration's deletes pass the shared delete verdict in the check phase.

## Impact

- Code: `internal/inventory/` (the guard, prune, the admit set), `internal/kubernetes/` (delete, the protected-kind test), `internal/workflow/apply/` (apply flow, record entries, messages), `internal/operator/` (plan, migration, install).
- Imports: the cli starts to import `github.com/open-platform-model/library/opm/k8s/ownership`. It already imports `opm/k8s/inventory`, `opm/k8s/labels`, `opm/k8s/object` and `opm/k8s/health`. It does not import `opm/k8s/lifecycle` in this change. No `go.mod` change: the package is in the pinned `v1.0.0-beta.7`.
- Stability: whether the `opm/k8s` tier is in the library's v1 promise is still open (owner decision O11 of the swarm design, which recommends marking the tier experimental). The cli takes the dependency either way; a break in the tier reaches the cli as a library bump.
- Users: a migration note in the PR body and a docs page on adopting an existing object. Scripts that match the old refusal text need the new one.
- Enhancement: implements part of 0012 (`enhancement.yaml`); no decision is claimed until the operator side has merged.
