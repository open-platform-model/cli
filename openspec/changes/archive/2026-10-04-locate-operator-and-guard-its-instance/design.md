## Context

See proposal.md (Why) for the two gaps. The code they live in today:

- `operator.CheckReady` (`internal/operator/ready.go:41-57`) calls `EmbeddedManifest()`, keeps its `CustomResourceDefinition` and `Deployment` documents (`readinessTargets`, lines 63-72) and Gets each one live through `pendingObjects` (`internal/operator/wait.go:40-49`), where any Get error counts as pending and `DefaultPredicate` decides readiness (CRD `Established`, workload rollout). It returns `*NotReadyError` listing the pending objects.
- Its one caller is `deleteOperatorOwned` (`internal/cmd/instance/delete.go:140-149`), which fills the error's `Hint` and returns exit 2. A repository-wide search (`grep -rn CheckReady`, 2026-10-04) finds no other caller, so "every CLI command that checks for a running operator" is this one guard today.
- `runInstanceDelete` (`delete.go:89-128`) resolves the target, builds its client through `cmdutil.NewK8sClient` (line 100, so a test cannot inject a fake), prompts unless `--force` or `--dry-run`, reads the record with `query.ResolveInventory` (which exits 5 when no record exists, `internal/workflow/query/status.go:52-60`), then branches on `inventory.ResolveOwnership`. Neither branch looks at other instances' finalizers. Today's tests call `deleteOperatorOwned` and `executeInstanceDelete` directly (`delete_test.go:51-200`).
- On the operator-owned branch, a record with `spec.prune: true` makes the CLI print `Instance deleted — operator pruned N resources` once the CR is gone (`delete.go:188-190`).
- `CheckFinalizerGuard`, `RemoveCleanupFinalizer` and `FinalizerGuardError` (`internal/operator/uninstall.go:47-142`) are uninstall's guard (archived 0006:D34). `FinalizerGuardError.Error()` hard-codes "refusing to uninstall" and "use --remove-finalizers to proceed".
- `inventory.Record` already carries `ModulePath` (`spec.module.path`) and the recorded inventory entries, so recognising the operator's instance needs no render.
- The fixed names are what `dist/install.yaml` declares today: Namespace `opm-operator-system`, Deployment `opm-operator-controller-manager`, CRDs `moduleinstances`, `modulepackages`, `platforms`, `transformerregistrations` in `opmodel.dev`. `TestEmbeddedManifest_CRDNamesAreExpected` (`internal/operator/manifest_test.go:31-48`) already pins the four CRD names. The fourth CRD, `transformerregistrations`, first shipped in opm-operator `v1.0.0-alpha.18` (commit `06b197a`, 2026-09-15); every later release carries all four.
- The operator's `refuse-own-instance` change (opm-operator) makes the operator never add its cleanup finalizer to the instance that deploys it, release a leftover one without pruning, and refuse an operator-owned one with reason `SelfManagementRefused` and the message "set `spec.owner` to `cli`".

### Evidence from the module-install trials (2026-10-04)

Two throwaway experiments, run on a local kind cluster against a draft of the operator module, back this change. Their material is not in any repository, so the findings this change relies on are stated here.

- **Render trial.** A draft operator module, rendered with no cluster and no Platform for the instance `opm-operator` in `opm-operator-system`, produced 19 objects whose Namespace, Deployment, ServiceAccount, Service, roles and CRDs carry exactly the names of today's manifest: every name derives from the instance's name and namespace, and the manifest's name prefix equals `opm-operator-`. Only the three role bindings took catalog-derived names. The four CRDs do not vary with the instance. So the fixed names below are the same for a manifest install and a module install.
- **Install trial, delete step.** With the operator installed as the CLI-owned instance `opm-operator` and an operator-managed fixture `default/hello` carrying `opmodel.dev/cleanup`, `opm instance delete opm-operator -n opm-operator-system --force` (and its `--dry-run`) deleted 14 objects, listed the Namespace and the four CRDs as left behind, and never mentioned `hello`. `hello` kept a stale `Ready=True` and its finalizer. A later `kubectl delete mi hello --wait=false` wedged in Terminating. Re-running install brought the operator back (14 created, 5 unchanged), it ran the pending cleanup, and `hello` was gone 18 s after install returned.
- **Install trial, readiness guard.** With the operator gone, `opm instance delete hello` refused with exit 2 (`the opm operator is not ready (Deployment/opm-operator-controller-manager in opm-operator-system) …`), so the existing per-instance guard works; it located the operator only through the embedded manifest's names.
- **Install trial, self-ownership.** Flipping the operator's instance to `owner: operator` by hand let the operator add its cleanup finalizer to its own instance; with cluster-admin it adopted all 19 objects, and deleting that instance with `spec.prune: true` made it prune its own Deployment and die mid-cleanup, leaving the record stuck on a finalizer only the dead operator could clear. This is why the guard below also covers the target itself and refuses an operator-owned record of the operator's module outright.

