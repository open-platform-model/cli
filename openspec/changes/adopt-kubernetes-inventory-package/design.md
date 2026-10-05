## Context

The library's `opm/k8s/inventory` (library PR 203, in `v1.0.0-beta.6`) was written from both frontends' copies, so for the cli this is mostly a move. Three things are not a plain move:

- the two stored digests change encoding, once and on purpose (0012:D6, 0012:D7);
- the stale set changes its relation from component-aware plus a rename filter to component-blind, which must not change a single prune decision (0012:D7:R1);
- the library `Entry` carries no struct tags, while the cli's `InventoryEntry` carried JSON tags that the legacy inventory Secret reader depends on.

State at the base of this change (`bc84595e`, cli PR 328 merged), re-checked by grep:

- `pkg/inventory/entry.go`: `NewEntryFromResource`, `IdentityEqual` (component-aware), `K8sIdentityEqual`, `K8sIdentity`, `IdentityOf`, `AdmitSet`, `ComputeStaleSet`, `ComputeDigest` (sorts, then hashes `json.Marshal` of the entries). `pkg/inventory/types.go`: `InventoryEntry` (JSON tags `group`, `kind`, `namespace`, `name`, `v`, `component`) and `Inventory` (`revision`, `digest`, `count`, `entries`).
- `internal/inventory/aliases.go` re-exports all of them into `internal/inventory`.
- `internal/inventory/digest.go`: `ComputeRenderDigest(objs []object.Exported)`, sorted by group, kind, namespace and name, hashing each object's CUE-export JSON, managed-by value included. Called once, in `internal/workflow/render/render.go` (`renderInstance`, after the single `object.Export`).
- `internal/inventory/stale.go`: `ApplyComponentRenameSafetyCheck`, called only from `internal/workflow/apply/apply.go` (`ComputeStaleInventorySet`, after `ComputeStaleSet`) and from `tests/integration/inventory-apply`.
- `internal/workflow/apply/apply.go` (`WriteInstanceRecord`): writes `status.inventory.digest` as `ComputeDigest(currentEntries)` and `status.lastAppliedRenderDigest` as `Result.RenderDigest`.
- `internal/inventory/legacy.go`: decodes the legacy Secret's JSON straight into `pkginventory.Inventory`, relying on its tags (`v` for the version).
- `internal/operator/migration_plan.go`: `objKey = inventory.K8sIdentity`, and `Admit()` builds an `inventory.AdmitSet`.
- Nothing in the cli reads back either stored digest to compare it. The operator does not reconcile a CLI-owned ModuleInstance.

## Goals / Non-Goals

**Goals:**
- No cli source declares an inventory entry type, a stale-set relation, an inventory digest or a render digest; all four are the library's.
- `pkg/inventory` is gone, and a lint rule stops it coming back.
- Every prune decision is unchanged, proven by tests over the apply's stale-set function.
- The CRD wire mapping and the legacy Secret reader read and write exactly the same JSON as before.
- The two stored digests are the library values, proven by tests that compare the stored value with the library function's result.

**Non-Goals:**
- The ownership verdict (`opm/k8s/ownership`, 0012:D8) and the deletion protocol (`opm/k8s/lifecycle`, 0012:D4). Later adoption changes replace `PreApplyExistenceCheck`, `AdmitSet` and `PruneStaleResources`.
- Any user-visible change other than the two stored values.

## Decisions

### KI1. The library type is used directly, with no alias

