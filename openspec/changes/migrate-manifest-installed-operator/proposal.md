## Why

Every cluster running OPM today runs an operator installed from a release manifest, by `opm operator install` (server-side apply as `opm-cli`) or by `kubectl apply`. Once `opm operator install` deploys the operator module as a CLI-owned ModuleInstance (change `install-operator-from-module`, 0028:D3), its first run on such a cluster refuses: the apply guard stops at the first object without an OPM `managed-by` label (`internal/inventory/stale.go`, `PreApplyExistenceCheck`), and even past that guard the Deployment's selector is immutable and the three old role bindings would stay behind unrecorded (measured in 0028 experiment 02, step 6). 0028:D8 makes that first install migrate the operator in place; this change implements it.

## What Changes

- `opm operator install` gains install's own migration of a manifest-installed operator (0028:D8). It is not a flag and not a general override: it runs on every install and does nothing on a cluster with no earlier manifest objects.
- **Proof list.** The CLI carries a fixed list of the kind, namespace, name and labels of every object of every opm-operator release that published an install manifest (v0.5.0 to v1.0.0-beta.5: 29 objects), and the Deployment selector those manifests set. An object is proven only when its kind and name are on the list, its labels include the listed ones and it carries no OPM instance identity (0028:D8:R8, R14).
- **Adoption.** Proven objects the module renders are admitted through the CLI's existing managed-by guard by an explicit admission set; every other foreign object is still refused, and install offers no override. No adopt annotation is written: 0012:D8's annotation is not implemented in any frontend yet (0028 06-operational.md).
- **Deployment recreate and binding delete.** Install deletes the proven earlier Deployment once (selector change) and the three proven superseded `*-rolebinding` objects (0028:D8:R6, R7, R13).
- **Order and resumability.** The proof runs with install's other refusing checks before the first write; the deletes run after the CRD step and before the instance apply. A re-run after an interrupted migration completes it (0028:D8:R11, R12).
- **Report.** Install names the adopted objects, the recreated Deployment, the deleted bindings and every other earlier-manifest object left in place (0028:D8:R9).
- **Client-side installs.** For objects installed with client-side `kubectl apply`, install moves field ownership from `kubectl-client-side-apply` to `opm-cli` before the instance apply, so the module's apply drops fields the earlier manifest set and the module does not.

SemVer: MINOR after GA (a refusal becomes a migration; the breaking switch to the module install is `install-operator-from-module`'s). On the beta line it ships as the next `beta.N`.

## Dependencies / gates

- **GATED: after `install-operator-from-module` is merged.** This change plugs into that change's two-step install: its check phase (before the CRD step) and its write order (CRD step, then migration writes, then instance apply). Implementation starts only once that change is on `main`; task 1.1 checks it.
- The delivery measurement needs a published operator module whose render goes through D12's catalog release (seccomp profile, subject-less roles), which `install-operator-from-module` already depends on.
- Handoff and ownership transfer are out of scope (0028:D4).

## Capabilities

### New Capabilities

- `operator-migration`: install's one-time migration of an operator installed from an earlier release's manifest: the proof list, adoption through the apply guard, the Deployment recreate, the binding delete, the order of checks and writes, resumption, and the report.

### Modified Capabilities

None. The migration is specified on its own; the install flow it plugs into is specified by `install-operator-from-module`.

## Impact

- **Command:** `opm operator install` (full form). `--crds-only` deletes nothing and is not a migration.
- **Packages:** `internal/operator` (new proof list and migration planner/executor, report lines), `internal/inventory` (`PreApplyExistenceCheck` takes an admission set), `internal/workflow/apply` (`RunPreApplyExistenceCheck` passes it through), `internal/cmd/operator/install.go` (report wiring).
- **Dependency:** `k8s.io/client-go/util/csaupgrade` (already in the module graph via client-go v0.36.4) for the client-side ownership move.
- **Testing:** unit tests on a fake dynamic client; an integration program for both install origins and an interrupted run. The local kind cluster has no internet egress at proposal time, so the cluster measurement 0028's graduation note requires is a delivery criterion recorded at archive, run where the operator image can be pulled or preloaded.
