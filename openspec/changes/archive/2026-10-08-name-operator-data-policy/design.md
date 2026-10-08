## Context

See `proposal.md` for the motivation. The facts the design rests on, each read from source:

- `opm instance delete` reads the `ModuleInstance` once, before the prompt (`query.ResolveInventory`), into `inventory.Record`. `Record.Prune` is the `spec.prune` of that object (`internal/inventory/store.go`, `recordFromObject`). For an operator-managed instance the CLI then deletes only the `ModuleInstance` and waits; the operator's finalizer does the rest.
- The operator branch `feat/protect-pvcs-by-default` adds `spec.dataPolicy` (`api/v1alpha1/common_types.go`): an optional enum `Keep`, `Delete`. Its method `DeletesClaims` is true for `Delete` only: an absent value, `Keep` and any unknown value keep claims. The field has no effect unless `spec.prune` is true. It covers pruning and deletion only: with `spec.rollout.forceConflicts`, an apply that the API server refuses as a change to an immutable field deletes and recreates the object, a claim included.
- The CLI writes the `ModuleInstance` spec with server-side apply under its own field manager (`inventory.ApplySpec`) and does not write `spec.prune` or `spec.dataPolicy`, so a value another manager set is not removed by a CLI edit. Read from source, not run against a cluster.

## Goals / Non-Goals

**Goals:**

- Every text the CLI prints or documents about an operator-managed instance and PersistentVolumeClaims is true for the operator that runs in the cluster, with the field or without it.
- One place decides how a `spec.dataPolicy` value is read and worded.

**Non-Goals:**

