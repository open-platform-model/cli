## Why

`opm operator install` carries a one-time migration of an opm-operator installed from an earlier release manifest (capability `operator-migration`, 0012:D8:R6 and 0012:D8:R7): a proof list, a read-only proof, field-ownership moves, a Deployment recreate, deletes of superseded role bindings, and an admission set that the migration hands to the ownership guard. Nobody runs OPM yet, so no manifest-installed operator exists to take over. The owner decided on 2026-10-05 that all legacy-migration code goes, and confirmed it on 2026-10-09 ("Remove both"): the cli removes its migration and stops passing `Admit` to the guard, then the library removes `Admit`. The enhancement withdraws 0012:D8:R6 and 0012:D8:R7.

This change was first built on 2026-10-05 on `refactor/retire-legacy-manifest-migration` (archived there as `2026-10-05-retire-legacy-manifest-migration`). Main has moved since: cli#347 and cli#349 put the ownership guard on every apply and every install, and the guard now takes the migration's result (`internal/inventory/guard.go:127`, fed by `internal/operator/plan_install.go:253` and `internal/operator/install.go:91`). Deleting the migration files is no longer enough. This proposal is the earlier one corrected for today's main.

The migration proof is also the reason an install over objects that carry another instance identity cannot be repaired: the proof refuses first, reads neither the record nor the adopt annotation, and prints "remove or rename these objects". With the proof gone the ownership guard decides, as it does for every other apply.

## What Changes