Every file that holds an entry imports `github.com/open-platform-model/library/opm/k8s/inventory` as `k8sinventory` (the cli's own package is also named `inventory`, and most callers import both). `internal/inventory` declares no alias for `Entry`: 0012 asks the frontends to use the tier types directly, and an alias would be a second name for the same thing.

Signatures move from `[]InventoryEntry` to `[]k8sinventory.Entry`: `DiscoverResourcesFromInventory`, `UnreadableEntry.Entry`, `PreApplyExistenceCheck`, `SplitProtected`, `PruneStaleResources`, `WriteInstanceRecord`, `CurrentInventoryEntries`, `ComputeStaleInventorySet`, `GuardEmptyRender`, `RunPreApplyExistenceCheck`, `previewPrune`, `logLeftBehind` and the query and status helpers.

### KI2. The cli keeps its record block, apply-guard key and wire mapping

These are cli concerns, not shared facts, so they move into `internal/inventory` instead of being deleted:

```go
// internal/inventory/record.go
// Inventory is the status.inventory block the CLI records on a ModuleInstance.
type Inventory struct {
    Revision int
    Digest   string
    Count    int
    Entries  []k8sinventory.Entry
}

// internal/inventory/admit.go
type K8sIdentity struct{ Group, Kind, Namespace, Name string }
func IdentityOf(e k8sinventory.Entry) K8sIdentity
type AdmitSet map[K8sIdentity]struct{}
func (s AdmitSet) Has(e k8sinventory.Entry) bool
```

`Inventory` loses its JSON tags: the CR path already maps it explicitly in `wire.go`, and nothing else marshals it. `K8sIdentity` stays a comparable struct because the library's identity key is unexported and `migration_plan.go` uses the key as a map key for unstructured objects too. It is the same four fields `inventory.SameObject` compares, and its doc comment says the ownership adoption deletes it; a test asserts `IdentityOf(a) == IdentityOf(b)` exactly when `SameObject(a, b)` over a table that varies each field. The ownership adoption later replaces `AdmitSet` with the tier's admission input.

`wire.go` maps `k8sinventory.Entry` field by field to the CRD names and back, exactly as now. Its round-trip tests stay as they are, with the type renamed.

### KI3. The legacy Secret is decoded through its own tagged struct

`legacy.go` declares the Secret's JSON shape privately:

```go
type legacyEntry struct {
    Group     string `json:"group"`
    Kind      string `json:"kind"`
    Namespace string `json:"namespace"`
    Name      string `json:"name"`
    Version   string `json:"v,omitempty"`
    Component string `json:"component,omitempty"`
}
type legacyInventory struct {
    Revision int          `json:"revision,omitempty"`
    Digest   string       `json:"digest,omitempty"`
    Count    int          `json:"count,omitempty"`
    Entries  []legacyEntry `json:"entries"`
}
```

and maps it to `Inventory` with library entries. The JSON keys are the ones `pkg/inventory/types.go` declares at the base of this change. `legacy_test.go` already decodes a payload whose second entry carries `"v":"v1"` and asserts the version, so it guards the mapping unchanged. The legacy digest read from the Secret is carried as read; it is replaced on the migrating apply's status write, as every apply replaces it.

### KI4. The stale set is `k8sinventory.StaleSet`, and the rename filter goes

`ComputeStaleInventorySet(prev, current)` becomes `k8sinventory.StaleSet(prev, current)`. The two rules give the same set: an entry the component-aware relation marks stale while a current entry has its group, kind, namespace and name is exactly an entry the rename filter then removes, and an entry with no such current entry stays in both. The library pins this in `TestStaleSet_EqualsTheCLIRuleWithItsRenameFilter`. The cli keeps its own guard at the call site it owns: the existing `internal/inventory/stale_test.go` cases (removed, renamed, first apply, idempotent, version change) and the rename-filter cases (rename filtered, genuine removal kept, mixed set, empty stale, same component) move to a table test over `ComputeStaleInventorySet` in `internal/workflow/apply`, each asserting the same expected set it asserts now; a filter case's stale input becomes `previous`. The one exception is the same-component case (stale and current both `[my-app/web]`, expecting the entry kept): that input cannot come from a stale set, and through the stale-set function the same pair is the idempotent case, so it expects empty. A mixed set with a component rename and an API version change on the same object is added.

The wrapper stays because the apply flow, the dry-run prune preview and their tests call it; it is one line.

### KI5. Both digests are the library's, and the stored value changes once

- **Render digest.** `renderInstance` computes `k8sinventory.RenderDigest(exported)` over the same single export. The export and the digest move into one small function, `exportAndDigest(resources []*object.Resource) ([]object.Exported, string, error)`, called with `object.Resources(out.Compiled)`, so a hermetic test can drive it with `object.Resource` values compiled from CUE source: its digest equals `k8sinventory.RenderDigest` of its export; two sets that differ only in the managed-by value (`opm-cli`, `opm-controller`) get the same digest (0012:D6:R2); a set that differs in another label gets a different one (0012:D6:R3). Errors keep the general exit code; a digest failure (an object whose JSON is not an object) is unreachable after a successful export, as before.
- **Inventory digest.** `WriteInstanceRecord` writes `k8sinventory.Digest(currentEntries)`. A test in `writerecord_test.go` captures the status patch and asserts `status.inventory.digest` equals `k8sinventory.Digest` of the written entries, and that the value differs from the digest the retired `ComputeDigest` gave for the same entries, recorded as a literal at the base of this change. The literal documents the one-time change, so a reviewer sees it in the test, not only in the release note.
- `internal/inventory/digest.go` and its tests are deleted. The goldens recorded by cli PR 328 pinned the old algorithm until this change; the library's own goldens (`TestDigest_EncodingIsTheOneDefined`, `TestRenderDigest_EncodingIsTheOneDefined`) pin the new encodings.

Alternative considered: keep the old values through a compatibility path. Rejected by 0012:D7: nothing in the cli compares a stored digest, so one change with a migration note costs less than two encodings.

### KI6. Sections

1. Digests from the library (the stored-value break, `feat!`). The entry type is still `pkg/inventory.InventoryEntry`, which has the library `Entry`'s fields in the same order, so `WriteInstanceRecord` converts each entry with a Go struct conversion (`k8sinventory.Entry(e)`, which ignores tags) for the one `Digest` call. That conversion is deleted in section 2.
2. Entry type, stale set and package deletion (the Go API break, `feat!`).

Each ends green. Section 1 alone is releasable: the stored digests are the library's, and the rest of the cli is unchanged.

### KI7. depguard

The `retired-kubernetes-copies` rule gains `github.com/open-platform-model/cli/pkg/inventory` with the message `use github.com/open-platform-model/library/opm/k8s/inventory`. Its comment names 0012:D7 beside 0012:D1/D5:R2. The rule is proven to fire the way cli PR 328 proved it: a stub package under the old path in a scratch copy outside the repo, never committed.

## Risks / Trade-offs

- [A user compares stored digests across the upgrade] → the migration note in the PR body and the release CHANGELOG names both fields, the cause and the rollback behaviour.
- [The operator's adoption lands in a different library release] → both frontends already pin `v1.0.0-beta.6`; the cli release that carries this change should be the one the operator's inventory adoption ships beside, so a mixed cluster never records two encodings of the render digest for long. Neither frontend compares the other's stored render digest today, so a gap is cosmetic.
- [A branch elsewhere still imports `pkg/inventory`] → it fails to compile on merge, and the depguard message names the replacement.
- [`cli-e4` and the lifecycle adoption touch the same files] → they replace `PreApplyExistenceCheck`, `AdmitSet` and `PruneStaleResources`; this change only renames their entry type, so whichever lands second rebuilds on the other.

## Migration Plan

The migration note in proposal.md goes in the PR body and the squash body, because the squash message setting is `BLANK` and a commit-body `BREAKING CHANGE` footer does not reach `main` otherwise; whoever merges the release PR carries it into `CHANGELOG.md` by hand. No cluster migration: the next apply records the new values.

Archive checklist (the archive rides the PR): `retire_capabilities: true` retires the emptied `public-inventory-package` spec, and the archiver rewrites the `pkg-types` Purpose to name `opm/k8s/inventory` beside the object, label and weight packages it already lists as the library's.

## Research & Decisions

### Keep `pkg/inventory` as a thin re-export of the library

**Context**: deleting a public package breaks Go importers.
**Explored**: importers of `github.com/open-platform-model/cli/pkg` in the workspace's opm-operator, library and opm-modules checkouts: none.
**Options considered**:
1. Re-export the library names from `pkg/inventory` - no break for importers; keeps a second import path for one definition.
2. Delete it - one import path; a break with no known importer.
**Decision**: option 2.
**Rationale**: 0012 asks the frontends to delete their copies, cli PR 328 deleted `pkg/core` and `pkg/resourceorder` the same way, and nobody imports it.

### Where the cli's record block lives

**Context**: `Inventory` (revision, digest, count, entries) was in `pkg/inventory` with JSON tags.
**Explored**: its users: `Record`, `LegacyInventory`, `StatusInput`, the CR wire mapping and the integration programs.
**Options considered**:
1. Keep it in a slimmed `pkg/inventory` - keeps the package alive for one cli-only struct.
2. Move it to `internal/inventory`, without tags - the CR wire mapping is explicit already; only the legacy reader needs tags, and it gets its own struct.
**Decision**: option 2.
**Rationale**: the block is the cli's record shape, not a shared contract, and the library package states that it owns no wire shape.
