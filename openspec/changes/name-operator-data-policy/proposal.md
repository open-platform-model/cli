## Why

cli#345 made the CLI keep PersistentVolumeClaims unless `--delete-data`, and told the user that the operator does not: the delete prompt of an operator-managed instance with `spec.prune` set says "the operator deletes its tracked resources, PersistentVolumeClaims and the data on them included", and the page `kept-volume-claims.md` says the same. The operator change opm-operator#267 makes both untrue: the operator keeps claims on prune and on deletion, and deletes them only when `spec.dataPolicy` of the instance is `Delete`. The CLI must say what the operator will do, read from the instance.

## What Changes

- The delete prompt of an operator-managed instance reads `spec.prune` and `spec.dataPolicy`, and, when the instance tracks a claim, reads from the installed `ModuleInstance` CRD whether the operator has `spec.dataPolicy` at all. No released operator had the field when this was written. When the CRD lacks it, the prompt says that the operator deletes claims. When the CRD has it: with `dataPolicy: Delete` claims are deleted; with `Keep`, no value, or an unknown value they are kept, the value is shown as it is, and one sentence says that an operator older than its CRDs deletes them.
- The dry run and the run with `--yes` state the same outcome, and the closing output of a delete after which the operator may have kept claims no longer says that the operator pruned every tracked resource: it names the claims and prints the `kubectl delete pvc` command for each.
- The warning for `--delete-data` on an operator-managed instance (`opm instance delete`, `opm instance apply`, `opm module apply`) says that the flag does not change what the operator does, names `spec.dataPolicy` as the setting of an operator that has it, and says that an older operator deletes claims. It stays a warning; the command goes on.
- The help of the three commands and the page `docs/site/diagnostics/kept-volume-claims.md` say the same, and the page names the exception the operator documents: a forced recreate under `spec.rollout.forceConflicts` deletes and recreates a claim whatever the data policy says.
- The CLI reads `spec.dataPolicy` from the unstructured `ModuleInstance` it already reads, and does not import the operator's Go types. The CRD read is one more GET of an object the delete of an operator-managed instance reads already for its readiness gate.

What does not change: everything the CLI does for an instance it owns (cli#345), the flags, the exit codes, and what the operator does.

SemVer: PATCH (corrected messages and documentation, no new flag, no changed exit code). During beta it ships as the next `1.0.0-beta.N`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: the two requirements on `opm instance delete` and PersistentVolumeClaims change what the prompt, the warning and the closing output say for an operator-managed instance.
- `apply-pruning`: the requirement on the prune and `--delete-data` changes what the warning for an operator-managed instance says.

## Impact

- `internal/cmd/instance/delete.go`: the prompt, the dry-run and progress lines and the closing output of the operator-owned branch; the help text.
- `internal/cmd/instance/apply.go`, `internal/cmd/module/apply.go`: one help sentence each.
- `internal/workflow/apply/delete.go`: the text of the constant `DeleteDataOperatorManagedNote`, nothing else in that package.
- `internal/operator/crdfield.go` (new): `ModuleInstanceSpecField`, a read of the ModuleInstance CRD schema.
- `internal/inventory/record.go`, `internal/inventory/store.go`: one new field `Record.DataPolicy` and its read from `spec.dataPolicy`, beside `Record.Prune`. No logic there changes.
- `docs/site/diagnostics/kept-volume-claims.md`.
- Tests in `internal/cmd/instance`, `internal/inventory` and `internal/workflow/apply`.
- No dependency, no operator change.
