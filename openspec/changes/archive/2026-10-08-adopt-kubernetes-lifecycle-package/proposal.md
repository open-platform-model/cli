## Why

The library holds one deletion protocol, `opm/k8s/lifecycle`, in library `v1.0.0-beta.7`, which the cli already pins: a plan built from inventory entries, a transition that names one action at a time, and a verdict on releasing the instance's hold. Enhancement 0012 decided that both frontends delete only what that transition names (0012:D4:R1).

The cli does not import the package. It runs two delete loops of its own, `kubernetes.Delete` (instance delete, operator uninstall) and `inventory.PruneStaleResources` (prune), over one shared helper, `kubernetes.JudgedDelete`, that cli#347 added. The helper already does what the protocol does for one object: read, ask `ownership.CanDelete`, delete with Foreground propagation and a UID precondition. The order, the "CRDs and Namespaces stay" rule, the failure bookkeeping and the decision to delete the instance record are still written twice in the cli and a third time in the library. A rule added to the library plan later would not reach the cli.

This change makes the library plan the only driver of those deletes. It is a move of control, not of behaviour: the cli has sent Foreground deletes in descending weight order since its first delete command, and cli#347 added the verdict and the precondition.

## What Changes

One runner in `internal/kubernetes` drives a `lifecycle.DeletionPlan` to its end with the cli's own client. Every path below builds a plan and hands it to the runner.

| cli path | Today | With this change: the lifecycle package | Stays in the cli |
| --- | --- | --- | --- |
| `opm instance delete`, CLI-owned instance (`internal/cmd/instance/delete.go` → `workflowapply.DeleteRecorded` → `kubernetes.Delete`) | Sorts the live objects, skips protected kinds, calls `JudgedDelete` per object, deletes the record when no object failed | `NewDeletionPlan` over the recorded objects the command found or could not read (order, CRD and Namespace skip); `Advance` names every read, delete and skip; `MayReleaseHold` decides whether the record is deleted | The prompt and `--yes`; the dry run; the PersistentVolumeClaim rule of cli#345 (claims are taken out before the plan is built, unless `--delete-data`); the reads and deletes themselves, through the discovery resolver of cli#342; every output line; the exit codes; the record delete |
| `opm operator uninstall` (`internal/operator/uninstall.go` → `DeleteRecorded`) | The same code as instance delete | The same plan, transition and hold verdict | The finalizer guard and `--remove-finalizers`; the no-record refusal; output; exit codes |
| Prune in `opm instance apply` and `opm module apply`, and in the instance apply inside `opm operator install` (`internal/workflow/apply/apply.go` `pruneStale` → `inventory.PruneStaleResources`) | Sorts the stale entries, calls `JudgedDelete` per entry, stops at a failed discovery request | `NewDeletionPlan` over the prunable stale entries, with the recorded identity; `Advance` names every action. No hold verdict: a prune releases nothing | The stale set split (protected, kept claims, `--delete-data`); the stop at a failed discovery request (cli#342); `--no-prune`; the dry-run preview; which entries stay in the written record; output; exit codes |
| `opm instance delete`, operator-owned instance (`deleteOperatorOwned`) | Deletes the ModuleInstance and waits for the operator | Nothing. The operator runs the protocol | Unchanged |
| Migration deletes of `opm operator install` (`internal/operator/migration_execute.go` `deleteProven`) | Deletes the earlier Deployment and the superseded bindings under a verdict taken in the check phase | Nothing. 0012:D4:R1 names this as the one exception; the objects are not inventory entries and their verdict needs `Admit`, which a plan cannot carry | Unchanged |
| The record delete (`inventory.DeleteCR`) | Sent when no object failed | Sent only on a release verdict of `MayReleaseHold` | The delete itself |

- **No changed delete behaviour.** Foreground propagation, descending weight order, the read before each delete, the ownership verdict and the UID precondition are all in `main` today (`internal/kubernetes/delete.go`, `internal/inventory/stale.go`). The plan gives the same order: both sort with the library's `object.Sort`, descending and stable, over the record's entry order.
- **Output and exit codes stay.** `design.md` has the table of every line that moves or is new. There is one, a new failure line for a read that returns another object than the one asked for, which a conforming API server never does. No exit code changes.
- **Waiting is not part of this change.** The cli sends each delete and reports it when the API server accepts it, as today. A script that runs `opm instance delete` and then expects the objects to be gone has always had to wait itself. A bounded wait is a possible later change; nothing here rules it out. `design.md` states what happens to an object that holds a finalizer nobody removes.
- **What of cli#347 this replaces.** `kubernetes.JudgedDelete`, `DeleteOutcome`, the live-read error marker and the two loops around them go; the runner takes their place as the cli's single delete site. `ErrReplaced`, the protected-kind test, the recorded identity for prune, the left-behind reporting, the migration's check-phase verdict, the call-site test and the integration script stay. `design.md` lists both sets by symbol.
- **Order of landing.** The apply half of the ownership work, the change `guard-every-apply-by-ownership`, merged as cli#349. This change is built on a base that holds it.
- No new command, no new flag, no `go.mod` change.

Not in this change: waiting for deleted objects to disappear; the apply path; flags; the dry-run prune preview (it still reads no stale object); the operator repo.

SemVer: PATCH after GA (no user-visible contract changes). Before GA it ships in the next beta as `refactor:` in the PR title.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: `opm instance delete` of a CLI-owned instance takes every read, delete and skip from the shared deletion plan and deletes the record only on its release verdict; a resource counts as deleted when its delete is accepted.
- `apply-pruning`: prune takes every read, delete and skip from the shared deletion plan.
- `operator-lifecycle`: `opm operator uninstall` follows the same plan and release verdict as instance delete.

## Impact

- Code: `internal/kubernetes/` (new runner; `delete.go` shrinks), `internal/inventory/stale.go` (prune loop), `internal/workflow/apply/delete.go` and `apply.go` (hold verdict, prune call), `internal/kubernetes/deletesites_test.go` (allowed list), `AGENTS.md` (one line on the delete site). `internal/operator/uninstall.go` and `internal/cmd/instance/delete.go` change only where a signature does.
- Imports: the cli starts to import `github.com/open-platform-model/library/opm/k8s/lifecycle`. It is in the pinned `v1.0.0-beta.7`.
- Stability: as for the rest of the `opm/k8s` tier, whether the package is in the library's v1 promise is open. A break reaches the cli as a library bump.
- Users: none expected. No migration note.
- Enhancement: the cli's part of 0012:D4:R1, R5 and R6 (`enhancement.yaml`); no decision is claimed until the operator side has merged.
