---
title: "Adopt an existing object"
description: "opm instance apply, opm module apply and opm operator install refuse to apply over an object that another owner holds: what each refusal means, how to adopt an object with the opmodel.dev/adopt annotation, how to hand an object from one instance to another, and what a dry run shows."
type: how-to
weight: 30
---

For an instance that the CLI manages, every apply checks every object it renders against the cluster before it changes anything. `opm` applies over an existing object only when the object belongs to the instance. When an object belongs to someone else, the apply stops before its first change and tells you how to adopt the object.

> [!WARNING]
> **The check changed after `v1.0.0-beta.10`**
>
> Releases up to `v1.0.0-beta.10` ran this check only on the first apply of an instance, and let the apply take over any object that carried OPM labels, whichever instance owned it. Later releases run the check on every apply, and on every dry run of an apply. An apply that took over an existing object before can now exit 1, and `opm operator install` can exit 2. A dry run of such an apply exits 1 too, where it exited 0 before: see "What a dry run shows". Nothing is changed in the cluster when that happens. No flag turns the check off.

## The message

A refused apply names every object it refuses, one line each, and says that it stopped before any change:

```text
apply refused: 1 object(s) cannot be applied by this instance:
  ConfigMap/default/settings exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c0a52-8d1e-5c1b-9a61-0d4c6c2f7be2
apply stopped before any change
```

An object that another instance adopted is not a refusal. The apply prints one warning for it and goes on:

```text
WARN ConfigMap/default/settings was adopted by module instance 9a40c1de-52f0-5b0e-8c11-4f2a9f3d1e17; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=6f1c0a52-8d1e-5c1b-9a61-0d4c6c2f7be2
```

## What it means

| Message | Meaning | Exit code |
| --- | --- | --- |
| `exists and is not managed by OPM` | The object is in the cluster and no OPM apply wrote it. The instance does not have it in its inventory. | 1 |
| `belongs to module instance <UUID>` | An OPM apply of another instance wrote the object. The instance does not have it in its inventory. | 1 |
| `is being deleted` | The object has a deletion timestamp. Kubernetes removes it when its finalizers are done. No annotation lifts this refusal. | 1 |
| `cannot check whether ... already exists` | The read of the object failed, so `opm` cannot tell who owns it. | 4 when the read was denied, 3 when the API server was unavailable, 1 otherwise |
| `was adopted by module instance <UUID>`, `is being adopted by module instance <UUID>` | The `opmodel.dev/adopt` annotation of the object names another instance. This instance does not apply the object, does not delete it, and removes it from its inventory. | 0 |

An object that the inventory of the instance already lists is always applied, also when its labels name another instance. Only a deletion timestamp or an `opmodel.dev/adopt` annotation for another instance stops it.

`opm operator install` runs the same check on every install, before it changes anything. It differs from the table in four points:

- An object that OPM does not manage, or that belongs to another instance, exits 2.
- An object that another instance adopted is a refusal too, with exit 2: the operator needs every object that its module renders.
- An object that is being deleted is not refused at once. Install waits for it to go, up to `--timeout`, and fails when it is still there.
- An object that install cannot read exits by the read error, as in the table: 4, 3 or 1.

Install does not take over an operator that was applied from an opm-operator release manifest. It refuses each object of that operator as an object that OPM does not manage, the CustomResourceDefinitions and the Namespace included, also with `--crds-only`. To keep those objects, annotate each one as the refusal prints, then run install again. Delete the earlier controller Deployment first: its selector differs from the selector of the module, Kubernetes does not let a selector change, and install fails on it after it applied the CustomResourceDefinitions. Install prints only objects that the module renders. It does not delete or report the other objects of the release manifest, such as the role bindings `opm-operator-manager-rolebinding`, `opm-operator-metrics-auth-rolebinding` and `opm-operator-leader-election-rolebinding`: delete them yourself.

## Causes and fixes

### You want the instance to own the object

Set the annotation that the refusal prints, then run the apply again:

```bash
kubectl annotate configmap settings -n default opmodel.dev/adopt=6f1c0a52-8d1e-5c1b-9a61-0d4c6c2f7be2
```

The value is the UUID of the instance. The apply then writes the object as the module renders it and adds it to the inventory. `opm` never sets this annotation, and no `opm` flag does.

### The object must stay as it is

Change the module or its values so that it renders the object under another name, or does not render it.

