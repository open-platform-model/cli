# Tasks: retire-legacy-manifest-migration

Ships as its own PR, merged before the controller rename; `rename-operator-command-to-controller` follows in a separate PR once the controller's first release under the new name is published.

**Local gate:** `task fmt`, `task lint`, `go test ./internal/... ./pkg/...` (unit), `go vet -tags e2e ./tests/e2e/`, `task openspec:check`.

## 1. Delete the migration

- [x] 1.1 Delete `internal/operator/legacy.go`, `legacy_test.go`, `migration_{plan,proof,execute,report}.go` and their tests, `migration_fixtures_test.go`, `internal/operator/testdata/legacy-manifests.json` and `hack/operator-legacy/`.
- [x] 1.2 Delete `tests/e2e/operator_migration_test.go` and `tests/e2e/testdata/legacy-operator/`.
- [x] 1.3 Delete `internal/inventory/admit.go` and `admit_test.go`; drop the `admit` parameter of `PreApplyExistenceCheck` (`internal/inventory/stale.go`) and its test `TestPreApplyExistenceCheck_AdmitSet`.
- [x] 1.4 Drop `Request.Admit` and the `admit` argument of `RunPreApplyExistenceCheck` in `internal/workflow/apply/apply.go`.
- [x] 1.5 `internal/operator/plan_install.go`: drop `Plan.Migration`, the `PlanMigration` call and the admission in the guard call; delete `operatorTagShape`, `looksLikeOperatorTag`, `operatorTagRefusal`. `install.go`: drop the migration writes and `Admit:`. Reword the names.go and ready.go comments that named manifest installs.
- [x] 1.6 `internal/cmd/operator/install.go`: drop the migration paragraph of the help, the `MigrationStoppedError` exception in `withRerunHint` and the migration errors in `installError`; update `operator_test.go` and `plan_install_test.go` (the old-tag case becomes an unserved-version case).
- [x] 1.7 README (Operator Lifecycle bullets) and AGENTS (`hack/operator-legacy` entry).
- [x] 1.8 `go build ./...`, `go vet ./...`, `go vet -tags e2e ./tests/e2e/`, `task lint` and the unit tests green, then commit `refactor(operator)!: retire the legacy-manifest migration`.

## 2. Specs and archive

- [x] 2.1 `task openspec:check` passes with this change's deltas (`operator-migration` REMOVED-only with `retire_capabilities: true`; `apply-pruning` and `operator-lifecycle` REMOVED + ADDED retitles).
- [x] 2.2 Verify the implementation against the change (opsx verify), archive it (`openspec archive retire-legacy-manifest-migration --yes`), confirm `openspec/specs/operator-migration` is gone and `task openspec:check` still passes, then commit `chore(openspec): archive retire-legacy-manifest-migration`.
