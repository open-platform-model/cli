---
title: "Kept PersistentVolumeClaims"
description: "opm instance delete and the prune of apply keep PersistentVolumeClaims and their data by default: what the kept lines mean, how to delete a claim, what the controller does for the instances it manages, and what changed for users of older releases."
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
- **Controller-managed instances.** There the controller deletes, not `opm`, and `--delete-data` does not change what it does. See [Controller-managed instances](#controller-managed-instances).
- **Other kinds.** Only a `PersistentVolumeClaim` of the core API group is kept. Namespaces and CustomResourceDefinitions are never deleted, and no flag changes that.
- **`opm controller uninstall`.** The controller module renders no claim, so uninstall has nothing to keep.

## Controller-managed instances

For an instance that the controller manages, `opm instance delete` deletes only the `ModuleInstance`. The controller then removes what the instance tracks, or leaves it. What it does with PersistentVolumeClaims depends on the controller release.

### A controller without the field deletes claims

`spec.dataPolicy` is new in the controller. A controller released before the field deletes PersistentVolumeClaims and the data on them whenever `spec.prune` is set. Its `ModuleInstance` CRD has no `spec.dataPolicy` in its schema, and `opm` reads the field's absence there as "this controller deletes claims", whatever the instance carries.

To see the release of the controller that runs in a cluster, read the `Controller` column of the Platform:

```sh
kubectl get platform cluster
```

### A controller with the field keeps claims

A controller that has `spec.dataPolicy` reads two fields of the `ModuleInstance`:

| `spec.prune` | `spec.dataPolicy` | The controller on delete |
| --- | --- | --- |
| not set | any | Leaves every tracked resource running, claims included. |
| `true` | not set, or `Keep` | Deletes the tracked resources and keeps PersistentVolumeClaims and the data on them. |
| `true` | `Delete` | Deletes the tracked resources, PersistentVolumeClaims and the data on them included. |

The same holds for the prune of the controller: a claim that a new render no longer produces is kept unless `spec.dataPolicy` is `Delete`. A kept claim stays in the cluster and the controller no longer tracks it.

Only `Delete` makes the controller delete claims. `opm` shows the value as it is written; a value other than `Keep` and `Delete` is shown in quotes and read as `Keep`, which is also how the controller reads it.

To have such a controller delete the claims of an instance, set the field before you delete the instance. This needs a controller and CRDs that have the field:

```sh
kubectl patch moduleinstance jellyfin -n media --type=merge -p '{"spec":{"dataPolicy":"Delete"}}'
```

### What the confirmation prompt says

Before it asks, `opm instance delete` reads the `ModuleInstance`, with its `spec.prune`, its `spec.dataPolicy` and its inventory, and the resources that the inventory tracks. For a controller-managed instance it then checks that the controller is ready: it reads the four CRDs of the controller and the Deployment of its controller. A controller that is not ready, or that `opm` cannot read, ends the command with a refusal, before any question. From the `ModuleInstance` CRD that this check reads, `opm` also takes whether the controller has `spec.dataPolicy`.

When `spec.prune` is set and the inventory tracks a PersistentVolumeClaim, the prompt says one of these:

| The cluster | The prompt says |
| --- | --- |
| The CRD has no `spec.dataPolicy` | The controller has no `spec.dataPolicy` and deletes the claims and the data on them. |
| `spec.dataPolicy` is `Delete` | The controller deletes the claims and the data on them. |
| The CRD has the field, and the value is `Keep`, not set, or unknown | The controller keeps the claims. One more sentence says that a controller older than its CRDs deletes them. |
| The CRD has no schema that `opm` can read | A controller with the field keeps the claims, an older one deletes them, and `opm` could not tell which runs. |

An instance that tracks no PersistentVolumeClaim gets no sentence about claims.

On a controller without the field:

```text
This instance is controller-managed: spec.prune is set and the controller in this cluster has no spec.dataPolicy, so the controller deletes its tracked resources, PersistentVolumeClaims and the data on them included.
Delete the ModuleInstance for instance "jellyfin" in namespace "media"? [y/N]:
```

On a controller with the field, for an instance without `spec.dataPolicy`:

```text
This instance is controller-managed: spec.prune is set and spec.dataPolicy is not set, so the controller deletes its tracked resources and keeps PersistentVolumeClaims and the data on them.
The ModuleInstance CRD has spec.dataPolicy, but a controller older than its CRDs deletes the claims whatever the field says.
Delete the ModuleInstance for instance "jellyfin" in namespace "media"? [y/N]:
```

The second sentence is there because the CRDs can be newer than the controller that runs beside them: `opm controller install --crds-only` updates the CRDs alone. `opm instance delete` decides from the CRD only. It compares no controller release, because no release with the field existed to compare with when this was written. If you are not sure, read the running release with `kubectl get platform cluster` before you answer.

The opposite state, a controller that is newer than its CRDs, is not supported. In that state the CRD has no `spec.dataPolicy`, so the prompt says that the controller deletes the claims, while a controller with the field keeps them.

With `--yes` and with `--dry-run` there is no prompt; the same outcome is on the `INFO` line of the run. After a delete that may have kept claims, the closing output lists them with the `kubectl get pvc` and `kubectl delete pvc` commands. It does not say that the claims are still there: `opm` does not read them back.

### The flag does not change what the controller does

`--delete-data` steers only what `opm` itself deletes. On a controller-managed instance, `opm instance delete`, `opm instance apply` and `opm module apply` print a warning and go on as without the flag:

```text
WARN --delete-data does not change what the controller does with a controller-managed instance: a controller that has spec.dataPolicy keeps PersistentVolumeClaims unless that field of the ModuleInstance is Delete, and an older controller deletes them
```

### A forced recreate deletes a claim under any data policy

`spec.dataPolicy` covers pruning and deletion only. With `spec.rollout.forceConflicts` set, an apply that the API server refuses as a change to an immutable field makes the controller delete the object and create it again. This includes a PersistentVolumeClaim, for example when a new module version changes its `storageClassName` or `accessModes`, and the data on the old claim is lost, with `Keep` too. Until the controller closes this gap, do not set `spec.rollout.forceConflicts` on an instance whose claims hold data that you need.
