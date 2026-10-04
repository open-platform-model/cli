# Tasks: locate-operator-and-guard-its-instance

Three sections, each one commit. Section 1 checks the gate and names the operator's fixed coordinates without changing behaviour; section 2 moves the running-operator check onto its instance record and the fixed names (0028:D3:R13/R15); section 3 adds the finalizer guard to `opm instance delete` of the operator's own instance (0028:D9:R5). Each section writes its failing tests first. Cluster e2e is not part of any gate here: the local kind cluster has no internet egress, so unit tests on the fake dynamic client are the bar, and the PR says so.

## 1. Gate check and the operator's fixed names (internal/operator)

- [ ] 1.1 Gate check. This change's gate is **none (wave 1)**: confirm it depends on no other 0028 change and on no unreleased version by checking that `internal/operator/dist/install.yaml` on this branch declares the Namespace `opm-operator-system`, the Deployment `opm-operator-controller-manager` and the CRDs `moduleinstances`, `modulepackages`, `platforms` and `transformerregistrations` in `opmodel.dev` (`grep -n "^  name:" internal/operator/dist/install.yaml`), and that `go.mod` needs no bump. Verify: the names match and no dependency moves; write a one-line result under this task.
- [ ] 1.2 Red first. In `internal/operator/manifest_test.go`, extend `TestEmbeddedManifest_CRDNamesAreExpected` (or add `TestEmbeddedManifest_UsesTheFixedNames`) to assert the manifest's CRD names equal `CRDNames`, and that its one `Deployment` is `ControllerDeploymentName` in `OperatorNamespace` and its one `Namespace` is `OperatorNamespace`. Add `TestIsOperatorInstance` (table: `opm-operator`/`opm-operator-system` true; `opm-operator`/`default` false; `hello`/`opm-operator-system` false). Verify: the tests fail to compile on today's code.
- [ ] 1.3 Add `internal/operator/names.go` per design.md § 1: `OperatorInstanceName`, `OperatorNamespace`, `ControllerDeploymentName`, `CRDNames`, `IsOperatorInstance`, with a doc comment that cites 0028:D2:R10 and 0028:D3:R18 once and says the names are a contract that outlives the embedded manifest. Verify: `go test ./internal/operator -run 'TestEmbeddedManifest|TestIsOperatorInstance'` passes.
- [ ] 1.4 `task fmt`, `task lint` and `task test` green, then commit `refactor(operator): name the operator's fixed coordinates`.

## 2. The running-operator check locates the operator through its record, else its fixed names (internal/operator, internal/cmd/instance)

- [ ] 2.1 Red first, in `internal/operator/ready_test.go` on `fakeClientWith` (`wait_test.go:72-86`, which already registers the ModuleInstance list kind) with `crdFixture`/`deploymentFixture`-style objects:
  - `TestCheckReady_NoRecordUsesFixedNames`: four Established CRDs and a rolled-out `opm-operator-controller-manager`, no ModuleInstance: nil.
  - `TestCheckReady_NoRecordMissingDeploymentIsNotReady`: same without the Deployment: `*NotReadyError` naming `opm-operator-controller-manager`.
  - `TestCheckReady_RecordTargetsComeFromItsInventory`: a ModuleInstance `opm-operator` in `opm-operator-system` whose inventory lists a fifth CRD `extras.opmodel.dev` (not Established) plus the four CRDs and the Deployment, all others ready: `*NotReadyError` naming `extras.opmodel.dev`.
  - `TestCheckReady_RecordReadyOperatorPasses`: the same record with everything ready: nil.
  - `TestCheckReady_RecordWithoutDeploymentFailsClosed`: a record whose inventory lists only CRDs, while a ready Deployment exists under the fixed name: `*NotReadyError` naming the record, proving there is no fallback.
  - `TestCheckReady_RecordReadErrorFailsClosed`: a reactor returning `apierrors.NewForbidden` on `get moduleinstances`: a non-`NotReadyError` error that names `opm-operator-system/opm-operator`, and `apierrors.IsForbidden` holds through the wrap.
  - Delete `TestReadinessTargets_SelectsCRDsAndTheControllerDeployment` with `readinessTargets`; keep `TestCheckReady_AbsentOperatorIsNotReady` and `TestNotReadyError_IncludesTheCallerHint`.
  Verify: the new tests fail on today's code.
