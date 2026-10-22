---
title: "Install over the earlier controller"
description: "A cluster that still runs the controller of the earlier install, opm-operator: why opm controller install prints a hint or a warning, how to remove the earlier controller by hand, and what happens to the CRDs and the ModuleInstances."
type: how-to
weight: 32
---

Earlier releases installed the controller as `opm-operator`, in the namespace `opm-operator-system`. `opm controller install` installs the controller as `opm-controller`, in the namespace `opm-system`. It does not migrate the earlier install and it does not remove it. The earlier controller keeps running until you remove it.

> [!WARNING]
> **Do not run two controllers**
>
> Two controllers on the same four CustomResourceDefinitions (CRDs) reconcile the same ModuleInstances. Remove the earlier controller before you install the new one.

## The message

`opm controller install` and `opm controller install --crds-only` read one Deployment, `opm-operator-controller-manager` in `opm-operator-system`. When the read succeeds and finds it, the command prints one line that names it and this page. The exit code and the text of a refusal do not change.

After a refusal, the line is the last line, and it starts with `hint:`:

```text
refusing to install: CustomResourceDefinition/moduleinstances.opmodel.dev exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=6f1c0a52-8d1e-5c1b-9a61-0d4c6c2f7be2
nothing was changed
hint: the cluster still runs the controller of the earlier install (Deployment opm-operator-controller-manager in namespace opm-operator-system); this install does not migrate it, and the opm docs page "Install over the earlier controller" says how to move to this one
```

When every check passes, for example because the four CRDs carry the adopt annotation, the same words are one warning, printed before the first change. The install then goes on. A later failure keeps the warning and has its own exit code.

When the Deployment is absent, or the read fails for any reason (for example, access is denied), nothing is printed. A cluster that lost the Deployment but kept the CRDs gets the refusal without the hint.

## Remove the earlier controller

Do the steps in this order. Workloads that the earlier controller deployed keep running while no controller runs.

1. Look at what the cluster holds:

   ```bash
   kubectl get deployment opm-operator-controller-manager --namespace opm-operator-system
   kubectl get moduleinstances.opmodel.dev --all-namespaces
   ```

2. Delete the earlier controller. The namespace `opm-operator-system` holds the Deployment, its Service, its ServiceAccount and its leader-election Role and RoleBinding, and only those:

   ```bash
   kubectl delete namespace opm-operator-system
   ```

3. Delete the cluster-scoped objects of the earlier install. Releases from `v1.0.0-alpha.18` to `v1.0.0-beta.8` created these names, and a release may have created fewer of them, so ignore a missing one:

   ```bash
   kubectl delete --ignore-not-found clusterrole \
     opm-operator-manager-role opm-operator-metrics-auth-role opm-operator-metrics-reader \
     opm-operator-moduleinstance-admin-role opm-operator-moduleinstance-editor-role opm-operator-moduleinstance-viewer-role \
     opm-operator-modulepackage-viewer-role opm-operator-platform-viewer-role \
     opm-operator-transformerregistration-admin-role opm-operator-transformerregistration-viewer-role
   kubectl delete --ignore-not-found clusterrolebinding \
     opm-operator-manager-rolebinding opm-operator-metrics-auth-rolebinding
   ```

4. Remove `spec.owner` from the ModuleInstances that hold the old value. See the section below.
5. Annotate the four CRDs for adoption. See the section on the CRDs.
6. Run `opm controller install`.

## The four CRDs

The earlier install and this install use the same four CRDs: `moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev` and `transformerregistrations.opmodel.dev`. The install refuses to apply over them, because it did not create them, and it prints the annotation to set.

Keep them. Run `opm controller install` once. It refuses, and the refusal prints the line `opmodel.dev/adopt=<UUID>` for each CRD. Set that annotation on the four CRDs, then run the install again:

```bash
kubectl annotate crd moduleinstances.opmodel.dev opmodel.dev/adopt=<UUID>
kubectl annotate crd modulepackages.opmodel.dev opmodel.dev/adopt=<UUID>
kubectl annotate crd platforms.opmodel.dev opmodel.dev/adopt=<UUID>
kubectl annotate crd transformerregistrations.opmodel.dev opmodel.dev/adopt=<UUID>
```

The install replaces the CRD schemas and keeps every ModuleInstance. Check the instances after the install: the new controller reconciles them, and this install does not test what the earlier controller left in them.

> [!CAUTION]
> **Deleting a CRD deletes every object of that kind**
>
> `kubectl delete crd moduleinstances.opmodel.dev` deletes every ModuleInstance in the cluster. An instance that holds a finalizer stays until a controller removes it, so with no controller running the delete waits. Delete the CRDs only on a cluster with nothing to keep. Then no adopt annotation is needed.

## The owner value of kept ModuleInstances

The ModuleInstance CRD of the earlier install accepted `spec.owner: cli` and `spec.owner: operator`. The CRD of this install accepts `cli` and `controller`, and an absent owner is valid in both and means controller-owned in the new one. The install replaces the schema, and the new schema then rejects `owner: operator` in a stored instance and in a manifest in Git.

So, after step 3 and before the install, remove the field from every ModuleInstance that holds `operator`:

```bash
kubectl get moduleinstances.opmodel.dev --all-namespaces \
  --output jsonpath='{range .items[?(@.spec.owner=="operator")]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}'
kubectl patch moduleinstance.opmodel.dev <name> --namespace <namespace> --type=json \
  --patch '[{"op":"remove","path":"/spec/owner"}]'
```

Remove `owner: operator` from the manifests in your GitOps repository too, or the next sync fails after the install.

## Uninstall

`opm controller uninstall` works from the record `opm-controller` in `opm-system`. An earlier install has no such record, so the command exits 2 and deletes nothing. Use the `kubectl delete` steps above.
