## Context

See proposal.md, Why. The code this change touches, as read on `main` at 048d6807:

- **The apply guard.** `inventory.PreApplyExistenceCheck` (`internal/inventory/stale.go`) GETs every rendered entry on a first apply and refuses at the first object without an OPM `app.kubernetes.io/managed-by` value (`pkgcore.IsOPMManagedBy`). A read error other than NotFound is logged at debug level and skipped. `apply.RunPreApplyExistenceCheck` (`internal/workflow/apply/apply.go`) skips it entirely when a previous inventory exists or on dry-run. The guard admits any OPM-managed object, whoever's instance it belongs to.
- **The writes.** `kubernetes.ApplyOne` server-side-applies with field manager `opm-cli` and `Force: true` (`internal/kubernetes/apply.go`, `internal/kubernetes/labels.go`). `inventory.PruneStaleResources` deletes with foreground propagation and never deletes a CRD or Namespace (`kubernetes.IsProtectedKind`). `kubernetes.WaitAbsent` waits for objects to disappear (used by `operator.waitForTerminating`, `internal/operator/install.go`).
- **The install being replaced.** Today `operator.Install` server-side-applies the embedded `dist/install.yaml` (`internal/operator/manifest.go`, `PinnedOperatorVersion = "v1.0.0-beta.5"`). Change `install-operator-from-module` replaces it with 0028:D3's two-step module install: a check phase with no writes, the CRD step, then the instance apply. This change adds a stage to that flow; the exact function names come from that change once merged (task 1.1).
- **Identity labels.** `pkg/core/labels.go`: `module-instance.opmodel.dev/{name,namespace,uuid}`. The operator module's render stamps them and `app.kubernetes.io/managed-by: opm-cli` on every object (0028 experiment 02, `out/render.yaml`).

Measured for this proposal (2026-10-04) from `dist/install.yaml` at every opm-operator tag and from the GitHub releases:

- Releases v0.5.0 to v1.0.0-beta.5 publish an `install.yaml` asset; v0.2.0 to v0.4.0 publish none (`gh release view`). v0.2.0 to v0.4.4 name everything `poc-controller-*` in `poc-controller-system`; from v0.5.0 every name carries the `opm-operator` prefix.
- The union over v0.5.0 to v1.0.0-beta.5 is 29 objects. 19 are the current manifest's. 10 only older releases shipped: the ClusterRoles `opm-operator-{bundlerelease,modulerelease}-{admin,editor,viewer}-role` and the CRDs `bundlereleases`, `modulereleases`, `releases` and `platforms` in group `releases.opmodel.dev`.
- Every object keeps the same labels in every release that ships it, and the Deployment keeps the selector `{app.kubernetes.io/name: opm-operator, control-plane: controller-manager}` throughout. The only role bindings any release ships are the three D8 deletes.
- The 10 CLI-embedded manifests (alpha.2 to beta.5, `git log -- internal/operator/dist/install.yaml`) agree with the tag trees.

## Goals / Non-Goals

**Goals:** 0028:D8 as specified in `specs/operator-migration/spec.md`, measured over an `opm-cli` install, a client-side `kubectl apply` install and an interrupted run.

**Non-Goals:**

- The 0012:D8 adopt annotation. Not implemented in any frontend; this change admits through the current guard (0028 06-operational.md, "The migration does not wait for the adopt annotation"). When the annotation lands in the CLI, a later change can switch admission to it with no change to the observable rule (0028:D8:R8).
- Closing the guard's general gap that it admits another instance's OPM-managed object. The migration refuses such an object on its own list; for other objects the guard is unchanged here.
- Migrating `poc-controller-*` installs (v0.2.0 to v0.4.4). No release with those names published a manifest, so they are not on the list.
- Ownership transfer (0028:D4), version checks (0028:D10), the operator locator (0028:D3:R13/R15).

## Decisions

### Flow

```text
opm operator install
  resolve + render module (install-operator-from-module)
  check phase, no writes:
    D10 checks, D5 value check, CRD floor ...       (install-operator-from-module)
    migration.Plan(ctx, client, rendered, instance) (this change)
       GET every proof-list entry and every rendered object
       -> Plan{Adopt, RecreateDeployment, DeleteBindings, LeftInPlace, MoveOwnership}
       -> or *migration.RefusalError naming every unproven object
    apply guard over rendered objects, with Plan.Admit()
  write phase:
    CRD step, wait until served                      (install-operator-from-module)
    migration.Execute(ctx, client, plan)             (this change)
       1. move CSA field ownership of adopted objects to opm-cli
       2. delete superseded bindings
       3. delete earlier Deployment (foreground), WaitAbsent
    instance apply (guard admits Plan.Admit())       (install-operator-from-module)
  report: migration lines + ordinary lines
```

