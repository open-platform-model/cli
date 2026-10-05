## Why

`opm instance delete` can orphan an object, and the read-only instance commands can hide one, when a tracked object cannot be read (cli issue #283).

- **Discovery drops what it cannot read.** `inventory.DiscoverResourcesFromInventory` (`internal/inventory/discover.go:59-71`) performs one GET per inventory entry. A GET that fails with anything but NotFound (Forbidden, a timeout, a 5xx) is logged at debug level and the entry is dropped: it is reported neither live nor missing. The function's error return is always nil (`:87`).
- **Delete then orphans the object.** `opm instance delete` passes only the live list to `kubernetes.Delete` (`internal/cmd/instance/delete.go:125`, `:270-281`). The dropped entry never reaches the in-loop re-read, whose rule "a read error keeps the `ModuleInstance` so a re-run retries" therefore does not apply. Delete reports no error and removes the `ModuleInstance`, so the object stays on the cluster with no record that tracked it. A persistent RBAC denial does this every time.
- **The read-only commands under-report.** `opm instance status` (`internal/workflow/query/status.go:63`), `opm instance list` (`internal/workflow/query/list.go:57`) and the orphan detection of `opm instance diff` (`internal/cmd/instance/diff.go:108`) show fewer resources than the instance tracks, with no warning. `status` and `list` can report `Ready` for an instance whose unreadable objects they never saw. `opm instance tree` and `opm instance events` read through the same `query.ResolveInventory` and drop the entry the same way.

## What Changes

- **Discovery returns the entries it could not read.** `DiscoverResourcesFromInventory` gains a third result, `unreadable []UnreadableEntry` (`Entry InventoryEntry`, `Err error`), in inventory order. NotFound stays "missing". `query.ResolveInventory` passes the slice through to its callers.
- **Instance delete keeps the `ModuleInstance` when a tracked object could not be read.** Each unreadable entry is a per-resource failure, listed with its read error, exactly like a failed in-loop re-read today: the readable resources are still deleted, the `ModuleInstance` is kept so its inventory still tracks the unread objects, and the command exits 1. The closing output says that the instance was kept and that re-running is safe. A dry run counts an unreadable entry among the resources it "could not check". An unreadable `Namespace` or `CustomResourceDefinition` is listed as left behind instead, since delete never deletes those kinds. The operator-owned path is unchanged: it deletes only the `ModuleInstance` and the operator prunes with its own credentials.
- **The read-only commands warn and count the entry as not ready.** `status`, `list`, `diff`, `tree` and `events` print one warning per unreadable entry (for `list`, one line per affected instance) naming the resource and the read error. `status` lists each unreadable resource with health `Unknown`, so the instance's aggregate is `NotReady` and the command exits 2. `list` counts it toward the total and not toward the ready count, so the instance shows `NotReady (r/t)`. `diff` warns that orphan detection could not check those resources.

**Behaviour change for users.** `opm instance delete` now exits 1 and keeps the `ModuleInstance` when any tracked object cannot be read, where it previously reported success and orphaned the object silently. Fix the cause (usually RBAC) and re-run; the re-run retries only what is still tracked. `opm instance status` and `opm instance list` now report such an instance as not ready instead of omitting the object. Under the repo's squash settings only the PR title reaches `main` and CHANGELOG.md, so the PR title carries this: `fix: keep the instance when delete cannot read a tracked resource; read-only commands warn`. The PR body says `Closes #283`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `resource-discovery`: inventory discovery reports the tracked resources it could not read, and every command that reads them warns instead of dropping them silently.
- `deploy`: `opm instance delete` treats a resource discovery could not read as a per-resource failure and keeps the `ModuleInstance`.
- `mod-status`: `opm instance status` lists an unreadable tracked resource as `Unknown` and reports the instance as not ready.
- `mod-list`: `opm instance list` counts an unreadable tracked resource as not ready.
- `status-exit-codes`: a failed read of one tracked resource exits 2 (an `Unknown` row), not 1 or 4; exit 1 and exit 3 are rescoped to the `ModuleInstance` record read and client setup.

## Impact

- **Release class: `fix`, PATCH (after GA as well), shipped as the next beta.N.** No flag, command or exit code is added or removed. The change is a safety fix: no input that worked stops working, but `instance delete` now fails where it used to orphan silently, so the PR title is the changelog line above (`fix`, not `feat!`).
- Commands: `opm instance delete` (CLI-owned path), `opm instance status`, `opm instance list`, `opm instance diff`, `opm instance tree`, `opm instance events`.
- Packages: `internal/inventory` (`discover.go`), `internal/kubernetes` (`delete.go`, `status.go`, `health.go`), `internal/workflow/query` (`status.go`, `list.go`), `internal/cmd/instance` (`delete.go`, `diff.go`, `status.go`, `tree.go`, `events.go`). The integration programs under `tests/integration/` that call `DiscoverResourcesFromInventory` take the extra result.
- Tests: a fake dynamic client whose GET reactor returns `apierrors.NewForbidden` drives every new unit test. The operator-owned delete e2e test (`TestE2E_Delete_OperatorOwnedDelegates`) must stay green.
- Coordination: cli PR #307 (another session, `feat!`, branch `feat/install-operator-from-module`) overlaps. It moves `executeInstanceDelete`'s body into `workflowapply.DeleteRecorded`/`DeleteRequest` (`internal/workflow/apply/delete.go`) and adds a second delete caller, `opm operator uninstall` (`internal/operator/uninstall.go`), which calls `DiscoverResourcesFromInventory` and then `DeleteRecorded`. After either rebase that caller compiles with the unreadable result discarded as `_`, so `operator uninstall` would still orphan an unreadable object. If #307 merges first, this change adds `Unreadable []kubernetes.UnreadableResource` to `workflowapply.DeleteRequest`, passes it into `DeleteOptions`, and has operator uninstall pass the unreadable slice from discovery, with a unit test. If this change merges first, the PR stage leaves a comment on #307 naming the uninstall caller so its rebase does not discard the slice. The wave-2 sibling `cli-followups` has no file overlap with this change. The later library delete protocol (kernel plan e4/f2) may take this rule over for both frontends; this change does not wait for it because the bug orphans objects today.
- Out of scope: typed classification of read errors (transient vs terminal; that arrives with the library's typed fetch errors), a retry inside discovery, and any operator change.
