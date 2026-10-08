## 1. Warn on a first install over resources OPM already manages

- [x] 1.1 Add tests: `inventory.FirstInstallCheck` returns the entries that exist with an OPM managed-by label and leaves out absent and admitted ones; `Execute` on a first install prints one warning naming "2 of 3" when two rendered ConfigMaps exist with the label, applies all three and exits 0; no warning on a clean first install or with a record. See the `Execute` warning test fail on the current code
- [x] 1.2 Add `FirstInstallCheck` to `internal/inventory/stale.go` with `PreApplyExistenceCheck` as its wrapper; return the list from `RunPreApplyExistenceCheck`; print the warning in `Execute`
- [x] 1.3 `task lint` and `task test:unit` green, then commit `feat(apply): warn when a first install finds resources OPM already manages`

## 2. Remove the legacy inventory Secret path

- [x] 2.1 Add a test in `internal/workflow/apply`: an instance with a legacy Secret (entries A, B, C), no record, and A, B, C in the cluster with the label; `Execute` rendering A and B applies them, records exactly A and B at revision 1, deletes nothing, prints the warning, and makes no request on Secrets. See the "no request on Secrets" and "C is not deleted" assertions fail on the current code
- [x] 2.2 Delete `internal/inventory/legacy.go` and `legacy_test.go`; drop the `legacy` value from `LoadPreviousInventory`, `WriteInstanceRecord`, `nextRevision` and `previousEntries`; delete `cleanupLegacySecret`; update or delete the tests that exist for the path; correct comments that still describe a Secret inventory
- [x] 2.3 Delete `tests/integration/migration/main.go` and its `go run` lines in `Taskfile.yml` and `.github/workflows/pr.yml`; `go vet` the remaining integration programs
- [x] 2.4 `task fmt`, `task vet`, `task lint`, `task test:unit`, `task openspec:check` and `task cascade:wiring:check` green, then commit `feat(apply)!: remove the legacy inventory Secret path`

## 3. Migration note

- [x] 3.1 Write `docs/site/diagnostics/legacy-inventory-secret.md`: the warning, what it means, the migration step with opm v1.0.0-beta.10, and the cleanup by hand when the step was skipped
- [x] 3.2 `task docs:bundle:check` and `task lint` green, then commit `docs(apply): add the migration note for the removed legacy inventory Secret`
