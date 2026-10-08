---
title: "Kept PersistentVolumeClaims"
description: "opm instance delete and the prune of apply keep PersistentVolumeClaims and their data by default: what the kept lines mean, how to delete a claim, what the operator does for the instances it manages, and what changed for users of older releases."
type: how-to
weight: 29
---

For an instance that the CLI manages, `opm instance delete` and the prune of `opm instance apply` and `opm module apply` do not delete a PersistentVolumeClaim. Deleting a claim deletes the data on its volume under the usual reclaim policy, so `opm` does it only when you pass `--delete-data`.

> [!WARNING]
> **The default changed after `v1.0.0-beta.10`**
>
> Releases up to `v1.0.0-beta.10` deleted a tracked PersistentVolumeClaim on `opm instance delete`, and pruned one that the module no longer rendered. Later releases keep it. A script that relied on delete or prune to remove claims must now pass `--delete-data`. Nothing else changes: the exit code of a run that keeps a claim is 0.

## The message

A kept claim is listed with the status `kept`, on an `INFO` line. It is not a warning and not an error.

`opm instance delete` lists each kept claim, deletes everything else, and prints the command that deletes each claim:

```text
INFO r:PersistentVolumeClaim/media/config                = kept
INFO r:Deployment/media/jellyfin                         - deleted
Instance deleted

Kept 1 PersistentVolumeClaim(s) and the data on them. OPM no longer tracks them.
Applying the instance again takes them back. To delete a claim and its data:
  kubectl delete pvc config -n media
To delete claims together with an instance, pass --delete-data.
```

`opm instance apply` and `opm module apply` list each stale claim that the prune kept:

```text
INFO keeping 1 stale PersistentVolumeClaim(s) and the data on them; pass --delete-data to prune them
INFO r:PersistentVolumeClaim/media/old-cache             = kept
Instance applied
```

With `--dry-run`, each command lists the same claims and says how many it would keep.

## What it means

The claim and its data are still in the cluster. The command did everything else it was asked to do and exited 0.

What happens to the claim next depends on the command:

| Command | The claim afterwards |
| --- | --- |
| `opm instance delete` | The `ModuleInstance` is deleted, so OPM no longer tracks the claim. No later `opm` command deletes it. If you apply the instance again, the apply takes the claim back and warns that it was not recorded. |
| `opm instance apply`, `opm module apply` | The claim stays in the inventory of the instance. Every later apply lists it as kept again, until an apply with `--delete-data` prunes it or you delete the claim yourself. An apply drops a claim that is no longer in the cluster from the inventory and does not list it. |

## Causes and fixes

### You want to keep the data

Do nothing. To reuse the data, apply the instance again with a module that renders a claim of the same name.

### You want to delete the data with the instance

Pass `--delete-data` to `opm instance delete`. Without `--yes`, the confirmation prompt names every claim that the run deletes:

```text
--delete-data: these PersistentVolumeClaims and the data on them will be deleted:
  media/config
Delete the resources for instance "jellyfin" in namespace "media" (CRDs and Namespaces are left behind)? [y/N]:
```

`--yes` alone skips the prompt and still keeps the claims. Run the command with `--dry-run` first to see which claims a real run deletes.

### You already deleted the instance and want to delete a kept claim

Run the `kubectl delete pvc` command that the delete printed. `opm` has no record of the claim any more.

### A claim that the module no longer renders is listed on every apply

Run the apply once with `--delete-data`. It prunes every stale claim of the instance, also one that an earlier apply kept. `--delete-data` and `--no-prune` exclude each other.

## What the flag does not cover

