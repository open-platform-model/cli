## Why

Three commands report success after a cluster call failed:

- `opm instance delete` (CLI-owned instance) and `opm operator uninstall` share one delete routine. When the delete of the `ModuleInstance` record fails, the routine logs a warning and returns normally. `instance delete` then prints `Instance deleted` and `operator uninstall` prints `opm-operator uninstalled`, both with exit 0. The record is still in the cluster and still lists objects that are gone.
- `opm instance diff` turns a failed read of a rendered resource (any error other than NotFound) into a warning and skips the resource. A skipped resource is no difference, so with every read failing the command prints `No differences found` and exits 0. A failed comparison, a failed read of the instance record and a failed read of a tracked resource (orphan detection) have the same outcome.

## What Changes

- `opm instance delete` and `opm operator uninstall`: when the record delete fails, the command prints no success line and exits non-zero. The error says that the tracked resources were deleted, that the record remains, and that re-running is safe.
- `opm instance diff`: a rendered resource that cannot be read or compared, an instance record that cannot be read, and a tracked resource that cannot be read are each reported as an error naming the object. The command still prints the differences it found, never prints `No differences found`, and exits non-zero.
- Exit codes come from the existing table (`internal/exit`), as cli#332 chose for the same causes: 4 when the API server denied the call (Forbidden or Unauthorized), 3 on a server timeout or service unavailable, 1 otherwise. A diff with failures of more than one class exits 1.

This reverses one sentence of the `resource-discovery` spec for `diff` only: an unreadable tracked resource was a warning that did not change the exit code. `tree` and `events` keep that rule.

No flag, no command and no output format changes. The success paths do not change. The operator-owned delete path already fails on a failed record delete and does not change.

Not in this change: the apply path; the "no resources to diff" return for an empty render; the orphan wording; resource-name resolution.

SemVer class: PATCH (bug fix). Before GA it ships in the next `1.0.0-beta.N`. Not breaking: only runs that already hit one of these failures change outcome, from exit 0 to a non-zero exit.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: a new requirement fails `opm instance delete` when the record delete fails.
- `operator-lifecycle`: a new requirement fails `opm operator uninstall` when the record delete fails.
- `inst-commands`: a new requirement fails `opm instance diff` when an object could not be read or compared.
- `resource-discovery`: the unreadable-entry requirement no longer lets `diff` exit 0.

## Impact

- `internal/workflow/apply/delete.go`: `DeleteRecorded` returns a `*RecordDeleteError`.
- `internal/cmd/instance/delete.go`: reports that error.
- `internal/operator/uninstall.go`, `internal/cmd/operator/uninstall.go`: no code change; the error reaches the existing exit mapping.
- `internal/kubernetes/diff.go`: `DiffResult` gains `Errors` and loses `Warnings`.
- `internal/cmd/instance/diff.go`: collects the failures and exits non-zero.
- Unit tests in those packages. No dependency change.
