---
title: "Wait for a deleted instance to be gone"
description: "opm instance delete returns when the API server has accepted the deletes. With --wait it returns only when the deleted resources are gone: what it waits for, what a timeout prints, and what to do then."
type: how-to
weight: 31
---

`opm instance delete` returns when the API server has accepted every delete. A deleted resource can still exist at that moment: it is terminating until its dependents are gone and every finalizer on it is removed. A script that deletes an instance and applies it again at once can apply onto a resource that is about to go.

Pass `--wait` to return only when the deleted resources are gone:

```bash
opm instance delete jellyfin -n media --yes --wait
```

`--timeout` bounds the wait. The default is `5m0s`, and the time starts when the wait starts:

```bash
opm instance delete jellyfin -n media --yes --wait --timeout 2m
```

Without `--wait` the command does what it did before the flag existed.

## What the command waits for

The command waits for each resource whose delete the API server accepted in this run. A resource is gone when:

- its read returns NotFound, or
- another object, with another UID, has its name. The resource that the run deleted no longer exists then.

The command does not wait for a resource that it did not delete:

| Resource | Why it is not waited for |
| --- | --- |
| A kept PersistentVolumeClaim | It is not deleted without `--delete-data`. See [Kept PersistentVolumeClaims](/docs/diagnostics/kept-volume-claims/). |
| A resource listed as `left behind` | It is a CustomResourceDefinition or a Namespace, or it no longer belongs to the instance. |
| A resource that was already gone | No delete was sent. |

`--wait` does nothing in two cases:

- With `--dry-run`. A dry run sends no delete.
- When a resource failed to delete. The command fails as it does without `--wait`. Fix the cause and run the command again; that run waits.

## When the wait times out

The command lists each resource that is still there, with its finalizers, and exits 1:

```text
INFO r:Deployment/media/jellyfin                         - deleted
INFO r:ConfigMap/media/jellyfin-config                   - deleted
INFO waiting for 2 deleted resource(s) to be gone timeout=2m0s
INFO 1 of 2 deleted resource(s) still terminating
WARN r:ConfigMap/media/jellyfin-config                   ! terminating finalizers=example.io/backup
ERRO timed out after 2m0s: 1 deleted resource(s) are still terminating

The ModuleInstance was kept, so it still tracks these resources.
A resource stays until its dependents are gone and every finalizer on it is removed.
Run the same command again to wait again; re-running is safe.
Without --wait the command deletes the ModuleInstance and does not wait for them.
```

While it waits, the command prints how many resources are left each time one goes, and every 30 seconds otherwise.

The command does not print `Instance deleted`, and it keeps the `ModuleInstance`. `opm instance list` still shows the instance, because resources of the instance still exist.

A line that ends with `error=` in place of `finalizers=` is a resource that the command could not read at the end of the wait; `not read before the timeout` means that the timeout passed before the first read of it answered. The command does not count a resource that it cannot read as gone.

The exit code is 1, not 3. Exit code 3 means that the cluster did not answer. Here the cluster answered each read and said that the resource exists.

### What to do

1. Read the finalizers on the line. Each one belongs to a controller that must finish its work and remove the finalizer.
2. Find out why that controller does not remove it. Often the controller is not running, or it waits for another resource.
3. Run the same command again. It finds the instance, sends the deletes again and waits again.

To stop tracking the instance without waiting, run the command again without `--wait`. It deletes the `ModuleInstance` and exits 0, and the terminating resources stay until their finalizers are removed.

`opm` never removes a finalizer from a resource.

## Operator-managed instances

For an instance that the operator manages, `opm instance delete` deletes only the `ModuleInstance`, and it always waits until the `ModuleInstance` is gone. `--timeout` bounds that wait too. `--wait` changes nothing there. The command does not read the resources that the operator prunes.
