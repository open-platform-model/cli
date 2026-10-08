## Why

The cli decides whether it may delete a Kubernetes object in three places, and the three disagree. `opm instance delete` reads each object again and checks the managed-by label and the instance UUID. Prune deletes a stale entry by its recorded name with no live read, so it can delete an object that another instance or a user created under that name since (audit finding B1). The operator install's migration deletes on its own proof. None of them knows the adopt annotation, so an object a user handed to another instance is still deleted.

The library holds one rule for this, `opm/k8s/ownership`, in library `v1.0.0-beta.7`, which the cli already pins. Enhancement 0012 decided that both frontends ask it on every prune and delete path (0012:D4:R1, 0012:D8:R8). The cli does not import it today.

This change is the delete half of the cli's adoption. The apply half (the guard on every apply, the adopt annotation on apply, `opm operator install`'s guard) is the change `guard-every-apply-by-ownership`, which lands after this one.

## What Changes

Every cli path that deletes an object reads the live object, asks the library's delete verdict and acts on the answer. The cli keeps its own reads, its own delete calls, its own order and its own exit codes.

| Path | Verdict and identity | What the cli does with the answer |
| --- | --- | --- |
| Prune of stale entries (`opm instance apply`, `opm module apply`, the instance apply inside `opm operator install`) | `ownership.CanDelete` on a live read, with the identity stored in the instance's record | Proceed: delete with the verdict's UID precondition. `already-absent`: done. `not-opm-managed`, `owner-mismatch`, `adopted-elsewhere`: leave behind, report, drop from the inventory. |
| `opm instance delete` | `ownership.CanDelete` on its existing re-read, with the record's identity | The same outcomes; the delete carries the UID precondition. |
| `opm operator uninstall` | `ownership.CanDelete`, through the code `instance delete` uses | The same outcomes as `instance delete`. |
| `opm operator install`: delete of the earlier Deployment and the superseded bindings | `ownership.CanDelete` with `Admit`, in the check phase, on the object the migration proof read, with the operator instance's identity | Proceed: delete with the UID precondition. Any skip: refuse the install with exit 2 before any change, naming the object and its owner. |

- **BREAKING** Prune reads before it deletes. It leaves behind, and drops from the inventory, a stale object that OPM does not manage, that carries another instance's UUID or that is adopted by another instance. Until now it deleted every stale name.
- **BREAKING** `opm instance delete` and `opm operator uninstall` leave behind an object whose `opmodel.dev/adopt` annotation names another instance. The left-behind reasons are the library's wording.
- **Prune judges with the recorded identity.** The identity is the one stored in the instance's record, the identity that applied the stale objects, also when the render's identity differs. So the first apply after a module moved to a new path still prunes what the instance owned. The record takes the new identity in the write that follows the prune.
- **Delete preconditions.** Every delete the cli sends after a verdict carries the UID of the object that was judged, so an object recreated under the same name since the read is not deleted. The library asks for the UID only; the cli sends no resourceVersion precondition.
- **Exit codes.** `design.md` holds the table of every case whose code or message changes. No code changes for a case that cli#332 to cli#346 settled.
- **Built on, not replaced.** The PersistentVolumeClaim rule of cli#345 (kept unless `--delete-data`) runs before the verdict, unchanged. Every read and delete goes through the discovery resolver of cli#342. The fail-safe rules of cli#332, cli#334 and cli#338 stay. Errors are told apart by type, as cli#346 requires: the new code matches no message text.
- No new command and no new flag. The dry run is unchanged; a dry run can therefore preview a prune that the real run leaves behind. A follow-up prints "would leave behind" lines.

Why this half is safe to ship alone: it only removes deletes. Every case that changes ends with an object left in the cluster and reported, never with an object deleted that was kept before. Apply behaviour does not change. Until the apply half lands, the cli honours the adopt annotation when it deletes and not when it applies: an instance whose object was annotated for another instance still re-applies it. The release note says so.

Not in this change: the apply guard and everything else in `guard-every-apply-by-ownership`; the deletion protocol (order, finalizer holds, propagation, `opm/k8s/lifecycle`), which is a later change; flags; publish; the operator repo.

SemVer: MAJOR after GA (a prune or delete that removed an object now leaves it). Before GA it ships as the next beta, as `feat!` in the PR title, with the migration note in the PR body.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `apply-pruning`: prune asks the delete verdict on a live read, with the recorded identity, and sends a UID precondition; the apply flow's prune and record-write steps say so.
- `deploy`: `opm instance delete` takes its per-object decision from the shared delete verdict, leaves an object adopted elsewhere behind, and sends a UID precondition.
- `operator-lifecycle`: `opm operator uninstall` leaves behind what the shared delete verdict leaves behind.
- `operator-migration`: the migration's deletes pass the shared delete verdict in the check phase; a skip refuses the install.

## Impact

- Code: `internal/kubernetes/` (delete, the protected-kind test), `internal/inventory/` (prune), `internal/workflow/apply/` (prune identity, left-behind lines, record entries), `internal/operator/` (migration plan and deletes), one new integration script under `tests/integration/`.
- Imports: the cli starts to import `github.com/open-platform-model/library/opm/k8s/ownership`. It already imports `opm/k8s/inventory`, `opm/k8s/labels`, `opm/k8s/object` and `opm/k8s/health`. It does not import `opm/k8s/lifecycle` in this change. No `go.mod` change: the package is in the pinned `v1.0.0-beta.7`.
- Stability: whether the `opm/k8s` tier is in the library's v1 promise is still open; the library may mark the tier experimental. The cli takes the dependency either way; a break in the tier reaches the cli as a library bump.
- Users: a migration note in the PR body (text in `design.md`). Scripts that match the old left-behind reasons need the new ones.
- Enhancement: implements part of 0012 (`enhancement.yaml`); no decision is claimed until the operator side has merged. The refusal of `opm operator install` on an object annotated for another instance needs a revision note on 0012:D8:R8, which words that case as a skip.
