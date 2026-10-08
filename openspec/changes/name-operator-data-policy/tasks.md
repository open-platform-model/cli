## 1. The CLI names the operator's data policy

- [x] 1.1 `internal/inventory`: add `Record.DataPolicy` and read `spec.dataPolicy` beside `spec.prune`; a test reads `Delete`, an unknown value, an absent value and a wrong-typed value from a `ModuleInstance` object
- [x] 1.2 `internal/workflow/apply/delete.go`: reword `DeleteDataOperatorManagedNote` so that it names `spec.dataPolicy`; a test asserts the constant names the field, and the thin-editor test that reads the constant still passes
- [x] 1.3 `internal/cmd/instance/delete.go`: `operatorDeletesClaims`, `describeDataPolicy` and the prompt read from `spec.prune` and `spec.dataPolicy`; a table test covers prune off, `Delete`, `Keep`, absent and an unknown value, and fails on the old text
- [x] 1.4 `internal/cmd/instance/delete.go`: the dry-run line, the progress line and the closing output of the operator-owned branch state the claim outcome; tests cover a completed delete that tracked a claim under no policy and under `Delete`, and the dry run
- [x] 1.5 Help of `opm instance delete`, `opm instance apply` and `opm module apply`: name `spec.dataPolicy` for an operator-managed instance; `TestDeleteDataFlagOnDeleteAndBothApplies` asserts that the three help texts name the field
- [x] 1.6 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix(cmd): name spec.dataPolicy for operator-managed claims in delete and apply`

## 2. The docs page

- [x] 2.1 `docs/site/diagnostics/kept-volume-claims.md`: the operator keeps claims unless `spec.dataPolicy` is `Delete`, `--delete-data` does not change that, what an older operator does and how to check, and the forced recreate under `spec.rollout.forceConflicts`; `task docs:bundle:check` passes
- [x] 2.2 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `docs(site): say that the operator keeps claims unless spec.dataPolicy is Delete`
