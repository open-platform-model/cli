## 1. Checks before the namespace create

- [ ] 1.1 Write `internal/workflow/apply/createns_test.go`: one case per refusing check (cluster gate, unreadable record, denied status permission, unreadable object, untracked cluster-scoped object, terminating object) with `CreateNS` set and the namespace missing, each asserting the refusal, no `create namespaces` call and no other write; see them fail on the current code
- [ ] 1.2 Split `EnsureNamespaceIfRequested` in `internal/workflow/apply/apply.go` into the read (first) and the create (after the existence check, before `kubernetes.Apply`, and before the nothing-to-apply return); the cases of 1.1 pass
- [ ] 1.3 Add success cases to the same file: the namespace is created before the first patch and the record is written at revision 1; an existing namespace is not created; nothing to apply still creates it; `go test ./internal/workflow/apply` passes, the dry-run tests in `newnamespace_test.go` unchanged
- [ ] 1.4 Reword the two refusals to "apply stopped before any change" (`internal/inventory/stale.go`, `unreadableRecordError`), replace `TestExecute_RefusalAfterNamespaceCreateClaimsNoMore` by the cases of 1.1, and assert the text in the tests
- [ ] 1.5 `task fmt`, `task vet`, `task lint`, `task openspec:check` and `task test:unit` green, then commit fix(apply): run the checks before --create-namespace creates the namespace

## 2. Install-level test for an unreadable object

- [ ] 2.1 Add `internal/operator/unreadable_test.go`: `PlanInstall` refuses an object whose read is denied, at each of its three reads, with the error type of that read and no write; `go test ./internal/operator` passes
- [ ] 2.2 Add the rows to `TestInstallErrorMapping` in `internal/cmd/operator/operator_test.go` that map those error shapes to their exit codes (a `GuardError` wrapping a Forbidden read: 2)
- [ ] 2.3 Update the comment on `GuardError` in `internal/operator/plan_install.go` to name the unreadable and terminating cases and its exit code
- [ ] 2.4 `task lint`, `task test:unit`, `task docs:bundle:check` and `task cascade:wiring:check` green, then commit test(operator): show that install refuses an object it cannot read
