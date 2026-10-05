## Why

The cli carries two packages whose contents now live in the library's Kubernetes tier (library ADR-011, library PR 196, released in `v1.0.0-beta.5` and pinned by the cli at `v1.0.0-beta.6`):

- `pkg/core`: the `Resource` wrapper over a rendered `cue.Value` with its instance, component and transformer provenance, `MarshalJSON` and `ToUnstructured`, and the OPM label vocabulary with `IsOPMManagedBy`. The library's `opm/k8s/object.Resource` and `opm/k8s/labels` are the same code, field for field and value for value; the library copied them from this package and the operator's identical copy.
- `pkg/resourceorder`: the kind-class weight table, `Sort` and `Direction`. The library's `opm/k8s/object` ported them unchanged from this package (`GetWeight` became `Weight`).

The owner decided that the frontends delete their copies in the release that adopts the tier: the Kubernetes facts OPM decides have one home, used by both frontends (0012:D1), and kind-class order has one definition, a single weight table in the tier's object package that both frontends apply and delete by (0012:D5:R1/R2). A change to the order is then made once, in the library, and reaches the cli through a library bump.

The render workflow also exports each compiled object from CUE twice: once for the render digest (`MarshalJSON` in `inventory.ComputeRenderDigest`) and once for the apply objects (`ToUnstructured`). The library's `object.Export` exports each object once and hands back the bytes and the decoded object together, so the digest and the apply objects read one export.

## What Changes

