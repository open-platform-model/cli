## Context

See proposal.md for the defect. Verified against `origin/main` f10cb73f:

- `kubernetes.Apply` (`internal/kubernetes/apply.go:58-95`) loops its input in slice order, calls `ApplyOne` for each, collects a per-resource error and continues. Its only production caller is `internal/workflow/apply/apply.go:146`, which passes `result.Resources` unsorted; the integration programs under `tests/integration/` call it directly with hand-built objects.
- `ApplyOne` derives the GVR from the object (`GVRFromUnstructured`), with no RESTMapper, so there is no discovery cache to refresh after a CRD becomes established. Its pre-apply GET sets status `created` on any GET error, not only NotFound.
- On any per-resource error the workflow skips prune and the inventory write (`apply.go:170-174`); on a returned error it exits through `exitCodeFromK8sError` before either.
- `internal/operator/wait.go` holds `Wait`, `WaitAbsent`, `waitUntil`, `pollObjects` and the predicates; `internal/operator` imports `internal/kubernetes` (`GVRFromUnstructured`, `ApplyOne`, `EvaluateHealth`), so `internal/kubernetes` cannot import it back. `internal/workflow/apply/wait.go:34` already reuses `operator.Wait` with `operator.HealthyPredicate` for `--wait`, under a fresh `--timeout`.
- Weight sorts: `sortByWeightDescending` (`internal/kubernetes/delete.go:141`), `sortByWeightAscending`/`sortByWeightDescending` (`internal/operator/plan.go:53-63`, used by `InstallPlan`, `CRDsOnlyPlan`, `UninstallPlan` and `install.go:65`), and an inline `sort.SliceStable` over `InventoryEntry` (`internal/inventory/stale.go:109-114`). All stable, all on `resourceorder.GetWeight`. `internal/output/manifest.go:46` is a three-key display sort (weight, namespace, name) specified by `cmd-structure`; it is not an apply or delete order and stays as it is, but its doc names a "5-key apply order" that does not exist.
- The operator's staging (Flux `ssa` v0.77.0, `ApplyAllStaged`, `manager_apply.go:347-401`): first the cluster definitions (`utils.IsClusterDefinition`: CRDs, Namespaces, ClusterRoles), applied and waited on with `WaitForSet`; then the class definitions (`utils.IsClassDefinition`: any kind ending in `Class`), applied and waited on; then optional custom stages; then everything else.
- `pkg/resourceorder` imports only `k8s.io/apimachinery/pkg/runtime/schema`; its main spec ("No CLI dependencies") requires a dependency tree of only the standard library and `k8s.io/apimachinery`. `k8s.io/apimachinery/pkg/apis/meta/v1/unstructured` would pull klog, structured-merge-diff, kube-openapi and more into it.
- The tree (`internal/kubernetes/tree.go:259-262`) keeps each component's resources in input order, with a note claiming the inventory is stored weight-ascending. It is not: `CurrentInventoryEntries` follows render order.

## Goals / Non-Goals

**Goals:**

- A module carrying a CRD and its custom resources, or a Namespace and objects in it, applies in one run.
- A dry run of a module carrying a new CRD does not report false errors for that CRD's custom resources, and says what it did not check.
- One weight sort in `pkg/resourceorder`; one readiness wait in `internal/kubernetes`.
- Every comment that claims an apply order which does not exist is made true or rewritten.

**Non-Goals:**

- Flux `ssa` or controller-runtime in the CLI (owner decision).
- Moving the weight table to the library or deleting the CLI copy; module-declared ordering of any kind.
- Changing what prune and instance delete remove (the parallel change `protect-crds-and-namespaces-in-prune-and-delete`).
- A dry run of a namespaced object in a Namespace the same apply creates. The server-side dry run of that object goes through `NamespaceLifecycle` admission and fails with `namespaces "<ns>" not found`; this change does not skip it.
- Sorting `opm instance tree` output explicitly. The `mod-tree` main spec asks for weight-then-name order; the tree follows inventory order. Only the false comment is fixed here.
- New flags, new exit codes.

