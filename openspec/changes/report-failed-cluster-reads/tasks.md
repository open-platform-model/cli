## 1. A failed record delete fails delete and uninstall

- [x] 1.1 Add a test in `internal/workflow/apply` that `DeleteRecorded` returns a `*RecordDeleteError` (cause reachable through unwrapping, objects deleted) when the record delete fails, a test in `internal/cmd/instance` that `executeInstanceDelete` prints no success line, says the record remains and exits 4 (Forbidden) and 1 (internal error), and a test in `internal/operator` that `Uninstall` returns the error and keeps the record; see them fail on the current code
- [x] 1.2 Add `RecordDeleteError` to `internal/workflow/apply/delete.go` and return it from `DeleteRecorded`; report it in `executeInstanceDelete`; verify the 1.1 tests pass
- [x] 1.3 `task lint` and `task test:unit` green, then commit `fix(delete): fail when the instance record cannot be deleted`

## 2. Diff fails when an object could not be read

- [ ] 2.1 Add a test in `internal/kubernetes` that `Diff` returns a failed read and a failed comparison in `DiffResult.Errors` and still treats NotFound as added, and tests in `internal/cmd/instance` for the diff report: every read Forbidden (no `No differences found`, exit 4), one modified beside one unreadable (difference printed, exit 1), unreadable record (exit 4), unreadable tracked resource (non-zero), two classes (exit 1), clean diff (exit 0); see them fail on the current code
- [ ] 2.2 Replace `DiffResult.Warnings` with `DiffResult.Errors` in `internal/kubernetes/diff.go`; in `internal/cmd/instance/diff.go` collect the record, tracked-resource and diff failures, print the differences found and return the exit error; verify the 2.1 tests pass and `go vet ./...` compiles the integration programs
- [ ] 2.3 `task lint`, `task test:unit` and `task openspec:check` green, then commit `fix(diff): fail when an object cannot be read or compared`