## Goals / Non-Goals

**Goals:**

- The running-operator check works unchanged whether the operator came from a manifest or will come from the module, and stops depending on the embedded manifest.
- No CLI path deletes an instance that deploys the operator past an armed `opmodel.dev/cleanup` finalizer unless the user has made uninstall's explicit choice to orphan those instances.
- No CLI path deletes an instance that deploys the operator through the operator-owned branch, which would wait on, and report, a cleanup the operator never performs for that instance.

**Non-Goals:**

- The module install, the operator module, removing the embedded manifest, and `opm operator uninstall` acting on the instance's inventory. Uninstall keeps its manifest-driven delete set and its existing guard and override.
- Ownership transfer of any instance. This change neither adds nor touches an owner write; it only names `spec.owner: cli` as the remedy, as the operator does.
- A generic cross-instance finalizer guard for other instances. Only the operator's instance runs the operator that clears the finalizer.

## Decisions

### 1. Locator: fixed names

```go
// internal/operator/names.go
const (
    OperatorInstanceName     = "opm-operator"
    OperatorNamespace        = "opm-operator-system"
    ControllerDeploymentName = "opm-operator-controller-manager"
    OperatorAPIGroup         = "opmodel.dev"
    OperatorModulePath       = "opmodel.dev/modules/opm_operator" // without @<major>
)

// CRDNames are the CRDs every operator release since v1.0.0-alpha.18 serves.
var CRDNames = []string{
    "moduleinstances.opmodel.dev", "modulepackages.opmodel.dev",
    "platforms.opmodel.dev", "transformerregistrations.opmodel.dev",
}

// internal/operator/ready.go
func CheckReady(ctx context.Context, client *kubernetes.Client) error {
    pending := pendingObjects(ctx, client, fixedTargets(), DefaultPredicate)
    ...
}
```

`fixedTargets` builds unstructured stubs from the constants: `apiextensions.k8s.io/v1` `CustomResourceDefinition` for each of `CRDNames` (cluster-scoped), and `apps/v1` `Deployment` `ControllerDeploymentName` in `OperatorNamespace`. `readinessTargets` and `CheckReady`'s call to `EmbeddedManifest()` go; `readinessTargets` has no other user. The `Namespace` is not a target: it is only the Deployment's namespace, as it is today. Error surfaces do not change: a not-ready operator keeps `*NotReadyError` and its wording (`the opm operator is not ready (...) — <hint>; install or repair it with 'opm operator install', then retry`), and any Get error still counts as pending.

**Options considered:**

1. Fixed names only. Chosen. The render trial shows the module keeps every name the check reads, and the module install change's downgrade check already reads the live Deployment at the same fixed names. It adds no read, no permission and no failure path the check does not have today.
2. Record first, fixed names second. Not chosen: the only gain is following a later module version that adds a CRD or renames the Deployment, which no planned version does. It costs three extra fail-closed paths, an exit-4 path for a tenant who cannot read `moduleinstances` in `opm-operator-system`, and a dead end after `opm operator uninstall` (the record still lists the Deployment uninstall removed, so every operator-owned delete refused). When a module version renames something, that change updates the constants.
3. Record only. Not chosen: every cluster installed today has no record.

### 2. Guard on `opm instance delete` of an instance that deploys the operator

**Which instance deploys the operator.** Any one of three signals, read from the record without rendering:

```go
// internal/operator/names.go
// DeploysOperator reports whether rec is an instance that deploys the operator,
// and which signal matched ("coordinates", "module path", "inventory CRD").
func DeploysOperator(rec *inventory.Record) (signal string, ok bool)
```

1. coordinates: name `opm-operator`, namespace `opm-operator-system`;
2. module path: `rec.ModulePath` with any `@<major>` suffix removed equals `opmodel.dev/modules/opm_operator`;
3. inventory CRD: an entry of group `apiextensions.k8s.io`, kind `CustomResourceDefinition`, whose name after its first `.` is exactly `opmodel.dev`.

These are the signals the operator's own `refuse-own-instance` change uses to recognise the instance it must never reconcile, so the CLI and the operator agree on what "the operator's instance" is. Coordinates alone (the earlier draft of this change) were not chosen: a record of the operator module installed under another name, or one whose inventory holds the operator's CRDs, removes the same operator when deleted and strands the same finalizers. The module path is the one the operator module's own change (`add-operator-module` in opm-operator) declares, `opmodel.dev/modules/opm_operator@v0`; stripping the major keeps the check valid when it moves to `v1`. Signal 3 compares the whole suffix, so `widgets.example.opmodel.dev.io` does not match.

**Testable seam.** The part of `runInstanceDelete` after `ResolveInventory` moves into a function that takes the client, so the guard's wiring is tested and not only the guard:

