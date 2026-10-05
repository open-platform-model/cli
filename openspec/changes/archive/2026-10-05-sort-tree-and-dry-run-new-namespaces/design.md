## Context

Verified against `origin/main` 5180cad1:

- `BuildTree` (`internal/kubernetes/tree.go:215`) groups `opts.InventoryLive` with `groupByComponent` and then sorts only component names (`sortedComponentNames`) and child nodes. `SortObjects` (`internal/kubernetes/sort.go`) is a stable in-place weight sort over unstructured objects, on `resourceorder.Sort`.
- `kubernetes.Apply` (`internal/kubernetes/apply.go`) sorts a copy of its input, applies stage 1 (CustomResourceDefinitions and Namespaces, `splitClusterDefinitions`), and on a dry run builds `newKinds` from the stage-1 outcomes whose pre-apply read returned NotFound (`stageOutcome.absentBefore`, `kindsOfNewCRDs`). `applyStage` skips objects whose group and kind are in that map, warns, and counts `Skipped`.
- `internal/workflow/apply.Execute` calls `EnsureNamespaceIfRequested` first (`apply.go:66`); on a dry run `Client.EnsureNamespace` reads the namespace, returns `true` without creating it when it is missing, and the workflow logs `namespace "<ns>" would be created`. `Execute` later records `budgetStart := time.Now()` and passes `EstablishDeadline: budgetStart.Add(timeout)` and `BudgetStart: budgetStart` to `kubernetes.Apply`. `FormatDryRunSummary` names only the CustomResourceDefinition reason.
- `kubernetes.Wait` reports `timed out after <time.Since(since) rounded to the second>`.
- `tests/integration/apply-staging` asserts the old limit: a dry run of CRD + Namespace + custom resource + ConfigMap reports one skip and exactly one error, the ConfigMap's namespace NotFound.
- `runInstanceVet` (`internal/cmd/instance/vet.go:64`) resolves the namespace with `config.ResolveKubernetes` (which reads `OPM_NAMESPACE`) and refuses an override after `AcquireInstanceFromDir`, so a command test needs the skip-unprovided fixture's dependencies (GHCR or the CUE cache), like `internal/workflow/render/skip_test.go`.
- `config.Resolve*` read the environment with `os.Getenv`, so an empty value and an unset variable behave alike.

## Goals / Non-Goals

**Goals:**

- `opm instance tree` lists each component's resources in the order the `mod-tree` spec promises.
- A dry run reports no error that the real apply would not hit because of a namespace the same apply creates, and says what it did not check.
- The three test gaps from the wave-1 reviews are closed without slowing the unit suite.

**Non-Goals:**

- A Namespace readiness wait in the real apply (the Namespace is usable as soon as it is created).
- Skipping anything else on a dry run.
- Touching the weight table (cli-e2e5 moves it to the library later; only the import changes then).
- New flags or exit codes.

## Research & Decisions

### 1. Tree order: weight, then name, then kind, API group and namespace

**Context**: the spec names weight then name. Two resources of equal weight can share a name (a ConfigMap and a Secret named `config`, or one name in two namespaces under the no-component group).
**Options considered**:
1. Sort the inventory when it is written - changes the stored record and the operator-shared format, and does nothing for records already written.
2. Sort in `groupByComponent` - local to the tree, works for every record.
**Decision**: option 2. Each group is first sorted by name, kind, API group and namespace (`sort.SliceStable`), then by weight with the stable `SortObjects(group, resourceorder.Ascending)`.
**Rationale**: the stable weight sort keeps the name order within a weight, and the extra keys make the order total, so two runs over the same inventory print the same tree.

```go
func groupByComponent(resources []*unstructured.Unstructured, componentMap map[string]string) map[string][]*unstructured.Unstructured {
	// ... grouping as today ...
	for _, group := range groups {
		sortByWeightThenName(group)
	}
	return groups
}

// sortByWeightThenName orders objs ascending by resource weight, then name,
// kind, API group and namespace.
func sortByWeightThenName(objs []*unstructured.Unstructured)
```

The `groupByComponent` doc comment says each group is in weight-then-name order. Child nodes (ReplicaSets, Pods) keep their current name sort.

### 2. The dry-run namespace skip lives beside the CRD skip

**Context**: `applyStage` already skips objects of a new kind; the new reason is a namespace instead of a kind.
**Options considered**:
1. Make `EnsureNamespace` create the namespace with a server-side dry run - the dry run persists nothing, so later objects still fail admission.
2. Record the new namespaces and skip in `applyStage` - same shape as the CRD skip.
**Decision**: option 2.

