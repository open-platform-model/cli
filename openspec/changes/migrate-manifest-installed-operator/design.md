## Context

proposal.md, Why, describes the problem. The code this change touches, as read on `main` at 048d6807:

- **The apply guard.** `inventory.PreApplyExistenceCheck` (`internal/inventory/stale.go`) GETs every rendered entry on a first apply. It refuses at the first object without an OPM `app.kubernetes.io/managed-by` value (`pkgcore.IsOPMManagedBy`). It logs a read error other than NotFound at debug level and skips the object. `apply.RunPreApplyExistenceCheck` (`internal/workflow/apply/apply.go`) skips the guard entirely when a previous inventory exists or on a dry run. The guard admits any OPM-managed object, whichever instance it belongs to.
- **The writes.** `kubernetes.ApplyOne` server-side-applies with field manager `opm-cli` and `Force: true` (`internal/kubernetes/apply.go`, `internal/kubernetes/labels.go`). `inventory.PruneStaleResources` deletes with foreground propagation and never deletes a CRD or a Namespace (`kubernetes.IsProtectedKind`). `kubernetes.WaitAbsent` waits for objects to disappear; `operator.waitForTerminating` in `internal/operator/install.go` uses it.
- **The install being replaced.** Today `operator.Install` server-side-applies the embedded `dist/install.yaml` (`internal/operator/manifest.go`, `PinnedOperatorVersion = "v1.0.0-beta.5"`). Change `install-operator-from-module` replaces it with a two-step module install: a check phase with no writes, the CRD step, then the instance apply. That change leaves a slot for this one in the check phase and in the write order. The exact function names come from that change once it is merged (task 1.2).
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

- Writing the 0012:D8 adopt annotation. No frontend implements it. This change admits through the current guard instead. When the annotation lands in the CLI, a later change can switch admission to it without changing the observable rule.
- Closing the guard's general gap: it admits another instance's OPM-managed object. The migration refuses such an object when it is on the proof list; for other objects the guard is unchanged here.
- Migrating `poc-controller-*` installs (v0.2.0 to v0.4.4). No release with those names published a manifest, so they are not on the list.
- Ownership transfer of the operator's instance, version and downgrade checks, and locating the operator. Other changes own these.

## Decisions

### Flow

```text
opm operator install
  resolve + render module                            (install-operator-from-module)
  check phase, no writes:
    version, value and CRD checks                    (install-operator-from-module)
    migration.Plan(ctx, client, rendered, instance)  (this change)
       GET every proof-list entry and every rendered object
       -> Plan{Adopt, MoveOwnership, RecreateDeployment, DeleteBindings, LeftInPlace}
       -> or *migration.RefusalError naming every unproven object
    apply guard over the rendered objects, admitting Plan.Admit()
  write phase:
    migration.MoveOwnership(ctx, client, plan)       (this change)
    CRD step, wait until served                      (install-operator-from-module)
    migration.Delete(ctx, client, plan, budget)      (this change)
       1. delete the superseded bindings
       2. delete the earlier Deployment (foreground), WaitAbsent
    instance apply (guard admits Plan.Admit())       (install-operator-from-module)
  report: migration lines, then the ordinary lines
```

The ownership move comes before the CRD step because the CRD step is the first forced apply to touch the CRDs. If a client-side manager still owned fields of a CRD at that point, the forced apply would leave those fields in place. The move rewrites `managedFields` only, so a run that stops right after it leaves the earlier operator running unchanged. The deletes come after the CRD step, so that a CRD step which fails leaves the controller running.

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

- `Ours` when `module-instance.opmodel.dev/uuid` equals the operator instance's UUID.
- `Unproven` when any identity label is set to anything else, or when a listed label is missing or differs.
- An object with no listed labels, such as the CRDs, is proven by kind, name and the absence of an identity.
- A GET error other than NotFound refuses; it is never skipped. This is unlike `PreApplyExistenceCheck`, which skips. The migration cannot prove what it cannot read.

### Admission through the current guard

```go
// internal/inventory/stale.go
func PreApplyExistenceCheck(ctx context.Context, client *kubernetes.Client,
	entries []InventoryEntry, admit AdmitSet) error

type AdmitSet map[K8sIdentity]struct{} // group, kind, namespace, name
```

An entry in `admit` passes the managed-by test. Every other caller passes `nil`, which keeps today's behaviour. `apply.Request` gains the set, so the instance apply in the write phase admits the same objects that the check phase proved.

**Options considered:**

1. **Relabel the proven objects `managed-by=opm-cli` before the apply**, as the install experiment did by hand. That is one more write per object. The label is also the guard's signal for every instance, so a relabel outlives the install.
2. **Write the 0012:D8 adopt annotation and teach the guard to read it.** That is 0012's design. Doing it here would implement half of 0012:D8 in one frontend, and the migration would then wait on that decision's implementation.
3. **An explicit admission set computed from the proof.** It needs no write, nothing outlives the run, and a resumed run recomputes it. Chosen.

### Field ownership of client-side installs