```go
// internal/cmd/instance/delete.go
func runInstanceDelete(...) error {
    ... // resolve target, build client, prompt, ResolveInventory (unchanged)
    return deleteResolvedInstance(ctx, k8sClient, rsf, namespace, inv, liveResources, timeout, dryRun, instanceLog)
}

// deleteResolvedInstance guards an instance that deploys the operator, then
// branches on ownership (0006:D18).
func deleteResolvedInstance(ctx context.Context, c *kubernetes.Client, rsf *cmdutil.InstanceSelectorFlags, namespace string,
    inv *inventory.Record, live []*unstructured.Unstructured, timeout time.Duration, dryRun bool, log *log.Logger) error {
    if err := guardOperatorInstanceDelete(ctx, c, inv); err != nil {
        return err
    }
    if inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned {
        return deleteOperatorOwned(ctx, c, inv, timeout, dryRun, log)
    }
    return executeInstanceDelete(ctx, c, rsf, namespace, inv, live, dryRun, log)
}
```

**The guard.**

```go
func guardOperatorInstanceDelete(ctx context.Context, c *kubernetes.Client, inv *inventory.Record) error {
    signal, ok := operator.DeploysOperator(inv)
    if !ok {
        return nil
    }
    if inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned {
        return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: &operator.OwnInstanceOwnerError{
            Namespace: inv.Namespace, Name: inv.Name, Signal: signal,
        }}
    }
    armed, err := operator.CheckFinalizerGuard(ctx, c)
    if err != nil {
        return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
    }
    if len(armed) == 0 {
        return nil
    }
    return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: &operator.FinalizerGuardError{
        Armed:  armed,
        Action: fmt.Sprintf("delete %s/%s, which deploys the operator", inv.Namespace, inv.Name),
        Target: operator.ArmedInstance{Namespace: inv.Namespace, Name: inv.Name},
        Remedy: "run 'opm operator uninstall --remove-finalizers' to remove that finalizer, orphaning their workloads, then retry",
    }}
}
```

**Operator-owned record of the operator's instance.** The operator never adds its cleanup finalizer to the instance that deploys it and, if one is left over, removes it without pruning. The operator-owned branch would therefore delete the CR, wait, and with `spec.prune: true` print "operator pruned N resources" when nothing was pruned, which the `inventory-ownership` rule "MUST NOT claim a prune it has not established" forbids; and with the operator down it refuses on readiness for an instance whose delete needs no operator. So such a record is refused before the ownership branch, with exit 2 and `OwnInstanceOwnerError`:

```text
refusing to delete <ns>/<name>: it deploys the operator (matched by <signal>) and is operator-owned, but the operator never reconciles or prunes the instance that deploys it; set spec.owner to cli (kubectl patch moduleinstance <name> -n <ns> --type=merge -p '{"spec":{"owner":"cli"}}'), then retry
```

The remedy is the one the operator's `SelfManagementRefused` condition names, so the two read the same. Once the owner is `cli`, the delete takes the CLI-owned branch, which deletes the recorded objects itself and still meets the finalizer guard. This check runs before the cluster-wide list, so it needs no extra permission. The CLI does not write the owner itself: transferring ownership is not part of this change.

**Finalizer guard.** `FinalizerGuardError` gains `Action`, `Target` and `Remedy`. `Error()` reads `refusing to <Action>: N instance(s) still carry the opmodel.dev/cleanup finalizer: <list> (<Remedy>)`; the list marks `Target` with ` (the instance being deleted)` when it is armed. Empty `Action` and `Remedy` give today's `uninstall` and `use --remove-finalizers to proceed; this orphans their workloads`, so uninstall's message and its scenarios are unchanged.

The guard runs after `ResolveInventory` because only then is the target's name, module and owner known for every identifier form (a UUID or a file carries them only through the record), and before the ownership branch so both branches are covered. It runs on a dry run with the same outcome, because a dry run that passes where the real run refuses misleads. The confirmation prompt still comes first, as it does today for the operator-owned readiness refusal; a refusal after a "yes" deletes nothing.

**Override shape.** Uninstall's guard refuses while any instance is armed unless the user passes `--remove-finalizers`, which strips that finalizer only and orphans those instances' workloads. Two ways to give `instance delete` the same explicit choice were weighed:

1. A `--remove-finalizers` flag on `opm instance delete`, scoped to the operator's instance. Not chosen: it adds a finalizer-stripping flag to the generic delete, which every user sees, to serve one instance; scoping it needs a usage error for every other instance; and it duplicates a choice `opm operator uninstall` already owns.
2. Refuse and point at `opm operator uninstall --remove-finalizers`. Chosen. The choice stays in the one command that owns it. The path works today: uninstall with the flag strips the finalizers (including a leftover one on the target) and deletes the operator's manifest objects; a following `opm instance delete` of the leftover CLI-owned record then finds nothing armed and takes the CLI-owned branch, which needs no running operator. Once uninstall acts on the instance's inventory (a later change), it deletes the record itself and the pointer is the whole path.

