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

An instance needs the step only when a release up to `v1.0.0-alpha.1` applied it last. Any apply since then with `v1.0.0-alpha.2` to `v1.0.0-beta.10` has already moved its inventory.

List the Secrets by their name:

```bash
kubectl get secrets --all-namespaces | grep ' opm\.'
```

For each instance you find, run `opm instance apply` (or `opm module apply`) once with `opm` `v1.0.0-beta.10`. That apply writes the `ModuleInstance`, prunes what the module no longer renders, and deletes the Secret. A dry run moves nothing. Then upgrade.

## The message

A release after `v1.0.0-beta.10` treats an instance without a `ModuleInstance` as a first install. When resources it renders are already in the cluster and carry the OPM managed-by label, apply prints this warning once, on standard error, before it applies. The exit code does not change.

```text
WARN 2 of 3 rendered resource(s) already exist and are managed by OPM, but the instance has no ModuleInstance record. Apply updates them in place and records them; it prunes nothing, so a resource an earlier apply created and this render no longer produces stays in the cluster untracked. If opm v1.0.0-alpha.1 or older last applied this instance, its inventory is in a Secret this release does not read: apply the instance once with opm v1.0.0-beta.10 first
```

A dry run does not check the cluster for existing resources and prints no such warning.

## What it means

Apply found resources that an earlier OPM apply created, and no record that lists them. It goes on: it updates the resources it renders, and writes a new `ModuleInstance` that records exactly those. It deletes nothing. It cannot prune, because it does not know what else the earlier apply created.

## Causes and fixes

### An old release recorded the instance in a Secret

You upgraded before the migration step. Nothing that runs was removed, and the instance now has a `ModuleInstance`. Two things are left over, and no later apply cleans them up:

- The Secret `opm.<instance name>.<instance id>`. Delete it with `kubectl delete secret`.
- Resources the old inventory listed and the module no longer renders. They still carry the label `app.kubernetes.io/managed-by` with the value `opm-cli` or `open-platform-model`. Compare the resources with that label in the namespace against `opm instance status`, and delete by hand what the instance does not list.

To avoid the manual cleanup, run the migration step above before the first apply with the new release.

### An earlier first apply stopped part way

When a first apply fails on one resource, it writes no `ModuleInstance`. The next apply finds the resources the first one created and prints the warning. No action is needed: this apply records them.

### The `ModuleInstance` was deleted and its resources were kept

Apply records the resources again. Resources the deleted record listed and the module no longer renders stay in the cluster; find them by the label as above.
