## Why

`opm operator install` carries a one-time migration of an opm-operator installed from an earlier release manifest (capability `operator-migration`, 0012:D8:R6/R7): a proof list of every object of every manifest release (`legacy.go`, recorded by `hack/operator-legacy` into a 6986-line `testdata/legacy-manifests.json`), a read-only proof, field-ownership moves, a Deployment recreate, superseded role-binding deletes, and an admission set threaded through the apply guard. Nobody runs OPM yet, so there is no manifest-installed operator to take over. The owner withdrew 0012:D8:R6 and 0012:D8:R7 on 2026-10-05 and asked for every piece of legacy-migration code to go, before the operator is renamed to the controller, so about 1,700 lines of dead code are not renamed.

## What Changes

- **BREAKING** `opm operator install` no longer migrates an operator installed from a release manifest. Such an operator's objects now fail the apply guard like any foreign object.
- Delete `internal/operator/legacy*.go`, `migration_*.go` (and tests), `internal/operator/testdata/legacy-manifests.json`, `hack/operator-legacy/`, the e2e migration test and its `tests/e2e/testdata/legacy-operator/` manifest.
- Delete the admission plumbing: `internal/inventory/admit.go` (`K8sIdentity`, `IdentityOf`, `AdmitSet`), the `admit` parameter of `inventory.PreApplyExistenceCheck`, `workflowapply.Request.Admit` and the `admit` argument of `RunPreApplyExistenceCheck`.
- Delete `operatorTagShape`, `looksLikeOperatorTag` and `operatorTagRefusal`: a `--version` shaped like an opm-operator release tag is refused like any other unserved selector.
- Trim `plan_install.go`, `install.go` and `cmd/operator/install.go` (help text, `MigrationStoppedError`/`MigrationRefusalError` mapping), plus README and AGENTS.

Release class: MAJOR after GA (a removed behaviour of `opm operator install`); during beta it ships as the next `-beta.N`. Commit `refactor(operator)!:`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `operator-migration`: retired (every requirement REMOVED, `retire_capabilities: true`).
- `apply-pruning`: the first-install existence check loses its admission set (REMOVED + ADDED under a new title, since three scenarios go).
- `operator-lifecycle`: the check-order requirement drops the migration proof and writes, the version-selection requirement drops the old-tag message, and the running-operator check drops the manifest-install case (each REMOVED + ADDED under a new title).

## Impact

- Packages: `internal/operator`, `internal/cmd/operator`, `internal/inventory`, `internal/workflow/apply`, `hack/`, `tests/e2e`.
- The homelab's two manifest installs (alpha.7) can no longer be migrated by the CLI; their move is a manual runbook outside this change.
- The `deploy` spec has no migration or admission text, so it is not touched.
- No enhancement decision is claimed delivered: this change removes an implementation of withdrawn requirements.
