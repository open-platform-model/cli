## Context

The cli deletes tracked objects in two places: `kubernetes.Delete` (used by `opm instance delete` and `opm operator uninstall` through `workflow/apply.DeleteRecorded`) and `inventory.PruneStaleResources` (the prune of `opm instance apply` and `opm module apply`). Both skip only `kubernetes.IsProtectedKind`: core Namespaces and CustomResourceDefinitions, with no override. A PersistentVolumeClaim is deleted like a ConfigMap, and with the usual `Delete` reclaim policy the volume and its data go with it. The owner decided to protect claims by default before v1.0.0 and to offer one explicit flag that deletes them.

## Goals / Non-Goals

**Goals:**

- `opm instance delete` and both prunes keep core PersistentVolumeClaims unless `--delete-data` is set.
- A kept claim is never silent and never a failure: it is listed, the exit code is 0, and the output says how to delete it.
- The record handling of a kept claim is defined and tested for delete and for prune.

**Non-Goals:**

- The operator's delete and prune (another repo). An operator-managed instance is not protected by this change.
- PersistentVolumes, Secrets and any other kind.
- Claims a StatefulSet creates from `volumeClaimTemplates`. The cli never tracks them: they are in no inventory, so delete and prune never see them, with or without the flag. Kubernetes keeps them when the StatefulSet is deleted or scaled down unless the StatefulSet sets `persistentVolumeClaimRetentionPolicy`.
- `opm operator uninstall`. The operator module (`opm-operator/modules/opm_operator/components.cue`, read at `59537b7`) renders one `emptyDir` volume and no claim, so uninstall has nothing to keep and stays as it is. It shares `DeleteRecorded`, which keeps a claim by default, so a future operator module with a claim is safe too.

## Research & Decisions

### Flag name

