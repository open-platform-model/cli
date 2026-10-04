## Context

See proposal.md (Why) for the two gaps. The code they live in today:

- `operator.CheckReady` (`internal/operator/ready.go:41-57`) calls `EmbeddedManifest()`, keeps its `CustomResourceDefinition` and `Deployment` documents (`readinessTargets`, lines 63-72) and Gets each one live through `pendingObjects` (`internal/operator/wait.go:40-49`), where any Get error counts as pending and `DefaultPredicate` decides readiness (CRD `Established`, workload rollout). It returns `*NotReadyError` listing the pending objects.
- Its one caller is `deleteOperatorOwned` (`internal/cmd/instance/delete.go:140-149`), which fills the error's `Hint` and returns exit 2. A repository-wide search (`grep -rn CheckReady`) finds no other caller, so "every CLI command that checks for a running operator" (0028:D3:R13/R15) is this one guard today.
- `runInstanceDelete` (`delete.go:89-128`) resolves the target, prompts unless `--force` or `--dry-run`, reads the record with `query.ResolveInventory` (which exits 5 when no record exists, `internal/workflow/query/status.go:52-60`), then branches on `inventory.ResolveOwnership`. Neither branch looks at other instances' finalizers.
- `CheckFinalizerGuard`, `RemoveCleanupFinalizer` and `FinalizerGuardError` (`internal/operator/uninstall.go:47-142`) are uninstall's guard. `FinalizerGuardError.Error()` hard-codes "refusing to uninstall".
- `inventory.GetRecord` (`internal/inventory/store.go:30-39`) returns `(nil, nil)` on NotFound and a wrapped error otherwise. A dynamic Get against a resource whose CRD is not installed reaches the API server as an unknown path and comes back 404, which `apierrors.IsNotFound` recognises, so "no CRD" and "no record" take the same branch. The fake dynamic client the unit tests use also returns NotFound for an object it does not hold.
- The fixed names are what `dist/install.yaml` declares today: Namespace `opm-operator-system`, Deployment `opm-operator-controller-manager`, CRDs `moduleinstances`, `modulepackages`, `platforms`, `transformerregistrations` in `opmodel.dev`. `TestEmbeddedManifest_CRDNamesAreExpected` (`internal/operator/manifest_test.go:31-48`) already pins the four CRD names. 0028's experiment 01 rendered the operator module with the same names for instance `opm-operator` in `opm-operator-system`, and 0028:D2:R10 makes them part of the contract.

## Goals / Non-Goals

**Goals:**

- The running-operator check works unchanged whether the operator came from a manifest or will come from the module, and stops depending on the embedded manifest.
- No CLI path deletes the operator's own instance past an armed `opmodel.dev/cleanup` finalizer without the user's explicit `--remove-finalizers`.

**Non-Goals:**

- The module install, the operator module, removing the embedded manifest, and `opm operator uninstall` acting on the instance's inventory (0028:D9:R3/R4/R6). Uninstall keeps its manifest-driven delete set and its existing guard.
- Ownership transfer of any instance. 0028 puts it out of scope, and this change neither adds nor touches an owner write.
- A generic cross-instance finalizer guard for other instances. Only the operator's instance runs the operator that clears the finalizer.

## Decisions

### 1. Locator: record first, fixed names second

```go
// internal/operator/names.go
const (
    OperatorInstanceName     = "opm-operator" // 0028:D3:R18
    OperatorNamespace        = "opm-operator-system"
    ControllerDeploymentName = "opm-operator-controller-manager"
)

// CRDNames are the CRDs every released operator install serves (0028:D2:R10).
var CRDNames = []string{
    "moduleinstances.opmodel.dev", "modulepackages.opmodel.dev",
    "platforms.opmodel.dev", "transformerregistrations.opmodel.dev",
}

// IsOperatorInstance reports whether name/namespace are the operator's own instance.
func IsOperatorInstance(name, namespace string) bool

// internal/operator/ready.go
func CheckReady(ctx context.Context, client *kubernetes.Client) error {
    targets, err := locateOperator(ctx, client)
    if err != nil {
        return err // record read failed: fail closed, not a NotReadyError
    }
    pending := pendingObjects(ctx, client, targets, DefaultPredicate)
    ...
}

// locateOperator returns the objects whose readiness defines a running operator.
func locateOperator(ctx context.Context, client *kubernetes.Client) ([]*unstructured.Unstructured, error) {
    rec, err := inventory.GetRecord(ctx, client, OperatorInstanceName, OperatorNamespace)
    switch {
    case err != nil:
        return nil, fmt.Errorf("reading the operator's instance record %s/%s: %w", OperatorNamespace, OperatorInstanceName, err)
    case rec == nil:
        return fixedTargets(), nil
    default:
        return recordTargets(rec) // NotReadyError when it lists no CRD or no Deployment
    }
}
```

