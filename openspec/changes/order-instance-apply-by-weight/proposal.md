## Why

`opm instance apply` and `opm module apply` send the rendered resources to the cluster in build order. `internal/kubernetes.Apply` (`apply.go:58-62`) says "Resources are assumed to be already ordered by weight (from RenderResult)", but nothing sorts them: `internal/workflow/apply/apply.go:146` passes `result.Resources` straight through, and the kernel emits build pair order. A module that ships a CustomResourceDefinition and a custom resource of it, or a Namespace and objects in it, can half-apply on the first run: the custom resource or namespaced object fails, the error skips prune and the inventory write, and a rerun succeeds. The `deploy` spec has promised ascending weight order (FR-D-003, User Story 1 scenario 3) all along.

The operator does not have the defect: it applies through Flux `ApplyAllStaged`, which applies cluster definitions first and waits for them, then class definitions, then the rest. The owner decided (2026-10-02 planning walkthrough) that the CLI stages by hand, without taking Flux into the CLI: sort by weight, CRDs and Namespaces first, wait for every CRD to be `Established`, then the rest. A server-side dry run cannot validate a custom resource whose CRD the same apply would create, so a dry run skips it with a warning. This is kind staging (facts of the Kubernetes API), not module ordering. The weight table stays in `pkg/resourceorder` for now; moving it to the library is separate, later work.

Three weight sorts already exist, each hand-written (`internal/kubernetes/delete.go:141`, `internal/operator/plan.go:53-63`, `internal/inventory/stale.go:109-114`). The CRD `Established` wait exists only in `internal/operator/wait.go`, and `internal/operator` imports `internal/kubernetes`, so `Apply` cannot reuse it in place.

## What Changes

- **Instance apply is staged.** `kubernetes.Apply` sorts a copy of its input ascending by `resourceorder.GetWeight` (stable, so build order breaks ties), applies the CustomResourceDefinitions and Namespaces first, waits for every CRD of that stage that applied to report `Established=True`, then applies the rest. A timeout fails the apply before the second stage, so neither prune nor the inventory write runs. Per-resource errors keep their current treatment: collected, the stage continues.
- **`--timeout` bounds the CRD wait from the start of the apply.** The CRD establish wait is charged to a `--timeout` budget (default 5m) that starts when the apply starts, and its timeout reports the time since then. The `--wait` readiness wait keeps a fresh full `--timeout`, unchanged.
- **A dry run skips what it cannot validate.** On `--dry-run`, a custom resource whose group and kind are served by a CRD that the same apply would create (the CRD's pre-apply read returned NotFound) is not sent; the command logs a warning naming the resource and its CRD and counts it as skipped in the dry-run summary. Nothing is waited on in a dry run. A namespaced object in a Namespace that the same apply creates still fails a dry run (the server's namespace admission rejects it); that stays a documented limit.
- **One weight sort helper.** `pkg/resourceorder` gains a generic stable ascending/descending sort by weight; `internal/kubernetes` wraps it for unstructured objects. Instance delete, stale prune, the operator install and uninstall plans, and the new apply staging use it.
- **The readiness wait moves.** `Wait`, `WaitAbsent`, the poll loop and the `ReadyPredicate`, `CRDEstablishedPredicate`, `HealthyPredicate` and `AbsentPredicate` move from `internal/operator` to `internal/kubernetes`; `internal/operator` and `internal/workflow/apply` call them there. No behavior change.
- **The false ordering comments are fixed.** `kubernetes.Apply`'s doc states the staging it now does. The `groupByComponent` note in `internal/kubernetes/tree.go`, which claims the inventory is stored in weight order, is rewritten to say the tree follows inventory order. The `sortResources` doc in `internal/output/manifest.go` stops naming a five-key apply order that does not exist.
- **`--timeout` help** on `instance apply` and `module apply` names the CRD wait and when its budget starts, and the command reference is regenerated.

Not in this change: deleting the weight table or moving it to the library; ClusterRoles or class definitions in staged waits (Flux stages them, but they have no readiness to wait for and weight order already puts them before what uses them); a Namespace readiness wait; skipping namespaced objects of a new Namespace on a dry run; sorting `opm instance tree` output explicitly; prune and delete protection of CRDs and Namespaces (the parallel change `protect-crds-and-namespaces-in-prune-and-delete`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: apply stages CRDs and Namespaces before everything else, waits for the CRDs to be established, applies the rest in ascending weight order, and a dry run skips a custom resource whose CRD the apply would create.
- `mod-apply`: `--timeout` also bounds the CRD establish wait, counted from the start of the apply.
- `pkg-resourceorder`: exports a stable weight sort used by every ordered apply and delete path.

## Impact

- **Release class: `fix(kubernetes)`, PATCH; ships as the next `1.0.0-beta.N`.** No flag, command or exit code is added or removed. A first apply that used to half-apply now succeeds; a dry run of a module carrying a new CRD and its custom resources now skips them with a warning where it used to report errors for them.
- Commands: `opm instance apply`, `opm module apply` (staging, CRD wait bound, dry-run skip); `opm instance delete`, `opm operator install|uninstall` and stale prune move to the shared helper unchanged in behavior.
- Packages: `pkg/resourceorder` (new `sort.go`), `internal/kubernetes` (`apply.go`, new `wait.go` and `sort.go`, `delete.go`, `tree.go` comment), `internal/operator` (`wait.go`, `plan.go`, `install.go`, `ready.go`), `internal/inventory` (`stale.go`), `internal/workflow/apply` (`apply.go`, `wait.go`), `internal/output` (`manifest.go` comment), `internal/cmd/instance` and `internal/cmd/module` (`--timeout` help), `docs/site/reference/cli/`, `tests/integration/`.
- Coordination: the parallel change `protect-crds-and-namespaces-in-prune-and-delete` edits `internal/inventory/stale.go`, `internal/kubernetes/delete.go` and `previewPrune` in `internal/workflow/apply/apply.go`, and adds `kubernetes.IsProtectedKind`. This change touches only the sort lines of the first two and the `kubernetes.Apply` call site of the third; this change is stacked on it and its partition uses `IsProtectedKind`.
- No enhancement decision backs this change, so it carries no `enhancement.yaml`.
