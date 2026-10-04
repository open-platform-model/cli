## Context

proposal.md, Why, describes the problem. The code this change touches, as read on `main` at 048d6807:

- **The apply guard.** `inventory.PreApplyExistenceCheck` (`internal/inventory/stale.go`) GETs every rendered entry on a first apply. It refuses at the first object without an OPM `app.kubernetes.io/managed-by` value (`pkgcore.IsOPMManagedBy`). It logs a read error other than NotFound at debug level and skips the object. `apply.RunPreApplyExistenceCheck` (`internal/workflow/apply/apply.go`) skips the guard entirely when a previous inventory exists or on a dry run. The guard admits any OPM-managed object, whichever instance it belongs to.
- **The writes.** `kubernetes.ApplyOne` server-side-applies with field manager `opm-cli` and `Force: true` (`internal/kubernetes/apply.go`, `internal/kubernetes/labels.go`). `inventory.PruneStaleResources` deletes with foreground propagation and never deletes a CRD or a Namespace (`kubernetes.IsProtectedKind`). `kubernetes.WaitAbsent` waits for objects to disappear; `operator.waitForTerminating` in `internal/operator/install.go` uses it.
- **The install being replaced.** Today `operator.Install` server-side-applies the embedded `dist/install.yaml` (`internal/operator/manifest.go`, `PinnedOperatorVersion = "v1.0.0-beta.5"`). Change `install-operator-from-module` replaces it with a two-step module install: a check phase with no writes, the CRD step, then the instance apply. That change leaves a slot for this one in the check phase and in the write order. Its check phase runs `inventory.PreApplyExistenceCheck` over every rendered object inside `PlanInstall` when no record exists, and its instance apply runs the same guard again through `apply.RunPreApplyExistenceCheck`. Its planning sketch places the proof slot after that guard; this change needs it before the guard (see "Flow"). Its instance apply applies the four CRDs again with the same field manager after the CRD step: `splitClusterDefinitions` (`internal/kubernetes/apply.go`) applies protected kinds first, and the install experiment reported `15 created, 4 unchanged` for those CRDs. The exact function names come from that change once it is merged (task 1.2).
- **Identity labels.** `pkg/core/labels.go` defines `module-instance.opmodel.dev/{name,namespace,uuid}`. The operator module's render stamps them, and `app.kubernetes.io/managed-by: opm-cli`, on every object.

### What the earlier manifests contain (measured 2026-10-04)

These figures come from `dist/install.yaml` at every one of the 47 opm-operator tags and from the GitHub release assets:

- Releases v0.5.0 to v1.0.0-beta.5 publish an `install.yaml` asset; v0.2.0 to v0.4.0 publish none. From v0.2.0 to v0.4.4 every name is `poc-controller-*` in `poc-controller-system`. From v0.5.0 every name carries the `opm-operator` prefix.
- The union over v0.5.0 to v1.0.0-beta.5 is 29 objects. 19 belong to the current manifest. The other 10 shipped only in older releases:
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

- Writing the 0012:D8 adopt annotation. The amended 0012:D8 lets `opm operator install` set it on exactly the objects this change proves, but no frontend's guard reads it yet. This change admits the same objects through an admission set instead. When the guard reads the annotation, a later change can switch install to writing it; the spec names which objects pass, not how, so it does not change.
- Closing the guard's general gap: it admits another instance's OPM-managed object. The migration refuses such an object when it is on the proof list; for other objects the guard is unchanged here.
- Migrating `poc-controller-*` installs (v0.2.0 to v0.4.4). No release with those names published a manifest, so they are not on the list.
- Ownership transfer of the operator's instance, version and downgrade checks, and locating the operator. Other changes own these.

## Decisions

### Flow

```text
opm operator install
  resolve + render module                            (install-operator-from-module)
  check phase, no writes:
    version, value and CRD checks, terminating wait  (install-operator-from-module)
    migration.Plan(ctx, client, rendered, instance)  (this change)
       GET every proof-list entry and every rendered object
       -> Plan{Adopt, Ours, MoveOwnership, RecreateDeployment, DeleteBindings, LeftInPlace}
       -> or *migration.RefusalError naming every object that blocks it
    apply guard over the rendered objects, admitting Plan.Admit()
                                                     (install-operator-from-module guard,
                                                      moved after the proof by this change)
  write phase:
    CRD step, wait until served                      (install-operator-from-module)
    migration.MoveOwnership(ctx, client, plan)       (this change)
    migration.Delete(ctx, client, plan, budget)      (this change)
       1. delete the earlier Deployment (foreground), WaitAbsent
       2. delete the superseded bindings
    instance apply, guard admitting Plan.Admit()     (install-operator-from-module)
  report: migration lines, then the ordinary lines
```

