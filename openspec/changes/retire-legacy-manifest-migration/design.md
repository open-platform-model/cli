## Context

All `path:line` references are on `origin/main` at `03ec4374`.

`PlanInstall` (`internal/operator/plan_install.go:188`) runs its checks in this order: record read, values merge, render, target rules, status-RBAC gate, wait for terminating objects, the migration proof (`PlanMigration`, `:234`), the ownership guard (`inventory.Guard`, `:249`, fed `Admit: plan.Migration.Admit()` at `:253`). `Install` (`internal/operator/install.go:50`) then writes: the CRD step, the migration's writes (`MoveOwnership` `:73`, `DeleteSuperseded` `:76`, `MigrationReport` `:79`), the instance apply (`workflowapply.Execute`, `:87`, fed `Admit` at `:91`), the `--rbac` objects, the rollout wait.

The guard asks the library's `ownership.CanApply` for every rendered object (`internal/inventory/guard.go:122`). `Admit` matters in exactly one branch of the verdict: an object outside the inventory that OPM does not manage, which carries no UUID label or this instance's (library v1.0.0-beta.8, `opm/k8s/ownership/apply.go:109` and `:151`). It never lifts terminating, other-instance or adopted-elsewhere.

## Goals / Non-Goals

**Goals:** delete the migration and every type, field, message, test, fixture and doc that exists only for it; make the ownership guard the one judge of an existing object on install; keep every other install check and every `opm instance` and `opm module` verdict unchanged.

**Non-Goals:** the operator to controller rename; the library's removal of `Admit`; a hint line or docs page for the install over an earlier identity; `opm operator uninstall`.

## What is removed

Whole files:

