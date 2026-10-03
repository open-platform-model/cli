## Context

Three CLI paths delete cluster objects for an instance: stale prune after apply (`inventory.PruneStaleResources`, called from `internal/workflow/apply/apply.go`), its dry-run preview (`previewPrune`), and CLI-owned instance delete (`kubernetes.Delete`, called from `internal/cmd/instance/delete.go`). The operator-owned delete path deletes only the `ModuleInstance` and lets the operator's finalizer prune, so it is out of scope here.

The operator's `Prune` (`opm-operator/internal/apply/prune.go`) is the reference this change follows. It never deletes a Namespace or a CRD, and before each delete it GETs the live object and skips it when the object is not OPM-managed or carries a different instance UUID.

## Goals / Non-Goals

**Goals:**

- One predicate decides which kinds the CLI never deletes, used by prune, preview and delete.
- Every object the CLI declines to delete is listed as `left behind`, in the dry run and the real run.
- Instance delete deletes an object only after re-reading it and confirming it still belongs to the instance.
- The first-install refusal stops naming a flag that does not bypass it.

**Non-Goals:**

- An override flag. The owner decided there is none.
- A live ownership re-check in prune, and shared `CanApply`/`CanDelete` verdicts or an adopt annotation. Those belong to the later ownership change that serves both frontends.
- Keeping left-behind prune entries in the recorded inventory. A stale Namespace already drops out today; CRDs follow the same rule.
- Changing the operator.
- Discovery-time read errors on instance delete. `inventory.DiscoverResourcesFromInventory` drops an entry whose GET fails with anything but NotFound (debug log only), so that entry never reaches the in-loop re-read and the `ModuleInstance` is still deleted. That gap predates this change and belongs to the later delete protocol that serves both frontends; the re-read error rule below covers only errors raised inside `Delete`.

## Decisions

### 1. `kubernetes.IsProtectedKind` in `internal/kubernetes`

```go
// IsProtectedKind reports whether the CLI never deletes objects of this kind:
// a Namespace takes everything inside it, and a CustomResourceDefinition takes
// every custom resource of its kind cluster-wide.
func IsProtectedKind(group, kind string) bool {
	switch {
	case group == "" && kind == "Namespace":
		return true
	case group == "apiextensions.k8s.io" && kind == "CustomResourceDefinition":
		return true
	}
	return false
}
```

It lives in `internal/kubernetes` because `internal/inventory` imports `internal/kubernetes`, and `kubernetes.Delete` needs the predicate, so `inventory` cannot host it without an import cycle. It matches group and kind, unlike the operator's kind-only switch, so an unrelated `Namespace` kind in another API group is not protected by accident.

### 2. Prune and preview split the stale set once

`inventory.SplitProtected(stale []InventoryEntry) (prunable, protected []InventoryEntry)` partitions by `IsProtectedKind`, keeping order. `Execute` in `internal/workflow/apply` calls it after `ComputeStaleInventorySet`:

- Real apply: `PruneStaleResources` gets `prunable`. When `protected` is non-empty, the instance logger prints `leaving N resource(s) behind` at warn level and one `output.FormatResourceLine(kind, ns, name, output.StatusLeftBehind)` line per entry.
- Dry run: `previewPrune` prints the `would prune` block for `prunable` (unchanged wording) and then a `would leave N resource(s) behind` block with the same left-behind lines.
- `--no-prune`: nothing is listed, since nothing would be pruned.

`PruneStaleResources` keeps its signature (the integration programs call it) and keeps skipping protected kinds itself, through the same predicate, so a caller that does not split cannot delete one.

`output.StatusLeftBehind = "left behind"` is added beside `StatusDeleted`, with a warn-tone (yellow) style in the status colour map and the `!` icon in `statusIcon`, so a left-behind line is never rendered with a blank icon.

### 3. Instance delete re-reads, checks ownership, and lists what it leaves

`DeleteOptions` gains `InstanceUUID string`, the instance's recorded `status.instanceUUID`, which `executeInstanceDelete` passes from `inv.InstanceUUID`. `DeleteResult` gains `LeftBehind []LeftBehindResource` (`Kind`, `Namespace`, `Name`, `Reason`).

Per object, in the existing reverse-weight order, on both the dry run and the real run:

