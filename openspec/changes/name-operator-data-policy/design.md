## Context

See `proposal.md` for the motivation. The facts the design rests on, each read from source:

- `opm instance delete` reads the `ModuleInstance` once, before the prompt (`query.ResolveInventory`), into `inventory.Record`. `Record.Prune` is the `spec.prune` of that object (`internal/inventory/store.go`, `recordFromObject`). For an operator-managed instance the CLI then deletes only the `ModuleInstance` and waits; the operator's finalizer does the rest.
- The operator branch `feat/protect-pvcs-by-default` adds `spec.dataPolicy` (`api/v1alpha1/common_types.go`): an optional enum `Keep`, `Delete`. Its method `DeletesClaims` is true for `Delete` only: an absent value, `Keep` and any unknown value keep claims. The field has no effect unless `spec.prune` is true. It covers pruning and deletion only: with `spec.rollout.forceConflicts`, an apply that the API server refuses as a change to an immutable field deletes and recreates the object, a claim included.
- The CLI writes the `ModuleInstance` spec with server-side apply under its own field manager (`inventory.ApplySpec`) and does not write `spec.prune` or `spec.dataPolicy`, so a value another manager set is not removed by a CLI edit. Read from source, not run against a cluster.

## Goals / Non-Goals

**Goals:**

- Every text the CLI prints or documents about an operator-managed instance and PersistentVolumeClaims is true for an operator that carries `spec.dataPolicy`, and says what an older operator does.
- One place decides how a `spec.dataPolicy` value is read and worded.

**Non-Goals:**

- No change to the CLI-owned delete and prune (cli#345).
- No new flag, no refusal, no new cluster read.
- No change to the logic of `internal/inventory`, `internal/kubernetes` or `internal/workflow/apply`. The change adds one read field to `inventory.Record` and rewords one constant in `internal/workflow/apply`.
- The CLI does not set `spec.dataPolicy`.

## Decisions

### Read the field into `inventory.Record`

`Record` gains `DataPolicy string`, read with `unstructured.NestedString(obj.Object, "spec", "dataPolicy")` beside `spec.prune`. A missing or wrong-typed value reads as empty. The value is kept as it is: the record does not interpret it.

Alternative: read the `ModuleInstance` a second time in the delete command. Rejected: it is a second cluster read of an object the command already holds, and the two reads can disagree. Alternative: import the operator's `v1alpha1` types. Rejected: the CLI reads the CR as unstructured data everywhere, and the brief of this change excludes the import.

### Interpret the value in the command package, as the operator does

`internal/cmd/instance` holds the interpretation: only the exact string `Delete` deletes claims. Everything else keeps them, which is the operator's own rule (`DeletesClaims`), so the CLI and the operator cannot disagree about an unknown value. The value is shown as it is: `Keep` and `Delete` bare, no value as "is not set", any other value quoted with Go `%q` and the words "not a value opm knows, read as Keep". `%q` also neutralises control characters, since the value comes from the cluster.

```go
// operatorDeletesClaims reports whether spec.dataPolicy lets the operator
// delete PersistentVolumeClaims.
func operatorDeletesClaims(dataPolicy string) bool

// describeDataPolicy words the value for a message: "spec.dataPolicy is Delete".
func describeDataPolicy(dataPolicy string) string

// operatorManagedDeletePrompt gains the dataPolicy argument.
func operatorManagedDeletePrompt(instanceName, instanceID, namespace string, prune bool, dataPolicy string) string
```

### The operator version: the CLI cannot tell, and says so

An operator released before `spec.dataPolicy` ignores the field and deletes claims under `spec.prune`. The brief asks for a sentence about that "if the cli can tell". It cannot, for three reasons:

- Before the prompt the command reads only the `ModuleInstance` and the resources of its inventory. The operator version is in `Platform/cluster` `status.operatorVersion`, which this command does not read; reading it is a new cluster read of a cluster-scoped object that a namespace-scoped user is often denied (`GateOperatorVersionCeiling` already degrades on that).
- The managed fields of the `ModuleInstance` name a field manager, not a version.
- The release that carries `spec.dataPolicy` has no number yet: opm-operator#267 is open. A comparison needs a constant that does not exist.

So the sentence is unconditional wherever the CLI says that claims are kept: "An operator released before spec.dataPolicy deletes PersistentVolumeClaims whatever the field says, and opm cannot tell which operator runs here." It is not printed when the policy is `Delete` (both operators delete) or when `spec.prune` is not set (both operators delete nothing). The docs page says how a user checks: `kubectl explain moduleinstance.spec.dataPolicy` fails on a cluster whose CRDs come from an older release.

Alternative: a version gate with a constant, added when the operator release exists. Deferred: it is a follow-up when the number is known, and it needs the Platform read.

### Texts

Prompt, `spec.prune` set, `dataPolicy: Delete`:

```text
This instance is operator-managed: spec.prune is set and spec.dataPolicy is Delete, so the operator deletes its tracked resources, PersistentVolumeClaims and the data on them included.
Delete the ModuleInstance for instance "jellyfin" in namespace "media"? [y/N]:
```

Prompt, `spec.prune` set, no `dataPolicy`:

```text
This instance is operator-managed: spec.prune is set and spec.dataPolicy is not set, so the operator deletes its tracked resources and keeps PersistentVolumeClaims and the data on them.
An operator released before spec.dataPolicy deletes PersistentVolumeClaims whatever the field says, and opm cannot tell which operator runs here.
Delete the ModuleInstance for instance "jellyfin" in namespace "media"? [y/N]:
```

Prompt, `spec.prune` not set (unchanged):

```text
This instance is operator-managed: spec.prune is not set, so the operator leaves its tracked resources running.
```

Warning for `--delete-data`, all three commands:

```text
WARN --delete-data does not change what the operator does with an operator-managed instance: spec.dataPolicy on the ModuleInstance decides whether the operator deletes PersistentVolumeClaims
```

Closing output of a completed delete, `spec.prune` set, the policy keeps, the inventory tracked the claim `apps/data`:

```text
Instance deleted - the operator pruned its tracked resources and keeps PersistentVolumeClaims

The instance tracked 1 PersistentVolumeClaim(s). The operator keeps them and the data on them
(spec.dataPolicy is not set), and OPM no longer tracks them.
An operator released before spec.dataPolicy deleted them. To see what is left:
  kubectl get pvc -n apps
To delete a claim and its data:
  kubectl delete pvc data -n apps
```

The closing output claims nothing the CLI did not establish: the disappearance of the `ModuleInstance` proves that the finalizer completed, not which claims are left. When the inventory tracked no claim, or the policy is `Delete`, the closing line stays "Instance deleted - operator pruned N resources". The tracked claims are counted from the inventory entries of the record with `kubernetes.IsDataClaim`, which the CLI-owned branch already uses.

Syntax, flags and exit codes: `opm instance delete <file|name|uuid> [flags]` is unchanged. No flag is added or changed. Exit codes are unchanged: 0 on success and on a declined prompt, 2 when the operator is not ready, 5 for a missing instance. No new error is introduced.

## Risks / Trade-offs

- [The operator PR is not merged] The texts describe an operator that is not released. Mitigation: the prompt and the page carry the sentence on older operators, and the docs page names no operator version.
- [The sentence on older operators is printed to users of a current operator too] It costs one line in a prompt for a destructive action and is true. It goes away with the version gate, when the release number exists.
- [`Record.DataPolicy` touches `internal/inventory`, where another change is in preparation] The addition is one field and one read, beside `Prune`. A conflict there is two adjacent lines.
- [A forced recreate deletes a claim under `Keep`] Not a CLI behaviour. The docs page names it.
