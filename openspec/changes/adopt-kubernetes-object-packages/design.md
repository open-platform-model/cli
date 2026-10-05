## Context

The library's Kubernetes tier (`opm/k8s/object`, `opm/k8s/labels`, library PR 196) was built from the cli's `pkg/core` and `pkg/resourceorder`, so the cli's adoption is a move, not a redesign. Two things need care: the apply, delete and output order must not move, and the stored render digest must not change in this change (the inventory adoption changes it once, on purpose, later).

Importers at the base of this change (`be1157d5`), re-checked by grep for both import paths:

- `pkg/core` labels: `internal/inventory/{cr,legacy,stale,store}.go`, `internal/kubernetes/delete.go`, `internal/operator/migration_proof.go`, `internal/platform/cluster.go`, `pkg/inventory/entry.go`; tests `internal/cmd/instance/delete_test.go`, `internal/inventory/legacy_test.go`, `internal/kubernetes/delete_test.go`, `pkg/inventory/types_test.go`; programs `tests/integration/{deploy,inst-list,inst-tree,inventory-apply,inventory-ops,migration}`.
- `pkg/core.Resource`: `internal/workflow/render/render.go`, `internal/inventory/digest.go`, `internal/inventory/digest_test.go`, `tests/integration/render-parity`.
- `pkg/resourceorder`: `internal/kubernetes/{sort,apply,delete,tree}.go`, `internal/kubernetes/delete_test.go`, `internal/inventory/stale.go`, `internal/output/manifest.go`.

`internal/operator/plan.go` and `install.go` (the operator install from cli PRs 307 and 309) import neither package; the install path reaches labels only through `migration_proof.go`. The duplicate check already uses `object.Duplicates` (cli PR 325).

## Goals / Non-Goals

**Goals:**
- No cli source imports `pkg/core` or `pkg/resourceorder`, and both packages are gone.
- Apply, delete, prune, tree and build output order are unchanged, proven by a test against the retired table.
- The render digest is byte-identical, proven by a golden value recorded before the change.
- The render exports each object from CUE once.

**Non-Goals:**
- Adopting `opm/k8s/inventory`, `ownership`, `health` or `object.Stages` (see proposal, "Not in this change").
- Any user-visible change.

## Decisions

### KO1. The order proof is a literal table, written while both tables exist

A test in `internal/kubernetes` lists every constant, every GVK entry and every kind entry of `pkg/resourceorder/weights.go` at `be1157d5` as literals, plus the two fallbacks (an unknown GVK weighs `WeightDefault`, an unknown version of a known kind weighs the kind). Each GVK entry is asserted through its exact GVK; each kind entry through a GVK whose group and version miss the GVK table (`fallback.invalid/v9`), since every kind entry also has an exact GVK row and the canonical GVK would never reach the kind table. In section 1 it asserts both `resourceorder.GetWeight` and `object.Weight` against the literals, which proves the literals are the cli's table and every entry of it keeps its weight in the library's table. The other direction (a row the library adds) is the library's own `TestWeightTableGuard`; at the base of this change the two `weights.go` files differ only in the `GetWeight` to `Weight` rename. A second test sorts one mixed, shuffled set with `resourceorder.Sort` and `object.Sort`, ascending and descending, and requires identical results, which also covers stability on equal weights. When section 5 deletes `pkg/resourceorder`, the `resourceorder` side of both tests goes and the literals stay, so the cli keeps a guard that the library table did not move under it. The library already pins the same values in its own `TestWeightTableGuard`; the cli test pins what the cli applies by, independently of a library edit.

### KO2. The render digest keeps its algorithm and takes the exported set

`ComputeRenderDigest(objs []object.Exported)` sorts a copy by group, kind, namespace and name, stable, and hashes each `Exported.JSON` in that order. The sort key reads the decoded object the way the old CUE accessors read the value: the group is `GetAPIVersion()` up to its last `/` (the old `parseAPIVersion`), and kind, namespace and name are `GetKind()`, `GetNamespace()` and `GetName()`. It does not use `Object.GroupVersionKind()`, whose `schema.ParseGroupVersion` returns an empty group-version-kind, kind included, for an `apiVersion` with more than one `/`. Both the old accessors and the new key read "" for a missing field. `object.Export` produces `JSON` with the same `Resource.MarshalJSON` the old digest called, so the bytes hashed and their order are the same; a digest test with a multi-slash `apiVersion` pins that case. Section 1 records the digest of the existing three-object test set as a literal at the base of this change; that literal must stay green through section 3. The digest changes once, later, when the cli adopts `inventory.RenderDigest` together with the operator.

Alternative: adopt `inventory.RenderDigest` now. Rejected: it changes every stored digest, which belongs to the inventory adoption, where the operator changes its digest in the same library release.

### KO3. One export, error path worded once