- [ ] 2.2 `internal/operator/ready.go` per design.md § 1: `locateOperator`, `recordTargets`, `fixedTargets`; `CheckReady` stops calling `EmbeddedManifest()`; remove `readinessTargets`; rewrite `CheckReady`'s doc comment (record first, fixed names second, fail closed), citing 0028:D3:R13/R15 once. Verify: `go test ./internal/operator/...` passes and `grep -n EmbeddedManifest internal/operator/ready.go` finds nothing.
- [ ] 2.3 `internal/cmd/instance/delete.go` `deleteOperatorOwned`: keep exit 2 for `*NotReadyError`; map any other `CheckReady` error with `cmdutil.ExitCodeFromK8sError`. Add `TestDeleteOperatorOwned_RecordReadErrorUsesKubernetesExitCode` in `delete_test.go` (Forbidden reactor: exit code 4, nothing deleted). Verify: `go test ./internal/cmd/instance/...` passes, including the two existing readiness tests.
- [ ] 2.4 `task fmt`, `task lint` and `task test` green, then commit `feat(operator): locate the running operator through its instance record or fixed names`.

## 3. Instance delete of the operator's instance meets the finalizer guard (internal/operator, internal/cmd/instance)

- [ ] 3.1 Red first.
  - `internal/operator/uninstall_test.go`: `TestFinalizerGuardError_DefaultsToUninstall` (empty `Action` keeps today's `refusing to uninstall:` text) and `TestFinalizerGuardError_NamesTheAction`.
  - `internal/cmd/instance/delete_test.go`, with `moduleInstanceFixture`-style ModuleInstances on a fake dynamic client and a CLI-owned record `opm-operator` in `opm-operator-system` tracking a ConfigMap:
    - `TestGuardOperatorInstanceDelete_RefusesWhileArmed`: `default/hello` carries `opmodel.dev/cleanup`; exit 2, message names `default/hello`, `--remove-finalizers` and `orphans`; the ConfigMap and the record still exist.
    - `TestGuardOperatorInstanceDelete_DryRunRefusesToo`.
    - `TestGuardOperatorInstanceDelete_OverrideStripsOnlyTheCleanupFinalizer`: `default/hello` carries `opmodel.dev/cleanup` and `example.com/keep`; after the guard only `example.com/keep` remains and the output names `default/hello` as orphaned.
    - `TestGuardOperatorInstanceDelete_DryRunOverridePatchesNothing`.
    - `TestGuardOperatorInstanceDelete_NoArmedInstancePasses`.
    - `TestGuardOperatorInstanceDelete_ListFailureFailsClosed`: Forbidden on `list moduleinstances`: exit 4.
    - `TestGuardOperatorInstanceDelete_OverrideOnOtherInstanceIsUsageError`: record `hello` in `default` with `removeFinalizers`: exit 1, no patch.
    - `TestGuardOperatorInstanceDelete_OtherInstancesAreNotGuarded`: record `hello` in `default`, an armed `demo/x`: nil.
    - `TestDeleteRemoveFinalizersFlag`: the flag exists, defaults to false, and its usage names the operator's instance and orphaning.
  Verify: the new tests fail to compile or fail on today's code.
- [ ] 3.2 `internal/operator/uninstall.go`: add `FinalizerGuardError.Action` and the `refusing to <Action>` wording with `uninstall` as the default. Verify: `go test ./internal/operator/...` passes and `internal/cmd/operator` tests are unchanged.
- [ ] 3.3 `internal/cmd/instance/delete.go` per design.md § 2: the `--remove-finalizers` flag, `guardOperatorInstanceDelete` called in `runInstanceDelete` right after `query.ResolveInventory` and before the ownership branch, the dry-run listing, and a `Long` help paragraph stating that deleting the operator's own instance refuses while instances carry the operator's cleanup finalizer, with `--remove-finalizers` as the override and its orphaning consequence, plus one example. No enhancement reference in help text or output strings. Verify: `go test ./internal/cmd/instance/...` passes and `go run ./cmd/opm instance delete --help` shows the paragraph and the flag.
- [ ] 3.4 Cross-cutting: `task openspec:check` and `task docs:bundle:check` (the generated command reference picks up the new flag and help text). Note in the task that cluster e2e (`task test:e2e`, `task test:integration`) was not run because the local kind cluster has no internet egress. Verify: both green.
- [ ] 3.5 `task fmt`, `task lint` and `task test` green, then commit `feat(instance): refuse deleting the operator's instance while instances wait on its cleanup` with a body for the changelog: deleting the instance opm-operator in opm-operator-system with opm instance delete now refuses while any instance carries the operator's cleanup finalizer; --remove-finalizers strips that finalizer, orphaning those instances' workloads, and proceeds.
