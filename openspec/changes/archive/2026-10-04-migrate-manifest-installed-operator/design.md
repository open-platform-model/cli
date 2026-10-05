## Context

proposal.md, Why, describes the problem. The code this change touches, as read on `main` at 048d6807:

- **The apply guard.** `inventory.PreApplyExistenceCheck` (`internal/inventory/stale.go`) GETs every rendered entry on a first apply. It refuses at the first object without an OPM `app.kubernetes.io/managed-by` value (`pkgcore.IsOPMManagedBy`). It logs a read error other than NotFound at debug level and skips the object. `apply.RunPreApplyExistenceCheck` (`internal/workflow/apply/apply.go`) skips the guard entirely when a previous inventory exists or on a dry run. The guard admits any OPM-managed object, whichever instance it belongs to.
- **The writes.** `kubernetes.ApplyOne` server-side-applies with field manager `opm-cli` and `Force: true` (`internal/kubernetes/apply.go`, `internal/kubernetes/labels.go`). `inventory.PruneStaleResources` deletes with foreground propagation and never deletes a CRD or a Namespace (`kubernetes.IsProtectedKind`). `kubernetes.WaitAbsent` waits for objects to disappear; `operator.waitForTerminating` in `internal/operator/install.go` uses it.
- **The install this change plugs into** (`install-operator-from-module`, cli PR #307, merged into this branch). `operator.PlanInstall` (`internal/operator/plan_install.go`) is the check phase and writes nothing: the record read, the values merge and render, `CheckTarget`, `inventory.GateStatusRBAC` (full install only), `waitForTerminating`, a comment marking the migration proof slot, and last, only when the operator instance has no record (`rec == nil`), `inventory.PreApplyExistenceCheck` over `workflowapply.CurrentInventoryEntries(plan.Objects())`, wrapped as `*operator.GuardError`. `plan.Objects()` is the whole render, or the four CRDs under `--crds-only`, so the `--crds-only` flow reaches the same slot and the same guard. `operator.Install` (`internal/operator/install.go`) is the write phase: the CRD step (`kubernetes.ApplyOne` per CRD, then `kubernetes.Wait` for Established), a comment marking the migration writes slot, then, unless `CRDsOnly`, `workflowapply.Execute` with a `workflowapply.Request` (no namespace creation, the operator ceiling skipped). `Execute` reaches `RunPreApplyExistenceCheck(ctx, client, hasPrevInventory, dryRun, currentEntries)` as its gate 6, which runs the guard only when the instance has no previous inventory. `kubernetes.Apply` server-side-applies every object, the CRDs first (`splitClusterDefinitions`), as field manager `opm-cli` with `Force: true` (`applyOne` in `internal/kubernetes/apply.go`), so the instance apply applies the CRDs again with force after the CRD step. The module 0.1.0 render (operator v1.0.0-beta.8) holds 22 objects: the 19 of the beta.5 manifest, renamed bindings aside, plus the ClusterRoles `opm-operator-{modulepackage,platform,transformerregistration}-viewer-role`.
- **Identity labels.** `pkg/core/labels.go` defines `module-instance.opmodel.dev/{name,namespace,uuid}`. The operator module's render stamps them, and `app.kubernetes.io/managed-by: opm-cli`, on every object.

### What the earlier manifests contain (measured 2026-10-04)

These figures come from `dist/install.yaml` at every one of the 47 opm-operator tags and from the GitHub release assets:

- Only operator releases (tags `v<semver>`) were read; the operator module's releases (tags `opm_operator-vX.Y.Z`, from opm-operator's `release-operator-module`) also attach an `install.yaml`, but it is a render of the module and is never a source for this list. `hack/operator-legacy` found 46 operator releases with an `install.yaml` asset: v0.4.2 to v1.0.0-beta.8 (v0.2.0 to v0.4.1 publish none). From v0.2.0 to v0.4.4 every name is `poc-controller-*` in `poc-controller-system`; from v0.5.0 every name carries the `opm-operator` prefix. The list starts at v0.5.0 (`FirstLegacyRelease`): v0.4.2 to v0.4.4 install another operator, in another namespace, under names the module never renders, and taking them over is a non-goal.
- The union over v0.5.0 to v1.0.0-beta.8 is 32 objects. 22 belong to the beta.8 manifest, which adds the ClusterRoles `opm-operator-{modulepackage,platform,transformerregistration}-viewer-role` to beta.5's 19. The other 10 shipped only in older releases:
  - the ClusterRoles `opm-operator-bundlerelease-{admin,editor,viewer}-role` (v0.5.0 to v0.6.4);
  - the ClusterRoles `opm-operator-modulerelease-{admin,editor,viewer}-role` (v0.5.0 to v0.7.5);
  - the CRDs `bundlereleases`, `modulereleases`, `releases` and `platforms` in group `releases.opmodel.dev`.
