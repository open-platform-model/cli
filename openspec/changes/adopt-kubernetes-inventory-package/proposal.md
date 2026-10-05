## Why

The cli keeps its own copy of the inventory facts both Kubernetes frontends must agree on: the inventory entry and its identity helpers (`pkg/inventory`), a component-aware stale set with a second filter that rescues component renames (`internal/inventory/stale.go`), the inventory digest (`pkg/inventory.ComputeDigest`) and the render digest (`internal/inventory/digest.go`). The library's Kubernetes tier now holds one definition of each in `opm/k8s/inventory` (library PR 203, released in `v1.0.0-beta.6`, which the cli already pins): `Entry`, `NewEntry`, `SameObject`, `StaleSet`, `Digest` and `RenderDigest`.

The owner decided that the frontends delete their copies and use the tier (0012:D1), and settled the two inventory questions the copies disagreed on:

- **The stale set is component-blind** (0012:D7, closing 0012:OQ7). An entry is the same object as a rendered one when group, kind, namespace and name agree, whatever its component or API version. The cli's rename filter existed only to rescue objects its component-aware relation marked stale, so it goes, and no prune decision changes.
- **The inventory digest hashes a canonical field encoding** (0012:D7:R2/R3), not the JSON form of the entries, so a change to how a frontend serialises an entry cannot move it.
- **The render digest ignores the managed-by value** (0012:D6:R2/R3), so the cli and the operator digest one render to the same value although each stamps its own runtime name.

Both stored values change once in the release that first records them, with a migration note (0012:D7:R4). The operator makes the same change against the same library release.

## What Changes

- **Digests.** The render workflow computes the render digest with `inventory.RenderDigest` over the render's single export, and the apply writes `status.inventory.digest` with `inventory.Digest`. `internal/inventory/digest.go` (`ComputeRenderDigest`) and `pkg/inventory.ComputeDigest` are deleted. The digest stored in `status.inventory.digest` and `status.lastAppliedRenderDigest` changes once (see Migration note).
- **Stale set.** The apply computes the stale set with `inventory.StaleSet`. `ApplyComponentRenameSafetyCheck` and its call in `internal/workflow/apply/apply.go` (`ComputeStaleInventorySet`) are deleted. Prune decisions do not change: a component rename or an API version change leaves nothing stale, as before.
- **Entry type.** Every cli package holds the library's `inventory.Entry` and builds it with `inventory.NewEntry`. `pkg/inventory` is deleted with its tests, and `internal/inventory/aliases.go` with it. The record block (`Inventory`: revision, digest, count, entries), `K8sIdentity` and `AdmitSet` move into `internal/inventory`, which is the cli's own record and apply-guard code, not a shared contract.
- **Wire mapping kept.** `internal/inventory/wire.go` keeps the explicit mapping between an entry and the ModuleInstance CRD's `status.inventory.entries[]` fields (`group`, `kind`, `namespace`, `name`, `v`, `component`). The library `Entry` carries no struct tags, so the legacy inventory Secret reader decodes the Secret's JSON through its own tagged struct and maps it the same way.
- **Lint and docs.** The `depguard` rule from cli PR 328 also refuses `github.com/open-platform-model/cli/pkg/inventory`. ADR-009 and ADR-012 get an "Amended" status note recording that the stale set is component-blind and the rename filter is gone (0012:D7); the ADR-012 note also records that the inventory digest is the library's canonical encoding. `AGENTS.md` names `opm/k8s/inventory` beside the other tier packages.
- **Specs.** `pkg-types` no longer lists `pkg/inventory` and states that the inventory entry, stale set and digests are the library's; it keeps every sentence cli PRs 311 and 328 added. `public-inventory-package` is retired. `instance-inventory`, `apply-pruning`, `kernel-render` and `resource-conversion` state the library rules where they stated the cli's own.

User-visible behaviour: every command, message, exit code, apply, prune and delete decision stays the same. Two stored status values change once (Migration note). A debug line (`component rename detected, skipping prune`) is no longer logged, since no entry needs rescuing.

**BREAKING** for Go importers of `github.com/open-platform-model/cli/pkg/inventory` (none known) and for anything that compares the two stored digests across the upgrade. SemVer class after GA: MAJOR (a public package is removed and a stored value changes encoding). Before GA it ships as the next `1.0.0-beta.N`. Release class of the PR: `feat!`.

