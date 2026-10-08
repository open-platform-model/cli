## 1. The bounded wait in the delete routine

- [ ] 1.1 Add tests in `internal/kubernetes`: `RunDeletion` records the UID of an accepted delete and none on a dry run, skip or failure; `WaitDeleted` returns at once when every object is NotFound, counts another UID under the same name as gone, polls until a terminating object goes, returns a `*TerminatingError` with finalizers at the deadline, keeps an object with a failing read pending, and returns the context error on cancellation
- [ ] 1.2 Add `StepResult.UID`, `WaitDeleted` and `TerminatingError`; add `DeleteOptions.Wait` and `Timeout` and `DeleteResult.Terminating` to `Delete`, with tests that a kept claim, a left-behind resource and a run with a failed resource are not waited for and that a dry run does not wait
- [ ] 1.3 Add `Wait` and `Timeout` to `workflowapply.DeleteRequest`; `DeleteRecorded` keeps the record when resources are still terminating; test both outcomes
- [ ] 1.4 `task fmt`, `task vet`, `task lint` and `task test:unit` green, then commit `feat(kubernetes): wait for deleted objects to be gone`

## 2. The flag on instance delete

- [ ] 2.1 Add tests in `internal/cmd/instance`: `--wait` is registered and reaches the delete; a timeout prints each terminating resource with its finalizers, prints no success line, keeps the `ModuleInstance` and exits 1; a wait that completes reports as without the flag; without `--wait` no resource is read after its delete and the output and exit code of a delete with a terminating resource are pinned
- [ ] 2.2 Add `--wait` to `opm instance delete`, pass it with `--timeout` to the delete, print the timeout report, and update the help of the command and of `--timeout`
- [ ] 2.3 `task fmt`, `task vet`, `task lint`, `task test:unit` and `task docs:bundle:check` green, then commit `feat(instance): add --wait to instance delete`

## 3. Documentation

- [ ] 3.1 Add `docs/site/diagnostics/wait-for-a-deleted-instance.md` and name the flag in the `README.md` command table
- [ ] 3.2 `task openspec:check`, `task docs:bundle:check` and `task cascade:wiring:check` green, then commit `docs(instance): describe delete --wait`