- Every object keeps the same labels in every release that ships it. The Deployment keeps the selector `{app.kubernetes.io/name: opm-operator, control-plane: controller-manager}` throughout.
- The only role bindings any release ships are `ClusterRoleBinding opm-operator-manager-rolebinding`, `ClusterRoleBinding opm-operator-metrics-auth-rolebinding` and `RoleBinding opm-operator-system/opm-operator-leader-election-rolebinding`.
- 11 of the current manifest's 19 objects carry `app.kubernetes.io/managed-by: kustomize`. The other 8 carry no managed-by label: the four CRDs, the ClusterRoles `manager-role`, `metrics-auth-role` and `metrics-reader`, and the ClusterRoleBinding `metrics-auth-rolebinding`, each under the `opm-operator-` prefix. None carries an OPM instance identity, so each is foreign to the new instance under the current guard.
- The 10 manifests the CLI has embedded (alpha.2 to beta.5, from `git log -- internal/operator/dist/install.yaml`) agree with the tag trees.

### How the operator module differs from the manifest (render experiment, 2026-10-04)

A catalog-path rendering of the operator module was compared with the beta.5 `install.yaml`:

- It renders the same 19 objects, with the same kinds and namespaces, and 16 of the 19 names. The four CRDs are byte-equal.
- **Bindings.** The catalog's role abstraction names each binding after its role. The module therefore renders `ClusterRoleBinding opm-operator-manager-role`, `ClusterRoleBinding opm-operator-metrics-auth-role` and `RoleBinding opm-operator-system/opm-operator-leader-election-role`. The manifest names them `*-rolebinding`.
- **Deployment selector.** The selector becomes `{app.kubernetes.io/name: controller-manager, component.opmodel.dev/name: controller-manager, core.opmodel.dev/workload-type: stateless, module-instance.opmodel.dev/name: opm-operator}`; the manifest's is `{app.kubernetes.io/name: opm-operator, control-plane: controller-manager}`. The catalog always adds the instance and component keys, so no module setting avoids the change. The pods keep the `control-plane: controller-manager` label.
- **Labels.** Every object loses `app.kubernetes.io/managed-by: kustomize` and `app.kubernetes.io/name: opm-operator`, and the Namespace loses `control-plane: controller-manager`. Every object gains the OPM labels.

### What install does today on a manifest-installed cluster (install experiment, 2026-10-04)

The experiment ran on kind v0.32.0 with node v1.36.1. It emulated the module install (CRD step, then `opm instance apply` of a CLI-owned instance) over today's `opm operator install`, which applies as `opm-cli`:

1. **First apply.** It refused with `pre-apply existence check failed: resource Namespace/opm-operator-system … is not managed by OPM`. It adopted nothing and created no duplicate. The emulated CRD step had already server-side-applied the four CRDs before the refusal, which shows that the guard must run before the CRD step.
2. **After every old object was relabelled `managed-by=opm-cli`,** standing in for adoption, the apply reported `applied 18 resources … 3 created, 11 configured` and then `Deployment … spec.selector … field is immutable`. It skipped the prune and wrote no inventory. The old operator kept running. The 3 created objects were the module's renamed bindings, which now sat beside the old ones.
3. **After the Deployment was deleted by hand,** the apply reported `applied 19 resources successfully (1 created, 18 unchanged)` and became healthy. That took 16.3 s, delete included. Every other object kept its uid. Both installs apply as `opm-cli`, so the old kustomize labels were dropped cleanly. The three old `*-rolebinding` objects were left with nothing recording them: functional duplicates that nothing would ever prune.

