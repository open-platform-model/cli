## Why

Every cluster running OPM today runs an operator installed from a release manifest, either by `opm operator install` (server-side apply as `opm-cli`) or by `kubectl apply`. Change `install-operator-from-module` replaces that manifest with a CLI-owned ModuleInstance of the operator module, pulled from a registry. On such a cluster its first run refuses. The apply guard stops at the first object without an OPM `managed-by` label (`internal/inventory/stale.go`, `PreApplyExistenceCheck`). Even with that guard passed, the module's Deployment selector differs from the manifest's and Kubernetes refuses to change a selector. The module's role bindings also carry new names, so the three old `*-rolebinding` objects would stay behind with nothing recording them. The design.md experiment findings measure all three. Without a migration, no existing cluster can move to the module install. The only alternative is hand work on about twenty objects: under `install-operator-from-module`, uninstall on a cluster with no operator instance record deletes nothing and refuses, naming install.

## What Changes

- `opm operator install` gains its own one-time migration of an operator installed from an earlier release manifest. It is not a flag and not a general override. It runs on every full install and does nothing on a cluster that holds no earlier manifest objects.
- **Proof list.** The CLI carries a fixed list of every object of every opm-operator release that published an install manifest, v0.5.0 to v1.0.0-beta.5: 29 objects. For each it records the kind, namespace, name and labels, plus the Deployment selector. An object is proven only when its kind and name are on the list, its labels include the listed ones and it carries no OPM instance identity.
- **Adoption.** Proven objects that the module renders pass the CLI's existing managed-by guard, in the check phase and in the instance apply alike. Every other foreign object is still refused, and install offers no override. The amended 0012:D8 lets install set the adopt annotation on exactly these proven objects; the CLI's guard does not read that annotation yet, so this change admits them through an admission set computed from the proof instead. The spec states the rule, which objects pass, and not the mechanism, so a later change can switch to the annotation without changing it.
- **Client-side installs.** For rendered objects that client-side `kubectl apply` wrote, install moves the fields of manager `kubectl-client-side-apply` to `opm-cli` after the CRD step and before the instance apply. The instance apply's forced apply then drops the labels and the `last-applied-configuration` annotation that the earlier apply set and the module does not render. Fields owned by any other manager are left alone.
- **Deployment recreate and binding delete.** Install deletes the proven earlier Deployment once, because its selector changes. It then deletes the three proven superseded `*-rolebinding` objects, each only when the render holds a binding of the same kind with the same `roleRef`. CRDs, custom resources, the Namespace and managed workloads are never touched.
- **Order and resumability.** The proof runs in install's check phase, before the apply guard and before the first write. A refused migration changes nothing. The migration's writes come after the CRD step and before the instance apply. A re-run after an interrupted migration completes it.
- **Report.** On a run that adopts, recreates or deletes anything, install names those objects and every other proven earlier-manifest object it leaves in place. A run with nothing to migrate prints no migration lines.
- **`--crds-only`.** It proves and admits the CRDs it applies and makes no other migration write, so it works on a manifest-installed cluster.

SemVer: MINOR after GA. A refusal becomes a migration; the breaking switch to the module install belongs to `install-operator-from-module`. On the beta line it ships as the next `beta.N`.

## Dependencies / gates

- **GATED: implementation starts only after `install-operator-from-module` is merged on `main`.** This change plugs into that change's install flow: its check phase (before the CRD step) and its write order (CRD step, then instance apply). Task 1.1 checks the gate.
- The cluster measurement needs a published operator module rendered through the catalog release with the seccomp profile and subject-less roles. `install-operator-from-module` already depends on that release.
- The operator's own instance stays CLI-owned. Handing it over to the operator, or any other ownership transfer, is out of scope.
- **Release coupling: UNDECIDED, owner or supervisor decision, to be recorded here and in `install-operator-from-module`'s proposal before either change is implemented.** Between the two merges, `opm operator install` refuses on every manifest-installed cluster (it changes nothing and names the earlier operator). The options are to hold cli releases from the merge of `install-operator-from-module` until this change merges, or to ship both changes in one PR.
- The user-facing migration note (what the recreate loses, and that an older CLI cannot reinstall over a migrated cluster) is written by this change in the cli `README.md` operator section (task 4.6). The `opm` docs pages are that repo's change.

## Capabilities

### New Capabilities

- `operator-migration`: install's one-time migration of an operator installed from an earlier release's manifest. It covers the proof list, adoption through the apply guard, field ownership, the Deployment recreate, the binding delete, the order of checks and writes, resumption, and the report.

### Modified Capabilities

- `apply-pruning`: the first-install existence check admits an explicit set of objects that operator install has proven, and nothing else.
- `operator-lifecycle`: install's check phase includes the migration proof before the apply guard, and its write order gains the migration writes between the CRD step and the instance apply. This modifies a requirement that `install-operator-from-module` adds, so it archives only after that change.

## Impact

- **Command:** `opm operator install`. `--crds-only` proves and admits only the CRDs and makes no migration write.
- **Packages:** `internal/operator` (the new proof list, the migration planner and executor, report lines); `pkg/inventory` (a comparable `K8sIdentity` key); `internal/inventory` (`PreApplyExistenceCheck` takes an admission set); `internal/workflow/apply` (`RunPreApplyExistenceCheck` passes the set through); `internal/cmd/operator/install.go` (report wiring); `README.md` (operator section migration note); and `hack/operator-legacy/`, a maintainer program that is not linked into `opm`.
- **Dependency:** `k8s.io/client-go/util/csaupgrade`, for the client-side ownership move. It is already in the module graph through client-go v0.36.4.
- **Testing:** unit tests on a fake dynamic client, plus an integration program covering both install origins and an interrupted run. At proposal time, kind clusters on the development host have no internet egress. The cluster measurement is a delivery criterion recorded in design.md before archive. It runs where the operator image can be pulled, or with the image preloaded.
