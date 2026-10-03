## Why

`opm instance apply` and `opm module apply` send the rendered resources to the cluster in build order. `internal/kubernetes.Apply` (`apply.go:58-62`) says "Resources are assumed to be already ordered by weight (from RenderResult)", but nothing sorts them: `internal/workflow/apply/apply.go:146` passes `result.Resources` straight through, and the kernel emits build pair order. A module that ships a CustomResourceDefinition and a custom resource of it, or a Namespace and objects in it, can half-apply on the first run: the custom resource or namespaced object fails, the error skips prune and the inventory write, and a rerun succeeds. The `deploy` spec has promised ascending weight order (FR-D-003, User Story 1 scenario 3) all along.

The operator does not have the defect: it applies through Flux `ApplyAllStaged`, which applies cluster definitions first, waits for them, then applies the rest. The owner decided (2026-10-02 walkthrough, task a1) that the CLI stages the same way, by hand, without taking Flux into the CLI: sort by weight, CRDs and Namespaces first, wait for every CRD to be `Established`, then the rest. A server-side dry run cannot validate a custom resource whose CRD the same apply would create, so a dry run skips it with a warning. This is kind staging (facts of the Kubernetes API), not module ordering; the weight table stays in `pkg/resourceorder` until it moves to the library with that work (task e5).

Three weight sorts already exist, each hand-written (`internal/kubernetes/delete.go:141`, `internal/operator/plan.go:53-63`, `internal/inventory/stale.go:109-114`). The CRD `Established` wait exists only in `internal/operator/wait.go`, and `internal/operator` imports `internal/kubernetes`, so `Apply` cannot reuse it in place.

## What Changes

- **Instance apply is staged.** `kubernetes.Apply` sorts a copy of its input ascending by `resourceorder.GetWeight` (stable, so build order breaks ties), applies the CustomResourceDefinitions and Namespaces first, waits for every CRD of that stage that applied to report `Established=True`, then applies the rest. The wait is bounded by the command's `--timeout` (default 5m); a timeout fails the apply before the second stage, so neither prune nor the inventory write runs. Per-resource errors keep their current treatment: collected, the stage continues.
- **A dry run skips what it cannot validate.** On `--dry-run`, a custom resource whose group and kind are served by a CRD that the same apply would create (the CRD's dry-run reports `created`) is not sent; the command logs a warning naming the resource and its CRD and counts it as skipped in the dry-run summary. Nothing is waited on in a dry run.
- **One weight sort helper.** `pkg/resourceorder` gains a stable ascending/descending sort by weight; instance delete, stale prune, the operator install and uninstall plans, and the new apply staging use it.
- **The readiness wait moves.** `Wait`, `WaitAbsent`, the poll loop and the `ReadyPredicate`, `CRDEstablishedPredicate`, `HealthyPredicate` and `AbsentPredicate` move from `internal/operator` to `internal/kubernetes`; `internal/operator` and `internal/workflow/apply` call them there. No behavior change.
- **The false ordering comments are fixed.** `kubernetes.Apply`'s doc states the staging it now does. `groupByComponent` in `internal/kubernetes/tree.go` claims the inventory is stored in weight order, which it is not (entries follow render order); the tree sorts each component's resources by weight, then name, through the helper, as the `mod-tree` spec already requires.
- **`--timeout` help** on `instance apply` and `module apply` names the CRD wait, and the command reference is regenerated.

Not in this change: deleting the weight table or moving it to the library (task e5); ClusterRoles in the first stage (Flux stages them, but weight order already puts them before everything that binds them and they have no readiness to wait for); a Namespace readiness wait; prune and delete protection of CRDs and Namespaces (task i2, a parallel change).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: apply stages CRDs and Namespaces before everything else, waits for the CRDs to be established, applies the rest in ascending weight order, and a dry run skips a custom resource whose CRD the apply would create.
- `pkg-resourceorder`: exports a stable weight sort used by every ordered apply and delete path.
- `mod-tree`: the per-component weight order comes from an explicit sort, not from the inventory's stored order.

## Impact

- **Release class: `fix(kubernetes)`, PATCH; ships as the next `1.0.0-beta.N`.** No flag, command or exit code is added or removed. A first apply that used to half-apply now succeeds; a dry run of a module carrying a new CRD and its custom resources now exits 0 with a warning where it used to report errors for the custom resources.
- Commands: `opm instance apply`, `opm module apply` (staging and dry-run skip), `opm instance tree` (explicit sort); `opm instance delete`, `opm operator install|uninstall` and stale prune move to the shared helper unchanged in behavior.
- Packages: `pkg/resourceorder` (new `sort.go`), `internal/kubernetes` (`apply.go`, new `wait.go`, `delete.go`, `tree.go`), `internal/operator` (`wait.go`, `plan.go`, `install.go`, `ready.go`), `internal/inventory` (`stale.go`), `internal/workflow/apply` (`apply.go`, `wait.go`), `internal/cmd/instance` and `internal/cmd/module` (`--timeout` help), `docs/site/reference/cli/`.
- Coordination: the parallel change for task i2 edits `internal/inventory/stale.go`, `internal/kubernetes/delete.go` and `previewPrune` in `internal/workflow/apply/apply.go`. This change touches only the sort lines of the first two and the `kubernetes.Apply` call site of the third; expect a rebase on it before the PR.
- No enhancement decision backs this change, so it carries no `enhancement.yaml`.
