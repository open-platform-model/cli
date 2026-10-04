## Why

Every cluster running OPM today runs an operator installed from a release manifest, either by `opm operator install` (server-side apply as `opm-cli`) or by `kubectl apply`. Change `install-operator-from-module` replaces that manifest with a CLI-owned ModuleInstance of the operator module, pulled from a registry. On such a cluster its first run refuses. The apply guard stops at the first object without an OPM `managed-by` label (`internal/inventory/stale.go`, `PreApplyExistenceCheck`). Even with that guard passed, the module's Deployment selector differs from the manifest's and Kubernetes refuses to change a selector. The module's role bindings also carry new names, so the three old `*-rolebinding` objects would stay behind with nothing recording them. The design.md experiment findings measure all three. Without a migration, no existing cluster can move to the module install, and the only alternatives are hand work on about twenty objects or an uninstall, which refuses while any instance still carries its cleanup finalizer.

## What Changes

- `opm operator install` gains its own one-time migration of an operator installed from an earlier release manifest. It is not a flag and not a general override. It runs on every full install and does nothing on a cluster that holds no earlier manifest objects.
- **Proof list.** The CLI carries a fixed list of every object of every opm-operator release that published an install manifest, v0.5.0 to v1.0.0-beta.5: 29 objects. For each it records the kind, namespace, name and labels, plus the Deployment selector. An object is proven only when its kind and name are on the list, its labels include the listed ones and it carries no OPM instance identity.
- **Adoption.** Proven objects that the module renders are admitted through the CLI's existing managed-by guard by an explicit admission set. Every other foreign object is still refused, and install offers no override. No adopt annotation is written. The 0012:D8 annotation, which install may set on proven objects under that decision's amendment, has no frontend implementation yet. The observable rule is the same either way.
- **Client-side installs.** For objects installed with client-side `kubectl apply`, install moves field ownership from `kubectl-client-side-apply` to `opm-cli` before its first apply. The module's forced apply then drops the labels and the `last-applied-configuration` annotation that the earlier manifest set and the module does not render.
- **Deployment recreate and binding delete.** Install deletes the proven earlier Deployment once, because its selector changes. It also deletes the three proven superseded `*-rolebinding` objects. CRDs, custom resources, the Namespace and managed workloads are never touched.
- **Order and resumability.** The proof runs with install's other refusing checks, before the first write. A refused migration changes nothing. A re-run after an interrupted migration completes it.
- **Report.** Install names the adopted objects, the recreated Deployment, the deleted bindings and every other earlier-manifest object that it leaves in place.

SemVer: MINOR after GA. A refusal becomes a migration; the breaking switch to the module install belongs to `install-operator-from-module`. On the beta line it ships as the next `beta.N`.

## Dependencies / gates

- **GATED: implementation starts only after `install-operator-from-module` is merged on `main`.** This change plugs into that change's install flow: its check phase (before the CRD step) and its write order (CRD step, then instance apply). Task 1.1 checks the gate.
- The cluster measurement needs a published operator module rendered through the catalog release with the seccomp profile and subject-less roles. `install-operator-from-module` already depends on that release.
- The operator's own instance stays CLI-owned. Handing it over to the operator, or any other ownership transfer, is out of scope.

## Capabilities

### New Capabilities

- `operator-migration`: install's one-time migration of an operator installed from an earlier release's manifest. It covers the proof list, adoption through the apply guard, field ownership, the Deployment recreate, the binding delete, the order of checks and writes, resumption, and the report.

### Modified Capabilities

None. The migration is specified on its own. The install flow it plugs into is specified by `install-operator-from-module`.

## Impact

- **Command:** `opm operator install` (full form). `--crds-only` performs no migration.
- **Packages:** `internal/operator` (the new proof list, the migration planner and executor, report lines); `internal/inventory` (`PreApplyExistenceCheck` takes an admission set); `internal/workflow/apply` (`RunPreApplyExistenceCheck` passes the set through); `internal/cmd/operator/install.go` (report wiring); and `hack/operator-legacy/`, a maintainer program that is not linked into `opm`.
- **Dependency:** `k8s.io/client-go/util/csaupgrade`, for the client-side ownership move. It is already in the module graph through client-go v0.36.4.
- **Testing:** unit tests on a fake dynamic client, plus an integration program covering both install origins and an interrupted run. At proposal time, kind clusters on the development host have no internet egress. The cluster measurement is a delivery criterion recorded in design.md before archive. It runs where the operator image can be pulled, or with the image preloaded.