```go
type ApplyOptions struct {
	DryRun            bool
	EstablishDeadline time.Time
	BudgetStart       time.Time

	// NewNamespaces names namespaces a dry run treats as created by this
	// apply although no Namespace object of it says so: the instance
	// namespace that --create-namespace would create. Apply adds every
	// Namespace of its first stage whose pre-apply read returned NotFound.
	// Ignored outside a dry run.
	NewNamespaces []string
}
```

In `Apply`, the dry-run branch builds `newNamespaces map[string]struct{}` from `opts.NewNamespaces` plus every stage-1 outcome that is a core `Namespace` with `absentBefore`. `applyStage` gains that set; an object is skipped when its namespace (non-empty) is in it, after the CRD check, so a custom resource that has both reasons warns once, for its CustomResourceDefinition. The warning follows the CRD one:

```
skipping ConfigMap/app-config in demo: namespace demo is created by this apply, so a dry run cannot validate it
```

Both reasons count in `ApplyResult.Skipped`, whose doc names both. `FormatDryRunSummary` becomes:

```
dry run complete: 2 resources would be applied, 2 skipped (their CustomResourceDefinition or Namespace is created by this apply)
```

A Namespace whose own dry-run apply failed is not in `applied`, so its objects are sent and fail as before: the error is real.

### 3. The workflow passes the instance namespace when `--create-namespace` would create it

`EnsureNamespaceIfRequested` returns `(wouldCreate bool, err error)`: `true` only on a dry run in which `EnsureNamespace` reported the namespace missing. `Execute` passes `NewNamespaces: []string{namespace}` when it is true. Outside a dry run nothing changes (the namespace exists by then). Without `--create-namespace`, a missing instance namespace is not skipped: the real apply would fail on it too, and the dry run should say so.

### 4. A clock seam instead of a sleep

**Context**: the staging test needs the elapsed time reported by the CRD timeout to count from the apply start, and today buys that with a one-second sleep in the CRD patch reactor.
**Options considered**:
1. Keep the sleep and assert a parsed duration `>= 1s` - still slow and still wall-clock dependent.
2. A package clock in `internal/workflow/apply`: `var now = time.Now`, and `budgetStart := now()`. The test sets `now` to return the real time minus one hour and `Options.Timeout` to one hour plus a few hundred milliseconds, so the deadline is a few hundred milliseconds away and the error must read `timed out after 1h0m`.
**Decision**: option 2.
**Rationale**: without `BudgetStart` the wait would count from its own start and report `0s`, so the line the reviewer deleted is now guarded, and the test takes well under a second. `EstablishDeadline` stays guarded: without it the 5m default deadline applies and the reported `1h5m` fails the `1h0m` assertion (slowly).

### 5. The `OPM_NAMESPACE` command test

`internal/cmd/instance` gains a test that sets `t.Setenv("OPM_NAMESPACE", "staging")`, runs `runInstanceVet` on `internal/workflow/render/testdata/skip-unprovided/instance` with `clusterLookup{offline: true}`, a config whose `Registry` routes `opmodel.dev` to GHCR, and no `-n`, and asserts an `*opmexit.ExitError` with `ExitValidationError` (2) whose error names `OPM_NAMESPACE`, `"staging"` and the fixture's namespace. It skips, like `skipFixture`, when `OPM_SKIP_REGISTRY_TESTS` is set or the core schema cannot be fetched. It does not run in parallel (it sets the environment).

`resolver_test.go` replaces every `os.Setenv(k, v)` plus deferred `os.Unsetenv` with `t.Setenv(k, v)`, and every bare `os.Unsetenv(k)` with `t.Setenv(k, "")`, which the resolvers read as unset.

## Error handling and output

No new error. Exit codes unchanged: a dry run with skipped objects exits 0; a real error in a dry run still exits through `exitCodeFromK8sError` or is reported per resource as before.

## Risks / Trade-offs

- Skipped objects are not validated by the server. The warning says so per object and the summary counts them.
- cli#307 edits `internal/workflow/apply/apply.go`; the hunks here are kept to the call site, the `ApplyOptions` literal, the clock seam, `EnsureNamespaceIfRequested` and `FormatDryRunSummary`.
- `opm instance tree` JSON/YAML consumers that relied on inventory order see the spec's order.