| Path | Lines | Holds |
| --- | --- | --- |
| `internal/operator/legacy.go` | 101 | `LegacyObject`, `LegacyObjects`, `SupersededBindings`, `legacyDeployment`, `FirstLegacyRelease` |
| `internal/operator/legacy_test.go` | 128 | the proof list against the recorded manifests |
| `internal/operator/migration_proof.go` | 156 | `Verdict`, `VerdictAbsent/Proven/Ours/Unproven`, `ProveLegacy`, `gitOpsOwner`, `instanceRef` |
| `internal/operator/migration_plan.go` | 361 | `MigrationPlan`, `Admit()` (`:86`), `MigrationBlock`, `MigrationRefusalError` (`:109`), `MigrationReadError` (`:124`), `PlanMigration` (`:145`), `judgeDeletes` (`:209`, the cli's one `DeleteInput.Admit: true`, `:225`), `getLive` |
| `internal/operator/migration_execute.go` | 125 | `MigrationStoppedError` (`:22`), `MoveOwnership` (`:40`), `DeleteSuperseded` (`:78`), `deleteProven` (`:108`) |
| `internal/operator/migration_report.go` | 31 | `MigrationReport` |
| `internal/operator/migration_{proof,plan,execute,report}_test.go`, `migration_fixtures_test.go` | 1093 | their tests; the helpers `findObj` and `paths` move to `module_fixtures_test.go` because `guard_test.go` uses them |
| `internal/operator/testdata/legacy-manifests.json` | 6986 | the recorded manifest objects |
| `hack/operator-legacy/main.go`, `main_test.go` | 247 | the only writer of that file |
| `tests/e2e/operator_migration_test.go` | 268 | `TestE2E_Operator_MigratesManifestInstall` |
| `tests/e2e/testdata/legacy-operator/install-v1.0.0-beta.8.yaml` | 1761 | its fixture |
| `internal/inventory/admit.go`, `admit_test.go` | 59 | `AdmitSet`, `AdmitSet.Has` |
| `openspec/specs/operator-migration/spec.md` | 272 | the capability, 13 requirements (removed by the archive) |

Edits:

| Path:line | Removed |
| --- | --- |
| `internal/inventory/guard.go:26-28` | field `GuardInput.Admit` |
| `internal/inventory/guard.go:127` | `Admit: in.Admit.Has(obj)` in the `ownership.ApplyInput` |
| `internal/workflow/apply/apply.go:95-98`, `:208` | field `Request.Admit` and its use; the comment at `:80` |
| `internal/operator/plan_install.go:73-76` | field `Plan.Migration` |
| `internal/operator/plan_install.go:231-237` | the `PlanMigration` call |
| `internal/operator/plan_install.go:253` | `Admit:` in the guard call; the comments at `:166-172`, `:180-187`, `:239-244` are reworded |
| `internal/operator/install.go:68-82` | the migration's writes and report lines |
| `internal/operator/install.go:91` | `Admit:` in the instance apply; the comment at `:43-49` is reworded |
| `internal/operator/plan_install.go:82-107`, `:121-123`, `:130-132` | `operatorTagShape`, `looksLikeOperatorTag`, `operatorTagRefusal` and their two uses in `ResolveTarget`; the test `TestLooksLikeOperatorTag`; `TestRunOperatorInstall_OldOperatorTagIsRefusedBeforeTheCluster` becomes an unserved-version test |
| `internal/operator/names.go:14-19`, `internal/operator/ready.go:40-42` | the comment text that names a manifest install |
| `internal/cmd/operator/install.go:69-75` | the help paragraph "An operator installed from an earlier release manifest ... is migrated" |
| `internal/cmd/operator/install.go:265-273` | the `MigrationStoppedError` exception in `withRerunHint` |
| `internal/cmd/operator/install.go:353`, `:355`, `:360`, `:362` | `MigrationRefusalError` (exit 2) and `MigrationStoppedError` (exit 1) in `installError` |
| `internal/kubernetes/applysites_test.go:29-30`, `:39` | the allowed apply site `migration_execute.go:MoveOwnership` |
| `internal/kubernetes/deletesites_test.go:20-24`, `:28` | the allowed delete site `migration_execute.go:deleteProven`; one exception fewer |
| `README.md:155`, `:156` | the bullet "Install migrates an operator installed from a release manifest"; "which migrates it" in the uninstall bullet |
| `AGENTS.md:131` | the `hack/operator-legacy/` entry |
| `docs/site/diagnostics/adopt-an-existing-object.md:50` | the paragraph on `operator migration refused` |

Messages that no longer exist: `no operator module version matches "<v>": --version now takes an operator module version ...` (the old-tag refusal); `operator migration refused: N object(s) of an earlier operator manifest cannot be proven:` with its per-object reasons (`carries the identity of instance ...`, `label ... is missing, earlier manifests set ...`, `is applied by Flux ...`, `the module renders no Deployment of this name ...`) and its last line `nothing was changed; remove or rename these objects, or stop the tool that applies them, then re-run 'opm operator install'`; `operator migration refused: cannot read ...`; the stopped-migration error; the install report lines of `MigrationReport`. No flag is removed: the migration had none.

`TestNoErrorTextMatch`: the migration matched no error text, so the list of named exceptions does not change.

## Rule the migration enforced, who enforces it after, test

| # | Rule today | Where | After this change | Test |
| --- | --- | --- | --- | --- |
| 1 | An existing object that OPM does not manage is refused unless proven to come from a release manifest | `migration_proof.go:45`, `guard.go:127` | The guard refuses it, proven or not: `foreign-object`, exit 2, names `opmodel.dev/adopt`. The proof exception is gone (0012:D8:R6 withdrawn) | new `TestPlanInstall_ManifestInstallIsRefused`; existing `TestGuard_Verdicts` foreign-object rows |
| 2 | An object on the proof list that carries another instance's UUID refuses the install, whatever the record or the annotation says, with "remove or rename" | `migration_proof.go:53-54`, `migration_plan.go:117` | The guard decides: recorded applies; annotated applies; neither is `other-instance`, exit 2, names the annotation | new `TestPlanInstall_ObjectsOfAnotherIdentity` (three rows); `TestPlanInstall_CRDOfAnotherInstanceRefusesBeforeAnyWrite` (error type changes) |
| 3 | An object another instance adopted refuses the install | `guard.go:134` (`RefuseLetGo`), and `migration_plan.go:209` for the deletes | Unchanged: the guard, `RefuseLetGo: true` | existing `TestPlanInstall_GuardRefusesWhatAnotherInstanceOwns`, unedited |
| 4 | A terminating object is never applied over | terminating wait `install.go:156`; guard `terminating` | Unchanged | existing `TestPlanInstall_WaitsOutTerminatingObjects`, `TestGuard_Verdicts` terminating rows |
| 5 | An object a GitOps tool applies (Flux or Argo CD marks, a foreign server-side-apply manager) is not taken over | `migration_proof.go:109` (`gitOpsOwner`), proof-list objects only | Gone with the proof. Such an object carries no OPM managed-by label, so rule 1 refuses it; a user who annotates it takes it over knowingly, as for any `opm instance apply` | rule 1's test |
| 6 | Fields of a client-side `kubectl apply` are handed to `opm-cli` before the instance apply, so earlier labels and annotations do not survive | `migration_execute.go:40` | Gone (0012:D8:R7 withdrawn). The instance apply is a forced server-side apply: it takes every field it sets; a field only the earlier manager set stays on the object | none; named as gone |
| 7 | The earlier Deployment (other selector) is deleted and recreated once; three superseded role bindings are deleted | `migration_execute.go:78`, `:108`; `migration_plan.go:209` | Gone. Install deletes nothing outside its prune. An annotated earlier Deployment fails in the instance apply on its immutable selector; the user deletes it first | `TestDeleteCallSites` with one site fewer |
| 8 | Deletes of the migration carry the judged UID | `migration_execute.go:108` | Gone with the deletes | removed |
| 9 | Objects of older releases the module does not render are reported and left | `migration_report.go:11` | Gone. Install does not read objects it does not render | removed |
| 10 | Every refusing check runs before the first write | `plan_install.go:188` | Unchanged, one check fewer | existing `TestPlanInstall_RefusalsWriteNothing`, minus its proof row |
| 11 | A read the proof fails exits 4, 3 or 1 | `migration_plan.go:345`, `cmd/operator/install.go:365` | The proof's read is gone. The terminating wait keeps 4, 3 or 1; the guard keeps 2 | `TestPlanInstall_UnreadableObjectRefuses` (two rows instead of three), `TestPlanInstall_GuardReadFailureWithARecord` (read count 2, was 3) |
| 12 | A migration refusal exits 2; a stopped migration exits 1 with no re-run hint | `cmd/operator/install.go:353-363`, `:268` | Both error types are gone. Guard refusals exit 2 (`GuardError`); a failure after the CRD step gets the re-run hint | `TestInstallErrorMapping` minus three rows; `TestWithRerunHint` minus one case |
| 13 | The instance's own unrecorded objects pass (`Ours` is admitted) | `migration_plan.go:86` | The guard passes them when they carry an OPM managed-by label. Without that label they are refused (see the proposal's verdict table) | new `TestPlanInstall_OwnUnrecordedObjects` (two rows) |

