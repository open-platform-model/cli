## Why

`opm instance delete` of a CLI-owned instance sends its deletes and returns. Each delete uses Foreground propagation, so a deleted resource can still exist, with a `deletionTimestamp`, when the command exits 0. A script that deletes an instance and then applies it again can apply onto a resource that is still terminating; the garbage collector then removes what the apply wrote. A resource that a finalizer holds is reported `deleted` and nothing tells the user that it is still there. The change that moved delete onto the shared deletion plan (cli#350) recorded this and left room for a bounded wait.

## What Changes

- `opm instance delete` gets `--wait`. With it, a CLI-owned delete polls the resources whose delete the API server accepted until each one is gone, or until `--timeout` passes. A resource is gone when its read returns NotFound, or when another object (a different UID) holds its name.
- `--timeout` (already on the command, default `5m0s`) also bounds this wait. Its budget starts when the wait starts.
- When the wait times out, the command lists each resource that is still terminating, with its finalizers, keeps the `ModuleInstance`, and exits 1. A re-run reads the same resources and waits again.
- Resources that the command does not delete are not waited for: kept PersistentVolumeClaims, resources left behind (CRDs, Namespaces, resources the ownership verdict let go), and resources that were already absent.
- A run with a per-resource failure does not wait: it fails as today.
- `--dry-run` sends no delete, so `--wait` does nothing there.
- Operator-managed instance: no behaviour change. The command already waits, with or without `--wait`, for the `ModuleInstance` to be gone, bounded by `--timeout`. The help now says so.
- `opm operator uninstall` does not get the flag (see design).
- Help text, the generated command reference and a new docs page describe the flag.

Without `--wait`, output and exit codes do not change; a test pins this.

SemVer class: MINOR (a new flag with a default that keeps today's behaviour). Before GA it ships in the next `1.0.0-beta.N`. Not breaking.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: a new requirement for the bounded wait of `opm instance delete`; the requirement "Instance delete reports a resource as deleted when its delete is accepted" is scoped to runs without `--wait`.

## Impact

- `internal/kubernetes`: the deletion run records the UID each accepted delete was sent with; a poll helper waits for deleted objects; `Delete` gets a wait option and reports what is still terminating.
- `internal/workflow/apply/delete.go`: `DeleteRecorded` passes the option and holds the record when resources are still terminating.
- `internal/cmd/instance/delete.go`: the flag, the help, the timeout report.
- `docs/site/diagnostics/`, `README.md`: user documentation.
- No new dependency. No change to the apply path, to `internal/config` or to `internal/operator`.