`recordTargets` turns inventory entries into unstructured stubs: group `apiextensions.k8s.io` kind `CustomResourceDefinition` and group `apps` kind `Deployment`, with the entry's recorded version or `v1` when the entry has none. `fixedTargets` builds the same stubs from the constants. `readinessTargets` and `CheckReady`'s call to `EmbeddedManifest()` go; `readinessTargets` has no other user.

**Options considered:**

1. Fixed names only. Simpler, and equal to the record today because 0028:D2:R10 keeps the names. Not chosen: 0028:D3:R13 asks for the record, and the record is the one source that follows a later module version adding a CRD.
2. Record only. Not chosen: every cluster installed today has no record (0028:D3:R15).
3. Record, falling back to fixed names whenever the record is unusable (empty inventory, read error). Not chosen: a record that exists but lists no Deployment is an install that did not finish, and a read error proves nothing. Falling back would let the guard pass on names the record contradicts. Failing closed matches today's `pendingObjects`, where any read error counts as pending.

**Error surfaces.** A not-ready operator keeps `*NotReadyError` and its wording (`the opm operator is not ready (...) — <hint>; install or repair it with 'opm operator install', then retry`). A record with no Deployment or CRD entry is a `*NotReadyError` whose pending line is `ModuleInstance/opm-operator in opm-operator-system (records no operator Deployment)` (or `CustomResourceDefinition`). A record read error is a plain wrapped error; `deleteOperatorOwned` maps it with `cmdutil.ExitCodeFromK8sError` (permission denied 4, connectivity 3) and keeps exit 2 for `*NotReadyError`.

### 2. Finalizer guard on `opm instance delete` of the operator's instance

```go
// internal/cmd/instance/delete.go, in runInstanceDelete after ResolveInventory
if err := guardOperatorInstanceDelete(ctx, k8sClient, inv, removeFinalizers, dryRun); err != nil {
    return err
}
// then the existing ownership branch

func guardOperatorInstanceDelete(ctx context.Context, c *kubernetes.Client, inv *inventory.Record, removeFinalizers, dryRun bool) error {
    if !operator.IsOperatorInstance(inv.Name, inv.Namespace) {
        if removeFinalizers {
            return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: errRemoveFinalizersScope}
        }
        return nil
    }
    armed, err := operator.CheckFinalizerGuard(ctx, c)
    if err != nil {
        return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
    }
    if len(armed) == 0 {
        return nil
    }
    if !removeFinalizers {
        return &opmexit.ExitError{Code: opmexit.ExitValidationError,
            Err: &operator.FinalizerGuardError{Armed: armed, Action: "delete the operator's instance opm-operator-system/opm-operator"}}
    }
    if dryRun {
        // list "would remove opmodel.dev/cleanup from <ns/name>" per instance; patch nothing
        return nil
    }
    return operator.RemoveCleanupFinalizer(ctx, c, armed) // exit 1 on any failure, nothing deleted
}
```

`FinalizerGuardError` gains `Action string`; `Error()` reads `refusing to <Action>: ...` and uses `uninstall` when `Action` is empty, so uninstall's message and its spec scenarios are unchanged.

The guard runs after `ResolveInventory` because only then is the target's name known for every identifier form (a UUID or a file carries it only through the record), and it runs before the ownership branch so both branches are covered. The confirmation prompt still comes first, as it does today for the operator-owned readiness refusal; a refusal after a "yes" deletes nothing.

**Which instance is "the operator's".** By its coordinates, `opm-operator` in `opm-operator-system`, which 0028:D3:R18 fixes and 0028:D3:R11 makes a singleton. Matching the record's module path (`opmodel.dev/modules/opm_operator`) was considered and not chosen: the path's major is not settled until the module's first release (0028:D1), the experiment's module used a test path, and a ModuleInstance at those coordinates runs the operator whatever path it names.