Not measured: a client-side `kubectl apply` origin. Its objects are owned by field manager `kubectl-client-side-apply`, so step 3's clean label drop does not carry over (see "Field ownership of client-side installs").

In the same experiment, an upgrade that rolled the operator pod produced a 28 s gap between the new pod starting and its controllers starting. Leader-lease handover is the likely cause; the experiment did not confirm it. `opm instance apply --wait` reported healthy before that gap ended. The Deployment recreate will show at least that gap.

## Goals / Non-Goals

**Goals:** the migration specified in `specs/operator-migration/spec.md`, measured over an `opm-cli` install, a client-side `kubectl apply` install and an interrupted run.

**Non-Goals:**

- Writing the 0012:D8 adopt annotation. 0012:D8:R6, as amended in enhancements PR #94, says no frontend sets it on the user's behalf and that installing the operator admits the proven objects as if adopted. This change does exactly that, through an admission set, and writes nothing to mark the objects.
- Closing the guard's general gap: it admits another instance's OPM-managed object. The migration refuses such an object when it is on the proof list; for other objects the guard is unchanged here.
- Migrating `poc-controller-*` installs (v0.2.0 to v0.4.4). Releases v0.4.2 to v0.4.4 published a manifest with those names, but they install another operator in another namespace, so they are not on the list.
- Ownership transfer of the operator's instance, version and downgrade checks, and locating the operator. Other changes own these.

## Decisions

### Flow

```text
opm operator install                                  (internal/cmd/operator/install.go)
  ResolveTarget: resolve the module version           (install-operator-from-module)
  PlanInstall, no writes:                             (internal/operator/plan_install.go)
    record read, values + render, CheckTarget,
    GateStatusRBAC, waitForTerminating                (install-operator-from-module)
    PlanMigration(ctx, client, render, crdsOnly)      (this change, the proof slot)
       GET every proof-list entry and every rendered object
       -> *MigrationPlan{Adopt, Ours, MoveOwnership, RecreateDeployment,
                         DeleteBindings, LeftInPlace}
       -> or *MigrationRefusalError naming every object that blocks it
    PreApplyExistenceCheck(..., plan.Migration.Admit())
       only when rec == nil                            (install-operator-from-module guard)
  Install, writes:                                    (internal/operator/install.go)
    CRD step: ApplyOne per CRD, Wait Established      (install-operator-from-module)
    MoveOwnership(ctx, client, migration)             (this change, the writes slot,
    DeleteSuperseded(ctx, client, migration, since)    skipped under --crds-only)
       1. delete the earlier Deployment (foreground), WaitAbsent
       2. delete the superseded bindings
    workflowapply.Execute, Request.Admit = Admit()    (install-operator-from-module)
       -> RunPreApplyExistenceCheck(..., admit)        only with no previous inventory
  report: migration lines, then the ordinary lines
```

Everything lives in package `internal/operator` (files `legacy.go`, `migration_proof.go`, `migration_plan.go`, `migration_execute.go`, `migration_report.go`), not in a `migration` subpackage: the proof list is built from the names in `names.go`, and `PlanInstall` calls the planner, so a subpackage would import its own importer. The plan travels on `operator.Plan` as the field `Migration`.

The proof runs before the check-phase guard, and both guard calls (the one in the check phase and the one inside the instance apply) admit `MigrationPlan.Admit()`. A proof placed after the guard would never run on a manifest-installed cluster: the guard refuses at `Namespace/opm-operator-system` first, exactly as the install experiment measured.

All migration writes sit in one slot after the CRD step. A CRD step that fails therefore leaves the earlier operator untouched. The ownership move can wait until after the CRD step because the instance apply applies the CRDs again as `opm-cli` with force (see Context): that second apply drops whatever the move handed to `opm-cli` and the module does not render. The move rewrites `managedFields` only, so a run that stops right after it leaves the earlier operator running unchanged.

