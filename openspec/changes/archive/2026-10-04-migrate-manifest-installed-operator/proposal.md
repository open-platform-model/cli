## Why

Every cluster running OPM today runs an operator installed from a release manifest, either by `opm operator install` (server-side apply as `opm-cli`) or by `kubectl apply`. Change `install-operator-from-module` replaces that manifest with a CLI-owned ModuleInstance of the operator module, pulled from a registry. On such a cluster its first run refuses. The apply guard stops at the first object without an OPM `managed-by` label (`internal/inventory/stale.go`, `PreApplyExistenceCheck`). Even with that guard passed, the module's Deployment selector differs from the manifest's and Kubernetes refuses to change a selector. The module's role bindings also carry new names, so the three old `*-rolebinding` objects would stay behind with nothing recording them. The design.md experiment findings measure all three. Without a migration, no existing cluster can move to the module install. The only alternative is hand work on about twenty objects: under `install-operator-from-module`, uninstall on a cluster with no operator instance record deletes nothing and refuses, naming install.

## What Changes

- `opm operator install` gains its own one-time migration of an operator installed from an earlier release manifest. It is not a flag and not a general override. It runs on every full install and does nothing on a cluster that holds no earlier manifest objects.
- **Proof list.** The CLI carries a fixed list of every object of every opm-operator operator release (tag `v<semver>`) that published an install manifest under the names the module keeps: v0.5.0 to v1.0.0-beta.8 as measured on 2026-10-04, 32 objects. The three releases before v0.5.0 that attach a manifest (v0.4.2 to v0.4.4) install a `poc-controller` in `poc-controller-system`, another operator the migration does not take over. It never reads a release of the operator module (tag `opm_operator-vX.Y.Z`): that release's `install.yaml` is a render of the module, whose objects carry the operator instance's identity and are its own, not foreign. For each it records the kind, namespace, name and labels, plus the Deployment selector. An object is proven only when its kind and name are on the list, its labels include the listed ones and it carries no OPM instance identity.
- **Adoption.** Proven objects that the module renders pass the CLI's existing managed-by guard, in the check phase and in the instance apply alike. Every other foreign object is still refused, and install offers no override. This is 0012:D8:R6 as amended in enhancements PR #94, and 0012:D8:R3 names it as the one exception to the ownership refusals that 0012:D1:R7 and 0012:D4:R2 apply: installing the operator admits, as if adopted, exactly the proven objects, and no frontend sets the adopt annotation on the user's behalf. The CLI admits them through an admission set computed from the proof and writes nothing to mark them.
- **Client-side installs.** For rendered objects that client-side `kubectl apply` wrote, install moves the fields of manager `kubectl-client-side-apply` to `opm-cli` after the CRD step and before the instance apply. The instance apply's forced apply then drops the labels and the `last-applied-configuration` annotation that the earlier apply set and the module does not render. Fields owned by any other manager are left alone.
- **Deployment recreate and binding delete.** Install deletes the proven earlier Deployment once, because its selector changes. It then deletes the three proven superseded `*-rolebinding` objects, each only when the render holds a binding of the same kind with the same `roleRef`. CRDs, custom resources, the Namespace and managed workloads are never touched. These are the only deletions outside an instance's inventory that 0012:D8:R7 allows, and the exceptions that 0012:D1:R4 and 0012:D4:R1 now name.
- **Order and resumability.** The proof runs in install's check phase, before the apply guard and before the first write. A refused migration changes nothing. The migration's writes come after the CRD step and before the instance apply. A re-run after an interrupted migration completes it.
- **Report.** On a run that adopts, recreates or deletes anything, install names those objects and every other proven earlier-manifest object it leaves in place. A run with nothing to migrate prints no migration lines.
- **`--crds-only`.** It proves and admits the CRDs it applies and makes no other migration write, so it works on a manifest-installed cluster.

SemVer: MINOR after GA. A refusal becomes a migration; the breaking switch to the module install belongs to `install-operator-from-module`. On the beta line it ships as the next `beta.N`.

## Dependencies / gates

- **Stacked on `install-operator-from-module` (cli PR #307).** This change plugs into that change's install flow: the proof slot its check phase leaves immediately before the apply guard, and its write order (CRD step, then instance apply). Its branch is merged into this one and this change's PR targets #307's branch; the two merge back to back (see "Release coupling"). Task 1.1 checks the gate.
- **GATED: enhancements PR #94 is merged**, so this change can declare 0012:D8:R6 and 0012:D8:R7 (task 1.1). `install-operator-from-module` already gates on it.
- The cluster measurement needs a published operator module. `install-operator-from-module` already depends on it: rendered through the catalog_opm release with the pod seccomp profile (catalog_opm PR #141), with the five unbound ClusterRoles rendered through `objects@v1alpha1` until catalog_opm's `add-subjectless-roles` is released.
- The operator's own instance stays CLI-owned. Handing it over to the operator, or any other ownership transfer, is out of scope.
- **Release coupling (decided 2026-10-04).** This change and `install-operator-from-module` ship in the same cli release. Between the two merges, `opm operator install` refuses on every manifest-installed cluster (it changes nothing and names the earlier operator), so the cli release PR is held from the merge of `install-operator-from-module` until this change merges. Workers never touch the release PR; this change's PR body says that its merge releases the hold.
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
- **Testing:** unit tests on a fake dynamic client, plus a kind e2e test covering both install origins, a refusal and an interrupted run. At proposal time, kind clusters on the development host have no internet egress. The cluster measurement is a delivery criterion recorded in design.md before archive. It runs where the operator image can be pulled, or with the image preloaded.