The proof runs before the check-phase guard, and both guard calls (the one in the check phase and the one inside the instance apply) admit `Plan.Admit()`. A proof placed after the guard would never run on a manifest-installed cluster: the guard refuses at `Namespace/opm-operator-system` first, exactly as the install experiment measured.

All migration writes sit in one slot after the CRD step. A CRD step that fails therefore leaves the earlier operator untouched. The ownership move can wait until after the CRD step because the instance apply applies the CRDs again as `opm-cli` with force (see Context): that second apply drops whatever the move handed to `opm-cli` and the module does not render. The move rewrites `managedFields` only, so a run that stops right after it leaves the earlier operator running unchanged.

The Deployment goes before the bindings. Deleting the bindings first would leave the old controller running for a moment with no RBAC, logging errors on every reconcile. The leader lease is unaffected either way, because the operator does not release it on cancel (`LeaderElectionReleaseOnCancel` is off in `opm-operator/cmd/main.go`).

### The proof list is a Go table, generated once and checked by a test

```go
// internal/operator/legacy.go
type LegacyObject struct {
	Group, Kind, Namespace, Name string
	Labels   map[string]string // the labels every manifest that shipped it set; may be empty
	Selector map[string]string // Deployment only
	Releases string            // "v0.5.0..v1.0.0-beta.5", for the report and the test
}

// LegacyObjects is the union of the objects of every opm-operator release
// that published an install manifest.
var LegacyObjects = []LegacyObject{ /* 29 entries */ }
```

`hack/operator-legacy/` is a `go run` program that is not linked into `opm`. It downloads `install.yaml` from every release that has that asset and writes `internal/operator/testdata/legacy-manifests.json`, with the kind, namespace, name, labels and selector per release and no specs. It needs the network and runs by hand, only when an earlier release is found to be missing. A unit test checks `LegacyObjects` against that file offline.

The list is closed. The operator now ships as a module, and any manifest a later release publishes is a render of that module (0021:D4). Such a manifest carries the instance identity and the module's selector, so its objects are this instance's own, not foreign.

**Options considered:**

1. Parse the embedded `install.yaml` at run time. It goes away with `install-operator-from-module`, and it covers only one release.
2. Fetch the manifests at install time. That needs GitHub at install, and the module install deliberately needs only the registry.
3. The Go table plus an offline test. Chosen.

### Proof

```go
// internal/operator/migration/proof.go
type Verdict int // Absent, Proven, Ours, Unproven

func Prove(live *unstructured.Unstructured, want LegacyObject, instanceUUID string) (Verdict, string /* reason */)
```

- `Absent` on NotFound.
- `Ours` when `module-instance.opmodel.dev/uuid` equals the operator instance's UUID.
- `Unproven` when any identity label is set to anything else, or when a listed label is missing or differs.
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

`K8sIdentity` is new (`pkg/inventory/entry.go` has only `K8sIdentityEqual`, which compares two entries); task 4.1 adds it beside that function. An entry in `admit` passes the managed-by test, and only that test: a terminating admitted object is still refused. Every other caller passes `nil`, which keeps today's behaviour. `apply.Request` gains the set, so the instance apply admits the same objects as the check-phase guard. `Plan.Admit()` is `Adopt` plus the rendered objects whose identity is this instance's (`Ours`); the latter already pass today, and listing them keeps a resumed run's set complete.

The amended 0012:D8 lets install set the adopt annotation on exactly these proven objects instead. Both mechanisms admit the same objects; the set needs no write, nothing of it outlives the run, and a resumed run recomputes it.

**Options considered:**

1. **Relabel the proven objects `managed-by=opm-cli` before the apply**, as the install experiment did by hand. That is one more write per object. The label is also the guard's signal for every instance, so a relabel outlives the install.
2. **Write the 0012:D8 adopt annotation and teach the guard to read it.** That is 0012's design. Doing it here would implement half of 0012:D8 in one frontend, and the migration would then wait on that decision's implementation.
3. **An explicit admission set computed from the proof.** It needs no write, nothing outlives the run, and a resumed run recomputes it. Chosen.

### Field ownership of client-side installs