### The proof list is a Go table, generated once and checked by a test

```go
// internal/operator/legacy.go
type LegacyObject struct {
	Group, Kind, Namespace, Name string
	Labels   map[string]string // labels every manifest that shipped it set
	Selector map[string]string // Deployment only
	Releases string            // "v0.5.0..v1.0.0-beta.5", for the report and the test
}

// LegacyObjects is the union of the objects of every opm-operator release
// that published an install manifest (0028:D8:R14).
var LegacyObjects = []LegacyObject{ /* 29 entries */ }
```

`hack/operator-legacy/` (a `go run` program, not linked into `opm`) downloads `install.yaml` of every release with that asset, a network step run by hand only when an earlier release is found missing, and writes `internal/operator/testdata/legacy-manifests.json` (kind, namespace, name, labels, selector per release; no specs). A unit test checks `LegacyObjects` against that file offline. The list is closed: no future release publishes a manifest of its own (0028:D2:R13), so this table never grows.

**Options considered:** (1) parse the embedded `install.yaml` at run time: it goes away with `install-operator-from-module` and covers one release only. (2) Fetch the manifests at install time: needs GitHub at install, which 0028:D3 removed. (3) The Go table plus an offline test: chosen.

### Proof

```go
// internal/operator/migration/proof.go
type Verdict int // Absent, Proven, Ours, Unproven

func Prove(live *unstructured.Unstructured, want LegacyObject, instanceUUID string) (Verdict, string /* reason */)
```

`Ours` when `module-instance.opmodel.dev/uuid` equals the operator instance's UUID. `Unproven` when any identity label is set to anything else, or a listed label is missing or differs. A GET error other than NotFound is a refusal, never a skip (unlike `PreApplyExistenceCheck`, which skips): the migration cannot prove what it cannot read.

### Admission through the current guard

```go
// internal/inventory/stale.go
func PreApplyExistenceCheck(ctx context.Context, client *kubernetes.Client,
	entries []InventoryEntry, admit AdmitSet) error

type AdmitSet map[K8sIdentity]struct{} // group, kind, namespace, name
```

An entry in `admit` passes the managed-by test. Every other caller passes `nil`, which keeps today's behaviour. `apply.Request` gains the set, so the instance apply in the write phase admits the same objects the check phase proved.

**Options considered:**

1. **Relabel the proven objects `managed-by=opm-cli` before the apply**, as experiment 02's `hack/relabel-old.sh` did. One more write per object, and the label is the guard's signal for every instance, so a relabel outlives the install.
2. **Write the 0012:D8 adopt annotation and teach the guard to read it.** That is 0012's design; doing it here would implement half of 0012:D8 in one frontend.
3. **An explicit admission set computed from the proof.** No write, nothing outlives the run, and a resumed run recomputes it. Chosen.

### Field ownership of client-side installs

Before the instance apply, for each adopted object whose `managedFields` hold an `Update` entry by `kubectl-client-side-apply`, install sends the patch `csaupgrade.UpgradeManagedFieldsPatch(obj, sets.New("kubectl-client-side-apply"), "opm-cli")` (`k8s.io/client-go/util/csaupgrade`, client-go v0.36.4). The forced apply that follows then owns every field and drops what the module does not render, `last-applied-configuration` included, as it already does for an earlier `opm-cli` install (experiment 02, step 6). This is what `kubectl apply --server-side` does when it takes over a client-side object. **Unverified:** that the patch takes the annotation and the kustomize labels along. Section 1 measures it before anything else is built.

**Options considered:** (1) Do nothing: the earlier labels and `last-applied-configuration` stay, owned by a manager nobody runs, and a later `kubectl apply` of the module manifest computes its removals from stale data. (2) Strip the annotation and labels with a JSON patch: covers the fields we know about, not the ones a user added through the manifest. (3) `csaupgrade`: chosen.

### Deployment recreate

The Deployment is deleted only when it is `Proven` and its live `spec.selector.matchLabels` equals the listed selector. Foreground propagation, then `WaitAbsent` under the install's `--timeout` budget. A Deployment that is `Ours` or carries any other selector is never deleted. An `Unproven` Deployment on the list refuses the install (0028:D8:R13).