For an install over objects that carry another identity, the case of the T9.31 report, the cli does exactly this after the change. `PlanInstall` reads the record `opm-operator` in `opm-operator-system`. For each rendered object that exists: listed in the record, it applies whatever its UUID label says; else with `opmodel.dev/adopt` equal to the operator instance's UUID, it applies and is recorded; else it is refused. Any refusal ends the command before its first write with exit 2 and:

```text
refusing to install: 5 object(s) cannot be applied by this instance:
  CustomResourceDefinition/moduleinstances.opmodel.dev belongs to module instance <other UUID>; to move it to this instance, remove it from module instance <other UUID>, then annotate it opmodel.dev/adopt=<instance UUID>
  ...
nothing was changed
```

An unlabelled CRD of a manifest install (no OPM managed-by label, no annotation) is refused the same way, also under `--crds-only`:

```text
refusing to install: 1 object(s) cannot be applied by this instance:
  CustomResourceDefinition/moduleinstances.opmodel.dev exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=<instance UUID>
nothing was changed
```

The cli adopts no unlabelled CRD by itself and offers no flag for it. The annotation stays the one override, as 0012:D8:R2 and 0012:D8:R3 say for every apply.

## Research & Decisions

### Remove the admission set, or keep it and pass nil

**Context**: `GuardInput.Admit` has one caller that fills it, the migration.
**Options considered**:
1. Keep the field, pass nil. No signature change; dead plumbing in three packages and one dead scenario set in `apply-pruning`.
2. Delete `AdmitSet`, `GuardInput.Admit` and `Request.Admit`.
**Decision**: 2.
**Rationale**: An override path nobody may use is a place where a later caller adds a bypass. The security rule is one override, the annotation.