## Research & Decisions

### 1. Staging lives in `kubernetes.Apply`

**Context**: the fix needs a sort, a partition, a wait and a dry-run skip around today's per-resource loop.
**Explored**: the one production caller (`internal/workflow/apply`) and the integration programs, which call `kubernetes.Apply` directly.
**Options considered**:
1. Stage in `internal/workflow/apply` and call `kubernetes.Apply` twice - keeps `Apply` dumb, but every other caller of `Apply` keeps the defect and the dry-run skip needs the stage-1 results.
2. Stage inside `kubernetes.Apply` - every caller gets it; the signature stays.
**Decision**: option 2.
**Rationale**: the ordering is a fact about the Kubernetes API, not about the workflow, and the integration programs exercise it for free.

`Apply` keeps its signature and gains one option and one result field:

```go
type ApplyOptions struct {
	DryRun bool

	// EstablishDeadline bounds the wait for the first stage's
	// CustomResourceDefinitions to report Established=True. Zero means
	// defaultEstablishTimeout (5m) from the start of the wait.
	EstablishDeadline time.Time
}

type ApplyResult struct {
	Applied, Created, Configured, Unchanged int

	// Skipped counts the custom resources a dry run did not send because
	// the same apply would create their CustomResourceDefinition.
	Skipped int

	Errors []resourceError
}
```

Algorithm:

1. Copy the input and sort the copy with `SortObjects(objs, resourceorder.Ascending)` (decision 5). The caller's slice is not reordered (the inventory entries were computed from it already, and the digest is order-independent anyway).
2. Partition, keeping order: stage 1 holds every object whose kind is protected (kind `CustomResourceDefinition` in group `apiextensions.k8s.io`, or kind `Namespace` in the core group); stage 2 holds the rest. Because CRDs weigh -100 and Namespaces 0, stage 1 is CRDs then Namespaces. The partition uses `IsProtectedKind(gvk.Group, gvk.Kind)` from the sibling change `protect-crds-and-namespaces-in-prune-and-delete` (same kind set) rather than repeating it.
3. Apply stage 1 with the existing per-resource loop (`applyStage`, factored out of today's body): log each line, collect errors, continue.
4. Not a dry run: collect the stage-1 CRDs that applied without error. If any, log `waiting for N CustomResourceDefinition(s) to be established` and call `Wait(ctx', client, crds, CRDEstablishedPredicate, start)` under `context.WithDeadline(ctx, deadline)`. A wait error returns `(result, fmt.Errorf("waiting for CustomResourceDefinitions to be established: %w", err))` before stage 2.
5. Dry run: record the group/kind pairs served by every stage-1 CRD whose pre-apply GET returned NotFound (`spec.group`, `spec.names.kind`; a CRD missing either is ignored). A GET refused for any other reason (RBAC, say) does not mark the CRD new, so its custom resources are sent and fail or pass on their own. No wait.
6. Apply stage 2 with the same loop, except that on a dry run an object whose `(group, kind)` is in the recorded set is not sent: log a warning `skipping <Kind>/<name>[ in <ns>]: its CustomResourceDefinition <crd-name> is created by this apply, so a dry run cannot validate it`, increment `Skipped`, continue.

A first stage with errors still proceeds to the wait (for the CRDs that applied) and to stage 2. That keeps today's "apply what can be applied, report every failure" behavior; a custom resource whose CRD failed fails on its own and is reported, and the workflow then skips prune and the inventory write as before.

The doc comment of `Apply` states this staging and drops the "already ordered by weight (from RenderResult)" sentence.

### 2. The CLI stages CRDs and Namespaces, with no ClusterRole or class stage

**Context**: Flux's first stage also holds ClusterRoles, and Flux has a second waited stage for class definitions (kinds ending in `Class`).
**Explored**: Flux `ssa` v0.77.0 `ApplyAllStaged` and `utils/is.go`.
**Options considered**:
1. Match Flux exactly (CRDs, Namespaces, ClusterRoles, then a class stage) - identical stage sets, but ClusterRoles and classes (StorageClass, IngressClass, PriorityClass) have no readiness condition, so their waits return at once; weight order already puts a ClusterRole (5) before every binding and workload that refers to it and a StorageClass (20) before every claim; and an object referring to a class that does not exist yet is accepted by the API server and resolved later by its controller.
2. CRDs and Namespaces only - what the owner's decision names; the two kinds whose absence makes a later object's request fail.
**Decision**: option 2.
**Rationale**: the owner's decision names CRDs and Namespaces; with weight sorting inside each stage the two definitions produce the same apply sequence for every kind the weight table knows. Recorded here so a later reader comparing with the operator does not take the difference for an oversight.

### 3. Dry-run policy: skip a custom resource of a new CRD, with a warning

**Context**: a server-side dry run of a custom resource whose CRD does not exist yet always fails with `no matches for kind`.
**Explored**: the dry-run path of `ApplyOne`; `NamespaceLifecycle` admission for the Namespace case.
**Options considered**:
1. Fail - a dry run that can never pass for a module shipping its own CRD.
2. Send anyway - a guaranteed error that reads like a real defect.
3. Skip with a warning - the owner's decision.
**Decision**: option 3. A skipped object is not an error: it does not add to `Errors`, and a dry run with only skips exits 0. The summary line becomes `dry run complete: N resources would be applied` followed, when `Skipped > 0`, by `, M skipped (CustomResourceDefinition created by this apply)`; it is built by `FormatDryRunSummary(*kubernetes.ApplyResult)` next to `FormatApplySummary`.
**Rationale**: the decision covers only custom resources whose CRD is new.

Limits, recorded and not fixed here:

- Only a CRD that does not yet exist on the cluster triggers the skip. A CRD that exists and is reconfigured (a new served version, say) is not detected; a custom resource of a version the live CRD does not serve yet fails the dry run as it does today. Detecting that needs a CRD diff.
- A namespaced object in a Namespace that the same apply creates fails the dry run with `namespaces "<ns>" not found`: the dry run of the Namespace persists nothing, and the object's own dry run goes through namespace admission. Extending the skip to these objects is outside the owner's decision; `--create-namespace` covers the instance namespace on a real apply; a dry run, which creates nothing, hits the same NotFound for it.

### 4. The wait moves to `internal/kubernetes/wait.go`

**Context**: `internal/operator` imports `internal/kubernetes`, so `Apply` cannot call `operator.Wait`.
**Explored**: every caller of the wait (`internal/operator/install.go`, `ready.go`, `internal/workflow/apply/wait.go`) and the tests that shorten the poll interval.
**Options considered**:
1. A second, smaller CRD wait in `internal/kubernetes` - two poll loops to keep in step.
2. Move the shared wait down into `internal/kubernetes` - one loop, the operator keeps only its own predicates.
**Decision**: option 2.
**Rationale**: the loop knows nothing operator-specific; it reads objects through `GVRFromUnstructured`, which already lives in `internal/kubernetes`.

Moved unchanged, with exported names kept: `ReadyPredicate`, `CRDEstablishedPredicate`, `HealthyPredicate`, `AbsentPredicate`, `Wait`, `WaitAbsent`; unexported `waitMode`, `waitUntil`, `pollObjects`, `describeObjects`, and `describeObjectList` (exported as `DescribeObjectList` returning `[]string`, because `internal/operator/ready.go` still uses it). Only the `conditionEstablished` constant moves; `conditionStatusTrue` already exists in `internal/kubernetes/health.go`. The poll interval becomes the exported variable `WaitPollInterval` (doc: tests shorten it), since `internal/operator` tests set it today and will set it across the package boundary.

`internal/operator` keeps what is operator-specific: `DefaultPredicate`, `WorkloadReadyPredicate`, `pendingObjects`, `CheckReady`, `NotReadyError`, and the kind constants. Its callers switch to `kubernetes.Wait`, `kubernetes.WaitAbsent`, `kubernetes.CRDEstablishedPredicate`. `internal/workflow/apply/wait.go` switches to `kubernetes.Wait` and `kubernetes.HealthyPredicate` and drops its `internal/operator` import. The wait tests move with the code.

### 5. The sort helper is generic in `pkg/resourceorder`, wrapped for unstructured in `internal/kubernetes`

**Context**: four call sites sort by weight, over two element types (`*unstructured.Unstructured` and `inventory.InventoryEntry`).
**Explored**: `go list -deps ./pkg/resourceorder` (only `k8s.io/apimachinery/pkg/runtime/schema` today) and the main spec's dependency rule.
**Options considered**:
1. `Sort[T]` plus `SortObjects([]*unstructured.Unstructured)` in `pkg/resourceorder` - one place, but `unstructured` pulls klog, structured-merge-diff and more into a package whose spec allows only the standard library and apimachinery's light core.
2. `Sort[T]` in `pkg/resourceorder`; a one-line `SortObjects` wrapper in `internal/kubernetes` - the package keeps its dependency tree.
**Decision**: option 2.
**Rationale**: the "No CLI dependencies" scenario reads its dependency tree; the wrapper costs one line.

```go
// pkg/resourceorder

// Direction selects ascending (apply) or descending (delete) weight order.
type Direction int

const (
	Ascending Direction = iota
	Descending
)

// Sort orders items in place by GetWeight of each item's GVK. The sort is
// stable: items of equal weight keep their relative order.
func Sort[T any](items []T, gvkOf func(T) schema.GroupVersionKind, dir Direction)

// internal/kubernetes

// SortObjects is resourceorder.Sort over unstructured objects.
func SortObjects(objs []*unstructured.Unstructured, dir resourceorder.Direction)
```

Callers: `kubernetes.Apply` (ascending), `kubernetes.Delete` (`delete.go`, descending, replacing `sortByWeightDescending`), `operator.InstallPlan`/`CRDsOnlyPlan`/`install.go` (ascending) and `UninstallPlan` (descending), `inventory.PruneStaleResources` (`resourceorder.Sort` over `InventoryEntry`, descending). The unexported sort functions are deleted. The helper moves to the library with the weight table later.

### 6. The tree's comment is rewritten, its order is not changed

**Context**: `groupByComponent`'s note says the inventory is stored weight-ascending, so input order already is weight order. It is not.
**Options considered**:
1. Sort each component's resources by weight then name, as the `mod-tree` spec asks - fixes a behavior gap, but the owner's decision is to fix the false comments, not to change the tree.
2. Rewrite the note to say what the function does: the group keeps input order, which is inventory (render) order.
**Decision**: option 2.
**Rationale**: stays inside the owner's decision. The gap against the `mod-tree` spec is recorded in Non-Goals and in the supervisor report for the owner to schedule.

### 7. The CRD wait shares the apply's `--timeout` budget; `--wait` keeps a fresh one

**Context**: `instance apply` and `module apply` resolve `--timeout` (`inventory.ResolveTimeout`, default 5m) for the `--wait` readiness wait. The CRD establish wait now needs a bound too.
**Explored**: `internal/operator/install.go:66-70` ("One budget ... the user reasons about a single --timeout per command") and the `operator-lifecycle` spec, which charges every wait to one budget; `kubernetes.Wait`, whose `since` argument is the start of the budget in force, so a timeout reports the time spent against it.
**Options considered**:
1. A full `--timeout` for each wait - one command can run for twice `--timeout`.
2. One budget per command for both waits - `--wait` then gets only what the apply left over, a user-visible change to `--wait` that no decision asked for.
3. The CRD wait is charged to a budget that starts with the apply; `--wait` keeps a fresh full `--timeout`, as before this change.
**Decision**: option 3 (supervisor triage of the review, 2026-10-03).
**Rationale**: the CRD wait is part of the apply, so it is bounded from the apply's start; `--wait` behaves exactly as it did. `Execute` records `budgetStart` before the apply and passes `EstablishDeadline: budgetStart + timeout` and `BudgetStart: budgetStart`; `waitEstablished` passes `BudgetStart` to `Wait` as `since`, so its timeout reports the time since the apply started. `waitForHealthy` takes the resolved `--timeout` and waits under a deadline of its own start plus that timeout, reporting from its own start. The applies, prune and inventory write are not cancelled by either bound.

The flag help on both commands becomes `Bound on the CustomResourceDefinition establish wait (counted from the start of the apply), on the --wait readiness wait, and on the operator-reconcile wait (operator-managed instances)`, and `task docs:reference` regenerates the reference pages. The `mod-apply` flag table follows.

## Errors

| Situation | Surfaced as | Exit |
| --- | --- | --- |
| A stage-1 or stage-2 object fails to apply | per-resource warning, `N resource(s) had errors`, prune and inventory write skipped (unchanged) | 1 |
| CRD not established within `--timeout` | `apply failed` with `waiting for CustomResourceDefinitions to be established: timed out after 5m0s waiting for CustomResourceDefinition/foos.example.com to become ready`; stage 2 not applied, no prune, no inventory write | 1 (via `exitCodeFromK8sError`) |
| A stage-1 CRD disappears during the wait | `... CustomResourceDefinition/foos.example.com was applied and has since disappeared` | 1 |
| Dry run, custom resource of a CRD new in this apply | warning line, counted as skipped | 0 |
| Dry run, namespaced object in a Namespace new in this apply | per-resource error `namespaces "<ns>" not found` (unchanged) | 1 |

Example dry run of a module carrying `foos.example.com` and one cluster-scoped `Foo` (resource lines illustrative; their format is `output.FormatResourceLine`, unchanged):

```text
dry run - no changes will be made
applying 2 resources
  customresourcedefinition/foos.example.com created
  skipping Foo/my-foo: its CustomResourceDefinition foos.example.com is created by this apply, so a dry run cannot validate it
dry run complete: 1 resources would be applied, 1 skipped (CustomResourceDefinition created by this apply)
```

## Testing

- `pkg/resourceorder/sort_test.go`: ascending and descending, stability on equal weights, `Sort` over a non-unstructured type.
- `internal/kubernetes/apply_test.go` (fake dynamic client with reactors): input in reverse order is patched CRD, Namespace, then the rest by weight; the caller's slice is unchanged; a CRD whose live object never reports `Established` times out under a near `EstablishDeadline` with stage 2 never patched; a CRD that reports `Established` lets stage 2 run; a dry run with a new CRD skips its custom resource with `Skipped == 1` and no error; a dry run with an existing CRD sends the custom resource.
- `internal/kubernetes/wait_test.go`: the moved operator wait tests.
- `internal/workflow/apply`: `FormatDryRunSummary` table test; an `Execute` test where the CRD never reports `Established` under a short `--timeout`: `ExitGeneralError`, no delete actions, no ModuleInstance create or patch.
- Integration: a new program `tests/integration/apply-staging/main.go` on `kind-opm-dev`, wired into `task test:integration`: a dry run of a CRD, a Namespace, a custom resource and a ConfigMap in the Namespace, in reverse order, against a clean cluster reports the custom resource skipped and exactly one error, the ConfigMap's namespace NotFound (the documented limit); one real `kubernetes.Apply` of the same set then succeeds on the first call; cleanup deletes all four through the dynamic client and waits for them to be gone with `WaitAbsent`.