## Migration note

BREAKING CHANGE: the cli records its inventory and render digests with the library's shared encodings, and no longer exports `pkg/inventory`.

On the first CLI apply with this release (`opm instance apply`, `opm module apply` or `opm operator install`), a CLI-owned ModuleInstance's `status.inventory.digest` and `status.lastAppliedRenderDigest` change once, even when no object changed:

- `status.inventory.digest` hashes a canonical field-by-field encoding of the entries instead of their JSON form;
- `status.lastAppliedRenderDigest` hashes a versioned canonical re-encoding of each object (sorted keys) with the `app.kubernetes.io/managed-by` value left out. Once the operator release that adopts the same package is running, it equals the value the operator records for the same render.

The cli never compares either value, so nothing is re-applied and no object changes. A rollback writes the old encodings again. External comparers must use `opm/k8s/inventory` from library v1.0.0-beta.6 or later.

Go API. Importers of `pkg/inventory` move to `github.com/open-platform-model/library/opm/k8s/inventory`:

| Removed (cli) | Use (library) |
| --- | --- |
| `InventoryEntry`, `NewEntryFromResource` | `inventory.Entry` (no struct tags), `inventory.NewEntry` |
| `K8sIdentityEqual` | `inventory.SameObject` |
| `IdentityEqual` | none (component-aware identity retired, 0012:D7) |
| `ComputeStaleSet` and the rename filter | `inventory.StaleSet` |
| `ComputeDigest` | `inventory.Digest` |
| `Inventory`, `K8sIdentity`, `AdmitSet` | none (cli-internal) |

## Not in this change

- **The apply guard** (`PreApplyExistenceCheck`, `AdmitSet`) and the every-apply ownership verdict of 0012:D8. That is the ownership adoption (`opm/k8s/ownership`), a later change; this one only moves `AdmitSet` and `K8sIdentity` into `internal/inventory`.
- **The deletion protocol** (`PruneStaleResources`, the instance-delete walk). That is the lifecycle adoption, a later change.
- **Any library change or library pin change.** The pin stays `v1.0.0-beta.6`.
- **The CRD wire shape** of `status.inventory`. Unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pkg-types`: `pkg/inventory` leaves the exported package list; the inventory entry, stale set and digests are the library's `opm/k8s/inventory`.
- `public-inventory-package`: retired; the shared inventory contract is the library's.
- `instance-inventory`: entry identity is the library's component-blind `SameObject`; the record block holds library entries; the inventory digest and render digest are the library's, and change once.
- `apply-pruning`: the stale set is the library's component-blind `StaleSet`; the component-rename filter and its flow step are removed.
- `kernel-render`: the render digest is the library's `RenderDigest`, which ignores the managed-by value.
- `resource-conversion`: the render digest computed from the single export is the library's, not the cli's earlier algorithm.

## Impact

- Code: delete `pkg/inventory/`, `internal/inventory/{aliases,digest}.go` and `internal/inventory/digest_test.go`; delete `ApplyComponentRenameSafetyCheck` from `internal/inventory/stale.go`. Switch `internal/inventory/{discover,legacy,record,stale,store,wire}.go`, `internal/workflow/apply/apply.go`, `internal/workflow/render/{render,types}.go`, `internal/workflow/query/status.go`, `internal/cmd/instance/{status,tree,delete}.go`, `internal/operator/{migration_plan,names,plan_install}.go`, their tests, and the `tests/integration/{deploy,inst-list,inst-tree,inventory-apply,inventory-ops,migration,module-apply,render-parity}` programs.
- API: `pkg/inventory` is removed (breaking). `internal/` signatures change from `InventoryEntry` to the library `Entry`.
- Stored data: `status.inventory.digest` and `status.lastAppliedRenderDigest` change once (Migration note). No CRD change.
- Lint: the `depguard` rule gains one denied path.
- Dependencies: none new; `opm/k8s/inventory` is in the pinned library.
- Archive: `retire_capabilities: true` in `.openspec.yaml` retires the emptied `public-inventory-package` spec.
- Release: ship in the cli release that pins the same library as the operator's inventory adoption, so both frontends change their stored digests on one library version.