The Deployment goes before the bindings. Deleting the bindings first would leave the old controller running for a moment with no RBAC, logging errors on every reconcile. The leader lease is unaffected either way, because the operator does not release it on cancel (`LeaderElectionReleaseOnCancel` is off in `opm-operator/cmd/main.go`).

### The proof list is a Go table, generated once and checked by a test

```go
// internal/operator/legacy.go
type LegacyObject struct {
	Group, Kind, Namespace, Name string
	Labels   map[string]string // the labels every manifest that shipped it set; may be empty
	Selector map[string]string // Deployment only
	Releases string            // "v0.5.0..v1.0.0-beta.8", for the report and the test
}

// LegacyObjects is the union of the objects of every opm-operator release
// that published an install manifest.
var LegacyObjects = []LegacyObject{ /* 32 entries */ }
```

`hack/operator-legacy/` is a `go run` program that is not linked into `opm`. It downloads `install.yaml` from every operator release, a tag matching `v<semver>`, that has that asset. It skips every other tag, the operator module's `opm_operator-vX.Y.Z` releases included: their `install.yaml` is the module's render, whose objects carry the operator instance's identity and so are never foreign. The tag filter is a function with its own offline test and writes `internal/operator/testdata/legacy-manifests.json`, with the kind, namespace, name, labels and selector per release and no specs. It needs the network and runs by hand, only when an earlier release is found to be missing. A unit test checks `LegacyObjects` against that file offline.