For each adopted object whose `managedFields` hold an `Update` entry by `kubectl-client-side-apply`, install sends the patch `csaupgrade.UpgradeManagedFieldsPatch(obj, sets.New("kubectl-client-side-apply"), "opm-cli")` (`k8s.io/client-go/util/csaupgrade`, client-go v0.36.4). The forced apply that follows then owns every field. It drops what the module does not render, `last-applied-configuration` included, as it already does over an earlier `opm-cli` install (install experiment, step 3). This is what `kubectl apply --server-side` does when it takes over a client-side object. Argo CD's `ClientSideApplyMigration` sync option does the same at each sync.

**Unverified:** that the patch takes the annotation and the kustomize labels along. Section 1 measures this before anything else is built.

**Options considered:**

1. Do nothing. The earlier labels and `last-applied-configuration` stay, owned by a manager nobody runs. A later `kubectl apply` of a module manifest then computes its removals from stale data.
2. Strip the annotation and labels with a JSON patch. That covers the fields we know about, but not fields a user added through the manifest.
3. `csaupgrade`. Chosen.

### Deployment recreate

The Deployment is deleted only when it is `Proven` and its live `spec.selector.matchLabels` equals the listed selector. The delete uses foreground propagation, then `WaitAbsent` under the install's `--timeout` budget. A Deployment that is `Ours`, or that carries any other selector, is never deleted. An `Unproven` Deployment on the list refuses the install.

The controller is down from the delete until the new pod holds the leader lease. The owner accepted this one-time outage when choosing the catalog-shaped module. The managed workloads keep running throughout. Only reconciliation pauses.

### Resumption

The plan is recomputed from the cluster on every run, and every step is idempotent:

- A deleted binding or Deployment reads NotFound (`Absent`).
- An object that the interrupted run applied reads `Ours`.
- The ownership patch is a no-op once no client-side manager is left.

Only a successful instance apply writes the record (`apply.WriteInstanceRecord`). An interrupted run therefore leaves no record, and the guard runs again on the next run with the recomputed admission set. Nothing is stored outside the cluster.

### Errors and exit codes

| Condition | Exit | Message |
| --- | --- | --- |
| An object fails the proof | 2 | `operator migration refused: <n> object(s) of an earlier operator manifest cannot be proven:`, then one line per object, such as `  ClusterRoleBinding/opm-operator-manager-rolebinding: carries the identity of instance team-a/web` or `: label app.kubernetes.io/name is "x", earlier manifests set "opm-operator"`, then the hint `nothing was changed; remove or rename these objects, then re-run 'opm operator install'` |
| A proof-list GET fails (RBAC, network) | 4 on Forbidden, 3 on a connectivity error, else 1 | `operator migration refused: cannot read Deployment/opm-operator-controller-manager in opm-operator-system: <err>` |
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

On a cluster with nothing to migrate, install prints no migration lines. The exact binding names come from the published module's render, which section 3 checks.

## Risks / Trade-offs

- [The operator is down from the Deployment delete until the new pod holds the leader lease; about 28 s were measured on a pod roll] → Accepted by the owner. The report says so, and the managed workloads keep running.
- [csaupgrade does not move the kustomize labels or the annotation] → Section 1 measures it first. If it fails, this design takes the JSON-patch option for exactly the fields the proof list names plus the annotation, and the spec is unchanged.
- [A release manifest is missing from the table] → The table is closed (see the proof-list decision), and the test against the downloaded set catches a gap among past releases.
- [Controller arguments a user patched onto the earlier Deployment, such as `--registry`, vanish with the recreate] → They become instance values under `install-operator-from-module`. The report's recreate line says the patches are not carried over, and the opm docs migration note says to pass them as values on this install.
- [An object on the list belongs to another instance and gets touched] → It reads `Unproven` and refuses the install before any write.
- [The delivery run needs a cluster that can pull the operator image] → At proposal time, kind nodes on the development host have no egress. `kind load docker-image` failed for the multi-arch operator image in the install experiment (`ctr: content digest … not found`). The run therefore uses a single-platform pull, a host-network pull-through mirror, or another cluster, and its result is recorded before archive.

## Migration Plan

The CLI migrates no data of its own. Rollback is a CLI downgrade. An older CLI's manifest install over a migrated cluster hits the same immutable-selector refusal in reverse. The opm docs migration note says so; this change does not address it.

## Research & Decisions

### Where the migration's list comes from

**Context**: The proof needs a source of truth for which objects an earlier operator release installed.
**Explored**: `dist/install.yaml` at all 47 opm-operator tags, the assets of the GitHub releases, and the 10 manifests the CLI has embedded.
**Options considered**:
1. Only the current manifest's 19 objects. This misses the 10 objects of v0.5.0 to v0.7.5 that a cluster upgraded through kubectl still holds.
2. The union over every release with a manifest asset (29 objects). Install can then name the older leftovers in its report.
**Decision**: Option 2.
**Rationale**: The extra 10 are never adopted or deleted, because the module does not render them, none is a binding, and four are CRDs. They are only reported. Naming them tells the user what is left over from an earlier install.

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