**Override shape.** 0028:D9 allows either uninstall's refusal with its override or a refusal that points at uninstall. Pointing at uninstall was not chosen: today's uninstall deletes the manifest's object list and leaves an instance record behind, so until the uninstall change lands it is not a way to delete the instance. A `--remove-finalizers` flag on `instance delete` gives the user the same explicit choice in the command they ran. It is scoped to the operator's instance so it cannot become a general finalizer-stripping tool (Principle VII); on any other instance it is a usage error rather than a silent no-op.

### Command syntax and flags

`opm instance delete <file|name|uuid> [flags]`, new flag:

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--remove-finalizers` | bool | false | Only when deleting the operator's own instance: strip the operator's cleanup finalizer from every ModuleInstance that still carries it, orphaning their workloads, and proceed |

Exit codes: 0 deleted; 1 `--remove-finalizers` on another instance, or a finalizer strip failed; 2 refused by the finalizer guard or the readiness guard; 3/4 connectivity or permission error from the list or the record read; 5 no record (unchanged).

### Example output

```text
$ opm instance delete opm-operator -n opm-operator-system --force
Error: refusing to delete the operator's instance opm-operator-system/opm-operator: 1 instance(s) still carry the opmodel.dev/cleanup finalizer: default/hello (use --remove-finalizers to proceed; this orphans their workloads)
exit 2

$ opm instance delete opm-operator -n opm-operator-system --force --remove-finalizers
WARN removed opmodel.dev/cleanup finalizer from default/hello — its workload is now orphaned (no longer cleaned up by the operator)
... per-resource delete lines, CRDs and the Namespace left behind ...

$ opm instance delete hello -n default --remove-finalizers
Error: --remove-finalizers applies only to the operator's instance opm-operator-system/opm-operator
exit 1
```

### Data flow

```text
instance delete ─ resolve target ─ prompt? ─ ResolveInventory (exit 5 if none)
                                             │
                          guardOperatorInstanceDelete (operator's instance only)
                                             │
                    ┌────────── ResolveOwnership ──────────┐
               CLI-owned                              operator-owned
          executeInstanceDelete             CheckReady ─ locateOperator
                                             record? ── yes ─ inventory CRDs + Deployments
                                                     └─ no ── fixed names
```

## Research & Decisions

### Where the running-operator check is used

**Context**: 0028:D3:R13/R15 bind "every CLI command that checks for a running operator".
**Explored**: `grep -rn "CheckReady\|EmbeddedManifest()" --include=*.go` in the cli tree, 2026-10-04.
**Options considered**:
1. Add the locator only to `deleteOperatorOwned` - matches the only caller.
2. Introduce a public `LocateOperator` used by install's wait as well - install's wait targets the applied plan, which is what it should wait on, and the module install rewrites it anyway.
**Decision**: Option 1, keeping `CheckReady` as the single entry point so any later caller gets the locator.
**Rationale**: `CheckReady` is the one call site (`internal/cmd/instance/delete.go:141`); `EmbeddedManifest()` is also called by install (`internal/operator/install.go:149`) and uninstall (`uninstall.go:184`), which keep the manifest until a later change removes it.

### Behaviour observed in experiment 02

**Context**: The guard and the locator are requirements because a real cluster showed the gaps.
**Explored**: `enhancements/0028/experiments/02-cli-bootstrap-install/README.md` step 7.
**Options considered**: n/a; the experiment is the evidence.
**Decision**: Guard the operator's instance on `instance delete`; locate through the record.
**Rationale**: Step 7 deleted 14 operator objects past an armed `default/hello`, which then wedged in Terminating until install ran again; the readiness guard there located the operator only through the embedded manifest's names.

## Risks / Trade-offs

- [A tenant user who may delete their own operator-owned instance but may not read ModuleInstances in `opm-operator-system` is now refused with a permission error.] → Today the same user already needs to read the CRDs and the Deployment in `opm-operator-system`, and the `opm-cli-user` ClusterRole that `opm operator install --rbac` emits grants every verb on `moduleinstances` cluster-wide. The error names the read that failed.
- [The record path is not exercised against a real cluster in this change.] → The local kind cluster has no internet egress, so the PR states that only unit tests on the fake dynamic client ran. The module install change exercises the record path end to end.
- [A user who names another instance `opm-operator` in `opm-operator-system` gets the guard.] → That namespace belongs to the operator, and 0028:D3:R11 makes the coordinates the operator's singleton.
- [The refusal comes after the confirmation prompt.] → Same as the existing readiness refusal; nothing is deleted.

## Migration Plan

None. A cluster installed today has no operator instance record, so the locator takes the fixed-name branch, which reads the same objects the manifest named. Rollback is reverting the change.