### Resumption

The plan is recomputed from the cluster on every run and every step is idempotent: a deleted binding or Deployment reads NotFound (`Absent`), an object the interrupted run applied reads `Ours`, and the ownership patch is a no-op once no client-side manager is left. The record is written only by a successful instance apply (`apply.WriteInstanceRecord`), so an interrupted run leaves no record, and the guard runs again with the recomputed admission set. Nothing outside the cluster is stored.

### Errors and exit codes

| Condition | Exit | Message |
| --- | --- | --- |
| An object fails the proof | 2 | `operator migration refused: <n> object(s) of an earlier operator manifest cannot be proven:` then one line per object, `  ClusterRoleBinding/opm-operator-manager-rolebinding: carries the identity of instance team-a/web` or `: label app.kubernetes.io/name is "x", earlier manifests set "opm-operator"`, and the hint `nothing was changed; remove or rename these objects, then re-run 'opm operator install'` |
| A proof-list GET fails (RBAC, network) | 2 or 4 | `operator migration refused: cannot read Deployment/opm-operator-controller-manager in opm-operator-system: <err>`; 4 on Forbidden |
| A delete fails after writes began | 1 | `operator migration stopped after it began: deleting <obj>: <err>; re-run 'opm operator install' to complete it` |
| `WaitAbsent` times out | 1 | the existing WaitAbsent timeout message, plus the re-run hint |

### Report

```text
migrating the operator installed from an earlier release manifest
  adopted   Namespace/opm-operator-system
  adopted   ServiceAccount/opm-operator-controller-manager (opm-operator-system)
  ...
  recreated Deployment/opm-operator-controller-manager (opm-operator-system): selector changed
  deleted   ClusterRoleBinding/opm-operator-manager-rolebinding: superseded by opm-operator-manager-role
  deleted   ClusterRoleBinding/opm-operator-metrics-auth-rolebinding: superseded by opm-operator-metrics-auth-role
  deleted   RoleBinding/opm-operator-leader-election-rolebinding (opm-operator-system): superseded by opm-operator-leader-election-role
  left      ClusterRole/opm-operator-modulerelease-viewer-role: not part of the operator module
```

On a cluster with nothing to migrate, no migration lines are printed. Exact binding names come from the published module's render (section 3 checks them).

## Risks / Trade-offs

- [The operator is down from the Deployment delete until the new pod holds the leader lease] → accepted by the owner (0028:D2, D8). The report says so; the managed workloads keep running.
- [csaupgrade does not move the kustomize labels or the annotation] → section 1 measures it first; if it fails, design.md takes the JSON-patch option for exactly the fields the proof list names and the annotation, and the spec is unchanged.
- [A release manifest outside the table] → the table is closed by 0028:D2:R13; the test against the downloaded set catches a gap in the past releases.
- [Controller arguments a user patched onto the earlier Deployment, such as `--registry`, vanish with the recreate] → they become instance values (0028:D5); the report's recreate line says the earlier Deployment's patches are not carried over, and the migration note in the opm docs says to pass them as values on this install.
- [Another instance's object on the list is touched] → `Unproven` refuses before writes (0028:D8:R10).
- [The delivery run needs a cluster that can pull the operator image] → the local kind cluster has no egress at proposal time; the run uses `kind load docker-image` of the pinned image, or another cluster, and is recorded at archive.

## Migration Plan

No data migration in the CLI. Rollback is a CLI downgrade, which 0028 06-operational.md, Rollback, covers.

## Research & Decisions

### Where the migration's list comes from

**Context**: 0028:D8 says the list covers "every operator release that published an install manifest, which are the names D2:R10 keeps plus the three superseded bindings."
**Explored**: `dist/install.yaml` at all 47 opm-operator tags, the assets of the GitHub releases, the 10 manifests the CLI embedded.
**Options considered**:
1. Only the current manifest's 19 objects: matches D8's parenthetical, misses the 10 objects of v0.5.0 to v0.7.5 that a cluster upgraded through kubectl still holds.
2. The union over every release with a manifest asset (29): matches 0028:D8:R14's wording, and lets install report the older leftovers (0028:D8:R9).
**Decision**: Option 2.
**Rationale**: R14 is the requirement, and the extra 10 are never adopted or deleted (the module does not render them, and none is a binding), only reported. The mismatch with D8's parenthetical goes back to the enhancement as a finding.

### How proven objects pass the guard

See Decisions, "Admission through the current guard".
