# Proposal: cli-operator-install-command

> Slice B2 of enhancement `0006` (CLI CR Inventory, Library Kernel Adoption, and Operator Handoff). Implements decisions D5, D23 (install-side half), D32, D33, D34, D35.

## Why

Enhancement 0006 moves the CLI's release inventory into the operator's `ModuleInstance` CR (slice C1), which makes the CRDs a hard prerequisite for every CLI apply — and handoff (slice C3) requires a running operator. Today the CLI has no way to put either onto a cluster; users must clone `opm-operator` or hand-fetch manifests. This slice gives the CLI an explicit, self-contained operator lifecycle surface so the C1/C3 slices land onto clusters the CLI itself can prepare.

## What Changes

- New `opm operator` command group (noun-first per 0006/D32):
  - `opm operator install` — SSA-applies the full embedded operator manifest (`dist/install.yaml` of a pinned opm-operator release) and waits, bounded, for completion (CRDs `Established`, operator Deployment rollout).
  - `opm operator install --crds-only` — applies only the `CustomResourceDefinition` documents filtered from the same artifact (the CLI-solo path); waits for `Established`.
  - `opm operator install --rbac [--user U | --group G]` — additionally emits an `opm-cli-user` ClusterRole (verbs on `moduleinstances`, `moduleinstances/status`, read on `platforms`), binding it when `--user`/`--group` is given (0006/D23). Off by default.
  - `opm operator install --version <tag>` — fetches `install.yaml` from the corresponding opm-operator GitHub release instead of using the embedded copy.
  - `opm operator uninstall` — deletes what install applied **except** CRDs and the Namespace (0006/D34). Refuses (naming the instances) while any `ModuleInstance` carries the operator's `opmodel.dev/cleanup` finalizer; `--remove-finalizers` strips that finalizer (only that one) and proceeds, stating the orphaning consequence.
- One embedded artifact: the pinned release's `dist/install.yaml` via `go:embed`; CRD subset derived by filtering, never a second embedded copy (0006/D35). Pin recorded in one constant; new `task operator:sync VERSION=<tag>` refreshes artifact + pin.
- **BREAKING (internal only):** the CLI's SSA field manager renames `opm` → `opm-cli` (0006/D33, D10 naming). Field ownership of previously-applied resources transfers on next apply via the existing `Force: true`; no user-visible flag or output changes. The CLI has no external users (0006/D14), so no compatibility shim.
- Explicitly **not** in this slice (0006/D33): no apply-path changes — the missing-CRD hint and version-skew gates land with the CR-inventory slice (C1).

SemVer: **MINOR** (new command group and flags with defaults; no existing command's flags or output change).

## Capabilities

### New Capabilities

- `operator-lifecycle`: the `opm operator install`/`uninstall` command surface — embedded pinned manifest, CRD-subset filtering, version fetch fallback, RBAC emission, bounded readiness waits, uninstall safety (CRD/Namespace preservation, finalizer refusal + `--remove-finalizers`), and the `opm-cli` field manager for all CLI server-side-apply writes.

### Modified Capabilities

- `cmd-structure`: the command-package organization requirement gains the `internal/cmd/operator/` package (new top-level command group joining `module`, `instance`, `config`).

## Impact

- **New packages:** `internal/cmd/operator/` (thin cobra commands), `internal/operator/` (embed, manifest parse/filter/plan, readiness wait, finalizer guard, release fetch).
- **Modified:** `internal/kubernetes/labels.go` (field-manager constant rename; the one SSA write site is `internal/kubernetes/apply.go`), `internal/cmd/root.go` (group wiring), `Taskfile.yml` (`operator:sync`), `README.md` (command groups).
- **Reused, unchanged behavior:** `internal/kubernetes` client/apply/delete/health primitives, `pkg/resourceorder` weights (already order CRD → Namespace → RBAC → Deployment; teardown already sorts descending), `cmdutil` k8s client + exit-code mapping, `internal/output` status lines.
- **Dependencies:** no new Go modules expected — multi-doc YAML decoding and polling come from `k8s.io/apimachinery` (already direct/transitive). First outbound HTTPS call in the k8s path (`--version` fetch to GitHub releases; no checksum verification at this stage per 0006/D35).
- **Coordination:** embeds opm-operator `v1.0.0-alpha.2` (first pin — already contains 0006 slices A1 and A4). Unblocks slice C1 (`cli-cr-inventory-backend`), which also consumes new slice A6 (`Platform.status.operatorVersion`).
