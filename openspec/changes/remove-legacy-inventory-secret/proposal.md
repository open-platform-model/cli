## Why

The cli still carries a read-and-migrate path for inventories that releases up to `v1.0.0-alpha.1` kept in a Secret named `opm.<name>.<id>`. Since `v1.0.0-alpha.2` every apply records the inventory in the instance's `ModuleInstance`, and an apply that finds no record looks for the Secret, uses it as the previous inventory and deletes it.

The path has two defects and no end date:

- It fails open. A Secret read that fails is logged as a warning and the apply goes on as a first install, so the recorded inventory is dropped.
- Closing that hole (the first attempt, branch `fix/deprecate-legacy-inventory-secret`) makes every first apply need `get` and `list` on Secrets in the namespace, for a Secret that almost no instance has.

The cli is in beta with one known user, and a removal after GA is a MAJOR change. The owner decided to remove the path now, in place of a deprecation.

## What Changes

- **BREAKING** `opm instance apply` and `opm module apply` no longer read, migrate or delete a legacy inventory Secret. No cli code makes a request on Secrets for inventory purposes, so an apply needs no permission on Secrets.
- `internal/inventory/legacy.go`, its tests, the `legacy` parameters of the apply workflow, and the integration program `tests/integration/migration` are deleted, with its lines in `Taskfile.yml` and `.github/workflows/pr.yml`.
- An instance whose only inventory is a legacy Secret has no record, so its next apply is a first install. Its resources carry the OPM managed-by label, so the first-install existence check lets them pass, as it does today for every resource OPM manages. The apply updates the rendered resources in place and records them. It deletes nothing and prunes nothing, and it leaves the Secret in place. A resource that only the Secret recorded, and that the module no longer renders, stays in the cluster untracked.
- New: a first install that finds rendered resources already in the cluster under OPM management prints one warning before it applies. The warning gives the count, says that nothing the render no longer produces is pruned, and names the release to apply with first when an old release installed the instance. This is the only signal such an instance gets, because the cli no longer reads the Secret.
- A new docs page, `docs/site/diagnostics/legacy-inventory-secret.md`, carries the migration note: apply every such instance once with `v1.0.0-beta.10`, the last release that migrates, then upgrade. It also says how to clean up by hand when that step was skipped.

Not changed: every path of an instance that has a `ModuleInstance` record; the fail-safe refusals of the record read, the existence check and the prune; the `ModuleInstance` record format; `opm operator install` and its manifest migration.

SemVer class: MAJOR after GA (a removed behaviour). Before GA it ships as the next `1.0.0-beta.N`, with a `!` in the PR title and the migration note in the PR body and the user docs.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `secret-inventory-migration`: the four migration requirements are removed; one requirement replaces them: the cli makes no request on Secrets for inventory purposes, and an instance recorded only in a legacy Secret is a first install.
- `instance-inventory`: the entry wire shape requirement loses its legacy Secret reader sentence and scenario.
- `apply-pruning`: a new requirement, the warning of a first install over resources OPM already manages.

## Impact

- Code: `internal/inventory/legacy.go` and `legacy_test.go` (deleted), `internal/inventory/stale.go`, `internal/workflow/apply/apply.go` and its tests.
- Tests and CI: `tests/integration/migration/main.go` (deleted), `Taskfile.yml` (`test:integration`), `.github/workflows/pr.yml` (integration job).
- Docs: `docs/site/diagnostics/legacy-inventory-secret.md` (new).
- Users: an instance last applied by `v1.0.0-alpha.1` or older must be applied once with `v1.0.0-beta.10` before the upgrade. RBAC: `get`, `list` and `delete` on Secrets are no longer used by apply.
- No dependency change, no flag change, no exit code change.