For each existing rendered object, adopted or `Ours`, whose `managedFields` hold an `Update` entry by `kubectl-client-side-apply`, install sends the patch `csaupgrade.UpgradeManagedFieldsPatch(obj, sets.New("kubectl-client-side-apply"), "opm-cli")` (`k8s.io/client-go/util/csaupgrade`, client-go v0.36.4). `Ours` objects are included because a client-side `kubectl apply` of a manifest rendered from the module leaves the same manager behind. The instance apply's forced apply then drops what the module does not render and the moved manager set, `last-applied-configuration` included, as it already does over an earlier `opm-cli` install (install experiment, step 3). This is what `kubectl apply --server-side` does when it takes over a client-side object. Argo CD's `ClientSideApplyMigration` sync option does the same at each sync.

The scope is deliberately narrow. Only the `kubectl-client-side-apply` manager moves. Fields owned by server-side `kubectl`, Flux, Argo CD or another tool's manager, by controllers, or set by API defaults stay where they are; the spec promises nothing about them. The move is cheap and is a no-op once no client-side manager is left, so it runs on every full install, not only on a migration run.

**Unverified:** that the patch takes the annotation and the kustomize labels along. Section 1 measures this before anything else is built.

**Options considered:**

1. Do nothing. The earlier labels and `last-applied-configuration` stay, owned by a manager nobody runs. A later `kubectl apply` of a module manifest then computes its removals from stale data.
2. Strip the annotation and labels with a JSON patch. That covers the fields we know about, but not fields a user added through the manifest.
3. `csaupgrade`. Chosen.

### Deployment recreate

The Deployment is deleted only when it is `Proven` and its live `spec.selector.matchLabels` equals the listed selector. The delete uses foreground propagation, then `WaitAbsent` under the install's `--timeout` budget. A Deployment that is `Ours`, or that carries any other selector, is never deleted. An `Unproven` Deployment on the list refuses the install. Its names come from `locate-operator-and-guard-its-instance`'s `internal/operator/names.go` (`OperatorNamespace`, `ControllerDeploymentName`), not from new literals.

### Binding delete

A superseded binding is deleted only when it is `Proven` and the render holds a binding of the same kind and namespace whose `roleRef` (API group, kind, name) equals the live binding's `roleRef`. The proof list does not hardcode the replacement names: the catalog's role abstraction decides them, and the render is the only source that cannot drift from it. A proven superseded binding with no replacement in the render refuses the install, naming the binding and its `roleRef`; deleting it would leave the operator without the rights it grants. The report names the replacement it found.

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
  adopted   Namespace/opm-operator-system
  adopted   ServiceAccount/opm-operator-controller-manager (opm-operator-system)
  ...
  recreated Deployment/opm-operator-controller-manager (opm-operator-system): selector changed; patches made to the earlier Deployment are not carried over
  deleted   ClusterRoleBinding/opm-operator-manager-rolebinding: superseded by opm-operator-manager-role
  deleted   ClusterRoleBinding/opm-operator-metrics-auth-rolebinding: superseded by opm-operator-metrics-auth-role
  deleted   RoleBinding/opm-operator-system/opm-operator-leader-election-rolebinding: superseded by opm-operator-leader-election-role
  left      ClusterRole/opm-operator-modulerelease-viewer-role: not part of the operator module
```

Install prints migration lines only on a run that adopts, recreates or deletes something. The `left` lines name proven list entries the module does not render. Those objects are never removed, so printing them on every later install would repeat the same lines forever; after a completed migration nothing is adopted, recreated or deleted, and install prints no migration lines. The replacement names come from the render (see "Binding delete").

### Migration note

The cli `README.md` operator section gains a short note (task 4.6): the first install over a manifest-installed operator recreates the Deployment, so patches made to it (such as `--registry`) are lost and must be passed as instance values; the controller pauses for about half a minute; and an older CLI cannot reinstall over a migrated cluster (see "Migration Plan"). The install pages in the `opm` docs are that repo's change.

## Risks / Trade-offs

- [The operator is down from the Deployment delete until the new pod holds the leader lease; about 28 s were measured on a pod roll] → Accepted by the owner. The report says so, and the managed workloads keep running.
- [csaupgrade does not move the kustomize labels or the annotation] → Section 1 measures it first. If it fails, this design takes the JSON-patch option for exactly the fields the proof list names plus the annotation, and the spec is unchanged.
- [A release manifest is missing from the table] → The table is closed (see the proof-list decision), and the test against the downloaded set catches a gap among past releases.
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
2. The union over every release with a manifest asset (29 objects). Install can then name the older leftovers in its report.
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