- No change to the CLI-owned delete and prune (cli#345).
- No new flag, no refusal, and no read of an object the command does not read already.
- No change to the logic of `internal/inventory`, `internal/kubernetes` or `internal/workflow/apply`. The change adds one read field to `inventory.Record` and one exported wrapper of the existing CRD schema walker there, rewords one constant in `internal/workflow/apply`, and lets the readiness gate in `internal/operator` return the CRD it read.
- The CLI does not set `spec.dataPolicy`.

## Decisions

### Read the field into `inventory.Record`

`Record` gains `DataPolicy string`, read with `unstructured.NestedString(obj.Object, "spec", "dataPolicy")` beside `spec.prune`. A missing or wrong-typed value reads as empty. The value is kept as it is: the record does not interpret it.

Alternative: read the `ModuleInstance` a second time in the delete command. Rejected: it is a second cluster read of an object the command already holds, and the two reads can disagree. Alternative: import the operator's `v1alpha1` types. Rejected: the CLI reads the CR as unstructured data everywhere and does not depend on the operator's Go module.

### Interpret the value in the command package, as the operator does

`internal/cmd/instance` holds the interpretation: only the exact string `Delete` is a request to delete claims. Everything else keeps them, which is the operator's own rule (`DeletesClaims`), so the CLI and the operator cannot disagree about an unknown value. The value is shown as it is: `Keep` and `Delete` bare, no value as "is not set", any other value quoted with Go `%q` and the words "not a value opm knows, read as Keep". `%q` also neutralises control characters, since the value comes from the cluster.

### The operator's CRD says whether the operator has the field

An operator released before `spec.dataPolicy` does not know the field and deletes claims under `spec.prune`. When this change was written no released operator had the field (opm-operator#267 was open), and the operator this CLI pins and installs was one of them. A prompt that leads with "keeps PersistentVolumeClaims" is then false for every released operator. So the CLI must find out, and it can, from an object the delete of an operator-managed instance already reads: its readiness gate reads the `moduleinstances.opmodel.dev` CRD. The schema of that CRD says whether the API has `spec.dataPolicy`, with no version number to compare.

```go
// internal/operator
type FieldSupport int // FieldUnknown, FieldAbsent, FieldPresent

// ReadyModuleInstanceCRD is CheckReady that also returns the live
// ModuleInstance CRD the gate read. It refuses exactly when CheckReady does.
func ReadyModuleInstanceCRD(ctx context.Context, client *kubernetes.Client) (*unstructured.Unstructured, error)

// SpecFieldSupport reports whether that CRD has spec.<field>.
func SpecFieldSupport(crd *unstructured.Unstructured, field string) FieldSupport

// internal/inventory: the one reading of the CRD schema (storage version,
// else the first served one), shared with the apply gate CheckCRDFieldFloor.
func ModuleInstanceCRDHasField(crd *unstructured.Unstructured, parent, child string) (bool, error)
```

One read, one schema walker. The gate reads the CRD once and the field is taken from that object. `internal/inventory/gates.go` already walks the CRD schema for the apply gate; the new exported function wraps that walker, so the two callers cannot apply different version rules.

The result has four cases, for an instance with `spec.prune` set that tracks a claim:

| CRD | `spec.dataPolicy` | The CLI says |
| --- | --- | --- |
| no such field | any | the operator has no `spec.dataPolicy` and deletes the claims |
| any | `Delete` | the operator deletes the claims |
| has the field | `Keep`, absent, unknown | the operator keeps the claims; an operator older than its CRDs deletes them |
| no readable schema | not `Delete` | neither: an operator with the field keeps them, an older one deletes them, opm could not read the schema |

The third row keeps one sentence of doubt. `opm operator install --crds-only` and a failed controller rollout both leave CRDs that are newer than the controller. The delete decides from the CRD only and compares no operator release: no release with the field existed to compare with. The release of the running controller is readable (`Platform/cluster` `status.operatorVersion`, which the apply gate reads, and the image of the controller Deployment, which the readiness gate reads); a comparison is a follow-up for when the release number exists.

The fourth row is a CRD that the gate read and found Established, but whose version or schema cannot be read. A CRD that cannot be read at all never reaches the prompt: the gate refuses first.

The opposite skew, a controller newer than its CRDs, is not supported. The CRD then has no field, the CLI says that the operator deletes the claims, and an operator with the field keeps them. The docs page says so.

### The question comes after every check that can refuse

The readiness gate refuses the delete of an operator-managed instance when the operator is not ready or cannot be read. It ran after the prompt, so a user answered a question about data loss and was then refused. The question now runs after the guard for an instance that deploys the operator and after the readiness gate, in one place:

```go
// confirmFunc asks whether the delete goes ahead, with what the operator does
// with the instance's claims. A run that skips the question answers yes.
type confirmFunc func(opClaims operatorClaims) bool

func deleteResolvedInstance(..., confirm confirmFunc, ...) error // guard, then: operator-owned -> deleteOperatorOwned; else confirm, delete
func deleteOperatorOwned(..., confirm confirmFunc, ...) error   // readiness gate, claims from its CRD, confirm, delete
```

The gate does not tell an RBAC denial from "operator not ready": both are the same refusal, as before this change. Whether a denied read should degrade is an owner decision and is not taken here.

The tracked claims are counted from the inventory entries of the record with `kubernetes.IsDataClaim`, which the CLI-owned branch already uses.

### Texts

Prompt, `spec.prune` set, a claim tracked, the CRD has no `spec.dataPolicy`:

```text
This instance is operator-managed: spec.prune is set and the operator in this cluster has no spec.dataPolicy, so the operator deletes its tracked resources, PersistentVolumeClaims and the data on them included.
Delete the ModuleInstance for instance "jellyfin" in namespace "media"? [y/N]:
```

Prompt, the CRD has the field, `dataPolicy: Delete`:

```text
This instance is operator-managed: spec.prune is set and spec.dataPolicy is Delete, so the operator deletes its tracked resources, PersistentVolumeClaims and the data on them included.
```

Prompt, the CRD has the field, no `dataPolicy`:

```text
This instance is operator-managed: spec.prune is set and spec.dataPolicy is not set, so the operator deletes its tracked resources and keeps PersistentVolumeClaims and the data on them.
The ModuleInstance CRD has spec.dataPolicy, but an operator older than its CRDs deletes the claims whatever the field says.
```

Prompt, the CRD has no readable schema:

```text
This instance is operator-managed: spec.prune is set and spec.dataPolicy is not set, so the operator deletes its tracked resources.
An operator that has spec.dataPolicy keeps PersistentVolumeClaims and the data on them, an older operator deletes them, and opm could not read the schema of the ModuleInstance CRD to tell which runs here.
```

Prompt, `spec.prune` set, no claim tracked:

```text
This instance is operator-managed: spec.prune is set, so the operator deletes its tracked resources.
```

Prompt, `spec.prune` not set (unchanged):

```text
This instance is operator-managed: spec.prune is not set, so the operator leaves its tracked resources running.
```

Warning for `--delete-data`, all three commands:

```text
WARN --delete-data does not change what the operator does with an operator-managed instance: an operator that has spec.dataPolicy keeps PersistentVolumeClaims unless that field of the ModuleInstance is Delete, and an older operator deletes them
```

Progress line of the run, and the dry-run line, `spec.prune` set (both exist today and gain the part after the comma; an instance that tracks no claim gains nothing):

```text
INFO deleting the ModuleInstance [...] the operator prunes its resources, PersistentVolumeClaims and the data on them kept (spec.dataPolicy is not set) unless the operator is older than its CRDs
INFO dry run complete: ModuleInstance "demo" would be deleted and the operator would prune its 2 tracked resource(s), PersistentVolumeClaims and the data on them included (the operator in this cluster has no spec.dataPolicy)
```

Closing output of a completed delete when the operator may have kept the claim `apps/data` (third and fourth row):

```text
Instance deleted: the operator finished its cleanup

The instance tracked 1 PersistentVolumeClaim(s). An operator that has spec.dataPolicy keeps them
and the data on them (spec.dataPolicy is not set), and OPM no longer tracks them.
An operator older than its CRDs deleted them. To see what is left:
  kubectl get pvc -n apps
To delete a claim and its data:
  kubectl delete pvc data -n apps
To have the operator delete claims with an instance, set spec.dataPolicy to Delete before deleting it.
```

The closing output claims nothing the CLI did not establish: the disappearance of the `ModuleInstance` proves that the finalizer completed, not which claims are left. In every other case the closing line stays as it is today ("Instance deleted", then "operator pruned N resources").

Syntax, flags and exit codes: `opm instance delete <file|name|uuid> [flags]` is unchanged. No flag is added or changed. Exit codes are unchanged: 0 on success and on a declined prompt, 2 when the operator is not ready, 5 for a missing instance. No new error is introduced. The operator-readiness refusal (exit 2) now comes before the question.

## Risks / Trade-offs

- [The operator PR is not merged] The texts for an operator with `spec.dataPolicy` describe an operator that is not released. Mitigation: the CLI says "kept" only when the installed CRD has the field, and the docs page names no operator version.
- [CRDs newer than the controller] The CLI then says "kept" and the operator deletes. Mitigation: the one sentence of doubt in the prompt, the progress line and the closing output, and the check on the docs page.
- [`Record.DataPolicy` touches `internal/inventory`, where another change is in preparation] The addition is one field and one read, beside `Prune`. A conflict there is two adjacent lines.
- [A forced recreate deletes a claim under `Keep`] Not a CLI behaviour. The docs page names it.
- [The names in the `kubectl delete pvc` lines come from `status.inventory`] They are printed as they are, as the CLI-owned closing output prints them. Not changed here.
