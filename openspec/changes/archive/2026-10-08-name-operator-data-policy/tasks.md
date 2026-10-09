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

## 3. The CLI reads from the CRD whether the operator has the field

- [x] 3.1 `internal/operator`: `ModuleInstanceSpecField` reads the ModuleInstance CRD and reports whether a served version has a field of `spec`; a test covers present, absent, not served, no CRD and a failed read
- [x] 3.2 `internal/cmd/instance/delete.go`: read the CRD before the prompt, only for an operator-managed instance with `spec.prune` set that tracks a claim; the prompt, the dry-run line, the progress line and the closing output say that claims are deleted when the CRD has no `spec.dataPolicy`, say "kept" only when it has, and say nothing about claims when none is tracked; tests cover the three CRD cases and count the CRD reads
- [x] 3.3 The `--delete-data` warning and the help of the three commands say what an older operator does; the docs page says what the command reads and what each case prints
- [x] 3.4 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix(cmd): say that an operator without spec.dataPolicy deletes claims`

## 4. Fix-check: the refusal comes before the question, one CRD read, one schema walker

- [x] 4.1 `internal/operator`, `internal/inventory`: `ReadyModuleInstanceCRD` returns the CRD the readiness gate read; `SpecFieldSupport` asks `inventory.ModuleInstanceCRDHasField`, the walker of the apply gate; tests cover present, absent, no schema, no CRD and the gate's refusal
- [x] 4.2 `internal/cmd/instance/delete.go`: the question is asked after the guard and the readiness gate; tests pin that a refused delete prints no question and that the ModuleInstance CRD is read once
- [x] 4.3 The docs page says what the command reads, names `kubectl get platform cluster`, drops the claims nobody ran, and names the unsupported skew
- [x] 4.4 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix(cmd): refuse an operator-managed delete before asking about it`