### The instance's own unrecorded objects

**Context**: `Admit()` adds `Ours`, so the admission also lifted the foreign-object refusal for an object with this instance's UUID and no OPM managed-by label.
**Explored**: a throwaway test over `PlanInstall` with the admission emptied through `go test -overlay`; results in the proposal's table. Only five existing tests fail with the admission emptied, all of them migration tests (`TestInstall_MigratesAManifestInstall`, `TestPlanInstall_AdmitsProvenObjects`, `TestInstall_CRDsOnlyThenFullInstallMigrates`, `TestInstall_ResumesAfterAPartialInstanceApply`, `TestInstall_RerunAfterMigrationPrintsNothing`).
**Options considered**:
1. Accept the refusal. The object fails closed with the annotation as remedy.
2. Keep a small admission for "own UUID" in the cli. It keeps `Admit` alive in the library, against the owner's decision.
**Decision**: 1, accepted by the proposal gate of 2026-10-09 (ruling 1).
**Rationale**: OPM stamps the UUID label and the managed-by label in one apply, so the state needs a third party to rewrite one label of an unrecorded object. A UUID label alone is not proof that OPM manages the object.

### What replaces the proof-list refusals of other tools

**Decision**: nothing. Rule 5 and rule 6 of the table go without a replacement.
**Rationale**: they protected a takeover that no longer happens by itself. A user who annotates an object that Flux applies takes the same risk as with `opm instance apply` today.

## Error handling and exit codes

`opm operator install [--crds-only] [--version <selector>] [-f <file>] [--reset-values] [--timeout <duration>] [--rbac] [--skip-platform]`: no flag changes.

| Failure | Exit | Changes |
| --- | --- | --- |
| Guard refusal (foreign, other instance, adopted elsewhere, terminating, guard read failure) | 2 | no |
| Terminating wait read failure | 4, 3 or 1 by the read error | no |
| Migration refusal | was 2 | gone |
| Migration read failure | was 4, 3 or 1 | gone; that read is now the guard's, exit 2 |
| Stopped migration | was 1 | gone |

## Risks / Trade-offs

- A manifest-installed operator fails install at the guard and needs one annotation for each object, or a delete of the earlier install. Accepted by the owner: "Nobody runs OPM".
- An annotated earlier controller Deployment fails in the instance apply, after the CRD step, on its immutable selector. Install is idempotent; the user deletes that Deployment and runs install again. `docs/site/diagnostics/adopt-an-existing-object.md` says to delete the earlier Deployment first; nothing is built for it (ruling 3). Not unit-tested here: the fake cluster does not enforce immutability.
- The cli's e2e suite loses its only test that starts from a release manifest. No other e2e test reads the deleted fixture.