The list closes once operator releases stop attaching `install.yaml` (opm-operator's `stop-operator-install-manifest`, gated on this cli). Until then an operator release after v1.0.0-beta.8 still attaches a manifest built from `config/default`; task 2.1 runs the program at implementation time, so the list covers every operator release published by then, and a release cut after that is added by re-running it. The module's own manifests never need an entry: they carry the instance identity and the module's selector, so their objects are this instance's own, not foreign (0021:D4 as amended).

**Options considered:**

1. Parse the embedded `install.yaml` at run time. It goes away with `install-operator-from-module`, and it covers only one release.
2. Fetch the manifests at install time. That needs GitHub at install, and the module install deliberately needs only the registry.
3. The Go table plus an offline test. Chosen.

### Proof

```go
// internal/operator/migration_proof.go
type Verdict int // Absent, Proven, Ours, Unproven

func ProveLegacy(live *unstructured.Unstructured, want LegacyObject, instanceUUID string) (verdict Verdict, reason string)
```

- `Absent` on NotFound.
- `Ours` when `module-instance.opmodel.dev/uuid` equals the operator instance's UUID.
- `Unproven` when any identity label is set to anything else, when a listed label is missing or differs, or, for the Deployment, when its `spec.selector.matchLabels` differs from the listed selector.
- An object with no listed labels, such as the CRDs, is proven by kind, name and the absence of an identity.
- A GET error other than NotFound refuses; it is never skipped. This is unlike `PreApplyExistenceCheck`, which skips. The migration cannot prove what it cannot read.

The plan refuses only for an object that blocks it: an `Unproven` or unreadable list entry that the migration would adopt (it is rendered), recreate (the Deployment) or delete (a superseded binding), and a superseded binding the render does not replace (see "Binding delete"). An `Unproven` list entry the module does not render, such as a ClusterRole of an older release that a user relabelled, is left alone and not reported: the migration neither needs it nor can vouch for it. Every object that blocks the plan is named in one refusal.

### Admission through the current guard

```go
// internal/inventory/stale.go
func PreApplyExistenceCheck(ctx context.Context, client *kubernetes.Client,
	entries []InventoryEntry, admit AdmitSet) error

// K8sIdentity is the comparable key of K8sIdentityEqual: no Component, no Version.
type K8sIdentity struct{ Group, Kind, Namespace, Name string }

type AdmitSet map[K8sIdentity]struct{}
```

`K8sIdentity` is new (`pkg/inventory/entry.go` has only `K8sIdentityEqual`, which compares two entries); task 4.1 adds it beside that function. An entry in `admit` passes the managed-by test, and only that test: a terminating admitted object is still refused. Every other caller passes `nil`, which keeps today's behaviour. `apply.Request` gains the set, so the instance apply admits the same objects as the check-phase guard. `MigrationPlan.Admit()` is `Adopt`, the Deployment the migration recreates (it still exists when the check-phase guard reads it), and the rendered objects whose identity is this instance's (`Ours`); the latter already pass today, and listing them keeps a resumed run's set complete.

This is the admission 0012:D8:R6 describes, and the one exception 0012:D8:R3 names to the ownership refusals of 0012:D1:R7 and 0012:D4:R2: the proven objects pass as if adopted, and no adopt annotation is written on the user's behalf. The set needs no write, nothing of it outlives the run, and a resumed run recomputes it.

**Options considered:**

1. **Relabel the proven objects `managed-by=opm-cli` before the apply**, as the install experiment did by hand. That is one more write per object. The label is also the guard's signal for every instance, so a relabel outlives the install.
2. **Write the 0012:D8 adopt annotation and teach the guard to read it.** Ruled out by 0012:D8:R6: no frontend sets the annotation on the user's behalf.
3. **An explicit admission set computed from the proof.** It needs no write, nothing outlives the run, and a resumed run recomputes it. Chosen.

### Field ownership of client-side installs

For each existing rendered object, adopted or `Ours`, whose `managedFields` hold an `Update` entry by `kubectl-client-side-apply`, install sends the patch `csaupgrade.UpgradeManagedFieldsPatch(obj, sets.New("kubectl-client-side-apply"), "opm-cli")` (`k8s.io/client-go/util/csaupgrade`, client-go v0.36.4). `Ours` objects are included because a client-side `kubectl apply` of a manifest rendered from the module leaves the same manager behind. The instance apply's forced apply then drops what the module does not render and the moved manager set, `last-applied-configuration` included, as it already does over an earlier `opm-cli` install (install experiment, step 3). This is what `kubectl apply --server-side` does when it takes over a client-side object. Argo CD's `ClientSideApplyMigration` sync option does the same at each sync.

`MoveOwnership` reads each object again before building its patch: the CRD step has changed the CRDs since the check phase read them, and the patch replaces `resourceVersion`, so a patch built from the planned copy would conflict. Every delete is preconditioned on the uid the proof read, so an object replaced since then is never deleted.

The scope is deliberately narrow. Only the `kubectl-client-side-apply` manager moves. Fields owned by server-side `kubectl`, a `kubectl label` or `kubectl annotate`, by controllers, or set by API defaults stay where they are; the spec promises nothing about them. An object that Flux, Argo CD or another server-side applier keeps applying never reaches this step: the proof refuses it, because that tool would re-apply the old-selector Deployment and the bindings install deletes. The move is cheap and is a no-op once no client-side manager is left, so it runs on every full install, not only on a migration run.

**Measured (spike, 2026-10-04, kind v1.34.3, client-go v0.36.4).** On a throwaway cluster, the v1.0.0-beta.5 `install.yaml` was applied client-side with `kubectl apply`. A program then server-side-applied the module 0.1.0 `moduleinstances.opmodel.dev` CRD as `opm-cli` with force (the CRD step), sent `UpgradeManagedFieldsPatch(live, {kubectl-client-side-apply}, "opm-cli")` as a JSON patch for the Namespace, `ClusterRole opm-operator-metrics-reader`, the metrics Service and that CRD, and then server-side-applied the module-rendered objects as `opm-cli` with force (the instance apply). Before the move each object was owned by `kubectl-client-side-apply` (`Update`) and carried `last-applied-configuration`; the Namespace and Service also carried `managed-by=kustomize`, `name=opm-operator` and `control-plane=controller-manager`. After the move only the manager changed (`opm-cli` `Apply`; labels and annotation untouched, so the earlier operator keeps running unchanged). After the forced apply every object carried `managed-by=opm-cli`, no `last-applied-configuration`, and only the managers `opm-cli` (plus `kube-apiserver` on the CRD, for status); the Namespace lost `control-plane`, which the module does not render, and the Service kept it, which the module renders. Control: `ClusterRole opm-operator-metrics-auth-role`, applied the same way with no move, kept `last-applied-configuration` and its `kubectl-client-side-apply` manager. (`kubectl apply --server-side` is no control: kubectl runs the same upgrade itself.) Over an `opm-cli` server-side install of the same manifest the move is a no-op for all four objects, and the forced apply drops `kustomize`, `name=opm-operator` and the Namespace's `control-plane` alike. The design stands: `csaupgrade`, not the JSON-patch fallback.

**Options considered:**

1. Do nothing. The earlier labels and `last-applied-configuration` stay, owned by a manager nobody runs. A later `kubectl apply` of a module manifest then computes its removals from stale data.
2. Strip the annotation and labels with a JSON patch. That covers the fields we know about, but not fields a user added through the manifest.
3. `csaupgrade`. Chosen.

### Deployment recreate

The Deployment is deleted only when it is `Proven` and its live `spec.selector.matchLabels` equals the listed selector (0012:D8:R7). The delete uses foreground propagation, then `WaitAbsent` under the install's `--timeout` budget. A Deployment that is `Ours`, or that carries any other selector, is never deleted. An `Unproven` Deployment on the list refuses the install. Its names come from `locate-operator-and-guard-its-instance`'s `internal/operator/names.go` (`OperatorNamespace`, `ControllerDeploymentName`), not from new literals.

### Binding delete

A superseded binding is deleted only when it is `Proven` (0012:D8:R7) and the render holds a binding of the same kind and namespace whose `roleRef` (API group, kind, name) equals the live binding's `roleRef`. The proof list does not hardcode the replacement names: the catalog's role abstraction decides them, and the render is the only source that cannot drift from it. A proven superseded binding with no replacement in the render refuses the install, naming the binding and its `roleRef`; deleting it would leave the operator without the rights it grants. The report names the replacement it found.

The controller is down from the delete until the new pod holds the leader lease. The owner accepted this one-time outage when choosing the catalog-shaped module. The managed workloads keep running throughout. Only reconciliation pauses.

### Resumption

The plan is recomputed from the cluster on every run, and every step is idempotent:

- A deleted binding or Deployment reads NotFound (`Absent`).
- An object that the interrupted run applied reads `Ours`.
- The ownership patch is a no-op once no client-side manager is left.

Only a successful instance apply writes the record (`apply.WriteInstanceRecord`). An interrupted run therefore leaves no record, and the guard runs again on the next run with the recomputed admission set. Nothing is stored outside the cluster.

### `--crds-only`

`install-operator-from-module` runs the guard over the CRDs in `--crds-only` mode. The earlier manifest's CRDs carry no managed-by label, so without the proof that guard refuses on every manifest-installed cluster. `--crds-only` therefore runs the proof over the four CRD entries only and admits the proven ones. It moves no ownership, deletes nothing and recreates nothing, and it prints no migration lines. A later full install finds those CRDs `Ours` and completes the migration.

**Options considered:** refusing `--crds-only` on a manifest-installed cluster with a hint to run a full install first. That breaks a form that works today, for no safety gain: the proof of a CRD is the same in both forms.

### Errors and exit codes

| Condition | Exit | Message |
| --- | --- | --- |
| An object fails the proof | 2 | `operator migration refused: <n> object(s) of an earlier operator manifest cannot be proven:`, then one line per object, such as `  ClusterRoleBinding/opm-operator-manager-rolebinding: carries the identity of instance team-a/web` or `: label app.kubernetes.io/name is "x", earlier manifests set "opm-operator"`, then the hint `nothing was changed; remove or rename these objects, then re-run 'opm operator install'` |
| A superseded binding has no replacement in the render | 2 | in the same refusal: `  ClusterRoleBinding/opm-operator-manager-rolebinding: the module renders no ClusterRoleBinding to ClusterRole/opm-operator-manager-role` |
| A proof-list GET fails (RBAC, network) | `cmdutil.ExitCodeFromK8sError`: 4 on Forbidden or Unauthorized, 3 on a server timeout or unavailable server, else 1 | `operator migration refused: cannot read Deployment/opm-operator-controller-manager in opm-operator-system: <err>` |
| A write fails after writes began | 1 | `operator migration stopped after it began: deleting <obj>: <err>; re-run 'opm operator install' to complete it` |
| `WaitAbsent` times out | 1 | the existing WaitAbsent timeout message, plus the re-run hint |

Exit codes follow `internal/exit`: 1 general error, 2 validation error (a refusal), 3 connectivity error, 4 permission denied.

### Report

```text
migrating the operator installed from an earlier release manifest
  adopted   CustomResourceDefinition/moduleinstances.opmodel.dev
  adopted   Namespace/opm-operator-system
  adopted   ServiceAccount/opm-operator-system/opm-operator-controller-manager
  ...
  recreated Deployment/opm-operator-system/opm-operator-controller-manager: selector changed; patches made to the earlier Deployment are not carried over
  deleted   RoleBinding/opm-operator-system/opm-operator-leader-election-rolebinding: superseded by opm-operator-leader-election-role
  deleted   ClusterRoleBinding/opm-operator-manager-rolebinding: superseded by opm-operator-manager-role
  deleted   ClusterRoleBinding/opm-operator-metrics-auth-rolebinding: superseded by opm-operator-metrics-auth-role
  left      ClusterRole/opm-operator-modulerelease-viewer-role: not part of the operator module
```

Objects are named as the apply lines name them (`Kind/namespace/name`), in proof-list order. `MigrationReport` builds the lines; install prints them after the migration's deletes, immediately before the instance apply's own lines.

Install prints migration lines only on a run that adopts, recreates or deletes something. The `left` lines name proven list entries the module does not render. Those objects are never removed, so printing them on every later install would repeat the same lines forever; after a completed migration nothing is adopted, recreated or deleted, and install prints no migration lines. The replacement names come from the render (see "Binding delete").

### Migration note

The cli `README.md` operator section gains a short note (task 4.6): the first install over a manifest-installed operator recreates the Deployment, so patches made to it (such as `--registry`) are lost and must be passed as instance values; the controller pauses for about half a minute; and an older CLI cannot reinstall over a migrated cluster (see "Migration Plan"). The install pages in the `opm` docs are that repo's change.

## Measured

Run 2026-10-04 with the change's binary on a throwaway kind cluster (kind, node v1.34.3, the operator image pullable), installing operator module 0.1.0 (operator v1.0.0-beta.8) over the v1.0.0-beta.8 `install.yaml`, with `--skip-platform`. The kind e2e job in CI repeats these runs (`TestE2E_Operator_MigratesManifestInstall`).

- **Client-side `kubectl apply`, earlier operator running, one full install.** 18.5 s wall clock, about 5 s of it the render. The report named 14 adopted objects (the four CRDs among them), the recreated Deployment and the three deleted bindings; the instance apply then reported `applied 22 resources successfully (4 created, 18 configured)` and the rollout completed. Afterwards the CRD, the Namespace, `ClusterRole opm-operator-manager-role` and the metrics Service were owned by `opm-cli` (`Apply`) alone (plus `kube-apiserver` on the CRD), with no `last-applied-configuration` and no `kustomize` label; the Service kept `control-plane`, which the module renders. The ownership move ran after the CRD step without a conflict, reading each object again.
- **Controller gap.** The earlier Deployment was deleted at 21:42:59; the new pod started its manager at 21:43:01 and acquired the leader lease at 21:43:18, when its controllers started: about 19 s without reconciliation, the old lease running out. The managed workloads were not touched.
- **Client-side, `--crds-only` first.** `--crds-only` reported the four CRDs `configured` and changed nothing else (the Deployment kept its `resourceVersion`, the three bindings stayed). The full install that followed took 22.7 s, adopted the 14 non-CRD objects, recreated the Deployment and deleted the bindings. The CRDs, the Namespace, the ServiceAccount and a CLI-owned ModuleInstance kept their uid (the instance its generation); the Deployment got a new uid; the record counted 22 entries, `spec.owner: cli`, and the new operator reconciled. A re-run printed no migration lines, reported `22 unchanged` and kept the Deployment's uid.
- **Refusal.** Over an `opm-cli` server-side install with `ClusterRoleBinding opm-operator-manager-rolebinding` carrying another instance's uuid and `ClusterRole opm-operator-platform-viewer-role` relabelled, install exited 2 naming both objects, and the `moduleinstances` CRD kept its `resourceVersion`: the CRD step never ran.
- **Interrupted run.** With those labels restored and the Deployment and the manager binding deleted by hand, as a migration stopped after its first deletes leaves them, one install completed in 17.8 s: it deleted the two remaining bindings, did not report a recreate, created the Deployment with the module's selector and kept the Namespace's uid.

## Risks / Trade-offs

- [The operator is down from the Deployment delete until the new pod holds the leader lease; about 28 s were measured on a pod roll, 19 s on the migration (see "Measured")] → Accepted by the owner. The report says so, and the managed workloads keep running.
- [csaupgrade does not move the kustomize labels or the annotation] → Measured in section 1: it does, once the forced instance apply follows (see "Field ownership of client-side installs").
- [An operator release manifest is missing from the table] → The program reads every operator release (`v<semver>`) at implementation time and the test against the downloaded set catches a gap among past releases; an operator release that still attaches a manifest after that is added by re-running the program, until `stop-operator-install-manifest` closes the list (see the proof-list decision). Module releases are never a source.
- [Controller arguments a user patched onto the earlier Deployment, such as `--registry`, vanish with the recreate] → They become instance values under `install-operator-from-module`. The report's recreate line says the patches are not carried over, and the README migration note says to pass them as values on this install.
- [An object on the list belongs to another instance and gets touched] → It reads `Unproven` and refuses the install before any write.
- [The delivery run needs a cluster that can pull the operator image] → At proposal time, kind nodes on the development host have no egress. `kind load docker-image` failed for the multi-arch operator image in the install experiment (`ctr: content digest … not found`). The run therefore uses a single-platform pull, a host-network pull-through mirror, or another cluster, and its result is recorded before archive.

## Migration Plan

The CLI migrates no data of its own. Rollback is a CLI downgrade. An older CLI's manifest install over a migrated cluster hits the same immutable-selector refusal in reverse. The README migration note says so; this change does not address it.

## Research & Decisions

### Where the migration's list comes from

**Context**: The proof needs a source of truth for which objects an earlier operator release installed.
**Explored**: `dist/install.yaml` at all 47 opm-operator tags, the assets of the GitHub releases, and the 10 manifests the CLI has embedded.
**Options considered**:
1. Only the current manifest's 19 objects. This misses the 10 objects of v0.5.0 to v0.7.5 that a cluster upgraded through kubectl still holds.
2. The union over every release with a manifest asset from v0.5.0 on (32 objects). Install can then name the older leftovers in its report.
**Decision**: Option 2.
**Rationale**: The extra 10 are never adopted or deleted, because the module does not render them, none is a binding, and four are CRDs. They are only reported, and only on the run that migrates. Naming them once tells the user what is left over from an earlier install. An unproven one never blocks the install.

### How proven objects pass the guard

See Decisions, "Admission through the current guard".

### Why install migrates instead of asking the user

**Context**: Every cluster running OPM takes this step once.
**Options considered**:
1. The user annotates each object, and install prints the commands. That is nineteen hand-made edits, which most users would get wrong or skip.
2. A separate migration subcommand. That adds a second command for a step every first module install needs, and install would still have to refuse and point at it.
3. Uninstall, then install fresh. The uninstall refuses while any instance carries the cleanup finalizer, so a cluster in use cannot take that path without orphaning its instances.
4. A command-wide force flag. It takes objects the user did not mean to take; 0012:D8 rejected it.
5. Install's own migration, limited to proven objects. Chosen.
**Rationale**: A migration that recreated CRDs would delete every instance on the cluster. Leaving the CRDs, the Namespace and the managed workloads untouched is the requirement that matters most. The controller's own Deployment is replaceable. Limiting adoption to objects proven to come from an earlier release keeps the guard whole, so nothing a user created is adopted by accident.
