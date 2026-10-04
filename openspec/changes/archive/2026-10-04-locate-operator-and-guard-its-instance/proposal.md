## Why

The operator is moving from an install manifest embedded in the CLI to an OPM module of its own: `opm operator install` will deploy a CLI-owned `ModuleInstance` named `opm-operator` in `opm-operator-system`, pulled from a registry, and the CLI will stop carrying the manifest. Two pieces of today's CLI do not survive that move. A trial of the module install on a kind cluster (2026-10-04, summarised in design.md) showed both:

- **The running-operator check reads the embedded manifest as its locator.** `operator.CheckReady` (`internal/operator/ready.go:41-57`) parses `dist/install.yaml` and takes its CRDs and its Deployment as the readiness targets. Its only caller is the operator-owned branch of `opm instance delete` (`deleteOperatorOwned`, `internal/cmd/instance/delete.go:140-149`). The check has no locator once the manifest is removed.
- **Deleting the operator's own instance skips the finalizer guard.** `opm operator uninstall` refuses while any ModuleInstance carries `opmodel.dev/cleanup` (`CheckFinalizerGuard`, `internal/operator/uninstall.go:64-84`). `opm instance delete opm-operator -n opm-operator-system` has no such check: in the trial it deleted 14 operator objects while an operator-managed instance was still armed, and that instance later wedged in Terminating until the operator was reinstalled.

Both land now, ahead of the module install, so that change only swaps what install applies: the locator and the guard already work for a manifest install and for a module install.

## What Changes

- **The running-operator check finds the operator by its fixed names.** The targets are the Deployment `opm-operator-controller-manager` in `opm-operator-system` and the CRDs `moduleinstances`, `modulepackages`, `platforms` and `transformerregistrations` in `opmodel.dev`. These are the names of every operator release since `v1.0.0-alpha.18`, and the operator module keeps them. The check no longer parses the embedded manifest and reads no instance record.
- **The fixed names become Go constants** in `internal/operator`, with a test that keeps the embedded manifest equal to them for as long as the CLI embeds one.
- **`opm instance delete` of an instance that deploys the operator is guarded.** Such an instance is recognised from its record by any of three signals: the coordinates `opm-operator`/`opm-operator-system`, the module path `opmodel.dev/modules/opm_operator` in any major, or a recorded CRD of the operator's API group `opmodel.dev`. Before any delete, and on a dry run:
  - an operator-owned record of such an instance is refused (exit 2) with the remedy "set `spec.owner` to `cli`", because the operator never reconciles or prunes the instance that deploys it, so the operator-owned delete path would report a cleanup that does not happen;
  - otherwise the command lists ModuleInstances cluster-wide and refuses (exit 2) while any carries `opmodel.dev/cleanup`, naming each one and pointing at `opm operator uninstall --remove-finalizers`, the existing explicit choice to orphan them. `--force` does not bypass either refusal, and `instance delete` gains no finalizer-stripping flag.
- The embedded manifest stays for `opm operator install`, `--crds-only` and `opm operator uninstall`; a later change removes it.

**Behaviour change for users.** `opm instance delete` of an instance that deploys the operator now refuses while it is operator-owned or while any instance still waits on the operator's cleanup. No cluster installed by a released CLI has such an instance yet. The running-operator check finds a manifest-installed operator exactly as before.

## Dependencies / gates

**Gate: none (wave 1).** This change depends on no other change in any repository and on no new release. It reads only objects every operator install since `v1.0.0-alpha.18` already has. Task 1.1 records the gate check.

Related changes, none of them blocking: opm-operator `refuse-own-instance` recognises the operator's own instance by the same three signals and refuses an operator-owned one with the same remedy; cli `install-operator-from-module` (later wave) replaces what install applies and relies on this change's fixed names and guard.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `operator-lifecycle`: the running-operator check locates the operator by its fixed names and no longer reads the embedded manifest.
- `inventory-ownership`: the operator-owned delete's readiness check uses those fixed names, so it finds an operator applied with kubectl.
- `deploy`: `opm instance delete` of an instance that deploys the operator refuses while that instance is operator-owned, and while instances carry the operator's cleanup finalizer, pointing at `opm operator uninstall`.

## Impact

- **Release class: MINOR after GA; ships as beta.N+1.** No flag or exit code is added or removed. The new refusals apply only to an instance that deploys the operator, which no released install creates.
- Commands: `opm instance delete` (the operator's instance on either ownership branch; operator-owned deletes through the fixed-name check). `opm operator install` and `opm operator uninstall` are unchanged.
- Packages: `internal/operator` (`ready.go`, new `names.go`, `uninstall.go`'s `FinalizerGuardError` wording), `internal/cmd/instance` (`delete.go`).
- Tests: `internal/operator/ready_test.go`, `manifest_test.go`, `names_test.go`, `uninstall_test.go`, `internal/cmd/instance/delete_test.go`. Unit tests on the fake dynamic client are the bar: the local kind cluster has no internet egress, so cluster e2e is not run for this change and the PR says so.
- Not in this change: the module install, the operator module, uninstall acting on the instance's inventory, removing the embedded manifest, and any transfer of an instance's ownership (the CLI only names the `spec.owner` remedy; it writes no owner).
- No enhancement entry backs this change; it is self-contained. Context it builds on: archived 0006:D34 (uninstall's finalizer guard and its `--remove-finalizers` override) and 0006:D18 (ownership as the delete's branch point).