- **Labels.** Every importer of the `pkg/core` label constants and `IsOPMManagedBy` reads them from `opm/k8s/labels` (imported as `opmlabels`, because several call sites hold a local `labels` map), and `render.RuntimeName` becomes `opmlabels.ManagedByCLI`. The values are byte-equal, so nothing written to or read from a cluster changes; a test pins the literal values.
- **One export in the render.** `internal/workflow/render/render.go` builds `object.Resources(out.Compiled)`, calls `object.Export` once, computes the render digest from the exported JSON and takes the apply objects from the exported objects. `inventory.ComputeRenderDigest` takes `[]object.Exported`; its algorithm and its bytes are unchanged (sort by group, kind, namespace and name, hash each object's CUE-export JSON), and a golden test pins the digest value across the change. The `tests/integration/render-parity` program moves with it.
- **Order.** Apply (`internal/kubernetes/apply.go`), delete (`delete.go`), prune (`internal/inventory/stale.go`), the tree view (`tree.go`) and the build output (`internal/output/manifest.go`) order by `object.Sort` and `object.Weight`. A test holds every entry of the table `pkg/resourceorder` carries at the base of this change at the same weight in the library table, so apply, delete and output order do not move. The two `weights.go` files are identical at the base of this change apart from the `GetWeight` to `Weight` rename.
- **Deletion.** `pkg/core` and `pkg/resourceorder` are deleted with their tests. A `depguard` rule in `.golangci.yml` refuses both import paths, with a message naming the library package to use.
- **Specs.** `pkg-types` no longer lists the two packages. `resource-conversion` drops the `pkg/core` requirements and states the library export, label and order rules the cli follows instead. `pkg-resourceorder` is retired. `deploy` and `cmd-structure` name the library weight order where they named `pkg/resourceorder`. `instance-identity-labeling` names `opm/k8s/labels.ManagedByCLI` where it named `core.LabelManagedByValue`.

No user-visible change: every command, message, exit code, output order, apply and delete order, label and stored digest stays the same, except the wording of an export failure, which a successful kernel render cannot reach (design KO3).

**BREAKING** for Go importers of `github.com/open-platform-model/cli/pkg/core` and `github.com/open-platform-model/cli/pkg/resourceorder` (none known). SemVer class after GA: MAJOR (a public package is removed). Before GA it ships as the next `1.0.0-beta.N`. Release class of the PR: `feat!`.

## Migration note

BREAKING CHANGE: the cli no longer exports `pkg/core` or `pkg/resourceorder`. Importers move to the library's Kubernetes tier, `github.com/open-platform-model/library/opm/k8s/object` and `.../opm/k8s/labels`:

| Removed (cli) | Use (library) |
| --- | --- |
| `core.Resource` and its methods | `object.Resource` (same fields and methods); build it from the kernel output with `object.NewResource` or `object.Resources` |
| `r.MarshalJSON()` then `r.ToUnstructured()` on the same resource | `object.Export`, which exports once and returns both |
| `core.LabelManagedBy` | `labels.ManagedBy` |
| `core.LabelManagedByValue`, `LabelManagedByControllerValue`, `LabelManagedByLegacyValue` | `labels.ManagedByCLI`, `labels.ManagedByController`, `labels.ManagedByLegacy` |
| `core.LabelComponent`, `LabelComponentName` | `labels.Component`, `labels.ComponentName` |
| `core.LabelModuleInstanceName`, `LabelModuleInstanceNamespace`, `LabelModuleInstanceUUID` | `labels.ModuleInstanceName`, `labels.ModuleInstanceNamespace`, `labels.ModuleInstanceUUID` |
| `core.IsOPMManagedBy` | `labels.IsOPMManagedBy` |
| `resourceorder.GetWeight` | `object.Weight` |
| `resourceorder.Sort`, `Direction`, `Ascending`, `Descending`, the `Weight*` constants | `object.Sort`, `object.Direction`, `object.Ascending`, `object.Descending`, the same `Weight*` constants |

Every value and weight is unchanged.

## Not in this change

- **The duplicate-identity check.** It already uses `object.Duplicates` (cli PR 325).
- **The inventory entry, stale set and digest packages** (`pkg/inventory`, `internal/inventory/digest.go`). They move to `opm/k8s/inventory` in a later change, which also changes the stored digest once. This change keeps the digest byte-for-byte.
- **`object.Stages`.** The cli keeps its two-stage apply (cluster definitions, then everything else in weight order), which is already what the `deploy` spec states. The cli applies object by object, so cutting the second stage per weight would change nothing it submits.
- **Ownership, health and the deletion protocol** (`opm/k8s/ownership`, `opm/k8s/health`, the library deletion plan). Later adoption changes.
- **Any library change or library pin change.** The pin stays `v1.0.0-beta.6`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pkg-types`: `pkg/core` and `pkg/resourceorder` leave the exported package list; the Kubernetes object wrapper, the label vocabulary and the weight table are the library's.
- `resource-conversion`: the `pkg/core.Resource` requirements are removed. The cli converts a render through the library's single export, reads label keys from the library and orders objects by the library weight table.
- `pkg-resourceorder`: retired; its requirements move into `resource-conversion` as the library order rule.
- `deploy`: the two-stage apply orders by the library weight table.
- `cmd-structure`: `opm module build` output orders by the library weight.
- `instance-identity-labeling`: the runtime identity the CLI injects is the library's `opm/k8s/labels.ManagedByCLI`.

## Impact

- Code: delete `pkg/core/` and `pkg/resourceorder/`. Switch `internal/inventory/{cr,digest,legacy,stale,store}.go`, `internal/kubernetes/{apply,delete,sort,tree}.go`, `internal/operator/migration_proof.go` (the operator-install migration proof, cli PRs 307 and 309), `internal/platform/cluster.go`, `internal/output/manifest.go`, `internal/workflow/render/{env,render}.go`, `pkg/inventory/entry.go`, their tests (`internal/cmd/instance/delete_test.go`, `internal/inventory/{digest,legacy}_test.go`, `internal/kubernetes/delete_test.go`, `pkg/inventory/types_test.go`) and the seven `tests/integration/{deploy,inst-list,inst-tree,inventory-apply,inventory-ops,migration,render-parity}` programs. `internal/operator/{plan_install,install,uninstall}.go` import neither package at the base of this change.
- API: `pkg/core` and `pkg/resourceorder` are removed (breaking). `inventory.ComputeRenderDigest` and `kubernetes.SortObjects` change parameter types (`internal/`).
- Lint: `.golangci.yml` enables `depguard` with one rule.
- Dependencies: none. `k8s.io/apimachinery` stays a direct dependency through the remaining code.
- Archive: the `pkg-resourceorder` main spec is left with no requirement; `retire_capabilities: true` in the change's `.openspec.yaml` lets the archive retire it. The `resource-conversion` Purpose is rewritten at archive to describe the library conversion (design, Migration Plan).
- Release: the operator's adoption deletes its own `pkg/core` against the same library release, so both frontends drop their copies on one library version.