- **BREAKING** `opm operator install` no longer migrates an operator installed from an opm-operator release manifest. Its objects are judged by the ownership guard as any other existing object: refused with exit 2 unless the record lists them or they carry `opmodel.dev/adopt=<instance UUID>`.
- **BREAKING** The ownership guard takes no admission set. `inventory.GuardInput.Admit`, `inventory.AdmitSet` and `workflowapply.Request.Admit` are deleted. The cli no longer sets `ownership.ApplyInput.Admit` or `ownership.DeleteInput.Admit`; library v1.0.0-beta.8 still exports both fields and no library call requires them (they are plain `bool` fields with a `false` zero value).
- Install over objects that carry another identity (the instance's module path, name or namespace changed): the guard decides. Recorded objects apply; objects annotated with `opmodel.dev/adopt=<instance UUID>` apply; objects with neither refuse with exit 2 and a message that names the annotation and the UUID to set. The text "operator migration refused" and the remedy "remove or rename these objects" no longer exist.
- Install sends no delete and no managed-fields patch outside its instance apply. `opm operator install` keeps its exit code 2 for every guard refusal.
- Delete the migration: 11 files in `internal/operator`, the 6986-line proof-list data, the tool `hack/operator-legacy`, the e2e migration test and its 1761-line manifest fixture. Full list with `path:line` in `design.md`.
- Delete the old-style `--version` tag message (`operatorTagShape`, `looksLikeOperatorTag`, `operatorTagRefusal`): a `--version` shaped like an opm-operator release tag is refused as any other selector the registry cannot satisfy, exit 2, before any cluster call. The running-operator requirement no longer names a manifest-installed operator; the check itself does not change (proposal gate of 2026-10-09, ruling 2).
- Docs: the install help, `README.md`, `AGENTS.md` and `docs/site/diagnostics/adopt-an-existing-object.md` lose their migration text.

Release class: MAJOR after GA (a removed behaviour of `opm operator install`); during beta it ships as the next `-beta.N`. PR title `refactor(operator)!: retire the legacy-manifest migration`.

Not in this change: any rename; the library (it removes `Admit` and `installDeletable` in its own breaking beta, after this merges); `opm operator uninstall`; the earlier-identity hint line and docs page for install (the reduced T9.31, a later unit).

## Two verdicts change for an object that is not a legacy-manifest object

The migration's admission set holds more than proven manifest objects. `MigrationPlan.Admit()` (`internal/operator/migration_plan.go:86`) also adds `Ours`: every existing rendered object whose `module-instance.opmodel.dev/uuid` label equals the operator instance's UUID. The library lifts the foreign-object refusal for an admitted object (`opm/k8s/ownership/apply.go:109`, `:151`). So today an object passes the guard when all of these hold: the record does not list it, it carries the operator instance's UUID label, and its `app.kubernetes.io/managed-by` label is absent or names another tool. After this change that object is refused as "not managed by OPM", exit 2, naming the adopt annotation.

Measured on the fake cluster with a throwaway test (the eight rendered fixture objects, `PlanInstall`, admission set emptied through a build overlay; nothing committed):

| Existing objects | Record | Today | After |
| --- | --- | --- | --- |
| instance's own (OPM managed-by, own UUID) | none | pass | pass |
| instance's own | lists them | pass | pass |
| own UUID, managed-by label removed | none | pass | **refused, exit 2** |
| own UUID, managed-by `Helm` | none | pass | **refused, exit 2** |
| own UUID, managed-by label removed | lists them | pass | pass |
| OPM managed-by, no UUID label (proof-list objects: the CRDs, the Namespace, the Deployment and the like) | none | refused by the proof, exit 2 | **pass** |

The last row is the second changed verdict, in the permissive direction, found by the review of the build. The proof refused a proof-list object that carried an OPM managed-by label, or an instance name or namespace label, without a UUID label (`internal/operator/migration_proof.go:56-69`): it was neither the instance's own nor proven. The guard passes it (library `opm/k8s/ownership/apply.go:109-126`: OPM-managed, no UUID label, no annotation naming another instance), as it does for every `opm instance apply`; install then applies it, stamps it and records it. An object without a UUID label predates UUID stamping or lost the label; no OPM runtime writes one today. The gate accepted it on 2026-10-09 (second ruling): the cli has one ownership rule, the library's, for install as for every `opm instance apply`. The test `TestPlanInstall_OwnUnrecordedObjects` pins it.

No `opm instance apply`, `opm module apply`, prune or delete passes an admission set (only `internal/operator` sets `Admit`), so none of their verdicts change. An install over its own recorded objects does not change. An install over its own unrecorded objects does not change as long as they carry the labels OPM stamps (a stopped run, a CRDs-only install, a `kubectl apply` of the module's render). The changed case needs a third party to have removed or rewritten the managed-by label on an unrecorded object while it kept the UUID label. It fails closed, with a remedy the user can apply. The proposal gate of 2026-10-09 accepted it (ruling 1): keeping it would mean keeping an admission path in the cli and `Admit` in the library. The way out is the annotation the refusal prints.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `operator-migration`: retired (all 13 requirements REMOVED, `retire_capabilities: true`).
- `apply-pruning`: "Ownership guard on every apply and dry run" loses its admission set (REMOVED and ADDED as "Ownership guard judges every apply and dry run", since three scenarios go).
- `operator-lifecycle`: "Every check that can refuse install runs before its first write" drops the migration proof, the admission and the migration's writes, and gains the identity cases (REMOVED and ADDED as "Install runs every refusing check before its first write", since two scenarios go). "Another module version is installed from the registry as served" loses the old-tag message (restated as "Install resolves another module version as the registry serves it"), and "The running-operator check locates the operator by its fixed names" loses the manifest-install case (restated as "The running-operator check finds the operator by its fixed names alone").
- `inventory-ownership`: "Operator-owned delete delegates to the operator's finalizer" loses its manifest-install scenario (restated as "Operator-owned delete hands cleanup to the operator's finalizer").

## Impact

- Packages: `internal/operator`, `internal/cmd/operator`, `internal/inventory`, `internal/workflow/apply`, `internal/kubernetes` (two call-site allow-lists), `hack/`, `tests/e2e`.
- For whom it breaks: a cluster whose operator was applied from an opm-operator release manifest (`install.yaml`, operator releases up to v1.0.0-beta.8) and never installed as the module. The owner's two homelab installs are the only known ones. They annotate each object, or delete the earlier install first.
- Exit codes: a failed read of a rendered object that the terminating wait could still read now fails in the guard (exit 2) and no longer in the migration proof (exit 4, 3 or 1).
- Enhancement: no decision is claimed delivered. This change removes the implementation of two withdrawn requirements (0012:D8:R6, 0012:D8:R7); `enhancement.yaml` declares 0012 with an empty decision list.
- Follow-up outside this change: the library removes `ApplyInput.Admit`, `DeleteInput.Admit` and `installDeletable`; the cli then bumps its pin with no code change.
