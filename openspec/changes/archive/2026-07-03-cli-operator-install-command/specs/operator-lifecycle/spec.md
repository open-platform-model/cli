# operator-lifecycle (delta)

Operator lifecycle surface: `opm operator install` / `opm operator uninstall`, the embedded pinned operator manifest, readiness waits, uninstall safety, and the CLI's server-side-apply field-manager identity. Slice B2 of enhancement 0006 (D5, D23, D32–D35).

## ADDED Requirements

### Requirement: Full operator install from the embedded manifest

`opm operator install` SHALL server-side-apply every document of the embedded operator manifest (`dist/install.yaml` of the pinned opm-operator release) with field manager `opm-cli`, ordered ascending by the resource weights of `pkg/resourceorder` (CRDs before Namespace before RBAC before Deployment). The command SHALL then wait, bounded by `--timeout` (default 5m), for every applied CRD to reach the `Established=True` condition and for the operator Deployment to complete its rollout, and SHALL exit non-zero with an actionable error if the timeout elapses first.

#### Scenario: Install onto an empty cluster

- **WHEN** `opm operator install` is run against a cluster with no OPM components
- **THEN** all manifest documents are applied via SSA with field manager `opm-cli`
- **AND** the command reports a per-resource status line (created/configured/unchanged) for each document
- **AND** the command waits until the CRDs are `Established` and the operator Deployment rollout completes, then exits zero

#### Scenario: Install is idempotent

- **WHEN** `opm operator install` is run a second time with no cluster-side changes in between
- **THEN** every document reports `unchanged` and the command exits zero

#### Scenario: Readiness timeout

- **WHEN** the operator Deployment cannot become ready within `--timeout`
- **THEN** the command exits non-zero with an error naming the resource still unready and the elapsed timeout

### Requirement: CRDs-only install via `--crds-only`

`opm operator install --crds-only` SHALL apply only the `CustomResourceDefinition` documents filtered from the same embedded manifest, and SHALL wait only for the `Established=True` condition on those CRDs. No Namespace, RBAC, Deployment, or Service objects are created.

#### Scenario: Solo-cluster CRD install

- **WHEN** `opm operator install --crds-only` is run against an empty cluster
- **THEN** exactly the manifest's `CustomResourceDefinition` documents are applied
- **AND** the command waits for each to report `Established=True` and exits zero
- **AND** no other kinds from the manifest exist on the cluster afterwards

### Requirement: Single embedded artifact with a pinned version

The CLI SHALL embed exactly one operator artifact — the pinned opm-operator release's `dist/install.yaml` — and SHALL derive every install/uninstall plan (full set, CRD subset, uninstall set) by filtering that one parsed artifact. The pinned release tag SHALL be recorded in a single Go constant adjacent to the embed directive, and a `task operator:sync VERSION=<tag>` task SHALL refresh both the embedded file and the constant from the corresponding opm-operator GitHub release asset.

#### Scenario: CRD subset cannot drift from the full install

- **WHEN** the source tree is inspected
- **THEN** `dist/install.yaml` is the only embedded operator manifest (no separately embedded CRD files)
- **AND** the CRDs-only plan is produced by filtering the parsed manifest for `kind: CustomResourceDefinition`

#### Scenario: Pin refresh is a single reviewable diff

- **WHEN** `task operator:sync VERSION=v1.0.0-alpha.3` is run
- **THEN** the embedded `install.yaml` is replaced with that release's asset and the pin constant is rewritten to `v1.0.0-alpha.3`, with no other source changes

### Requirement: `--version` fetches the release asset instead of the embed

`opm operator install --version <tag>` SHALL fetch `install.yaml` from the opm-operator GitHub release `<tag>` over HTTPS and feed it through the same parse/plan/apply path as the embedded artifact. Fetch failures SHALL produce a clear error naming the tag and URL; no checksum or signature verification is performed at this stage (0006/D35). The command output SHALL name which version was installed and whether it came from the embed or a fetch.

#### Scenario: Fetching a valid tag

- **WHEN** `opm operator install --version v1.0.0-alpha.1` is run with network access
- **THEN** the manifest is downloaded from that release's `install.yaml` asset and applied
- **AND** the output states the installed version and its fetched origin