1. `IsProtectedKind(gvk.Group, gvk.Kind)`: left behind, reason `CRDs and Namespaces are never deleted`.
2. GET the live object. NotFound: already gone, logged at debug, neither deleted nor an error. Any other error: a per-resource error, as a failed delete is today (the `ModuleInstance` is then kept, so a re-run retries).
3. `pkgcore.IsOPMManagedBy(labels[LabelManagedBy])` false: left behind, reason `no longer managed by OPM`.
4. `opts.InstanceUUID != "" && liveUUID != "" && liveUUID != opts.InstanceUUID`: left behind, reason `owned by another instance`.
5. Otherwise delete (real run) or print the existing dry-run line, and count it.
   a. A NotFound from the delete call itself (the object vanished between the GET and the DELETE) is "already gone", as in the operator's prune: neither deleted nor an error.

Step 4 copies the operator's tolerances: an object with no UUID label predates UUID stamping and passes on the managed-by check alone, and an instance with no recorded UUID falls back to the managed-by check. This is the reading of the owner's "like the operator" for the UUID match. A stricter rule (refuse when either side is empty) would leave legacy objects behind on every delete, which the operator does not do.

The re-read is done inside the loop rather than relying on the objects `ResolveInventory` fetched, because earlier foreground deletes in the same loop (and, without `--force`, the time the user takes at the prompt before discovery returns) sit between that read and the delete. The delete itself is unchanged (foreground propagation, by name).

`executeInstanceDelete` prints each left-behind object as `output.FormatResourceLine(kind, ns, name, StatusLeftBehind)` with its reason as a key-value field. Left-behind objects are not errors: the exit code is 0 and the `ModuleInstance` is deleted last as today. The progress line `all resources have been deleted` is printed only when nothing was left behind and nothing failed, and a real run with a per-resource failure prints no closing checkmark at all, since the `ModuleInstance` is kept for a re-run. The closing line becomes `Instance deleted — N resource(s) left behind` when N > 0, followed by `output.Details` naming `kubectl delete` as the way to remove them once nothing needs them. The dry run closes with `dry run complete: N resources would be deleted, M left behind`.

### 4. First-install refusal wording

`PreApplyExistenceCheck`'s untracked-object message becomes:

```text
resource <Kind>/<name> in namespace "<ns>" already exists and is not managed by OPM — remove or rename it, or change the module to render a different name
```

No flag is named. The guard's scope (first apply only, non-NotFound GET errors ignored) is unchanged here; it moves to the later ownership change.

## Research & Decisions

### Skip or allow an override on delete

**Context**: Deleting a Namespace or a CRD is unrecoverable and reaches past the instance; the review asked whether delete should skip with a warning or take an override flag.
**Explored**: kernel plan research for this task (re-checked against `origin/main` f10cb73f): the operator never deletes either kind, on prune or on instance delete.
**Options considered**:
1. Always skip, list as left behind - matches the operator; no new flag; the user removes a Namespace with `kubectl delete` when they mean it.
2. Skip by default, `--include-protected` to delete - one more flag on a destructive command, and the CLI and operator would still differ.
**Decision**: Option 1 (owner decision).
**Rationale**: One rule in both frontends; deleting a Namespace stays an explicit `kubectl` act.

### UUID match on delete

**Context**: The owner asked delete to skip an object "unless managed-by and instance UUID still match (like the operator)".
**Explored**: `opm-operator/internal/apply/prune.go`: managed-by must be an OPM value; a live UUID that differs from the owner's is skipped; an empty live UUID or an empty owner UUID disables the UUID comparison.
**Options considered**:
1. The operator's tolerances - legacy objects without the label still delete.
2. Strict match on both sides - legacy objects are left behind on every delete.
**Decision**: Option 1.
**Rationale**: "like the operator" names the reference; the strict rule would make CLI and operator delete differently again.

## Example output

The resource lines come from `output.FormatResourceLine` (`r:Kind/namespace/name`, padded to 48 columns, then icon and status); the closing line from `output.FormatCheckmark`.

```text
$ opm instance delete demo -n apps --force
demo  deleting resources in namespace "apps"
demo  r:Deployment/apps/web                           - deleted
demo  r:CustomResourceDefinition/widgets.example.io   ! left behind  reason="CRDs and Namespaces are never deleted"
demo  r:Namespace/apps                                ! left behind  reason="CRDs and Namespaces are never deleted"
✔ Instance deleted — 2 resource(s) left behind
  Remove them with 'kubectl delete' once nothing else needs them.
```

## Risks / Trade-offs

- [A user relied on `instance delete` removing the instance's Namespace] → it now stays, listed with a `kubectl delete` hint; the PR title and the draft release notes carry the changelog note (proposal.md § Behaviour change for users).
- [An extra GET per object on delete] → one request per tracked object, the same order as the existing discovery reads; acceptable for an interactive command.
- [A left-behind CRD or Namespace drops out of the inventory after prune] → unchanged rule (Namespaces already do this); the warn lines are the record.