**Context**: one flag must mean "delete claims and their data" on three commands.
**Explored**: `openspec/specs/flag-conventions/spec.md` (after cli#341): `--force` overrides a protective refusal of state (empty render, non-empty directory) and never answers a prompt; `--yes` only skips a prompt.
**Options considered**:
1. `--force`: already means "prune on an empty render" on both apply commands. A second meaning there would make `--force` on an empty render delete data, the worst case.
2. `--delete-data`: says what is lost, is new on all three commands, has one meaning.
3. `--delete-pvcs` or `--prune-pvcs`: names the mechanism, not the consequence; two spellings for one concept across delete and apply.
**Decision**: `--delete-data`, boolean, default false, no shorthand.
**Rationale**: a destructive flag is spelled out in full and names the consequence. One name per concept.

### Record handling when delete keeps a claim

**Context**: after `opm instance delete` keeps a claim, either the `ModuleInstance` record is deleted and the claim is left untracked, or the record stays and still lists the claim.
**Options considered**:
1. The record goes; the claim is left in the cluster, untracked.
   - The claim is in no inventory afterwards. Prune and delete act only on inventory entries, so no later `opm` command and no operator can delete it. Only an explicit `kubectl delete pvc` does.
   - It matches what delete already does with a Namespace, a CRD and an object that left OPM management: listed, left in the cluster, record deleted, exit 0.
   - Applying the instance again takes the claim back: the first-install check passes an object that carries an OPM managed-by label (`inventory.FirstInstallCheck`) and warns that it was unrecorded. The data is reattached.
   - Cost: the claim can no longer be deleted through `opm`; the output must print the `kubectl` command.
2. The record stays and lists the kept claims.
   - A later `opm instance delete --delete-data` could remove the claim.
   - The instance is then not deleted: `opm instance list` and `status` still show it, `opm operator uninstall` and CRD removal see a live `ModuleInstance`, and the record keeps a full `spec` (module, values). If its owner is switched to the operator, the operator renders it again and, with `spec.prune`, deletes what the record lists, the claim included. The record is the thing that makes a claim deletable by a tool.
   - Needs a new write path (rewrite the inventory to the kept claims) that can fail half-way.
**Decision**: option 1.
**Rationale**: it is the state in which no tool can delete the data by accident, it adds no new write path, and it keeps "delete" meaning that the instance is gone. The lost convenience is one printed `kubectl` line per claim.

### Record handling when prune keeps a claim

**Decision**: the kept claim stays in the written inventory (as an entry prune failed to delete already does). Before it keeps a claim, the apply reads it (`inventory.ClaimsInCluster`): a claim that is already gone (NotFound) is not reported and leaves the record, so a claim the user removed with `kubectl` is not listed for ever; any other read error keeps it. It is stale again on every later apply, is listed as kept each time, and an apply with `--delete-data` prunes it. `opm instance delete` then sees it through the inventory and applies its own rule.
**Rationale**: given by the task. An apply keeps the instance, so the record is the right place to remember the claim. With `--no-prune` nothing changes: the stale set is neither deleted nor recorded, as today.

### Exit code and log level

**Decision**: exit 0. Each kept claim is one INFO line with the new status `kept`; the count line is INFO; the closing block of delete goes to standard error through `output.Details`, like the existing left-behind hint. No WARN and no ERRO line is printed for a kept claim.
**Rationale**: the claim is kept on purpose. A script that greps for `WARN` or checks the exit code must not trip on the default path. The left-behind lines of protected kinds stay warnings, unchanged.

### Where the rule lives

**Decision**: `kubernetes.IsDataClaim(group, kind)` in `protected.go`, beside `IsProtectedKind` and kept apart from it: a protected kind has no override, a claim has one. Callers pass the override down as a bool:

```go
// internal/kubernetes
func IsDataClaim(group, kind string) bool // core PersistentVolumeClaim
type DeleteOptions struct { /* ... */ DeleteData bool }
type DeleteResult  struct { /* ... */ Kept []LeftBehindResource }

// internal/inventory
func SplitDataClaims(stale []k8sinventory.Entry) (prunable, claims []k8sinventory.Entry)

// internal/workflow/apply
type Options       struct { /* ... */ DeleteData bool }
type DeleteRequest struct { /* ... */ DeleteData bool }
```

`Delete` checks the claim rule before the live read, as it does for a protected kind, so a claim discovery could not read is kept and is not an error. `PruneStaleResources` is unchanged: `Execute` splits the stale set first and never passes it a claim it must keep.

### Prompt order in instance delete

**Context**: the prompt must name the claims `--delete-data` deletes, and today the prompt runs before the record is read.
**Decision**: read the record first (`query.ResolveInventory`), then prompt, then run the guards and the delete. The prompt lists claims only for a CLI-owned instance with `--delete-data`, from the live objects of the inventory.
**Rationale**: a prompt that names its targets needs them. Side effect, accepted: a missing instance is reported (exit 5) before the prompt instead of after a "y". The read is read-only.

### Operator-managed instances

**Decision**: `--delete-data` is accepted and ignored with one WARN line, on delete and on both apply commands: `--delete-data has no effect on an operator-managed instance: the operator decides what it removes`. On delete the line prints before the confirmation question. The delete prompt of an operator-managed instance has its own text (`operatorManagedDeletePrompt`): it never says that claims are kept, and it says what `spec.prune` makes the operator do, claims included. The operator's prune exempts only Namespaces and CustomResourceDefinitions (`opm-operator/internal/apply/prune.go`, read at `59537b7`).
**Options considered**: refusing (as `--skip-unprovided` is refused) would fail scripts that pass the flag to mixed fleets for no safety gain; staying silent would hide that the flag did nothing.

## Command syntax and output

```text
opm instance delete <file|name|uuid> [--delete-data] [--yes] [--dry-run] ...
opm instance apply  <instance.cue>   [--delete-data] ...
opm module   apply  [path|module]    [--delete-data] ...
```

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delete-data` | bool | false | Also delete PersistentVolumeClaims and the data on them (kept by default) |

Delete, default:

```text
INFO r:PersistentVolumeClaim/media/config   kept
INFO r:Deployment/media/jellyfin            deleted
Instance deleted

Kept 1 PersistentVolumeClaim(s) and the data on them. OPM no longer tracks them.
Applying the instance again takes them back. To delete a claim and its data:
  kubectl delete pvc config -n media
To delete claims together with an instance, pass --delete-data.
```

Prompt with `--delete-data`:

```text
--delete-data: these PersistentVolumeClaims and the data on them will be deleted:
  media/config
Delete the resources for instance "jellyfin" in namespace "media" (CRDs and Namespaces are left behind)? [y/N]:
```

Apply, default, with a stale claim:

```text
INFO keeping 1 stale PersistentVolumeClaim(s) and the data on them; pass --delete-data to prune them
INFO r:PersistentVolumeClaim/media/old-cache   kept
Instance applied
```

Exit codes: 0 when claims are kept; 1 for `--delete-data` with `--no-prune` (usage error); every other code unchanged.

## Risks / Trade-offs

- A user who relied on delete to free storage now leaves claims behind, and they cost money. Mitigation: every run lists them and prints the command; the migration note says it.
- A stale claim kept by prune is reported on every apply while it is in the cluster. Accepted: it is one INFO line, and it is the only reminder. Once the claim is gone the next apply drops the entry.
- The prompt reorder changes when "not found" is reported. Accepted, see above.
- Operator-managed instances are not protected. Out of scope here; the docs page says it. It is a question for the owner whether the operator gets the same default.
