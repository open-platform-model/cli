## 1. Unreadable record stops the apply

- [x] 1.1 Add a test in `internal/workflow/apply` that runs `Execute` with a `ModuleInstance` read failing (internal error, Forbidden; real run and dry run) and asserts an error, the exit code, no resource patch and no record write; see it fail on the current code
- [x] 1.2 Make `LoadPreviousInventory` return the read error and `Execute` return it as an `ExitError` on both reads; update the call in `tests/integration/migration/main.go`; verify the 1.1 test passes and `go vet ./...` compiles the integration programs
- [x] 1.3 `task lint` and `task test:unit` green, then commit `fix(apply): stop when the instance record cannot be read`

## 2. First-install existence check refuses an unreadable object

- [x] 2.1 Add a test in `internal/inventory` for `PreApplyExistenceCheck` with a read failing with Forbidden (error returned, names the resource, still Forbidden through unwrapping) and with NotFound (passes), and a test in `internal/workflow/apply` that `Execute` exits 4 and applies nothing; see them fail on the current code
- [x] 2.2 Return the read error from `PreApplyExistenceCheck` and map it to an exit code in `RunPreApplyExistenceCheck`; verify the 2.1 tests and the `internal/operator` tests pass
- [x] 2.3 `task lint` and `task test:unit` green, then commit `fix(apply): refuse a first install when an object cannot be read`

## 3. Failed prune keeps the entry and fails the command

- [x] 3.1 Add a test in `internal/inventory` that `PruneStaleResources` returns a `PruneError` listing exactly the entries whose delete failed, and a test in `internal/workflow/apply` that `Execute` writes a record holding the current entries plus the failed entry, names the resource in the output, prints no success line and exits 1; see them fail on the current code
- [x] 3.2 Add `PruneError` to `internal/inventory/stale.go` and return it from `PruneStaleResources`; in `Execute` keep the failed entries in the written record and return exit 1 after the write; verify the 3.1 tests pass
- [x] 3.3 `task lint`, `task test:unit` and `task openspec:check` green, then commit `fix(apply): keep entries a failed prune left behind and exit non-zero`