`renderInstance` calls `object.Export(object.Resources(out.Compiled))` once. A failure returns `ExitGeneralError` wrapping the `*object.ExportError`, which names the resource, its index and the failed step. Before, the digest's failure said `render digest: ...` and the conversion's said `converting resource <Kind>/<Name> to unstructured: ...`. Neither is reachable after a successful kernel render (the kernel refuses a non-concrete object first), so the wording change is on a path no user sees; the new message is `converting rendered resources: <ExportError>`. Result objects are `Exported.Object`, in render order, as before.

### KO4. Label import alias `opmlabels`, and a literal guard for the values

Deleting `pkg/core` also deletes `TestLabelConstants`, the cli's only check of the label values. Section 1 adds `TestLabelValuesMatchRetiredCopy`, which asserts the nine literal keys and values against both `pkgcore` and `opmlabels`, and `IsOPMManagedBy` for the three OPM values and one other; section 5 keeps the `opmlabels` side. `render.RuntimeName` becomes `opmlabels.ManagedByCLI`, so the CLI keeps no copy of the managed-by value either.

Import alias:

Several call sites hold a local variable named `labels` (`stale.go`, `delete.go`, `migration_proof.go`, test helpers), and `internal/kubernetes/pods.go` imports `k8s.io/apimachinery/pkg/labels`. Every file imports the library package as `opmlabels`, so the name reads the same everywhere and never shadows.

### KO5. `SortObjects` stays as the unstructured adapter

`kubernetes.SortObjects(objs, dir object.Direction)` keeps its name and becomes `object.Sort(objs, (*unstructured.Unstructured).GroupVersionKind, dir)`. Its three callers (`apply.go`, `delete.go`, `tree.go`) and one test keep one short call; `stale.go` calls `object.Sort` directly, as it called `resourceorder.Sort`. The display sort in `internal/output/manifest.go` keeps its three keys (weight, namespace, name) and reads the weight from `object.Weight`.

### KO6. depguard guards the deleted paths

`.golangci.yml` enables `depguard` with one rule over all files that denies `github.com/open-platform-model/cli/pkg/core` and `github.com/open-platform-model/cli/pkg/resourceorder`, each with a message naming `opm/k8s/object` or `opm/k8s/labels`. Once the packages are gone an import fails to compile anyway; the rule stops a local copy from coming back under the old path. The rule has no allow list, so depguard denies nothing else; section 5 checks that by running the full lint, and checks the rule fires by linting a throwaway stub package under the old path in the scratch area (never committed).

### KO7. Sections

1. Pins (tests only). 2. Labels. 3. One export. 4. Order. 5. Deletion, lint rule, specs (the `feat!` commit). Each ends green; sections 1 to 4 change no exported API, so `main` stays releasable after any of them.

## Risks / Trade-offs

- [The library table is edited in a later library release] → the cli literal test fails on the bump PR, so an order change is a reviewed edit in both repos.
- [Other sessions' branches still import the old paths] → none exist on origin at the base of this change; a branch that appears later fails to compile on merge and moves its imports by the migration table.
- [The golden digest pins a value, not the operator's] → the operator parity is the inventory adoption's job; this change only proves the cli digest did not move.

## Migration Plan

The migration table from the proposal goes in the PR body (cli AGENTS.md, beta promise): the squash message setting is `BLANK`, so a commit-body `BREAKING CHANGE` footer does not reach `main`, and whoever merges the release PR carries the note into `CHANGELOG.md` by hand. No cluster or stored-data migration.

Archive checklist (the archive rides the PR): `retire_capabilities: true` retires the emptied `pkg-resourceorder` spec, and the archiver rewrites the `resource-conversion` Purpose, which still describes `pkg/core.Resource`, to say the CLI converts a render through the library's `opm/k8s/object` single export, reads OPM label keys from `opm/k8s/labels` and orders objects by the library weight table.

## Research & Decisions

### Adopt `object.Stages` for the apply

**Context**: the library offers `Stages`, which cuts an apply set into the cluster-definition stage and one stage per weight.
**Explored**: `internal/kubernetes/apply.go` (`splitClusterDefinitions`, `applyStage`), library `opm/k8s/object/stages.go`.
**Options considered**:
1. Use `Stages` - one more library call; the cli applies object by object in sorted order, so per-weight stages change nothing it submits, and the CRD wait already hangs off the first stage.
2. Keep the cli's two stages over `object.Sort` - no behaviour change, smallest diff.
**Decision**: option 2.
**Rationale**: `Stages` exists for an engine that re-sorts within a call (the operator's Flux apply). The cli has no such engine.

### Adopt `inventory.RenderDigest` now

**Context**: the library also offers the shared render digest.
**Explored**: library `opm/k8s/inventory/render_digest.go` (a new encoding with its own tag line and the managed-by value blanked).
**Options considered**:
1. Adopt now - every stored `lastAppliedRenderDigest` changes in this release.
2. Keep the cli algorithm over the exported set - no stored value moves.
**Decision**: option 2.
**Rationale**: the stored digest changes once, in the inventory adoption, released against the same library version as the operator's.
