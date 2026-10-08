## Context

See proposal.md for the defects. Observed at the base commit:

- `DeleteRecorded` (`internal/workflow/apply/delete.go:81-86`) warns and returns `(result, nil)` when `inventory.DeleteCR` fails. `DeleteCR` returns every error other than NotFound (`internal/inventory/store.go:229-236`).
- `reportInstanceDelete` (`internal/cmd/instance/delete.go:300-325`) reads only `deleteResult.Errors`. `runOperatorUninstall` (`internal/cmd/operator/uninstall.go`) reads only `result.Errors`, and already maps an error from `operator.Uninstall` through `cmdutil.ExitCodeFromK8sError`.
- `Diff` (`internal/kubernetes/diff.go:351-378`) appends a failed read and a failed comparison to `DiffResult.Warnings`; `IsEmpty` ignores them. `Warnings` has no other writer.
- `discoverOrphanCandidates` (`internal/cmd/instance/diff.go`) logs a failed record read at debug level and warns about unreadable tracked resources.
- Exit codes are the constants in `internal/exit/exit.go` (0 success, 1 general, 2 validation, 3 connectivity, 4 permission denied, 5 not found). `cmdutil.ExitCodeFromK8sError` maps an API error to them.

## Goals / Non-Goals

**Goals:**

- Each failure fails the command, with a unit test that fails before the fix.
- No change to a success path, a flag or an output format.

**Non-Goals:**

- The apply path, the empty-render return of diff, the orphan wording.
- A changed exit code for `tree` or `events`.

## Research & Decisions

### How the record delete failure reaches the two commands

**Context**: `opm instance delete` and `opm operator uninstall` share `DeleteRecorded`. The first prints its own errors (`Printed: true`); the second lets the root command print.
**Options considered**:
1. Add the failure to `DeleteResult.Errors`. Both commands then say "1 resource(s) failed to delete" and "the ModuleInstance was kept", which is false: every resource was deleted.
2. Return a typed error. Each command words it; the uninstall command needs no change.
**Decision**: Option 2. `DeleteRecorded` MUST return `(nil, *RecordDeleteError)` and MUST NOT log the failure itself, so no command prints it twice.
**Rationale**: The typed error keeps the cause reachable through `errors.As`, so `ExitCodeFromK8sError` gives the same code as the operator-owned path does for a failed `DeleteCR`.

```go
// RecordDeleteError reports that every tracked object was deleted and the
// delete of the ModuleInstance record then failed.
type RecordDeleteError struct {
    Namespace, Name string
    Err             error
}
func (e *RecordDeleteError) Error() string
func (e *RecordDeleteError) Unwrap() error
```

A re-run is safe: absent objects count as deleted, then the record delete is tried again (the existing "Re-running uninstall" scenario).

### What diff does with an object it cannot read

**Context**: The diff of the readable objects is still useful, but the result is incomplete.
**Options considered**:
1. Stop at the first failed read. Simple, but hides the differences already found and the other failures.
2. Compare everything, print what was found, then fail.
**Decision**: Option 2. `Diff` MUST return each failed read and each failed comparison in `DiffResult.Errors`; `Warnings` is removed. The command MUST add a failed record read and each unreadable tracked resource, print the differences found, and return an `*exit.ExitError`. It MUST NOT print `No differences found` when any failure exists.
**Rationale**: One run shows every cause. A failed record read does not stop the comparison of the rendered resources; it only means orphan detection did not run, and the error says so.

```go
type DiffError struct {
    Kind, Namespace, Name string
    Err                   error
}
```

### Exit code of an incomplete diff

**Decision**: When every failure maps to the same code through `ExitCodeFromK8sError`, the command exits with that code (4 Forbidden or Unauthorized, 3 server timeout or service unavailable, 1 otherwise). Failures of more than one class exit 1.
**Rationale**: It is the table cli#332 uses for a failed read. One code cannot name two causes, so a mix is the general error.

## Command syntax, flags, output

`opm instance delete <name> [flags]`, `opm operator uninstall [flags]`, `opm instance diff <instance.cue> [flags]`. No flag changes.

```text
ERRO deleting ModuleInstance apps/demo: its tracked resources were deleted, but the record remains error=<cause>

The ModuleInstance still lists resources that are gone.
Fix the cause (for example missing RBAC) and re-run; re-running is safe.
```

```text
ERRO could not diff resource kind=ConfigMap namespace=apps name=web error="reading the live object: <cause>"
ERRO diff is incomplete: 1 object(s) could not be read or compared

Fix the cause (for example missing RBAC) and run the diff again.
```

## Risks / Trade-offs

- [A user who can read workloads but not ModuleInstances now gets a non-zero diff] -> The error names the record; without it the diff cannot see orphans, and saying "No differences found" was the defect.
- [A script that treats any non-zero diff exit as "differences exist"] -> Diff exits 0 when differences exist today, so no such contract exists.
- [The spec sentence that kept diff at exit 0 for unreadable tracked resources is reversed] -> Recorded in the proposal; `tree` and `events` keep it.

## Open Questions

None.