#### Scenario: Fetching a missing tag

- **WHEN** `opm operator install --version v9.9.9` is run and the release or asset does not exist
- **THEN** the command exits non-zero with an error naming the tag and the attempted URL, and applies nothing

### Requirement: Opt-in RBAC emission via `--rbac`

`opm operator install --rbac` SHALL additionally create a ClusterRole `opm-cli-user` granting full verbs on `moduleinstances`, `get/patch/update` on `moduleinstances/status`, and `get/list` on `platforms`. When `--user <U>` or `--group <G>` is supplied alongside `--rbac`, the command SHALL also create a ClusterRoleBinding binding that subject to the role. Without `--rbac`, no RBAC objects beyond the manifest's own are created. `--user`/`--group` without `--rbac` SHALL be rejected as a flag-validation error before any cluster interaction.

#### Scenario: RBAC off by default

- **WHEN** `opm operator install --crds-only` is run
- **THEN** no `opm-cli-user` ClusterRole or ClusterRoleBinding is created

#### Scenario: Role plus binding for a user

- **WHEN** `opm operator install --crds-only --rbac --user alice` is run
- **THEN** the `opm-cli-user` ClusterRole and a ClusterRoleBinding for user `alice` are applied with field manager `opm-cli`

#### Scenario: Subject flags require --rbac

- **WHEN** `opm operator install --user alice` is run without `--rbac`
- **THEN** the command fails flag validation with an error stating `--user`/`--group` require `--rbac`, before contacting the cluster

### Requirement: Uninstall preserves CRDs and the Namespace

`opm operator uninstall` SHALL delete the embedded manifest's documents except every `CustomResourceDefinition` and the `Namespace`, in descending resource-weight order. CRD removal remains a deliberate manual `kubectl delete crd`. The command SHALL NOT wait for deletion to complete (fire-and-report).

#### Scenario: Uninstall after a full install

- **WHEN** `opm operator uninstall` is run on a cluster with no `ModuleInstance` resources
- **THEN** the operator Deployment, Service, RBAC objects, and ServiceAccount are deleted
- **AND** the three OPM CRDs and the operator Namespace still exist afterwards

### Requirement: Uninstall refuses while operator cleanup finalizers are armed

Before deleting anything, `opm operator uninstall` SHALL list `ModuleInstance` resources cluster-wide; if any carries the `opmodel.dev/cleanup` finalizer, the command SHALL refuse (exit non-zero) and name each such instance as `namespace/name`. With `--remove-finalizers`, the command SHALL remove exactly the `opmodel.dev/cleanup` finalizer (leaving all other finalizers intact) from every such instance, state that the instances and their workloads are now orphaned, and then proceed. If the cluster-wide list fails (including RBAC denial), the command SHALL fail closed without deleting anything.

#### Scenario: Refusal names the armed instances

- **WHEN** `opm operator uninstall` is run while `default/jellyfin` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command exits non-zero without deleting anything
- **AND** the error names `default/jellyfin` and points at `--remove-finalizers` and its orphaning consequence

#### Scenario: Override strips only the operator's finalizer

- **WHEN** `opm operator uninstall --remove-finalizers` is run while an instance carries both `opmodel.dev/cleanup` and a foreign finalizer
- **THEN** `opmodel.dev/cleanup` is removed from the instance, the foreign finalizer remains
- **AND** the command states the orphaning consequence and proceeds with the uninstall

#### Scenario: List failure fails closed

- **WHEN** the uninstalling user lacks permission to list `moduleinstances` cluster-wide
- **THEN** the command exits non-zero without deleting anything, mapping the RBAC error to the standard permission-denied exit code

### Requirement: All CLI server-side-apply writes use field manager `opm-cli`

Every server-side-apply write the CLI performs — instance/module applies and operator installs alike — SHALL use the field manager `opm-cli`. The former manager name `opm` is retired; ownership of resources previously applied under `opm` transfers on their next apply via the existing forced-conflicts behavior, with no user-visible change.

#### Scenario: One manager identity across the CLI

- **WHEN** any resource is applied by any CLI command after this change
- **THEN** its `managedFields` attribute the CLI's fields to manager `opm-cli`
- **AND** no CLI code path applies with manager `opm`