- **Claims that a StatefulSet creates.** A StatefulSet creates one claim per replica from its `volumeClaimTemplates`. OPM does not track those claims, so `opm` never deletes them, with or without `--delete-data`. Kubernetes keeps them when the StatefulSet is deleted, unless the StatefulSet sets `persistentVolumeClaimRetentionPolicy`. Delete them with `kubectl delete pvc`.
- **Operator-managed instances.** There the operator deletes, not `opm`, and `--delete-data` does not change what it does. See [Operator-managed instances](#operator-managed-instances).
- **Other kinds.** Only a `PersistentVolumeClaim` of the core API group is kept. Namespaces and CustomResourceDefinitions are never deleted, and no flag changes that.
- **`opm operator uninstall`.** The operator module renders no claim, so uninstall has nothing to keep.

## Operator-managed instances

For an instance that the operator manages, `opm instance delete` deletes only the `ModuleInstance`. The operator then removes what the instance tracks, or leaves it. What it does with PersistentVolumeClaims depends on the operator release.

### An operator without the field deletes claims

`spec.dataPolicy` is new in the operator. An operator released before the field deletes PersistentVolumeClaims and the data on them whenever `spec.prune` is set. Its `ModuleInstance` CRD has no `spec.dataPolicy` in its schema, and `opm` reads the field's absence there as "this operator deletes claims", whatever the instance carries.

To see the release of the operator that runs in a cluster, read the `Operator` column of the Platform:

```sh
kubectl get platform cluster
```

### An operator with the field keeps claims

An operator that has `spec.dataPolicy` reads two fields of the `ModuleInstance`:

| `spec.prune` | `spec.dataPolicy` | The operator on delete |
| --- | --- | --- |
| not set | any | Leaves every tracked resource running, claims included. |
| `true` | not set, or `Keep` | Deletes the tracked resources and keeps PersistentVolumeClaims and the data on them. |
| `true` | `Delete` | Deletes the tracked resources, PersistentVolumeClaims and the data on them included. |

The same holds for the prune of the operator: a claim that a new render no longer produces is kept unless `spec.dataPolicy` is `Delete`. A kept claim stays in the cluster and the operator no longer tracks it.

Only `Delete` makes the operator delete claims. `opm` shows the value as it is written; a value other than `Keep` and `Delete` is shown in quotes and read as `Keep`, which is also how the operator reads it.

To have such an operator delete the claims of an instance, set the field before you delete the instance. This needs an operator and CRDs that have the field:

```sh
kubectl patch moduleinstance jellyfin -n media --type=merge -p '{"spec":{"dataPolicy":"Delete"}}'
```

### What the confirmation prompt says

Before it asks, `opm instance delete` reads the `ModuleInstance`, with its `spec.prune`, its `spec.dataPolicy` and its inventory, and the resources that the inventory tracks. For an operator-managed instance it then checks that the operator is ready: it reads the four CRDs of the operator and the Deployment of its controller. An operator that is not ready, or that `opm` cannot read, ends the command with a refusal, before any question. From the `ModuleInstance` CRD that this check reads, `opm` also takes whether the operator has `spec.dataPolicy`.

When `spec.prune` is set and the inventory tracks a PersistentVolumeClaim, the prompt says one of these:

| The cluster | The prompt says |
| --- | --- |
| The CRD has no `spec.dataPolicy` | The operator has no `spec.dataPolicy` and deletes the claims and the data on them. |
| `spec.dataPolicy` is `Delete` | The operator deletes the claims and the data on them. |
| The CRD has the field, and the value is `Keep`, not set, or unknown | The operator keeps the claims. One more sentence says that an operator older than its CRDs deletes them. |
| The CRD has no schema that `opm` can read | An operator with the field keeps the claims, an older one deletes them, and `opm` could not tell which runs. |

An instance that tracks no PersistentVolumeClaim gets no sentence about claims.

On an operator without the field:

```text
This instance is operator-managed: spec.prune is set and the operator in this cluster has no spec.dataPolicy, so the operator deletes its tracked resources, PersistentVolumeClaims and the data on them included.
Delete the ModuleInstance for instance "jellyfin" in namespace "media"? [y/N]:
```

On an operator with the field, for an instance without `spec.dataPolicy`:

```text
This instance is operator-managed: spec.prune is set and spec.dataPolicy is not set, so the operator deletes its tracked resources and keeps PersistentVolumeClaims and the data on them.
The ModuleInstance CRD has spec.dataPolicy, but an operator older than its CRDs deletes the claims whatever the field says.
Delete the ModuleInstance for instance "jellyfin" in namespace "media"? [y/N]:
```

The second sentence is there because the CRDs can be newer than the operator that runs beside them: `opm operator install --crds-only` updates the CRDs alone. `opm instance delete` decides from the CRD only. It compares no operator release, because no release with the field existed to compare with when this was written. If you are not sure, read the running release with `kubectl get platform cluster` before you answer.

The opposite state, an operator that is newer than its CRDs, is not supported. In that state the CRD has no `spec.dataPolicy`, so the prompt says that the operator deletes the claims, while an operator with the field keeps them.

With `--yes` and with `--dry-run` there is no prompt; the same outcome is on the `INFO` line of the run. After a delete that may have kept claims, the closing output lists them with the `kubectl get pvc` and `kubectl delete pvc` commands. It does not say that the claims are still there: `opm` does not read them back.

### The flag does not change what the operator does

`--delete-data` steers only what `opm` itself deletes. On an operator-managed instance, `opm instance delete`, `opm instance apply` and `opm module apply` print a warning and go on as without the flag:

```text
WARN --delete-data does not change what the operator does with an operator-managed instance: an operator that has spec.dataPolicy keeps PersistentVolumeClaims unless that field of the ModuleInstance is Delete, and an older operator deletes them
```

### A forced recreate deletes a claim under any data policy

`spec.dataPolicy` covers pruning and deletion only. With `spec.rollout.forceConflicts` set, an apply that the API server refuses as a change to an immutable field makes the operator delete the object and create it again. This includes a PersistentVolumeClaim, for example when a new module version changes its `storageClassName` or `accessModes`, and the data on the old claim is lost, with `Keep` too. Until the operator closes this gap, do not set `spec.rollout.forceConflicts` on an instance whose claims hold data that you need.
