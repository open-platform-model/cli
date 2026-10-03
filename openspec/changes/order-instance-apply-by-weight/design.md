## Context

See proposal.md for the defect. Verified against `origin/main` f10cb73f:

- `kubernetes.Apply` (`internal/kubernetes/apply.go:58-95`) loops its input in slice order, calls `ApplyOne` for each, collects a per-resource error and continues. Its only production caller is `internal/workflow/apply/apply.go:146`, which passes `result.Resources` unsorted; the integration programs under `tests/integration/` call it directly with hand-built objects.
- `ApplyOne` derives the GVR from the object (`GVRFromUnstructured`), with no RESTMapper, so there is no discovery cache to refresh after a CRD becomes established.
- On any per-resource error the workflow skips prune and the inventory write (`apply.go:170-174`); on a returned error it exits through `exitCodeFromK8sError` before either.
- `internal/operator/wait.go` holds `Wait`, `WaitAbsent`, `waitUntil`, `pollObjects` and the predicates; `internal/operator` imports `internal/kubernetes` (`GVRFromUnstructured`, `ApplyOne`, `EvaluateHealth`), so `internal/kubernetes` cannot import it back. `internal/workflow/apply/wait.go:34` already reuses `operator.Wait` with `operator.HealthyPredicate` for `--wait`.
- Weight sorts: `sortByWeightDescending` (`internal/kubernetes/delete.go:141`), `sortByWeightAscending`/`sortByWeightDescending` (`internal/operator/plan.go:53-63`, used by `InstallPlan`, `CRDsOnlyPlan`, `UninstallPlan` and `install.go:65`), and an inline `sort.SliceStable` over `InventoryEntry` (`internal/inventory/stale.go:109-114`). All stable, all on `resourceorder.GetWeight`. `internal/output/manifest.go:46` is a three-key display sort (weight, namespace, name) specified by `cmd-structure`; it is not an apply or delete order and stays as it is.
- The operator's stage definition (Flux `ssa` v0.77.0, `utils.IsClusterDefinition`): CRDs, Namespaces and ClusterRoles, applied and then waited on with `WaitForSet`.

## Goals / Non-Goals

**Goals:**

- A module carrying a CRD and its custom resources, or a Namespace and objects in it, applies in one run.
- A dry run of such a module completes without false errors and says what it could not check.
- One weight sort in `pkg/resourceorder`; one readiness wait in `internal/kubernetes`.
- Every comment that claims the input is already weight-ordered is either made true or removed.

**Non-Goals:**

- Flux `ssa` or controller-runtime in the CLI (owner decision, task a1).
- Moving the weight table to the library or deleting the CLI copy (task e5); module-declared ordering of any kind.
- Changing what prune and instance delete remove (task i2).
- New flags, new exit codes.

## Decisions

### 1. Staging lives in `kubernetes.Apply`

`Apply` keeps its signature and gains one option:

```go
type ApplyOptions struct {
	DryRun bool

	// EstablishTimeout bounds the wait for the first stage's
	// CustomResourceDefinitions to report Established=True. Zero means
	// defaultEstablishTimeout (5m, the commands' --timeout default).
	EstablishTimeout time.Duration
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

1. Copy the input and sort the copy with `resourceorder.SortObjects(objs, resourceorder.Ascending)`. The caller's slice is not reordered (the inventory entries were computed from it already, and the digest is order-independent anyway).
2. Partition, keeping order: stage 1 holds every object whose `isClusterDefinition` is true (kind `CustomResourceDefinition` in group `apiextensions.k8s.io`, or kind `Namespace` in the core group); stage 2 holds the rest. Because CRDs weigh -100 and Namespaces 0, stage 1 is CRDs then Namespaces.
3. Apply stage 1 with the existing per-resource loop (`applyStage`, factored out of today's body): log each line, collect errors, continue.
4. Not a dry run: collect the stage-1 CRDs that applied without error. If any, log `waiting for N CustomResourceDefinition(s) to be established` and call `Wait(ctx', client, crds, CRDEstablishedPredicate, start)` under `context.WithTimeout(ctx, EstablishTimeout)`. A wait error returns `(result, fmt.Errorf("waiting for CustomResourceDefinitions to be established: %w", err))` before stage 2.
5. Dry run: record the group/kind pairs served by every stage-1 CRD whose dry-run status was `created` (`spec.group`, `spec.names.kind`; a CRD missing either is ignored). No wait.
6. Apply stage 2 with the same loop, except that on a dry run an object whose `(group, kind)` is in the recorded set is not sent: log a warning `skipping <Kind>/<name>[ in <ns>]: its CustomResourceDefinition <crd-name> is created by this apply, so a dry run cannot validate it`, increment `Skipped`, continue.

A first stage with errors still proceeds to the wait (for the CRDs that applied) and to stage 2. That keeps today's "apply what can be applied, report every failure" behavior; a custom resource whose CRD failed fails on its own and is reported, and the workflow then skips prune and the inventory write as before.

The doc comment of `Apply` states this staging and drops the "already ordered by weight (from RenderResult)" sentence.

### 2. The CLI stage is CRDs and Namespaces, not ClusterRoles

**Context**: Flux's first stage also holds ClusterRoles.
**Options considered**:
1. Match Flux exactly (CRDs, Namespaces, ClusterRoles) - identical stage sets, but a ClusterRole has no readiness, and weight order (5) already puts it before every RoleBinding, ClusterRoleBinding and workload that refers to it, so the extra stage changes nothing observable.
2. CRDs and Namespaces only - what the owner's decision names; the two kinds whose absence makes a later object's request fail.
**Decision**: option 2.
**Rationale**: the owner's decision line names CRDs and Namespaces; with weight sorting inside each stage the two definitions produce the same apply sequence for every kind the weight table knows. Recorded here so a later reader comparing with the operator does not take the difference for an oversight.

### 3. Dry-run policy: skip with a warning

Owner decision (task a1). The alternatives were failing (a dry run that can never pass for a module shipping its own CRD) and sending anyway (a guaranteed `no matches for kind` error that reads like a real defect). A skipped object is not an error: it does not add to `Errors`, and a dry run with only skips exits 0. The summary line becomes `dry run complete: N resources would be applied` followed, when `Skipped > 0`, by `, M skipped (CustomResourceDefinition created by this apply)`.

Only a CRD that does not yet exist on the cluster triggers the skip. A CRD that exists and is reconfigured (a new served version, say) is not detected; a custom resource of a version the live CRD does not serve yet fails the dry run as it does today. Detecting that needs a CRD diff and is not part of the decision.

### 4. The wait moves to `internal/kubernetes/wait.go`

Moved unchanged, with exported names kept: `ReadyPredicate`, `CRDEstablishedPredicate`, `HealthyPredicate`, `AbsentPredicate`, `Wait`, `WaitAbsent`; unexported `waitMode`, `waitUntil`, `pollObjects`, `describeObjects`, and `describeObjectList` (exported as `DescribeObjects` returning `[]string`, because `internal/operator/ready.go` still uses it). The poll interval becomes the exported variable `WaitPollInterval` (doc: tests shorten it), since `internal/operator` tests set it today and will set it across the package boundary.

`internal/operator` keeps what is operator-specific: `DefaultPredicate`, `WorkloadReadyPredicate`, `pendingObjects`, `CheckReady`, `NotReadyError`, and the kind constants. Its callers switch to `kubernetes.Wait`, `kubernetes.WaitAbsent`, `kubernetes.CRDEstablishedPredicate`. `internal/workflow/apply/wait.go` switches to `kubernetes.Wait` and `kubernetes.HealthyPredicate` and drops its `internal/operator` import. The wait tests move with the code.

### 5. The sort helper lives in `pkg/resourceorder`

```go
// Direction selects ascending (apply) or descending (delete) weight order.
type Direction int

const (
	Ascending Direction = iota
	Descending
)

// Sort orders items in place by GetWeight of each item's GVK. The sort is
// stable: items of equal weight keep their relative order.
func Sort[T any](items []T, gvkOf func(T) schema.GroupVersionKind, dir Direction)

// SortObjects is Sort over unstructured objects.
func SortObjects(objs []*unstructured.Unstructured, dir Direction)
```

`unstructured` is part of `k8s.io/apimachinery`, so the package's "only apimachinery" rule holds. Callers: `kubernetes.Apply` (ascending), `kubernetes.Delete` (`delete.go`, descending, replacing `sortByWeightDescending`), `operator.InstallPlan`/`CRDsOnlyPlan`/`install.go` (ascending) and `UninstallPlan` (descending), `inventory.PruneStaleResources` (`Sort` over `InventoryEntry`, descending), and the tree (decision 6). The unexported sort functions are deleted. The helper moves to the library with the weight table under task e5.

### 6. The tree sorts explicitly

`groupByComponent`'s note says the inventory is stored weight-ascending, so input order already is weight order. It is not: `CurrentInventoryEntries` follows render order, and this change does not reorder the caller's slice. The `mod-tree` spec requires weight-then-name order within a component. The tree therefore sorts each group with a two-key sort (`GetWeight`, then name), through `resourceorder.Sort` applied after a name sort (stability gives weight-then-name), and the note is replaced by a one-line statement of what the function does. Reading the owner's "fix the false comments" as "make the code match the spec the comment defended", not "rewrite the comment to admit the spec is unmet".

### 7. `--timeout` covers the CRD wait

`instance apply` and `module apply` already resolve `--timeout` (`inventory.ResolveTimeout`, default 5m). The workflow passes the resolved value as `ApplyOptions.EstablishTimeout`. The flag help on both commands becomes `Bound on the CustomResourceDefinition establish wait, the --wait readiness wait and the operator-reconcile wait (operator-managed instances)`, and `task docs:reference` regenerates the reference pages. The two waits use separate budgets of the same length: the CRD wait runs before the inventory write, `--wait` after it, and the user sees two logged waits.

## Errors

| Situation | Surfaced as | Exit |
| --- | --- | --- |
| A stage-1 or stage-2 object fails to apply | per-resource warning, `N resource(s) had errors`, prune and inventory write skipped (unchanged) | 1 |
| CRD not established within `--timeout` | `apply failed` with `waiting for CustomResourceDefinitions to be established: timed out after 5m0s waiting for CustomResourceDefinition/foos.example.com to become ready`; stage 2 not applied, no prune, no inventory write | 1 (via `exitCodeFromK8sError`) |
| A stage-1 CRD disappears during the wait | `... CustomResourceDefinition/foos.example.com was applied and has since disappeared` | 1 |
| Dry run, custom resource of a CRD new in this apply | warning line, counted as skipped | 0 |

Example dry run of a module carrying `foos.example.com` and one `Foo` (resource lines illustrative; their format is `output.FormatResourceLine`, unchanged):

```text
dry run - no changes will be made
applying 3 resources
  customresourcedefinition/foos.example.com created
  namespace/demo created
  skipping Foo/my-foo in demo: its CustomResourceDefinition foos.example.com is created by this apply, so a dry run cannot validate it
dry run complete: 2 resources would be applied, 1 skipped (CustomResourceDefinition created by this apply)
```

## Testing

- `pkg/resourceorder/sort_test.go`: ascending and descending, stability on equal weights, `Sort` over a non-unstructured type.
- `internal/kubernetes/apply_test.go` (fake dynamic client with reactors): input in reverse order is patched CRD, Namespace, then the rest by weight; the caller's slice is unchanged; a CRD whose live object never reports `Established` times out under a short `EstablishTimeout` with stage 2 never patched; a CRD that reports `Established` lets stage 2 run; a dry run with a new CRD skips its custom resource with `Skipped == 1` and no error; a dry run with an existing CRD sends the custom resource.
- `internal/kubernetes/wait_test.go`: the moved operator wait tests.
- `internal/kubernetes/tree_test.go`: a component whose input is out of weight order prints weight-then-name.
- Integration: a new program `tests/integration/apply-staging/main.go` on `kind-opm-dev`, wired into `task test:integration`: one `kubernetes.Apply` of a CRD, a Namespace, a custom resource and a ConfigMap in the Namespace, in reverse order, succeeds on the first call; a dry run of the same set against a clean cluster reports the custom resource skipped; cleanup deletes all four.
