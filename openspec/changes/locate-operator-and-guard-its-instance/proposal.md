## Why

Enhancement 0028 moves `opm operator install` from the embedded manifest to a CLI-owned ModuleInstance of the operator module. Two pieces of today's CLI do not survive that move, and experiment 02 of 0028 (`enhancements/0028/experiments/02-cli-bootstrap-install`, step 7) showed both on a real cluster:

- **The running-operator check reads the embedded manifest as its locator.** `operator.CheckReady` (`internal/operator/ready.go:41-57`) parses `dist/install.yaml` and takes its CRDs and its Deployment as the readiness targets. The only caller is the operator-owned branch of `opm instance delete` (`deleteOperatorOwned`, `internal/cmd/instance/delete.go:140-149`). When a later change drops the manifest, that guard loses its locator, and without a replacement it would refuse every operator-owned delete on a cluster whose operator was applied with kubectl. 0028:D3:R13 and R15 make the locator the operator's instance record, else the fixed names every released manifest uses.
- **Deleting the operator's own instance skips the finalizer guard.** `opm operator uninstall` refuses while any ModuleInstance carries `opmodel.dev/cleanup` (`CheckFinalizerGuard`, `internal/operator/uninstall.go:64-84`). `opm instance delete opm-operator -n opm-operator-system` has no such check: in step 7 it deleted 14 operator objects while an operator-managed instance was still armed, and that instance then wedged in Terminating until install was re-run. 0028:D9:R5 says no CLI command may do that unless the user makes the same explicit choice uninstall offers.

Both land now, ahead of the module install, so the install change only has to swap what it applies: the guard and the locator already work for a manifest install and for a module install.

## What Changes

- **The running-operator check finds the operator through its instance record first.** It reads the ModuleInstance `opm-operator` in `opm-operator-system` (the operator's singleton coordinates, 0028:D3:R18). When the record exists, the CRDs and the Deployment its inventory lists are the readiness targets. When no record exists (every cluster installed today, or an operator applied with kubectl from any release's manifest), the targets are the fixed names: the Deployment `opm-operator-controller-manager` in the Namespace `opm-operator-system` and the four CRDs `moduleinstances`, `modulepackages`, `platforms` and `transformerregistrations` in `opmodel.dev`. The check no longer parses the embedded manifest. A record that lists no CRD or no Deployment, or a record read that fails for any reason other than NotFound, fails closed.
- **The fixed names become Go constants** in `internal/operator`, and a test keeps the embedded manifest equal to them for as long as the CLI embeds one. They are part of the contract once the manifest goes (0028:D2:R10).
- **`opm instance delete` of the operator's own instance meets the finalizer guard.** Before any delete, on both ownership branches and on a dry run, when the target is `opm-operator` in `opm-operator-system`, the command lists ModuleInstances cluster-wide and refuses (exit 2) while any carries `opmodel.dev/cleanup`, naming each one. A new flag, `--remove-finalizers`, is the same explicit choice `opm operator uninstall --remove-finalizers` offers: it strips that finalizer only, reports each orphaned instance, and then deletes. `--force` keeps its meaning (skip the prompt) and does not bypass the guard. Given for any other instance, `--remove-finalizers` is refused as a usage error before anything is deleted.
- The embedded manifest stays for `opm operator install`, `--crds-only` and `opm operator uninstall`; another change removes it.

**Behaviour change for users.** Deleting the instance named `opm-operator` in `opm-operator-system` with `opm instance delete` now refuses while any instance still waits on the operator's cleanup; add `--remove-finalizers` to orphan them and proceed. No cluster installed by a released CLI has that instance yet. The running-operator check finds a manifest-installed operator exactly as before.

## Dependencies / gates

**Gate: none (wave 1).** This change depends on no other 0028 change and on no new release of any repository. It reads only objects every released operator install already has. Task 1.1 records the gate check.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `operator-lifecycle`: the running-operator check locates the operator through its instance record, else by fixed names, and no longer reads the embedded manifest.
- `inventory-ownership`: the operator-owned delete's readiness check uses that locator, so it finds an operator applied with kubectl.
- `deploy`: `opm instance delete` of the operator's own instance refuses while instances carry the operator's cleanup finalizer, with `--remove-finalizers` as the explicit override.

## Impact

- **Release class: `feat`, MINOR after GA (a new flag with a safe default), shipped as the next beta.N.** No flag or exit code is removed. The new refusal applies only to the operator's own instance, which no released install creates.
- Commands: `opm instance delete` (the operator's instance on either ownership branch; operator-owned deletes through the new locator). `opm operator install` and `opm operator uninstall` are unchanged.
- Packages: `internal/operator` (`ready.go`, new `names.go`, `uninstall.go`'s `FinalizerGuardError` wording), `internal/cmd/instance` (`delete.go`).
- Tests: `internal/operator/ready_test.go`, `internal/operator/manifest_test.go`, `internal/operator/uninstall_test.go`, `internal/cmd/instance/delete_test.go`. Unit tests on the fake dynamic client are the bar: the local kind cluster has no internet egress, so cluster e2e is not run for this change and the PR says so.
- Not in this change: the module install itself, the operator module, uninstall acting on the instance's inventory (0028:D9:R3/R4/R6), removing the embedded manifest, and any ownership transfer (out of 0028's scope).
- Enhancement: `enhancement.yaml` declares 0028 without claiming a whole decision, because it delivers only 0028:D3:R13/R15 and 0028:D9:R5.
