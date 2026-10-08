## 1. The dry run runs the ownership guard

- [ ] 1.1 Add the status words `would refuse` and `would skip` to `internal/output/styles.go`; verify with a formatter test that each renders with the attention icon.
- [ ] 1.2 In `internal/workflow/apply/apply.go`, run `inventory.Guard` on a dry run with the same input as the real run, print one `would refuse` line per refused object, return exit 1 with the dry-run error, return the real run's code for an unreadable object, print `would skip` for a let-go object and leave it out of the server-side dry run; delete `previewAlreadyManaged`. Verify with tests in `guard_test.go`, `existence_test.go`, `letgo_test.go` and `firstinstall_test.go`: refusal lines, exit codes, no write, both first and later apply.
- [ ] 1.3 Print the closing line when every rendered object is let go, on a real run (exit 0, record written) and on a dry run; verify with `TestExecute_LetsGoOfItsOnlyObject` and a dry-run twin.
- [ ] 1.4 `task fmt`, `task vet`, `task lint`, `task test:unit` green, then commit `fix(apply): run the ownership guard on a dry run`.

## 2. The prune preview asks the delete verdict

- [ ] 2.1 In `internal/inventory/stale.go`, add `PreviewPruneStaleResources`, sharing the deletion-plan run with `PruneStaleResources` and sending no delete; `LeftBehind` carries the skip reason. Verify with a test in `internal/inventory` that asserts the verdicts and that no delete is sent.
- [ ] 2.2 In `internal/workflow/apply/apply.go`, make the dry-run prune preview list `would prune`, `would keep`, `would let go` and `cannot check` from that run, and exit as the real apply does when a stale object cannot be read. Verify with tests in `dryrunprune_test.go` and `pruneownership_test.go`.
- [ ] 2.3 `task fmt`, `task vet`, `task lint`, `task test:unit` green, then commit `fix(apply): judge stale objects in the dry-run prune preview`.

## 3. Docs

- [ ] 3.1 Rewrite the section "What a dry run does not show" of `docs/site/diagnostics/adopt-an-existing-object.md` as "What a dry run shows", with the status words and exit codes, and correct the page description; verify with `task docs:bundle:check`.
- [ ] 3.2 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit`, `task cascade:wiring:check` green, then commit `docs(apply): say what a dry run shows of the ownership check`.
