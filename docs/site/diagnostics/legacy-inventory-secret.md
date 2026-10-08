---
title: "Legacy inventory Secret"
description: "opm no longer reads the inventory Secret that old releases wrote: how to migrate an instance before you upgrade, and what the first-install warning means."
type: how-to
weight: 28
---

`opm` releases up to `v1.0.0-alpha.1` recorded what an instance owns in a Secret named `opm.<instance name>.<instance id>` in the instance namespace. Since `v1.0.0-alpha.2` the record is the instance's `ModuleInstance`, and releases up to `v1.0.0-beta.10` moved a Secret they found into it on the next apply.

> [!WARNING]
> **Releases after `v1.0.0-beta.10` do not read the Secret**
>
> Apply every instance that still has an inventory Secret once with `opm` `v1.0.0-beta.10`, then upgrade.

## Migrate before you upgrade

An instance needs the step only when a release up to `v1.0.0-alpha.1` applied it last. An apply since then with `v1.0.0-alpha.2` to `v1.0.0-beta.10` moved its inventory and deleted the Secret, unless that apply could not read the Secret. An instance that still has the Secret needs the step.

List the Secrets by the inventory label, and by name for a Secret that lost the label:

```bash
kubectl get secrets --all-namespaces --selector opmodel.dev/component=inventory
kubectl get secrets --all-namespaces | grep ' opm\.'
```

For each instance you find, run `opm instance apply` (or `opm module apply`) once with `opm` `v1.0.0-beta.10`. That apply writes the `ModuleInstance`, prunes what the module no longer renders, and deletes the Secret. A dry run moves nothing. Then upgrade.

To check an instance with the new release before you apply it, run the apply with `--dry-run`: it prints the warning below when the instance has no `ModuleInstance` and its resources are already in the cluster.

## The message

A release after `v1.0.0-beta.10` treats an instance without a `ModuleInstance` as a first install. When resources it renders are already in the cluster and carry the OPM managed-by label, `opm instance apply` and `opm module apply` print one warning on standard error, before anything is applied. The exit code does not change.

A dry run has written nothing, so it names the migration step:

```text
WARN 2 of 3 rendered resource(s) already exist and are managed by OPM, but the instance has no ModuleInstance record. A real apply would update them in place and record them; it would prune nothing, so a resource an earlier apply created and this render no longer produces would stay in the cluster untracked. If opm v1.0.0-alpha.1 or older last applied this instance, its inventory is in a Secret this release does not read: apply the instance once with opm v1.0.0-beta.10 before you apply it with this release
```

A real apply goes on and writes the record, so it names the Secret to keep:

```text
WARN 2 of 3 rendered resource(s) already exist and are managed by OPM, but the instance has no ModuleInstance record. This apply updates them in place and records them; it prunes nothing, so a resource an earlier apply created and this render no longer produces stays in the cluster untracked. If opm v1.0.0-alpha.1 or older last applied this instance, the Secret "opm.blog.0b9f3c1e-6a53-5f0b-9d0e-3a8f6f0c1d2e" in this namespace still lists what it owned: keep it, do not apply this instance with an older opm, and remove the leftovers as the opm docs page "Legacy inventory Secret" says
```

## What it means

Apply found resources that an earlier OPM apply created, and no record that lists them. It goes on: it updates the resources it renders, and writes a new `ModuleInstance` that records exactly those. It deletes nothing. It cannot prune, because it does not know what else the earlier apply created.

## Causes and fixes

### An old release recorded the instance in a Secret

You applied with the new release before the migration step. Nothing that runs was removed, and the instance now has a `ModuleInstance`. Resources that the old inventory listed and the module no longer renders are left over, and no later apply removes them.

> [!CAUTION]
> Do not apply this instance with `v1.0.0-beta.10` or older now. Those releases find the new `ModuleInstance`, skip the Secret, and delete it unread. The Secret is the only list of what the old inventory held.

Clean up by hand, in this order:

1. Read the old inventory from the Secret. Its `inventory` key holds a JSON record with one entry per resource (`group`, `kind`, `namespace`, `name`):

   ```bash
   kubectl get secret opm.<instance name>.<instance id> --namespace <namespace> --output jsonpath='{.data.inventory}' | base64 --decode
   ```

2. Compare the entries with what `opm instance status` lists for the instance. Delete, with `kubectl delete`, each resource the Secret lists and the instance does not.
3. Delete the Secret with `kubectl delete secret`.

When the Secret is gone, select by the instance, never by the managed-by label alone: other instances in the namespace carry that label too. The resources of one instance carry `module-instance.opmodel.dev/name=<instance name>`.

### An earlier first apply stopped part way

When a first apply fails on one resource, it writes no `ModuleInstance`. The next apply finds the resources the first one created and prints the warning. No action is needed: this apply records them.

### The `ModuleInstance` was deleted and its resources were kept

Apply records the rendered resources again. Resources that the deleted record listed and the module no longer renders stay in the cluster. Find them by the label `module-instance.opmodel.dev/name=<instance name>` and compare them with `opm instance status`.