### You want to move an object from one instance to another

1. Annotate the object with the UUID of the instance that takes it over. The refusal of that instance prints the annotation.
2. Apply the instance that takes it over. It applies the object and records it.
3. Apply the first instance. It prints the `was adopted by` warning, removes the object from its inventory, and does not delete it. Remove the object from the first instance's module or values to stop the warning.

To move the object back, set the annotation to the UUID of the first instance and apply it.

### The object is being deleted

Wait until the object is gone, then apply again. If it does not go away, find what holds its finalizer. A PersistentVolumeClaim, for example, stays until no pod mounts it. `opm` cannot change the workload while one of the instance's objects is in this state, so release the object with `kubectl`.

### The refused objects are the instance's own

Three cases refuse objects that the instance wrote itself.

- **Leftover CustomResourceDefinitions and Namespaces.** `opm instance delete` and `opm operator uninstall` never delete a CustomResourceDefinition or a Namespace, and leave the labels on them. If you install the module again under another instance name, another namespace or another module path, the new instance has another UUID and the leftovers are refused as `belongs to module instance`. Annotate each one as the refusal shows. If nothing else uses them, you can delete them instead.
- **An instance with no record whose identity changed.** The UUID of an instance comes from its module path, its name and its namespace. If the `ModuleInstance` of the instance was deleted, or `opm` `v1.0.0-alpha.1` or older recorded the instance in a Secret, and one of the three changed since, then every object is refused as `belongs to module instance`. The refusal adds a line that says so. Annotate each object as shown. You do not have to remove anything first.
- **An object of the instance that is being deleted.** See "The object is being deleted" above.

## What a dry run shows

`opm instance apply --dry-run` and `opm module apply --dry-run` run the same check as the real apply and change nothing. A dry run prints one line for each object that the real apply does not apply:

```text
ERRO m:demo: r:ConfigMap/default/settings                      ! would refuse reason="ConfigMap/default/settings exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c0a52-8d1e-5c1b-9a61-0d4c6c2f7be2"
WARN m:demo: r:ConfigMap/default/shared                        ! would skip reason="ConfigMap/default/shared is being adopted by module instance 9a40c1de-52f0-5b0e-8c11-4f2a9f3d1e17; this instance does not apply it; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c0a52-8d1e-5c1b-9a61-0d4c6c2f7be2"
```

| Line | Meaning | Exit code of the dry run |
| --- | --- | --- |
| `would refuse` | The real apply refuses this object. The reason is the message of the table above. | 1 |
| `would skip` | Another instance adopted the object. The real apply does not apply it and goes on. | 0 |

When a dry run prints a `would refuse` line, it stops as the real apply does. It does not send the other objects to the API server, and it exits 1. A dry run that cannot read an object exits by the read error: 4, 3 or 1. A script can thus use the exit code of the dry run to find out if the apply is refused.

The dry run also shows what the apply does with objects that the module no longer renders:

| Line | Meaning | Exit code of the dry run |
| --- | --- | --- |
| `would prune` | The real apply deletes the object. | 0 |
| `would keep` | The object is not managed by OPM, or it belongs to another instance. The real apply leaves it in the cluster and removes it from the inventory. | 0 |
| `would let go` | Another instance is adopting the object. The real apply leaves it in the cluster and removes it from the inventory. | 0 |
| `cannot check` | The dry run could not read the object, so the real apply cannot delete it. | 4 when a discovery request was denied, 3 when it was unavailable, 1 otherwise |

An object that is already gone has no line. The cluster can change between the dry run and the apply, and the apply checks again.

When another instance adopted every object that the module renders, the apply has nothing to apply. It prints `nothing applied: all <n> rendered resource(s) are adopted by another instance`, writes the inventory without them and exits 0. The dry run prints `nothing would be applied` with the same reason.

`opm operator install` has no `--dry-run` flag.

## What the check does not cover

- **Operator-managed instances.** The operator applies those instances. This page describes only what the CLI does.
- **The `--rbac` objects of `opm operator install`.** They belong to no instance and are applied without this check.
- **Who can set the annotation.** Everyone who can patch an object can set `opmodel.dev/adopt` on it. With the UUID of an instance, the next apply of that instance takes the object over. With another UUID, the instance stops applying the object, and `opm operator install` refuses while one of the operator's objects carries such an annotation. The warning and the refusal name the object and the UUID each time.