`--force` keeps its meaning (skip the prompt) and bypasses neither refusal.

### Command syntax and flags

`opm instance delete <file|name|uuid> [flags]`: no flag added or removed. A `Long` help paragraph states both refusals and their remedies.

Exit codes: 0 deleted; 2 refused (operator-owned instance of the operator, finalizer guard, or readiness guard); 3/4 connectivity or permission error from the cluster-wide list; 5 no record (unchanged); 1 any other failure (unchanged).

### Example output

```text
$ opm instance delete opm-operator -n opm-operator-system --force
Error: refusing to delete opm-operator-system/opm-operator, which deploys the operator: 1 instance(s) still carry the opmodel.dev/cleanup finalizer: default/hello (run 'opm operator uninstall --remove-finalizers' to remove that finalizer, orphaning their workloads, then retry)
exit 2

$ opm instance delete ops -n platform --force      # CLI-owned, module path opmodel.dev/modules/opm_operator@v0
Error: refusing to delete platform/ops, which deploys the operator: 2 instance(s) still carry the opmodel.dev/cleanup finalizer: default/hello, platform/ops (the instance being deleted) (run 'opm operator uninstall --remove-finalizers' …)
exit 2

$ opm instance delete opm-operator -n opm-operator-system --force      # spec.owner: operator
Error: refusing to delete opm-operator-system/opm-operator: it deploys the operator (matched by coordinates) and is operator-owned, … set spec.owner to cli (…), then retry
exit 2
```

### Data flow

```text
instance delete ─ resolve target ─ prompt? ─ ResolveInventory (exit 5 if none)
                                             │
                                  deleteResolvedInstance
                                             │
                     guardOperatorInstanceDelete (DeploysOperator only)
                        operator-owned? ── yes ─ refuse, exit 2 (set spec.owner: cli)
                        any armed?      ── yes ─ refuse, exit 2 (opm operator uninstall --remove-finalizers)
                                             │
                    ┌────────── ResolveOwnership ──────────┐
               CLI-owned                              operator-owned
          executeInstanceDelete             CheckReady ─ fixed names
```

## Research & Decisions

### Where the running-operator check is used

**Context**: The requirement binds "every CLI command that checks for a running operator before it acts".
**Explored**: `grep -rn "CheckReady\|EmbeddedManifest()" --include=*.go` in the cli tree, 2026-10-04.
**Options considered**:
1. Change the targets inside `CheckReady` - matches the only caller and covers any later one.
2. Introduce a public `LocateOperator` used by install's wait as well - install's wait targets the applied plan, which is what it should wait on, and the module install rewrites it anyway.
**Decision**: Option 1, keeping `CheckReady` as the single entry point.
**Rationale**: `CheckReady` has one call site (`internal/cmd/instance/delete.go:141`); `EmbeddedManifest()` is also called by install (`internal/operator/install.go:149`) and uninstall (`uninstall.go:184`), which keep the manifest until a later change removes it.

### Since when the fixed names hold

**Context**: The check must find every operator a user may still run.
**Explored**: `git log --diff-filter=A -- config/crd/bases/opmodel.dev_transformerregistrations.yaml` and `git tag --contains` in opm-operator, 2026-10-04.
**Decision**: The check requires all four CRDs; an operator older than `v1.0.0-alpha.18` is reported as not ready.
**Rationale**: This is not a behaviour change: the embedded manifest `CheckReady` reads today already requires all four. An operator that old cannot serve the CLI's current API anyway; reporting it as not ready points the user at `opm operator install`, which is the right fix.

## Risks / Trade-offs

- [A later module version renames the Deployment or adds a CRD the check should wait on.] → No planned version does; such a change updates the constants in `names.go` in the same release, and the manifest test catches drift while the manifest is embedded.
- [A user who names an unrelated instance `opm-operator` in `opm-operator-system` gets the guard.] → That namespace belongs to the operator, and the operator applies the same rule to that instance.
- [Pointing at uninstall makes deleting a leftover operator record a two-step path while other instances are armed.] → The user is removing the operator, which is what uninstall is for; the first step is the explicit choice, and the second needs no flag.
- [Deleting an operator-owned record of the operator's instance becomes a two-step path (set the owner, then delete).] → The operator already refuses that record and names the same step; the CLI-owned branch is the one that actually deletes the objects.
- [The refusal comes after the confirmation prompt.] → Same as the existing readiness refusal; nothing is deleted.

## Migration Plan

None. The fixed names are the objects the manifest named, so a cluster installed today sees the same check, and no instance there deploys the operator by any of the three signals. Rollback is reverting the change.
