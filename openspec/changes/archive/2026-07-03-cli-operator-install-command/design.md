# Design: cli-operator-install-command

## Context

Enhancement 0006 fixed the product surface in its decision log — noun-first `opm operator install [--crds-only] [--rbac]` / `opm operator uninstall [--remove-finalizers]` (D32), install/uninstall-only scope with the `opm-cli` field-manager rename (D33), uninstall safety semantics (D34), single embedded artifact + pin + fetch + readiness wait (D35). This design does not re-litigate those; it decides how they map onto the CLI codebase.

Current state that shapes the design:

- `internal/kubernetes` has the client (`client.go`, dynamic + clientset, cached), SSA apply (`apply.go`, `ApplyPatchType` + `Force: true`, field manager from the `fieldManagerName` constant in `labels.go`), delete (`delete.go`, already sorts descending by weight), and per-resource health evaluation (`health.go`).
- `pkg/resourceorder` weights already encode the install order this slice needs: CRD (−100) → Namespace (0) → ClusterRole/Binding (5) → SA/Role/Binding (10) → Service (50) → Deployment (100).
- Nothing in the CLI parses multi-document YAML (all manifests arrive via CUE render) and nothing polls a cluster for readiness (`k8s.io/apimachinery/pkg/util/wait` is only a transitive dep).
- opm-operator's `dist/install.yaml` (16 docs: 3 CRDs, 1 Namespace, RBAC set, Deployment, Service) is uploaded as a GitHub release asset by its `release.yml`. Tag `v1.0.0-alpha.2` contains 0006 slices A1 (dependency line) and A4 (`spec.owner` CRD).
- No existing spec pins the current `opm` field-manager string, so the rename is spec-clean; the new `operator-lifecycle` spec owns the `opm-cli` requirement.

## Goals / Non-Goals

**Goals:**

- `opm operator install/uninstall` per D32–D35, with all cluster writes attributed to field manager `opm-cli`.
- One embedded artifact; CRD subset always derived by filtering, structurally unable to drift from the full install.
- Bounded, observable readiness waits whose machinery slice C3 (`instance handoff`) can reuse for its "operator installed and ready" precondition.
- Uninstall that cannot wedge the cluster: CRDs and Namespace preserved, finalizer refusal by default.

**Non-Goals:**

- No apply-path changes: no missing-CRD hint, no version-skew gates — C1's scope (D33).
- No operator upgrade orchestration; `install` is idempotent SSA, not a package manager (0006 README out-of-scope).
- No checksum/signature verification of fetched artifacts (deferred by D35).
- No reconciliation of `--version`-fetched vs embedded CRD schema differences — the CLI-≥-cluster contract is C1's gate (D24).

## Decisions

### 1. Two packages: thin `internal/cmd/operator/`, logic in `internal/operator/`

`internal/cmd/operator/` holds cobra wiring only (`operator.go` group, `install.go`, `uninstall.go`), matching the `internal/cmd/instance/` pattern and Constitution II ("commands orchestrate"). All behavior — embed, parse, plan, apply/delete loops, waits, finalizer guard, fetch — lives in a new `internal/operator/` package. Not `pkg/`: this is CLI-product behavior with no reuse ambition outside the CLI.

*Alternative — fold logic into `internal/kubernetes`:* rejected; that package is generic cluster primitives, and the operator-manifest lifecycle (pinning, filtering, GitHub fetch) is a distinct concern that would bloat it.

### 2. Export a single-resource SSA helper instead of reusing `kubernetes.Apply`

`kubernetes.Apply` is instance-shaped: `instanceName` parameter, `output.InstanceLogger`, instance-flavored result counters. Rather than generalize its signature (touching the instance apply path this slice must not disturb), promote the existing unexported `applyResource` to an exported single-resource primitive (e.g. `ApplyOne(ctx, client, obj, opts) (status, error)`) and have both `kubernetes.Apply` and the operator install loop call it. The install loop owns its own reporting (per-doc status lines via `output.FormatResourceLine`).

*Alternative — parameterize `Apply` with a logger/label:* rejected; widens a stable API for one caller and couples operator output style to instance output style.

### 3. Manifest handling: parse once, plan by filtering

`internal/operator` embeds `dist/install.yaml` and parses it into `[]*unstructured.Unstructured` using `k8s.io/apimachinery/pkg/util/yaml`'s multi-doc decoder (no new dependency). Three pure functions derive plans from one parsed set:

- install plan = all docs, sorted ascending by `pkg/resourceorder` weight;
- CRDs-only plan = docs where `kind == CustomResourceDefinition`;
- uninstall plan = all docs minus CRDs minus the Namespace, sorted descending (matching `delete.go`'s teardown convention).

Pure functions over parsed docs keep everything unit-testable without a cluster, and make "uninstall never touches CRDs/Namespace" a property of the plan, not of scattered `if` checks in the delete loop.

### 4. Pin + refresh: one constant next to the embed, `task operator:sync`

The pinned tag lives as a single Go constant (e.g. `internal/operator/manifest.go: PinnedOperatorVersion = "v1.0.0-alpha.2"`) beside the `go:embed` directive. `task operator:sync VERSION=<tag>` downloads `https://github.com/open-platform-model/opm-operator/releases/download/<tag>/install.yaml` into the embed location and rewrites the constant — the whole upgrade is one reviewable diff. `--version <tag>` at runtime fetches the same URL over HTTPS (10–30s timeout, clear error on 404 naming the tag) and feeds the identical parse/plan path; per D35 no checksum verification.

*Alternative — infer the pin from `go.mod` or a config file:* rejected; the CLI has no Go dependency on opm-operator (0006/D13 forbids the module edge), and a config file adds a second source of truth for what is a build-time property.

### 5. Readiness waits: new bounded poll in `internal/operator`, condition checks from `health.go` where they fit

No polling machinery exists; build a small bounded wait (poll every 2s against `--timeout`, default `5m`, context-cancellable) in `internal/operator`:

- CRDs: wait for condition `Established=True` on each applied CRD (custom check — `health.go` has no CRD case).
- Full install: CRDs `Established`, then the operator Deployment ready — reuse `kubernetes.EvaluateHealth`'s workload path against the live Deployment object.

The wait function takes a set of target objects and a per-object readiness predicate, so C3's "operator installed and ready" precondition can call the same code with the same predicates. Uninstall does **not** wait for deletion to complete (fire-and-report): nothing downstream consumes "fully gone", and waiting adds failure modes (stuck pod termination) to a command whose job is done at delete-issuance.

### 6. Finalizer guard: read via dynamic client, strip only `opmodel.dev/cleanup`

Before deleting anything, uninstall lists `moduleinstances` cluster-wide via the existing dynamic client (GVR hardcoded in `internal/operator`; the CLI deliberately has no opm-operator type imports per 0006/D13). Any instance whose `metadata.finalizers` contains `opmodel.dev/cleanup` triggers the refusal, listing each `namespace/name`. With `--remove-finalizers`, the CLI removes exactly that finalizer via a JSON patch on each instance, prints the orphaning consequence, then proceeds. RBAC failure on the list (user can delete Deployments but not list moduleinstances) fails closed with the standard `cmdutil.ExitCodeFromK8sError` mapping — uninstall does not proceed blind.

*Alternative — strategic-merge or SSA for finalizer removal:* rejected; finalizers is a shared list not owned by `opm-cli`, a targeted JSON patch (test-and-remove) is the precise, minimal write.

### 7. `--rbac` is valid with and without `--crds-only`

D23/D32 word `--rbac` against the CRDs-only path (its motivating user), but a full-install cluster whose CLI drivers are non-admins is coherent. The flag emits the `opm-cli-user` ClusterRole (full verbs on `moduleinstances`, get/patch/update on `moduleinstances/status`, get/list on `platforms`) and, when `--user`/`--group` is given, one ClusterRoleBinding. Objects are constructed in Go as `unstructured` (not embedded YAML templates — they take runtime parameters). `--user`/`--group` without `--rbac` is a flag-validation error (fail-early, Constitution I).

### 8. Field-manager rename is a one-constant change, sequenced first

`internal/kubernetes/labels.go: fieldManagerName = "opm"` → `"opm-cli"`. The only SSA write site is `apply.go`; ownership of previously-applied resources transfers on next apply through the existing `Force: true`. Landed as the first task so every new write this slice adds is born under the final manager name. No transition handling: no external users (0006/D14).

## Risks / Trade-offs

- **[E2E needs the operator image]** The kind-cluster e2e's rollout wait requires pulling the pinned operator image from GHCR (or preloading it). → Verify pull access in the e2e environment first; if unavailable, e2e asserts through CRD `Established` + Deployment *created*, and rollout-ready is covered by a documented manual/integration path.
- **[Fetched `--version` artifact skew]** A newer fetched `install.yaml` may carry CRDs/fields this CLI predates; nothing gates that here. → Accepted for this slice; D24's CLI-≥-cluster gate lands in C1. `--version` output names the tag it installed.
- **[Finalizer strip races the operator]** If the operator is still running when `--remove-finalizers` strips, a concurrent reconcile may re-add finalizers. → Acceptable: uninstall deletes the Deployment immediately after; a re-added finalizer just re-triggers the refusal on a rerun, which is safe and visible.
- **[Field ownership transfer noise]** After the rename, the first re-apply of pre-existing releases shows all fields migrating `opm` → `opm-cli` in managedFields. → Cosmetic; no behavior change (`Force: true` preexists). Noted in the task so reviewers aren't surprised by e2e managedFields diffs.
- **[GitHub fetch is a new failure surface]** Air-gapped or rate-limited environments can't use `--version`. → The embedded default needs no network; the error message says to use the embedded version or `task operator:sync` from a connected machine.

## Migration Plan

Forward-only, additive. No data migration; no existing command changes flags or output. Rollback = revert the commits — the field-manager rename reverts symmetrically (ownership transfers back on next apply the same way).

## Open Questions

None blocking. Flag defaults chosen here (poll 2s, `--timeout 5m`, fetch timeout 30s) are implementation-tunable without spec impact.
